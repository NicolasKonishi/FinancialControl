package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/handlers"
	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

type memoryStore struct {
	mu           sync.Mutex
	categories   map[int]models.Category
	transactions map[int]models.Transaction
	members      map[int]models.Member
	bills        map[int]models.Bill
	payments     map[string]models.BillPayment
	goals        map[int]models.SavingsGoal
	goalMonths   map[string]models.SavingsMonthAmount
	wallets      map[int]models.Wallet
	nextCatID    int
	nextTxID     int
	nextMemberID int
	nextBillID   int
	nextGoalID   int
	nextWalletID int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		categories:   make(map[int]models.Category),
		transactions: make(map[int]models.Transaction),
		members:      make(map[int]models.Member),
		bills:        make(map[int]models.Bill),
		payments:     make(map[string]models.BillPayment),
		goals:        make(map[int]models.SavingsGoal),
		goalMonths:   make(map[string]models.SavingsMonthAmount),
		wallets:      make(map[int]models.Wallet),
		nextCatID:    1,
		nextTxID:     1,
		nextMemberID: 1,
		nextBillID:   1,
		nextGoalID:   1,
		nextWalletID: 1,
	}
}

func (m *memoryStore) CreateCategory(_ context.Context, input models.CreateCategoryInput) (models.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	icon := input.Icon
	if icon == "" {
		icon = "other"
	}
	c := models.Category{
		ID:          m.nextCatID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Icon:        icon,
		CreatedAt:   time.Now().UTC(),
	}
	m.nextCatID++
	m.categories[c.ID] = c
	return c, nil
}

func (m *memoryStore) ListCategories(_ context.Context) ([]models.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Category, 0, len(m.categories))
	for _, c := range m.categories {
		out = append(out, c)
	}
	return out, nil
}

func (m *memoryStore) GetCategoryByID(_ context.Context, id int) (models.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.categories[id]
	if !ok {
		return models.Category{}, repository.ErrNotFound
	}
	return c, nil
}

func (m *memoryStore) UpdateCategory(_ context.Context, id int, input models.UpdateCategoryInput) (models.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.categories[id]
	if !ok {
		return models.Category{}, repository.ErrNotFound
	}
	c.Name = strings.TrimSpace(input.Name)
	c.Description = strings.TrimSpace(input.Description)
	c.Icon = input.Icon
	m.categories[id] = c
	return c, nil
}

func (m *memoryStore) DeleteCategory(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.categories[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.categories, id)
	return nil
}

func (m *memoryStore) CreateMember(_ context.Context, input models.CreateMemberInput) (models.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member := models.Member{
		ID:            m.nextMemberID,
		Name:          strings.TrimSpace(input.Name),
		MonthlySalary: input.MonthlySalary,
		CreatedAt:     time.Now().UTC(),
	}
	m.nextMemberID++
	m.members[member.ID] = member
	return member, nil
}

func (m *memoryStore) ListMembers(_ context.Context) ([]models.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Member, 0, len(m.members))
	for _, member := range m.members {
		out = append(out, member)
	}
	return out, nil
}

func (m *memoryStore) GetMemberByID(_ context.Context, id int) (models.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.members[id]
	if !ok {
		return models.Member{}, repository.ErrNotFound
	}
	return member, nil
}

func (m *memoryStore) UpdateMember(_ context.Context, id int, input models.UpdateMemberInput) (models.Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	member, ok := m.members[id]
	if !ok {
		return models.Member{}, repository.ErrNotFound
	}
	member.Name = strings.TrimSpace(input.Name)
	member.MonthlySalary = input.MonthlySalary
	m.members[id] = member
	return member, nil
}

func (m *memoryStore) DeleteMember(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.members[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.members, id)
	return nil
}

func (m *memoryStore) SumMonthlySalaries(_ context.Context) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var total float64
	for _, member := range m.members {
		total += member.MonthlySalary
	}
	return total, nil
}

func (m *memoryStore) CreateTransaction(_ context.Context, tx models.Transaction) (models.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.applyTxWalletLocked(nil, &tx); err != nil {
		return models.Transaction{}, err
	}
	tx.ID = m.nextTxID
	tx.CreatedAt = time.Now().UTC()
	m.nextTxID++
	m.transactions[tx.ID] = tx
	return tx, nil
}

func (m *memoryStore) ListTransactions(_ context.Context) ([]models.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Transaction, 0, len(m.transactions))
	for _, tx := range m.transactions {
		out = append(out, tx)
	}
	return out, nil
}

func (m *memoryStore) ListTransactionsByMonth(_ context.Context, year, month int) ([]models.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Transaction, 0)
	for _, tx := range m.transactions {
		y, mo, _ := tx.Date.Date()
		if y == year && int(mo) == month {
			out = append(out, tx)
		}
	}
	return out, nil
}

func (m *memoryStore) GetTransactionByID(_ context.Context, id int) (models.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, ok := m.transactions[id]
	if !ok {
		return models.Transaction{}, repository.ErrNotFound
	}
	return tx, nil
}

func (m *memoryStore) UpdateTransaction(_ context.Context, id int, tx models.Transaction) (models.Transaction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.transactions[id]
	if !ok {
		return models.Transaction{}, repository.ErrNotFound
	}
	if err := m.applyTxWalletLocked(&existing, &tx); err != nil {
		return models.Transaction{}, err
	}
	tx.ID = id
	tx.CreatedAt = existing.CreatedAt
	m.transactions[id] = tx
	return tx, nil
}

func (m *memoryStore) DeleteTransaction(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.transactions[id]
	if !ok {
		return repository.ErrNotFound
	}
	if err := m.applyTxWalletLocked(&existing, nil); err != nil {
		return err
	}
	delete(m.transactions, id)
	return nil
}

func (m *memoryStore) applyTxWalletLocked(previous, next *models.Transaction) error {
	move := func(tx *models.Transaction, sign float64) error {
		if tx == nil || tx.WalletID == nil {
			return nil
		}
		wallet, ok := m.wallets[*tx.WalletID]
		if !ok {
			return repository.ErrNotFound
		}
		if tx.MemberID != nil && wallet.MemberID != nil && *wallet.MemberID != *tx.MemberID {
			if !models.IsCompanyWallet(wallet.Kind) {
				return repository.ErrWalletOwner
			}
		}
		delta := tx.Amount
		if tx.Type != models.TransactionTypeIncome {
			delta = -tx.Amount
		}
		wallet = models.ApplyWalletDelta(wallet, sign*delta)
		m.wallets[wallet.ID] = wallet
		return nil
	}
	if err := move(previous, -1); err != nil {
		return err
	}
	return move(next, 1)
}

func (m *memoryStore) CreateBill(_ context.Context, bill models.Bill) (models.Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bill.ID = m.nextBillID
	bill.CreatedAt = time.Now().UTC()
	if bill.MemberIDs == nil {
		bill.MemberIDs = []int{}
	}
	m.nextBillID++
	m.bills[bill.ID] = bill
	return bill, nil
}

func (m *memoryStore) ListBills(_ context.Context) ([]models.Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Bill, 0, len(m.bills))
	for _, bill := range m.bills {
		out = append(out, bill)
	}
	return out, nil
}

func (m *memoryStore) ListBillsActiveInMonth(_ context.Context, year, month int) ([]models.Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Bill, 0)
	for _, bill := range m.bills {
		if bill.IsActiveInMonth(year, month) {
			out = append(out, bill)
		}
	}
	return out, nil
}

func (m *memoryStore) GetBillByID(_ context.Context, id int) (models.Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bill, ok := m.bills[id]
	if !ok {
		return models.Bill{}, repository.ErrNotFound
	}
	return bill, nil
}

func (m *memoryStore) UpdateBill(_ context.Context, id int, bill models.Bill) (models.Bill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.bills[id]
	if !ok {
		return models.Bill{}, repository.ErrNotFound
	}
	bill.ID = id
	bill.CreatedAt = existing.CreatedAt
	m.bills[id] = bill
	return bill, nil
}

func (m *memoryStore) DeleteBill(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.bills[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.bills, id)
	return nil
}

func paymentKey(billID, year, month int) string {
	return fmt.Sprintf("%d-%d-%d", billID, year, month)
}

func (m *memoryStore) ListBillPayments(_ context.Context, year, month int) ([]models.BillPayment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.BillPayment, 0)
	for _, payment := range m.payments {
		if payment.Year == year && payment.Month == month {
			out = append(out, payment)
		}
	}
	return out, nil
}

func (m *memoryStore) SetBillPaid(_ context.Context, billID, year, month int, paid bool, paidByMemberID, walletID *int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	bill, ok := m.bills[billID]
	if !ok {
		return repository.ErrNotFound
	}
	key := paymentKey(billID, year, month)
	if existing, exists := m.payments[key]; exists && existing.WalletID != nil {
		wallet, wok := m.wallets[*existing.WalletID]
		if wok {
			wallet = models.ApplyWalletDelta(wallet, existing.Amount)
			m.wallets[wallet.ID] = wallet
		}
	}
	if !paid {
		delete(m.payments, key)
		return nil
	}
	if paidByMemberID == nil || *paidByMemberID < 1 {
		return fmt.Errorf("paid_by_member_id is required")
	}
	var candidates []models.Wallet
	for _, wallet := range m.wallets {
		if wallet.MemberID != nil && *wallet.MemberID == *paidByMemberID {
			candidates = append(candidates, wallet)
		}
	}
	chosen, ok := models.PreferredWallet(candidates)
	if walletID != nil && *walletID > 0 {
		w, wok := m.wallets[*walletID]
		if !wok {
			return repository.ErrNotFound
		}
		if w.MemberID == nil || *w.MemberID != *paidByMemberID {
			if !models.IsCompanyWallet(w.Kind) {
				return repository.ErrWalletOwner
			}
		}
		chosen = w
		ok = true
	}
	if !ok {
		return repository.ErrNoWallet
	}
	amount := bill.ChargeForMonth(year, month)
	chosen = models.ApplyWalletDelta(chosen, -amount)
	m.wallets[chosen.ID] = chosen
	wid := chosen.ID
	m.payments[key] = models.BillPayment{
		BillID:         billID,
		Year:           year,
		Month:          month,
		PaidAt:         time.Now().UTC(),
		PaidByMemberID: paidByMemberID,
		WalletID:       &wid,
		Amount:         amount,
	}
	return nil
}

func goalMonthKey(goalID, year, month int) string {
	return fmt.Sprintf("g-%d-%d-%d", goalID, year, month)
}

func (m *memoryStore) CreateSavingsGoal(_ context.Context, goal models.SavingsGoal) (models.SavingsGoal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	goal.ID = m.nextGoalID
	goal.CreatedAt = time.Now().UTC()
	if goal.MemberIDs == nil {
		goal.MemberIDs = []int{}
	}
	goal.SavedAmount = 0
	m.nextGoalID++
	m.goals[goal.ID] = goal
	goal.SavedAmount = goal.OpeningAmount
	return goal, nil
}

func (m *memoryStore) ListSavingsGoals(_ context.Context) ([]models.SavingsGoal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.SavingsGoal, 0, len(m.goals))
	for _, goal := range m.goals {
		goal.SavedAmount = m.sumSavedLocked(goal.ID)
		out = append(out, goal)
	}
	return out, nil
}

func (m *memoryStore) GetSavingsGoalByID(_ context.Context, id int) (models.SavingsGoal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	goal, ok := m.goals[id]
	if !ok {
		return models.SavingsGoal{}, repository.ErrNotFound
	}
	goal.SavedAmount = m.sumSavedLocked(id)
	return goal, nil
}

func (m *memoryStore) UpdateSavingsGoal(_ context.Context, id int, goal models.SavingsGoal) (models.SavingsGoal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.goals[id]
	if !ok {
		return models.SavingsGoal{}, repository.ErrNotFound
	}
	goal.ID = id
	goal.CreatedAt = existing.CreatedAt
	if goal.MemberIDs == nil {
		goal.MemberIDs = []int{}
	}
	goal.SavedAmount = m.sumSavedLocked(id)
	m.goals[id] = goal
	return goal, nil
}

func (m *memoryStore) DeleteSavingsGoal(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.goals[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.goals, id)
	for key, item := range m.goalMonths {
		if item.GoalID == id {
			delete(m.goalMonths, key)
		}
	}
	return nil
}

func (m *memoryStore) ListSavingsMonthAmounts(_ context.Context, year, month int) ([]models.SavingsMonthAmount, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.SavingsMonthAmount, 0)
	for _, item := range m.goalMonths {
		if item.Year == year && item.Month == month {
			out = append(out, item)
		}
	}
	return out, nil
}

func (m *memoryStore) SetSavingsMonthAmount(_ context.Context, goalID, year, month int, amount float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.goals[goalID]; !ok {
		return repository.ErrNotFound
	}
	key := goalMonthKey(goalID, year, month)
	if amount <= 0 {
		delete(m.goalMonths, key)
		return nil
	}
	m.goalMonths[key] = models.SavingsMonthAmount{
		GoalID:  goalID,
		Year:    year,
		Month:   month,
		Amount:  amount,
		SavedAt: time.Now().UTC(),
	}
	return nil
}

func (m *memoryStore) AdjustSavings(_ context.Context, goalID int, amount float64, walletID *int) (models.SavingsGoal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	goal, ok := m.goals[goalID]
	if !ok {
		return models.SavingsGoal{}, repository.ErrNotFound
	}
	if m.sumSavedLocked(goalID)+amount < -0.005 {
		return models.SavingsGoal{}, repository.ErrInsufficient
	}
	if walletID != nil && *walletID > 0 {
		wallet, wok := m.wallets[*walletID]
		if !wok {
			return models.SavingsGoal{}, repository.ErrNotFound
		}
		if !models.WalletCanFundGoal(wallet, goal.MemberIDs) {
			return models.SavingsGoal{}, repository.ErrWalletOwner
		}
		wallet.Balance = math.Round((wallet.Balance-amount)*100) / 100
		m.wallets[wallet.ID] = wallet
	}
	goal.OpeningAmount = math.Round((goal.OpeningAmount+amount)*100) / 100
	m.goals[goalID] = goal
	goal.SavedAmount = m.sumSavedLocked(goalID)
	goal.ApplyYield()
	return goal, nil
}

func (m *memoryStore) sumSavedLocked(goalID int) float64 {
	total := m.goals[goalID].OpeningAmount
	for _, item := range m.goalMonths {
		if item.GoalID == goalID {
			total += item.Amount
		}
	}
	return total
}

func (m *memoryStore) CreateWallet(_ context.Context, input models.CreateWalletInput) (models.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w := models.Wallet{
		ID:             m.nextWalletID,
		Name:           strings.TrimSpace(input.Name),
		Kind:           models.NormalizeWalletKind(input.Kind),
		MemberID:       input.MemberID,
		Balance:        input.Balance,
		ClosingDay:     input.ClosingDay,
		DueDay:         input.DueDay,
		CreditLimit:    input.CreditLimit,
		InvoiceBalance: input.InvoiceBalance,
		CreatedAt:      time.Now().UTC(),
	}
	m.nextWalletID++
	m.wallets[w.ID] = w
	return w, nil
}

func (m *memoryStore) ListWallets(_ context.Context) ([]models.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]models.Wallet, 0, len(m.wallets))
	for _, w := range m.wallets {
		out = append(out, w)
	}
	return out, nil
}

func (m *memoryStore) GetWalletByID(_ context.Context, id int) (models.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.wallets[id]
	if !ok {
		return models.Wallet{}, repository.ErrNotFound
	}
	return w, nil
}

func (m *memoryStore) UpdateWallet(_ context.Context, id int, input models.UpdateWalletInput) (models.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.wallets[id]
	if !ok {
		return models.Wallet{}, repository.ErrNotFound
	}
	existing.Name = strings.TrimSpace(input.Name)
	existing.Kind = models.NormalizeWalletKind(input.Kind)
	existing.MemberID = input.MemberID
	existing.Balance = input.Balance
	existing.ClosingDay = input.ClosingDay
	existing.DueDay = input.DueDay
	existing.CreditLimit = input.CreditLimit
	existing.InvoiceBalance = input.InvoiceBalance
	m.wallets[id] = existing
	return existing, nil
}

func (m *memoryStore) DeleteWallet(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.wallets[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.wallets, id)
	return nil
}

func (m *memoryStore) PayWalletInvoice(_ context.Context, creditWalletID int, input models.PayInvoiceInput) (models.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	amount := math.Round(input.Amount*100) / 100
	if !(amount > 0) || input.FromWalletID < 1 || input.FromWalletID == creditWalletID {
		return models.Wallet{}, repository.ErrInvalidAmount
	}
	credit, ok := m.wallets[creditWalletID]
	if !ok {
		return models.Wallet{}, repository.ErrNotFound
	}
	if !models.IsCredit(credit.Kind) {
		return models.Wallet{}, repository.ErrNotCredit
	}
	if credit.InvoiceBalance <= 0 {
		return models.Wallet{}, repository.ErrInvoiceEmpty
	}
	from, ok := m.wallets[input.FromWalletID]
	if !ok {
		return models.Wallet{}, repository.ErrNotFound
	}
	if models.IsCredit(from.Kind) {
		return models.Wallet{}, repository.ErrNotCredit
	}
	if amount > credit.InvoiceBalance {
		amount = credit.InvoiceBalance
	}
	from = models.ApplyWalletDelta(from, -amount)
	credit = models.ApplyWalletDelta(credit, amount)
	m.wallets[from.ID] = from
	m.wallets[credit.ID] = credit
	return credit, nil
}

func (m *memoryStore) ReconcileCardInvoice(
	_ context.Context,
	walletID, year, month int,
	amount float64,
	periodStart, periodEnd *string,
) (models.CardInvoice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	wallet, ok := m.wallets[walletID]
	if !ok {
		return models.CardInvoice{}, repository.ErrNotFound
	}
	wallet.InvoiceBalance = math.Round(amount*100) / 100
	m.wallets[walletID] = wallet
	cycle := models.CardCycleForDueMonth(wallet, year, month)
	return models.FinalizeCardInvoice(models.CardInvoice{
		WalletID:             walletID,
		Year:                 year,
		Month:                month,
		ClosingDate:          cycle.ClosingDate.Format("2006-01-02"),
		DueDate:              cycle.DueDate.Format("2006-01-02"),
		Amount:               amount,
		Source:               "statement",
		StatementPeriodStart: periodStart,
		StatementPeriodEnd:   periodEnd,
		StatementBalance:     &amount,
	}), nil
}

type stubCDI struct {
	rate float64
}

func (s stubCDI) AnnualRate(_ context.Context) (float64, error) {
	return s.rate, nil
}

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handlers.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body handlers.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "ok" {
		t.Fatalf("status = %q", body.Status)
	}
}

func TestCreateTransactionSuccess(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Food", Icon: "food"})
	h := &handlers.Transactions{Store: store, Categories: store, Members: store}

	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(
		`{"category_id":1,"type":"expense","description":"Lunch","amount":42.5,"date":"2026-08-15"}`,
	))
	rec := httptest.NewRecorder()
	h.ListOrCreate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateTransactionDebitsWallet(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Food", Icon: "food"})
	member, _ := store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Ana", MonthlySalary: 3000})
	wallet, _ := store.CreateWallet(context.Background(), models.CreateWalletInput{
		Name:     "Conta",
		Kind:     models.WalletChecking,
		MemberID: &member.ID,
		Balance:  500,
	})
	h := &handlers.Transactions{Store: store, Categories: store, Members: store}

	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(fmt.Sprintf(
		`{"category_id":1,"member_id":%d,"wallet_id":%d,"type":"expense","description":"Mercado","amount":80,"date":"2026-08-15"}`,
		member.ID, wallet.ID,
	)))
	rec := httptest.NewRecorder()
	h.ListOrCreate(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := store.GetWalletByID(context.Background(), wallet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Balance != 420 {
		t.Fatalf("wallet after expense = %v, want 420", updated.Balance)
	}

	incomeReq := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(fmt.Sprintf(
		`{"category_id":1,"member_id":%d,"wallet_id":%d,"type":"income","description":"Salário recebido","amount":3000,"date":"2026-08-05"}`,
		member.ID, wallet.ID,
	)))
	incomeRec := httptest.NewRecorder()
	h.ListOrCreate(incomeRec, incomeReq)
	if incomeRec.Code != http.StatusCreated {
		t.Fatalf("income status = %d body=%s", incomeRec.Code, incomeRec.Body.String())
	}
	credited, err := store.GetWalletByID(context.Background(), wallet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if credited.Balance != 3420 {
		t.Fatalf("wallet after income = %v, want 3420", credited.Balance)
	}
}

func TestCreateTransactionRequiresCategory(t *testing.T) {
	store := newMemoryStore()
	h := &handlers.Transactions{Store: store, Categories: store, Members: store}

	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(
		`{"category_id":99,"type":"expense","description":"Lunch","amount":10,"date":"2026-08-15"}`,
	))
	rec := httptest.NewRecorder()
	h.ListOrCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestMembersAndForecast(t *testing.T) {
	store := newMemoryStore()
	members := &handlers.Members{Store: store}
	forecast := &handlers.Forecast{Store: store}

	createReq := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(
		`{"name":"Nicolas","monthly_salary":5000}`,
	))
	createRec := httptest.NewRecorder()
	members.ListOrCreate(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create member status = %d", createRec.Code)
	}

	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Mercado", Icon: "market"})
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Casa", Icon: "home"})
	memberID := 1
	_, _ = store.CreateTransaction(context.Background(), models.Transaction{
		CategoryID:  1,
		MemberID:    &memberID,
		Type:        models.TransactionTypeExpense,
		Description: "Feira",
		Amount:      200,
		Date:        time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
	})
	_, _ = store.CreateBill(context.Background(), models.Bill{
		Name:       "Internet",
		Amount:     100,
		CategoryID: 2,
		DueDay:     10,
		Frequency:  models.BillFrequencyMonthly,
		Recurrence: models.BillRecurrenceOngoing,
		StartMonth: "2026-01",
		MemberIDs:  []int{memberID},
	})

	req := httptest.NewRequest(http.MethodGet, "/forecast/monthly?year=2026&month=8", nil)
	rec := httptest.NewRecorder()
	forecast.Monthly(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forecast status = %d body=%s", rec.Code, rec.Body.String())
	}

	var body models.MonthlyForecast
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PlannedSalary != 5000 {
		t.Fatalf("planned_salary = %v, want 5000", body.PlannedSalary)
	}
	if body.PlannedBills != 100 {
		t.Fatalf("planned_bills = %v, want 100", body.PlannedBills)
	}
	if body.TotalExpense != 300 {
		t.Fatalf("total_expense = %v, want 300", body.TotalExpense)
	}
	if len(body.ByMember) != 1 {
		t.Fatalf("by_member len = %d, want 1", len(body.ByMember))
	}
	mf := body.ByMember[0]
	if mf.BillShare != 100 {
		t.Fatalf("bill_share = %v, want 100", mf.BillShare)
	}
	if mf.VariableExpense != 200 {
		t.Fatalf("variable_expense = %v, want 200", mf.VariableExpense)
	}
	if mf.TotalToPay != 300 {
		t.Fatalf("total_to_pay = %v, want 300", mf.TotalToPay)
	}
	if mf.Remaining != 4700 {
		t.Fatalf("remaining = %v, want 4700", mf.Remaining)
	}
}

func TestCreateOngoingAndUntilBills(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Casa", Icon: "home"})
	h := &handlers.Bills{Store: store, Categories: store, Members: store}

	ongoingReq := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Luz","amount":180,"category_id":1,"due_day":15,"frequency":"monthly","recurrence":"ongoing","start_month":"2026-01","member_ids":[]}`,
	))
	ongoingRec := httptest.NewRecorder()
	h.ListOrCreate(ongoingRec, ongoingReq)
	if ongoingRec.Code != http.StatusCreated {
		t.Fatalf("ongoing status = %d body=%s", ongoingRec.Code, ongoingRec.Body.String())
	}
	var luz models.Bill
	if err := json.NewDecoder(ongoingRec.Body).Decode(&luz); err != nil {
		t.Fatal(err)
	}
	if luz.Frequency != models.BillFrequencyMonthly {
		t.Fatalf("frequency = %q, want monthly", luz.Frequency)
	}

	weeklyReq := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Feira","amount":50,"category_id":1,"due_day":5,"frequency":"weekly","recurrence":"ongoing","start_month":"2026-08","member_ids":[]}`,
	))
	weeklyRec := httptest.NewRecorder()
	h.ListOrCreate(weeklyRec, weeklyReq)
	if weeklyRec.Code != http.StatusCreated {
		t.Fatalf("weekly status = %d body=%s", weeklyRec.Code, weeklyRec.Body.String())
	}

	untilReq := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Netflix","amount":55,"category_id":1,"due_day":5,"frequency":"monthly","recurrence":"until","start_month":"2026-01","end_month":"2026-12","member_ids":[]}`,
	))
	untilRec := httptest.NewRecorder()
	h.ListOrCreate(untilRec, untilReq)
	if untilRec.Code != http.StatusCreated {
		t.Fatalf("until status = %d body=%s", untilRec.Code, untilRec.Body.String())
	}

	missingEnd := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Spotify","amount":30,"category_id":1,"due_day":5,"recurrence":"until","start_month":"2026-01","member_ids":[]}`,
	))
	missingRec := httptest.NewRecorder()
	h.ListOrCreate(missingRec, missingEnd)
	if missingRec.Code != http.StatusBadRequest {
		t.Fatalf("missing end status = %d, want 400", missingRec.Code)
	}

	// Create two members and attach both to one bill.
	_, _ = store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Ana", MonthlySalary: 3000})
	_, _ = store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Bruno", MonthlySalary: 4000})
	sharedReq := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Internet","amount":120,"category_id":1,"due_day":8,"recurrence":"ongoing","start_month":"2026-01","member_ids":[1,2]}`,
	))
	sharedRec := httptest.NewRecorder()
	h.ListOrCreate(sharedRec, sharedReq)
	if sharedRec.Code != http.StatusCreated {
		t.Fatalf("shared status = %d body=%s", sharedRec.Code, sharedRec.Body.String())
	}
	var shared models.Bill
	if err := json.NewDecoder(sharedRec.Body).Decode(&shared); err != nil {
		t.Fatal(err)
	}
	if len(shared.MemberIDs) != 2 {
		t.Fatalf("member_ids len = %d, want 2", len(shared.MemberIDs))
	}
}

func TestBillPaidChecklist(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Casa", Icon: "home"})
	member, _ := store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Nicolas", MonthlySalary: 6500})
	memberID := member.ID
	wallet, _ := store.CreateWallet(context.Background(), models.CreateWalletInput{
		Name:     "Conta",
		Kind:     models.WalletChecking,
		MemberID: &memberID,
		Balance:  1000,
	})
	h := &handlers.Bills{Store: store, Categories: store, Members: store}

	createReq := httptest.NewRequest(http.MethodPost, "/bills", strings.NewReader(
		`{"name":"Luz","amount":180,"category_id":1,"due_day":15,"frequency":"monthly","recurrence":"ongoing","start_month":"2026-01","member_ids":[]}`,
	))
	createRec := httptest.NewRecorder()
	h.ListOrCreate(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var bill models.Bill
	if err := json.NewDecoder(createRec.Body).Decode(&bill); err != nil {
		t.Fatal(err)
	}

	paidReq := httptest.NewRequest(http.MethodPut, "/bills/1/paid", strings.NewReader(
		`{"year":2026,"month":8,"paid":true,"paid_by_member_id":1}`,
	))
	paidReq.SetPathValue("id", "1")
	paidRec := httptest.NewRecorder()
	h.SetPaid(paidRec, paidReq)
	if paidRec.Code != http.StatusOK {
		t.Fatalf("paid status = %d body=%s", paidRec.Code, paidRec.Body.String())
	}

	updatedWallet, err := store.GetWalletByID(context.Background(), wallet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedWallet.Balance != 820 {
		t.Fatalf("wallet after pay = %v, want 820", updatedWallet.Balance)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/bills/payments?year=2026&month=8", nil)
	listRec := httptest.NewRecorder()
	h.ListPayments(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listRec.Code, listRec.Body.String())
	}
	var payments []models.BillPayment
	if err := json.NewDecoder(listRec.Body).Decode(&payments); err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 || payments[0].BillID != bill.ID {
		t.Fatalf("payments = %+v, want one for bill %d", payments, bill.ID)
	}
	if payments[0].PaidByMemberID == nil || *payments[0].PaidByMemberID != memberID {
		t.Fatalf("paid_by = %+v", payments[0].PaidByMemberID)
	}

	unpaidReq := httptest.NewRequest(http.MethodPut, "/bills/1/paid", strings.NewReader(
		`{"year":2026,"month":8,"paid":false}`,
	))
	unpaidReq.SetPathValue("id", "1")
	unpaidRec := httptest.NewRecorder()
	h.SetPaid(unpaidRec, unpaidReq)
	if unpaidRec.Code != http.StatusNoContent {
		t.Fatalf("unpaid status = %d body=%s", unpaidRec.Code, unpaidRec.Body.String())
	}

	refunded, err := store.GetWalletByID(context.Background(), wallet.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refunded.Balance != 1000 {
		t.Fatalf("wallet after unpay = %v, want 1000", refunded.Balance)
	}

	emptyReq := httptest.NewRequest(http.MethodGet, "/bills/payments?year=2026&month=8", nil)
	emptyRec := httptest.NewRecorder()
	h.ListPayments(emptyRec, emptyReq)
	var empty []models.BillPayment
	if err := json.NewDecoder(emptyRec.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("payments after unpay = %+v, want empty", empty)
	}

	missingReq := httptest.NewRequest(http.MethodPut, "/bills/99/paid", strings.NewReader(
		`{"year":2026,"month":8,"paid":true,"paid_by_member_id":1}`,
	))
	missingReq.SetPathValue("id", "99")
	missingRec := httptest.NewRecorder()
	h.SetPaid(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing bill status = %d, want 404", missingRec.Code)
	}
}

func TestSavingsGoalsAndForecast(t *testing.T) {
	store := newMemoryStore()
	members := &handlers.Members{Store: store}
	savings := &handlers.Savings{Store: store, Members: store, CDI: stubCDI{rate: 14.15}}
	forecast := &handlers.Forecast{Store: store}

	createMember := httptest.NewRequest(http.MethodPost, "/members", strings.NewReader(
		`{"name":"Nicolas","monthly_salary":5000}`,
	))
	createMemberRec := httptest.NewRecorder()
	members.ListOrCreate(createMemberRec, createMember)
	if createMemberRec.Code != http.StatusCreated {
		t.Fatalf("create member status = %d", createMemberRec.Code)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(
		`{"name":"Viagem","end_kind":"amount","target_amount":8000,"member_ids":[1]}`,
	))
	createRec := httptest.NewRecorder()
	savings.ListOrCreate(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create goal status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var goal models.SavingsGoal
	if err := json.NewDecoder(createRec.Body).Decode(&goal); err != nil {
		t.Fatal(err)
	}
	wantPlan := models.BuildSavingsPlan(8000, 0, 14.15, 12, 1, true)
	if goal.MonthlyAmount != wantPlan.MonthlyAmount || goal.TargetAmount != 8000 {
		t.Fatalf("goal = %+v, want monthly %v", goal, wantPlan.MonthlyAmount)
	}
	if goal.EndKind != models.SavingsEndAmount {
		t.Fatalf("end_kind = %q", goal.EndKind)
	}

	missingName := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(
		`{"name":"","end_kind":"amount","target_amount":1000,"member_ids":[1]}`,
	))
	missingRec := httptest.NewRecorder()
	savings.ListOrCreate(missingRec, missingName)
	if missingRec.Code != http.StatusBadRequest {
		t.Fatalf("missing name status = %d, want 400", missingRec.Code)
	}

	noPeople := httptest.NewRequest(http.MethodPost, "/savings", strings.NewReader(
		`{"name":"Reserva","end_kind":"amount","target_amount":1000,"member_ids":[]}`,
	))
	noPeopleRec := httptest.NewRecorder()
	savings.ListOrCreate(noPeopleRec, noPeople)
	if noPeopleRec.Code != http.StatusBadRequest {
		t.Fatalf("no people status = %d, want 400", noPeopleRec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/forecast/monthly?year=2026&month=8", nil)
	rec := httptest.NewRecorder()
	forecast.Monthly(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("forecast status = %d body=%s", rec.Code, rec.Body.String())
	}
	var body models.MonthlyForecast
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.PlannedSavings != wantPlan.MonthlyAmount {
		t.Fatalf("planned_savings = %v, want %v", body.PlannedSavings, wantPlan.MonthlyAmount)
	}
	wantRemaining := 5000 - wantPlan.MonthlyAmount
	if body.Remaining != wantRemaining {
		t.Fatalf("remaining = %v, want %v", body.Remaining, wantRemaining)
	}
	if len(body.ByMember) != 1 || body.ByMember[0].SavingsShare != wantPlan.MonthlyAmount {
		t.Fatalf("by_member = %+v", body.ByMember)
	}

	planReq := httptest.NewRequest(http.MethodGet, "/savings/plan?end_kind=amount&target=8000&members=1", nil)
	planRec := httptest.NewRecorder()
	savings.Plan(planRec, planReq)
	if planRec.Code != http.StatusOK {
		t.Fatalf("plan status = %d body=%s", planRec.Code, planRec.Body.String())
	}

	saveReq := httptest.NewRequest(http.MethodPut, "/savings/1/month", strings.NewReader(
		`{"year":2026,"month":8,"amount":500}`,
	))
	saveReq.SetPathValue("id", "1")
	saveRec := httptest.NewRecorder()
	savings.SetMonthAmount(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("save month status = %d body=%s", saveRec.Code, saveRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/savings/months?year=2026&month=8", nil)
	listRec := httptest.NewRecorder()
	savings.ListMonthAmounts(listRec, listReq)
	var months []models.SavingsMonthAmount
	if err := json.NewDecoder(listRec.Body).Decode(&months); err != nil {
		t.Fatal(err)
	}
	if len(months) != 1 || months[0].Amount != 500 {
		t.Fatalf("months = %+v", months)
	}

	goalsReq := httptest.NewRequest(http.MethodGet, "/savings", nil)
	goalsRec := httptest.NewRecorder()
	savings.ListOrCreate(goalsRec, goalsReq)
	var goals []models.SavingsGoal
	if err := json.NewDecoder(goalsRec.Body).Decode(&goals); err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 || goals[0].SavedAmount != 500 {
		t.Fatalf("goals = %+v", goals)
	}

	clearReq := httptest.NewRequest(http.MethodPut, "/savings/1/month", strings.NewReader(
		`{"year":2026,"month":8,"amount":0}`,
	))
	clearReq.SetPathValue("id", "1")
	clearRec := httptest.NewRecorder()
	savings.SetMonthAmount(clearRec, clearReq)
	if clearRec.Code != http.StatusNoContent {
		t.Fatalf("clear month status = %d body=%s", clearRec.Code, clearRec.Body.String())
	}
}

func TestWalletsCreateAndUpdate(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Nicolas", MonthlySalary: 6500})
	h := &handlers.Wallets{Store: store, Members: store}

	memberID := 1
	createReq := httptest.NewRequest(http.MethodPost, "/wallets", strings.NewReader(
		`{"name":"Conta","kind":"checking","member_id":1,"balance":1186.72}`,
	))
	createRec := httptest.NewRecorder()
	h.ListOrCreate(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", createRec.Code, createRec.Body.String())
	}
	var wallet models.Wallet
	if err := json.NewDecoder(createRec.Body).Decode(&wallet); err != nil {
		t.Fatal(err)
	}
	if wallet.Balance != 1186.72 || wallet.MemberID == nil || *wallet.MemberID != memberID {
		t.Fatalf("wallet = %+v", wallet)
	}

	jointReq := httptest.NewRequest(http.MethodPost, "/wallets", strings.NewReader(
		`{"name":"Caixinha conjunta","kind":"savings","member_id":null,"balance":1603.6}`,
	))
	jointRec := httptest.NewRecorder()
	h.ListOrCreate(jointRec, jointReq)
	if jointRec.Code != http.StatusCreated {
		t.Fatalf("joint status = %d body=%s", jointRec.Code, jointRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/wallets/1", strings.NewReader(
		`{"name":"Conta","kind":"checking","member_id":1,"balance":2000}`,
	))
	updateReq.SetPathValue("id", "1")
	updateRec := httptest.NewRecorder()
	h.Update(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update status = %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/wallets", nil)
	listRec := httptest.NewRecorder()
	h.ListOrCreate(listRec, listReq)
	var items []models.Wallet
	if err := json.NewDecoder(listRec.Body).Decode(&items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("wallets len = %d, want 2", len(items))
	}
}

type fakeStatementParser struct {
	result models.ParsedStatement
	err    error
}

func (f fakeStatementParser) ParseStatement(_ context.Context, _ []byte, _, _ int) (models.ParsedStatement, error) {
	return f.result, f.err
}

func TestStatementPreviewAndImport(t *testing.T) {
	store := newMemoryStore()
	food, _ := store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Comida", Icon: "food"})
	transport, _ := store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Transporte", Icon: "transport"})
	member, _ := store.CreateMember(context.Background(), models.CreateMemberInput{Name: "Ana", MonthlySalary: 3000})
	closing, due := 10, 17
	card, _ := store.CreateWallet(context.Background(), models.CreateWalletInput{
		Name:           "Nubank",
		Kind:           models.WalletCredit,
		MemberID:       &member.ID,
		ClosingDay:     &closing,
		DueDay:         &due,
		CreditLimit:    5000,
		InvoiceBalance: 1500,
	})
	_, _ = store.CreateTransaction(context.Background(), models.Transaction{
		CategoryID:  food.ID,
		MemberID:    &member.ID,
		Type:        models.TransactionTypeExpense,
		Description: "iFood",
		Amount:      42.90,
		Date:        time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})

	h := &handlers.Statements{
		Parser: fakeStatementParser{result: models.ParsedStatement{
			Issuer: "nubank",
			Items: []models.ParsedStatementItem{
				{Date: "2026-07-10", Description: "IFOOD", Amount: 42.90, Kind: "expense", SuggestedIcon: "food"},
				{Date: "2026-07-11", Description: "UBER TRIP", Amount: 18.50, Kind: "expense", SuggestedIcon: "transport"},
				{Date: "2026-07-08", Description: "Pagamento recebido", Amount: 1234.56, Kind: "payment", SuggestedIcon: "other"},
			},
		}},
		Transactions: store,
		Categories:   store,
		Members:      store,
		Wallets:      store,
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "fatura.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("%PDF-fake"))
	_ = writer.WriteField("wallet_id", fmt.Sprintf("%d", card.ID))
	_ = writer.WriteField("member_id", fmt.Sprintf("%d", member.ID))
	_ = writer.WriteField("year", "2026")
	_ = writer.WriteField("month", "8")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	previewReq := httptest.NewRequest(http.MethodPost, "/statements/preview", &buf)
	previewReq.Header.Set("Content-Type", writer.FormDataContentType())
	previewRec := httptest.NewRecorder()
	h.Preview(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status = %d body=%s", previewRec.Code, previewRec.Body.String())
	}
	var preview models.StatementPreview
	if err := json.NewDecoder(previewRec.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.NewCount != 1 || preview.MatchedCount != 1 || preview.SkippedCount != 1 {
		t.Fatalf("counts new=%d matched=%d skipped=%d", preview.NewCount, preview.MatchedCount, preview.SkippedCount)
	}

	importBody := fmt.Sprintf(
		`{"wallet_id":%d,"member_id":%d,"apply_to_invoice":false,"items":[{"date":"2026-07-11","description":"UBER TRIP","amount":18.5,"type":"expense","category_id":%d}]}`,
		card.ID, member.ID, transport.ID,
	)
	importReq := httptest.NewRequest(http.MethodPost, "/statements/import", strings.NewReader(importBody))
	importRec := httptest.NewRecorder()
	h.Import(importRec, importReq)
	if importRec.Code != http.StatusCreated {
		t.Fatalf("import status = %d body=%s", importRec.Code, importRec.Body.String())
	}

	updated, err := store.GetWalletByID(context.Background(), card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.InvoiceBalance != 1500 {
		t.Fatalf("invoice after import without apply = %v, want 1500", updated.InvoiceBalance)
	}
	txs, err := store.ListTransactions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(txs) != 2 {
		t.Fatalf("transactions after import = %d, want 2", len(txs))
	}
}

func TestStatementPreviewCSVWithoutParser(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Comida", Icon: "food"})
	h := &handlers.Statements{
		Transactions: store,
		Categories:   store,
		Members:      store,
		Wallets:      store,
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "extrato.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("Data;Descricao;Valor\n10/07/2026;IFOOD;42,90\n11/07/2026;UBER TRIP;18,50\n"))
	_ = writer.WriteField("year", "2026")
	_ = writer.WriteField("month", "7")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/statements/preview", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Preview(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var preview models.StatementPreview
	if err := json.NewDecoder(rec.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.NewCount != 2 {
		t.Fatalf("new_count = %d, want 2", preview.NewCount)
	}
}

func TestStatementPreviewOFXWithoutParser(t *testing.T) {
	store := newMemoryStore()
	_, _ = store.CreateCategory(context.Background(), models.CreateCategoryInput{Name: "Outro", Icon: "other"})
	h := &handlers.Statements{
		Transactions: store,
		Categories:   store,
		Members:      store,
		Wallets:      store,
	}

	ofx := []byte("OFXHEADER:100\n<OFX><STMTTRN><TRNTYPE>DEBIT</TRNTYPE><DTPOSTED>20260901000000</DTPOSTED><TRNAMT>-43.46</TRNAMT><MEMO>Compra no débito - SAO ROQUE</MEMO></STMTTRN></OFX>")
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "extrato.ofx")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(ofx)
	_ = writer.WriteField("year", "2026")
	_ = writer.WriteField("month", "9")
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/statements/preview", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.Preview(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var preview models.StatementPreview
	if err := json.NewDecoder(rec.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if preview.NewCount != 1 || preview.Items[0].Kind != "expense" {
		t.Fatalf("preview = %+v", preview)
	}
}
