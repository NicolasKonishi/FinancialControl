package models

import "testing"

func TestOccurrencesMonthly(t *testing.T) {
	bill := Bill{
		Amount:     100,
		DueDay:     10,
		Frequency:  BillFrequencyMonthly,
		Recurrence: BillRecurrenceOngoing,
		StartMonth: "2026-01",
	}
	if got := bill.OccurrencesInMonth(2026, 8); got != 1 {
		t.Fatalf("monthly occ = %d, want 1", got)
	}
	if got := bill.ChargeForMonth(2026, 8); got != 100 {
		t.Fatalf("monthly charge = %v, want 100", got)
	}
	if bill.IsActiveInMonth(2025, 12) {
		t.Fatal("expected inactive before start")
	}
}

func TestOccurrencesWeekly(t *testing.T) {
	// Anchor 2026-08-05 (Wed). August 2026 has Wednesdays: 5,12,19,26 = 4
	bill := Bill{
		Amount:     50,
		DueDay:     5,
		Frequency:  BillFrequencyWeekly,
		Recurrence: BillRecurrenceOngoing,
		StartMonth: "2026-08",
	}
	if got := bill.OccurrencesInMonth(2026, 8); got != 4 {
		t.Fatalf("weekly occ = %d, want 4", got)
	}
	if got := bill.ChargeForMonth(2026, 8); got != 200 {
		t.Fatalf("weekly charge = %v, want 200", got)
	}
}

func TestOccurrencesBiweekly(t *testing.T) {
	bill := Bill{
		Amount:     80,
		DueDay:     1,
		Frequency:  BillFrequencyBiweekly,
		Recurrence: BillRecurrenceOngoing,
		StartMonth: "2026-08",
	}
	// Aug 1, 15, 29 = 3
	if got := bill.OccurrencesInMonth(2026, 8); got != 3 {
		t.Fatalf("biweekly occ = %d, want 3", got)
	}
}

func TestOccurrencesDailyUntil(t *testing.T) {
	end := "2026-08"
	bill := Bill{
		Amount:     10,
		DueDay:     28,
		Frequency:  BillFrequencyDaily,
		Recurrence: BillRecurrenceUntil,
		StartMonth: "2026-08",
		EndMonth:   &end,
	}
	// Aug 28..31 = 4 days
	if got := bill.OccurrencesInMonth(2026, 8); got != 4 {
		t.Fatalf("daily occ = %d, want 4", got)
	}
	if bill.IsActiveInMonth(2026, 9) {
		t.Fatal("expected inactive after end_month")
	}
}

func TestBillInterestCharge(t *testing.T) {
	bill := Bill{
		AmountMode:   BillAmountModeInterest,
		Amount:       316.86,
		InterestRate: 0.27,
		StartMonth:   "2026-11",
		DueDay:       20,
		Frequency:    BillFrequencyMonthly,
		Recurrence:   BillRecurrenceOngoing,
	}
	if got := bill.ChargeForMonth(2026, 11); got != 316.86 {
		t.Fatalf("month 0 charge = %v, want 316.86", got)
	}
	got := bill.ChargeForMonth(2026, 12)
	if got <= 316.86 {
		t.Fatalf("month 1 should grow, got %v", got)
	}
	if bill.IsActiveInMonth(2026, 10) {
		t.Fatal("expected inactive before start")
	}
}

func TestOccurrencesWeekdaysAndYearly(t *testing.T) {
	bill := Bill{
		Amount:     10,
		DueDay:     24,
		Frequency:  BillFrequencyWeekdays,
		Recurrence: BillRecurrenceOngoing,
		StartMonth: "2026-08",
	}
	// Aug 24–31 weekdays: 24–28 + 31 = 6
	if got := bill.OccurrencesInMonth(2026, 8); got != 6 {
		t.Fatalf("weekdays occ = %d, want 6", got)
	}

	yearly := Bill{
		Amount:     200,
		DueDay:     24,
		Frequency:  BillFrequencyYearly,
		Recurrence: BillRecurrenceOngoing,
		StartMonth: "2026-08",
	}
	if got := yearly.OccurrencesInMonth(2026, 8); got != 1 {
		t.Fatalf("yearly Aug occ = %d, want 1", got)
	}
	if yearly.IsActiveInMonth(2026, 9) {
		t.Fatal("yearly should be inactive in September")
	}
}
