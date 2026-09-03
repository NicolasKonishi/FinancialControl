package models

import (
	"math"
	"time"
)

// CardInvoice is one credit-card cycle, identified by its due month.
type CardInvoice struct {
	WalletID             int        `json:"wallet_id"`
	Year                 int        `json:"year"`
	Month                int        `json:"month"`
	ClosingDate          string     `json:"closing_date"`
	DueDate              string     `json:"due_date"`
	Amount               float64    `json:"amount"`
	PaidAmount           float64    `json:"paid_amount"`
	Outstanding          float64    `json:"outstanding"`
	Paid                 bool       `json:"paid"`
	Source               string     `json:"source"`
	StatementPeriodStart *string    `json:"statement_period_start,omitempty"`
	StatementPeriodEnd   *string    `json:"statement_period_end,omitempty"`
	StatementBalance     *float64   `json:"statement_balance,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

// CardCycle contains the closing date and following payment due date.
type CardCycle struct {
	Year        int
	Month       int
	ClosingDate time.Time
	DueDate     time.Time
}

// CardCycleForPurchase assigns a purchase to the invoice that closes next.
// The closing day starts the next cycle: a purchase on day 14 belongs to the
// following invoice when the card closes on day 14.
func CardCycleForPurchase(wallet Wallet, purchase time.Time) CardCycle {
	closingDay := cardDay(wallet.ClosingDay, 1)
	dueDay := cardDay(wallet.DueDay, closingDay)
	closing := cardDate(purchase.Year(), int(purchase.Month()), closingDay)
	if purchase.Day() >= closingDay {
		closing = cardDate(purchase.Year(), int(purchase.Month())+1, closingDay)
	}
	dueYear, dueMonth := closing.Year(), int(closing.Month())
	if dueDay <= closingDay {
		dueMonth++
	}
	due := cardDate(dueYear, dueMonth, dueDay)
	return CardCycle{Year: due.Year(), Month: int(due.Month()), ClosingDate: closing, DueDate: due}
}

// CardCycleForDueMonth returns the cycle whose bill is due in year/month.
func CardCycleForDueMonth(wallet Wallet, year, month int) CardCycle {
	closingDay := cardDay(wallet.ClosingDay, 1)
	dueDay := cardDay(wallet.DueDay, closingDay)
	due := cardDate(year, month, dueDay)
	closingYear, closingMonth := year, month
	if dueDay <= closingDay {
		closingMonth--
	}
	closing := cardDate(closingYear, closingMonth, closingDay)
	return CardCycle{Year: due.Year(), Month: int(due.Month()), ClosingDate: closing, DueDate: due}
}

// FinalizeCardInvoice fills calculated fields before returning an invoice.
func FinalizeCardInvoice(invoice CardInvoice) CardInvoice {
	invoice.Amount = math.Round(invoice.Amount*100) / 100
	invoice.PaidAmount = math.Round(invoice.PaidAmount*100) / 100
	invoice.Outstanding = math.Round((invoice.Amount-invoice.PaidAmount)*100) / 100
	if invoice.Outstanding < 0 {
		invoice.Outstanding = 0
	}
	invoice.Paid = invoice.Amount > 0 && invoice.Outstanding <= 0
	if invoice.Source == "" {
		invoice.Source = "calculated"
	}
	return invoice
}

func cardDay(value *int, fallback int) int {
	if value == nil || *value < 1 || *value > 31 {
		return fallback
	}
	return *value
}

func cardDate(year, month, day int) time.Time {
	base := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > last {
		day = last
	}
	return time.Date(base.Year(), base.Month(), day, 0, 0, 0, 0, time.UTC)
}
