package handlers_test

import (
	"context"
	"encoding/json"
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
	nextCatID    int
	nextTxID     int
	nextMemberID int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		categories:   make(map[int]models.Category),
		transactions: make(map[int]models.Transaction),
		members:      make(map[int]models.Member),
		nextCatID:    1,
		nextTxID:     1,
		nextMemberID: 1,
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
	tx.ID = id
	tx.CreatedAt = existing.CreatedAt
	m.transactions[id] = tx
	return tx, nil
}

func (m *memoryStore) DeleteTransaction(_ context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.transactions[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.transactions, id)
	return nil
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
	_, _ = store.CreateTransaction(context.Background(), models.Transaction{
		CategoryID:  1,
		Type:        models.TransactionTypeExpense,
		Description: "Feira",
		Amount:      200,
		Date:        time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
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
	if body.TotalExpense != 200 {
		t.Fatalf("total_expense = %v, want 200", body.TotalExpense)
	}
}
