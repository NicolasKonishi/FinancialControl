package statement

import (
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

const amountEpsilon = 0.009

// BuildPreview matches parsed statement lines against the existing ledger and planned bills.
func BuildPreview(
	parsed models.ParsedStatement,
	existing []models.Transaction,
	categories []models.Category,
	walletID *int,
	memberID *int,
	bills []models.Bill,
) models.StatementPreview {
	used := make(map[int]bool)
	usedBills := make(map[int]bool)
	items := make([]models.StatementPreviewItem, 0, len(parsed.Items))
	newCount := 0
	matchedCount := 0
	skippedCount := 0
	creditCard := parsed.StatementType == "credit_card"

	for i, raw := range parsed.Items {
		kind := normalizeKind(raw.Kind)
		icon := raw.SuggestedIcon
		if kind == "income" && (icon == "" || icon == "other") {
			icon = "salary"
		}
		item := models.StatementPreviewItem{
			Index:         i,
			Date:          raw.Date,
			Description:   strings.TrimSpace(raw.Description),
			Amount:        math.Round(raw.Amount*100) / 100,
			Kind:          kind,
			CategoryID:    CategoryForIcon(categories, icon),
			SuggestedIcon: icon,
		}
		if item.Kind == "" {
			item.Kind = "expense"
		}

		if matchID, ok := findMatch(item, existing, used, walletID); ok {
			item.AlreadyRecorded = true
			item.MatchedTransactionID = &matchID
			item.Selected = false
			matchedCount++
		} else if billID, ok := findMatchingBill(item, bills, walletID, creditCard, usedBills); ok {
			item.MatchedBillID = &billID
			if creditCard {
				// Credit lines still need to land as transactions so the invoice
				// can confirm the planned bill instead of creating a second one.
				item.Selected = true
				newCount++
			} else {
				item.AlreadyRecorded = true
				item.Selected = false
				matchedCount++
			}
		} else if item.Kind == "expense" || item.Kind == "income" || (creditCard && item.Kind == "refund") {
			item.Selected = true
			newCount++
		} else {
			item.Selected = false
			skippedCount++
		}
		items = append(items, item)
	}

	return models.StatementPreview{
		Issuer:        parsed.Issuer,
		StatementType: parsed.StatementType,
		PeriodStart:   parsed.PeriodStart,
		PeriodEnd:     parsed.PeriodEnd,
		Balance:       parsed.Balance,
		WalletID:      walletID,
		MemberID:      memberID,
		NewCount:      newCount,
		MatchedCount:  matchedCount,
		SkippedCount:  skippedCount,
		Items:         items,
	}
}

// CategoryForIcon picks the household category that best matches a suggested icon.
func CategoryForIcon(categories []models.Category, icon string) int {
	icon = strings.ToLower(strings.TrimSpace(icon))
	if icon != "" {
		for _, cat := range categories {
			if strings.EqualFold(cat.Icon, icon) {
				return cat.ID
			}
		}
	}
	for _, fallback := range []string{"other", "market", "food"} {
		for _, cat := range categories {
			if strings.EqualFold(cat.Icon, fallback) {
				return cat.ID
			}
		}
	}
	for _, cat := range categories {
		if cat.Icon != "salary" && cat.Icon != "freelance" {
			return cat.ID
		}
	}
	if len(categories) > 0 {
		return categories[0].ID
	}
	return 0
}

func findMatch(item models.StatementPreviewItem, existing []models.Transaction, used map[int]bool, walletID *int) (int, bool) {
	itemDate := item.Date
	itemDesc := NormalizeDesc(item.Description)
	for _, tx := range existing {
		if used[tx.ID] {
			continue
		}
		if !sameAmount(tx.Amount, item.Amount) {
			continue
		}
		if formatDate(tx.Date) != itemDate {
			continue
		}
		if !walletsCompatible(tx.WalletID, walletID) {
			continue
		}
		if !SimilarDesc(itemDesc, tx.Description) {
			continue
		}
		used[tx.ID] = true
		return tx.ID, true
	}
	return 0, false
}

func findMatchingBill(item models.StatementPreviewItem, bills []models.Bill, walletID *int, creditCard bool, used map[int]bool) (int, bool) {
	if item.Kind != "expense" && item.Kind != "refund" {
		return 0, false
	}
	year, month, ok := parseItemYearMonth(item.Date)
	if !ok {
		return 0, false
	}
	candidate := BillCandidate(item.Description, item.Amount, walletID)
	idx := MatchingBill(bills, candidate, year, month, used, !creditCard)
	if idx < 0 {
		nextYear, nextMonth := addMonth(year, month)
		idx = MatchingBill(bills, candidate, nextYear, nextMonth, used, !creditCard)
	}
	if idx < 0 {
		return 0, false
	}
	used[bills[idx].ID] = true
	return bills[idx].ID, true
}

// BillCandidate is the planned-bill shape used to match an imported charge.
func BillCandidate(description string, amount float64, walletID *int) models.Bill {
	name, current, total, _ := models.ParseInstallment(description)
	return models.Bill{
		Name:             name,
		Amount:           math.Round(amount*100) / 100,
		WalletID:         walletID,
		InstallmentStart: current,
		InstallmentTotal: total,
	}
}

// MatchingBill finds a planned bill for an imported charge. One bill is used at most once.
// allowCashBill lets checking imports match bills without a wallet (à vista).
func MatchingBill(bills []models.Bill, candidate models.Bill, year, month int, used map[int]bool, allowCashBill bool) int {
	best := -1
	bestRank := 99
	targetMonth := year*12 + month
	for i, bill := range bills {
		if used[bill.ID] || !walletsAlign(bill.WalletID, candidate.WalletID, allowCashBill) {
			continue
		}
		if !sameAmount(bill.Amount, candidate.Amount) {
			continue
		}
		if !SimilarDesc(bill.Name, candidate.Name) {
			continue
		}
		if installmentSeries(bill.InstallmentTotal) != installmentSeries(candidate.InstallmentTotal) {
			continue
		}
		if installmentSeries(candidate.InstallmentTotal) {
			if bill.InstallmentTotal != candidate.InstallmentTotal {
				continue
			}
			start, err := time.Parse("2006-01", bill.StartMonth)
			if err != nil {
				continue
			}
			position := bill.InstallmentStart + targetMonth - (start.Year()*12 + int(start.Month()))
			if position != candidate.InstallmentStart {
				continue
			}
		}
		if !bill.IsActiveInMonth(year, month) {
			continue
		}
		rank := 1
		if bill.Recurrence == models.BillRecurrenceOngoing {
			rank = 0
		}
		if rank < bestRank {
			best = i
			bestRank = rank
		}
	}
	return best
}

// ExpenseCoveredByBill reports whether a ledger expense is already represented by a planned bill.
func ExpenseCoveredByBill(tx models.Transaction, bills []models.Bill, year, month int, used map[int]bool) bool {
	if tx.Type != "" && tx.Type != models.TransactionTypeExpense {
		return false
	}
	candidate := BillCandidate(tx.Description, tx.Amount, tx.WalletID)
	idx := MatchingBill(bills, candidate, year, month, used, true)
	if idx < 0 {
		return false
	}
	used[bills[idx].ID] = true
	return true
}

func walletsAlign(billWallet, importWallet *int, allowCashBill bool) bool {
	if billWallet != nil && importWallet != nil {
		return *billWallet == *importWallet
	}
	if billWallet == nil && importWallet != nil {
		return allowCashBill
	}
	if billWallet != nil && importWallet == nil {
		return false
	}
	return true
}

func installmentSeries(total int) bool {
	return total > 1
}

func parseItemYearMonth(value string) (int, int, bool) {
	day, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return 0, 0, false
	}
	return day.Year(), int(day.Month()), true
}

func addMonth(year, month int) (int, int) {
	if month == 12 {
		return year + 1, 1
	}
	return year, month + 1
}

func walletsCompatible(existing *int, importWallet *int) bool {
	if existing == nil || *existing < 1 {
		return true
	}
	if importWallet == nil || *importWallet < 1 {
		return true
	}
	return *existing == *importWallet
}

func sameAmount(a, b float64) bool {
	return math.Abs(a-b) <= amountEpsilon
}

func formatDate(value time.Time) string {
	return value.UTC().Format("2006-01-02")
}

// SimilarDesc reports whether two merchant names refer to the same charge.
func SimilarDesc(a, b string) bool {
	left := NormalizeDesc(a)
	right := NormalizeDesc(b)
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	if strings.Contains(left, right) || strings.Contains(right, left) {
		return true
	}
	aTokens := strings.Fields(left)
	bTokens := strings.Fields(right)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return false
	}
	bSet := make(map[string]struct{}, len(bTokens))
	for _, token := range bTokens {
		bSet[token] = struct{}{}
	}
	overlap := 0
	for _, token := range aTokens {
		if _, ok := bSet[token]; ok {
			overlap++
		}
	}
	smaller := len(aTokens)
	if len(bTokens) < smaller {
		smaller = len(bTokens)
	}
	return overlap > 0 && overlap*2 >= smaller
}

// NormalizeDesc lowercases and strips punctuation so "IFOOD *IFOOD" matches "iFood".
func NormalizeDesc(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	prevSpace := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "payment":
		return "payment"
	case "refund":
		return "refund"
	case "income":
		return "income"
	case "transfer":
		return "transfer"
	default:
		return "expense"
	}
}
