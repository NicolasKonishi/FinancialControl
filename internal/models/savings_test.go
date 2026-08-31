package models

import "testing"

func TestCDIDailyToAnnual(t *testing.T) {
	annual := CDIDailyToAnnual(0.054267)
	if annual < 14 || annual > 16 {
		t.Fatalf("CDIDailyToAnnual(0.054267) = %v, want ~14-16", annual)
	}
	if got := CDIDailyToAnnual(14.15); got != 14.15 {
		t.Fatalf("already annual = %v", got)
	}
}

func TestMonthsInclusive(t *testing.T) {
	n, err := MonthsInclusive(2026, 8, "2027-07")
	if err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("months = %d, want 12", n)
	}
	if _, err := MonthsInclusive(2026, 8, "2026-07"); err == nil {
		t.Fatal("expected error for past end_month")
	}
}

func TestMonthlyContribution(t *testing.T) {
	plain := MonthlyContribution(1200, 12, 0)
	if plain != 100 {
		t.Fatalf("zero rate = %v, want 100", plain)
	}
	withYield := MonthlyContribution(8000, 12, MonthlyRateFromCDIAnnual(14.15))
	if withYield >= 8000.0/12 || withYield < 500 {
		t.Fatalf("yield monthly = %v, want less than 666.67 and more than 500", withYield)
	}
}

func TestWalletCanFundGoal(t *testing.T) {
	member := 2
	other := 9
	checking := Wallet{ID: 1, Kind: WalletChecking, MemberID: &member}
	benefit := Wallet{ID: 2, Kind: WalletBenefit, MemberID: &member}
	joint := Wallet{ID: 3, Kind: WalletChecking, MemberID: nil}
	outsider := Wallet{ID: 4, Kind: WalletChecking, MemberID: &other}

	if !WalletCanFundGoal(checking, []int{1, 2}) {
		t.Fatal("checking of a goal member should fund")
	}
	if WalletCanFundGoal(benefit, []int{2}) {
		t.Fatal("benefit wallet should not fund a savings box")
	}
	if WalletCanFundGoal(joint, []int{2}) {
		t.Fatal("joint wallet should not fund a personal savings box")
	}
	if WalletCanFundGoal(outsider, []int{2}) {
		t.Fatal("checking of someone outside the box should not fund")
	}
}

func TestProjectedBalance(t *testing.T) {
	plain := ProjectedBalance(1000, 100, 0, 2)
	if plain != 1200 {
		t.Fatalf("zero yield = %v, want 1200", plain)
	}
	grown := ProjectedBalance(1000, 0, 14.15, 12)
	if grown <= 1000 {
		t.Fatalf("CDI projection = %v, want more than 1000", grown)
	}
}

func TestSavingsGoalMonthlyPlan(t *testing.T) {
	goal := SavingsGoal{TargetAmount: 1000, MonthlyAmount: 200, SavedAmount: 0, EndKind: SavingsEndAmount}
	if got := goal.MonthlyPlan(2026, 8); got != 200 {
		t.Fatalf("MonthlyPlan() = %v, want 200", got)
	}

	goal.SavedAmount = 900
	if got := goal.MonthlyPlan(2026, 8); got != 100 {
		t.Fatalf("MonthlyPlan() near target = %v, want 100", got)
	}

	goal.SavedAmount = 1000
	if got := goal.MonthlyPlan(2026, 8); got != 0 {
		t.Fatalf("MonthlyPlan() complete = %v, want 0", got)
	}

	end := "2026-06"
	goal = SavingsGoal{TargetAmount: 1000, MonthlyAmount: 200, EndKind: SavingsEndDate, EndMonth: &end}
	if got := goal.MonthlyPlan(2026, 8); got != 0 {
		t.Fatalf("MonthlyPlan() after end = %v, want 0", got)
	}

	goal = SavingsGoal{EndKind: SavingsEndNone, MonthlyAmount: 200, TargetAmount: 0}
	if got := goal.MonthlyPlan(2026, 8); got != 0 {
		t.Fatalf("MonthlyPlan() none = %v, want 0", got)
	}
}
