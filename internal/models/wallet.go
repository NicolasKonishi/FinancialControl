package models

import (
	"math"
	"time"
)

const (
	WalletChecking = "checking"
	WalletSavings  = "savings"
	WalletBenefit  = "benefit"
	WalletCompany  = "company"
	WalletCredit   = "credit"
)

// Wallet is money parked in an account, box, benefit, or credit card.
type Wallet struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Kind           string    `json:"kind"`
	MemberID       *int      `json:"member_id"`
	Balance        float64   `json:"balance"`
	ClosingDay     *int      `json:"closing_day"`
	DueDay         *int      `json:"due_day"`
	CreditLimit    float64   `json:"credit_limit"`
	InvoiceBalance float64   `json:"invoice_balance"`
	CreatedAt      time.Time `json:"created_at"`
}

// CreateWalletInput is the JSON body for POST /wallets.
type CreateWalletInput struct {
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	MemberID       *int    `json:"member_id"`
	Balance        float64 `json:"balance"`
	ClosingDay     *int    `json:"closing_day"`
	DueDay         *int    `json:"due_day"`
	CreditLimit    float64 `json:"credit_limit"`
	InvoiceBalance float64 `json:"invoice_balance"`
}

// UpdateWalletInput is the JSON body for PUT /wallets/{id}.
type UpdateWalletInput struct {
	Name           string  `json:"name"`
	Kind           string  `json:"kind"`
	MemberID       *int    `json:"member_id"`
	Balance        float64 `json:"balance"`
	ClosingDay     *int    `json:"closing_day"`
	DueDay         *int    `json:"due_day"`
	CreditLimit    float64 `json:"credit_limit"`
	InvoiceBalance float64 `json:"invoice_balance"`
}

// PayInvoiceInput is the JSON body for POST /wallets/{id}/pay-invoice.
type PayInvoiceInput struct {
	Amount       float64 `json:"amount"`
	FromWalletID int     `json:"from_wallet_id"`
	Year         int     `json:"year"`
	Month        int     `json:"month"`
}

// ValidWalletKind reports whether kind is supported.
func ValidWalletKind(value string) bool {
	switch value {
	case WalletChecking, WalletSavings, WalletBenefit, WalletCompany, WalletCredit:
		return true
	default:
		return false
	}
}

// NormalizeWalletKind defaults empty to checking.
func NormalizeWalletKind(value string) string {
	if value == "" {
		return WalletChecking
	}
	return value
}

// IsCredit reports whether the wallet is a credit card.
func IsCredit(kind string) bool {
	return NormalizeWalletKind(kind) == WalletCredit
}

// IsCompanyWallet reports whether the wallet is a company account.
func IsCompanyWallet(kind string) bool {
	return NormalizeWalletKind(kind) == WalletCompany
}

// ValidCardDay reports whether day is a calendar day of month.
func ValidCardDay(value *int) bool {
	if value == nil {
		return true
	}
	return *value >= 1 && *value <= 31
}

// AvailableCredit is limit minus the current invoice.
func AvailableCredit(wallet Wallet) float64 {
	return math.Round((wallet.CreditLimit-wallet.InvoiceBalance)*100) / 100
}

// ApplyWalletDelta moves cash or invoice according to wallet kind.
// Expense deltas are negative: cash falls, credit invoice rises.
func ApplyWalletDelta(wallet Wallet, delta float64) Wallet {
	if IsCredit(wallet.Kind) {
		next := math.Round((wallet.InvoiceBalance-delta)*100) / 100
		if next < 0 {
			next = 0
		}
		wallet.InvoiceBalance = next
		return wallet
	}
	wallet.Balance = math.Round((wallet.Balance+delta)*100) / 100
	return wallet
}

// WalletCanFundGoal reports whether cash can move between this wallet and a savings box.
func WalletCanFundGoal(wallet Wallet, memberIDs []int) bool {
	if NormalizeWalletKind(wallet.Kind) != WalletChecking {
		return false
	}
	if wallet.MemberID == nil {
		return false
	}
	for _, id := range memberIDs {
		if id == *wallet.MemberID {
			return true
		}
	}
	return false
}

// PreferredWallet picks the account to debit when paying a bill.
func PreferredWallet(wallets []Wallet) (Wallet, bool) {
	if len(wallets) == 0 {
		return Wallet{}, false
	}
	best := wallets[0]
	bestRank := walletPayRank(best.Kind)
	for _, wallet := range wallets[1:] {
		rank := walletPayRank(wallet.Kind)
		if rank < bestRank {
			best = wallet
			bestRank = rank
		}
	}
	return best, true
}

func walletPayRank(kind string) int {
	switch NormalizeWalletKind(kind) {
	case WalletChecking:
		return 0
	case WalletCompany:
		return 1
	case WalletCredit:
		return 2
	case WalletSavings:
		return 3
	default:
		return 4
	}
}
