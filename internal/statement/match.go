package statement

import (
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

const amountEpsilon = 0.009

// BuildPreview matches parsed statement lines against the existing ledger.
func BuildPreview(
	parsed models.ParsedStatement,
	existing []models.Transaction,
	categories []models.Category,
	walletID *int,
	memberID *int,
) models.StatementPreview {
	used := make(map[int]bool)
	items := make([]models.StatementPreviewItem, 0, len(parsed.Items))
	newCount := 0
	matchedCount := 0
	skippedCount := 0

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
		} else if item.Kind == "expense" || item.Kind == "income" || (parsed.StatementType == "credit_card" && item.Kind == "refund") {
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
		if !similarDesc(itemDesc, NormalizeDesc(tx.Description)) {
			continue
		}
		used[tx.ID] = true
		return tx.ID, true
	}
	return 0, false
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

func similarDesc(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	aTokens := strings.Fields(a)
	bTokens := strings.Fields(b)
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
