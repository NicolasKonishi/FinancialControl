package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// SavingsStore is the persistence dependency for savings handlers.
type SavingsStore interface {
	CreateSavingsGoal(ctx context.Context, goal models.SavingsGoal) (models.SavingsGoal, error)
	ListSavingsGoals(ctx context.Context) ([]models.SavingsGoal, error)
	GetSavingsGoalByID(ctx context.Context, id int) (models.SavingsGoal, error)
	UpdateSavingsGoal(ctx context.Context, id int, goal models.SavingsGoal) (models.SavingsGoal, error)
	DeleteSavingsGoal(ctx context.Context, id int) error
	ListSavingsMonthAmounts(ctx context.Context, year, month int) ([]models.SavingsMonthAmount, error)
	SetSavingsMonthAmount(ctx context.Context, goalID, year, month int, amount float64) error
	AdjustSavings(ctx context.Context, goalID int, amount float64, walletID *int) (models.SavingsGoal, error)
}

// CDISource provides the current CDI annual rate.
type CDISource interface {
	AnnualRate(ctx context.Context) (float64, error)
}

// Savings handles savings goal HTTP endpoints.
type Savings struct {
	Store   SavingsStore
	Members MemberStore
	CDI     CDISource
}

// ListOrCreate handles GET and POST /savings.
func (h *Savings) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Store.ListSavingsGoals(r.Context())
		if err != nil {
			writeStoreError(w, err, "savings goal not found")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Savings) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateSavingsGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	goal, ok := h.buildFromInput(w, r, input, 0)
	if !ok {
		return
	}

	created, err := h.Store.CreateSavingsGoal(r.Context(), goal)
	if err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Update handles PUT /savings/{id}.
func (h *Savings) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	existing, err := h.Store.GetSavingsGoalByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}

	var input models.UpdateSavingsGoalInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	goal, ok := h.buildFromInput(w, r, models.CreateSavingsGoalInput{
		Name:         input.Name,
		TargetAmount: input.TargetAmount,
		MemberIDs:    input.MemberIDs,
		Notes:        input.Notes,
		EndKind:      input.EndKind,
		EndMonth:     input.EndMonth,
	}, existing.SavedAmount)
	if !ok {
		return
	}

	updated, err := h.Store.UpdateSavingsGoal(r.Context(), id, goal)
	if err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /savings/{id}.
func (h *Savings) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.Store.DeleteSavingsGoal(r.Context(), id); err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Plan handles GET /savings/plan.
func (h *Savings) Plan(w http.ResponseWriter, r *http.Request) {
	target, err := strconv.ParseFloat(r.URL.Query().Get("target"), 64)
	if err != nil || target <= 0 {
		http.Error(w, "target must be greater than zero", http.StatusBadRequest)
		return
	}
	endKind := models.NormalizeEndKind(r.URL.Query().Get("end_kind"))
	if endKind == "" {
		http.Error(w, "end_kind must be none, date, or amount", http.StatusBadRequest)
		return
	}
	memberCount, _ := strconv.Atoi(r.URL.Query().Get("members"))
	saved, _ := strconv.ParseFloat(r.URL.Query().Get("saved"), 64)
	endMonth := strings.TrimSpace(r.URL.Query().Get("end_month"))

	cdiAnnual := h.cdiAnnual(r.Context())
	now := time.Now()
	plan, err := planFromInput(endKind, target, saved, endMonth, memberCount, now.Year(), int(now.Month()), cdiAnnual)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// ListMonthAmounts handles GET /savings/months?year=&month=.
func (h *Savings) ListMonthAmounts(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}

	items, err := h.Store.ListSavingsMonthAmounts(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// SetMonthAmount handles PUT /savings/{id}/month.
func (h *Savings) SetMonthAmount(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.SetSavingsMonthInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Year < 1 {
		http.Error(w, "invalid year", http.StatusBadRequest)
		return
	}
	if input.Month < 1 || input.Month > 12 {
		http.Error(w, "invalid month", http.StatusBadRequest)
		return
	}
	if input.Amount < 0 {
		http.Error(w, "amount must be >= 0", http.StatusBadRequest)
		return
	}

	if err := h.Store.SetSavingsMonthAmount(r.Context(), id, input.Year, input.Month, input.Amount); err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}

	if input.Amount <= 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := h.Store.ListSavingsMonthAmounts(r.Context(), input.Year, input.Month)
	if err != nil {
		writeStoreError(w, err, "savings goal not found")
		return
	}
	for _, item := range items {
		if item.GoalID == id {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeJSON(w, http.StatusOK, models.SavingsMonthAmount{
		GoalID: id,
		Year:   input.Year,
		Month:  input.Month,
		Amount: input.Amount,
	})
}

// Adjust handles PUT /savings/{id}/adjust.
func (h *Savings) Adjust(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.AdjustSavingsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Amount == 0 {
		http.Error(w, "amount must not be zero", http.StatusBadRequest)
		return
	}

	updated, err := h.Store.AdjustSavings(r.Context(), id, input.Amount, input.WalletID)
	if err != nil {
		if errors.Is(err, repository.ErrInsufficient) {
			http.Error(w, "not enough saved in this box", http.StatusBadRequest)
			return
		}
		if errors.Is(err, repository.ErrWalletOwner) {
			http.Error(w, "use a checking account of someone in this box", http.StatusBadRequest)
			return
		}
		writeStoreError(w, err, "savings goal not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Savings) buildFromInput(w http.ResponseWriter, r *http.Request, input models.CreateSavingsGoalInput, saved float64) (models.SavingsGoal, bool) {
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return models.SavingsGoal{}, false
	}

	endKind := models.NormalizeEndKind(input.EndKind)
	if endKind == "" {
		http.Error(w, "end_kind must be none, date, or amount", http.StatusBadRequest)
		return models.SavingsGoal{}, false
	}

	memberIDs := uniquePositiveIDs(input.MemberIDs)
	if len(memberIDs) == 0 {
		http.Error(w, "select at least one person", http.StatusBadRequest)
		return models.SavingsGoal{}, false
	}
	for _, memberID := range memberIDs {
		if _, err := h.Members.GetMemberByID(r.Context(), memberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return models.SavingsGoal{}, false
			}
			writeStoreError(w, err, "member not found")
			return models.SavingsGoal{}, false
		}
	}

	var endMonth *string
	if input.EndMonth != nil {
		trimmed := strings.TrimSpace(*input.EndMonth)
		if trimmed != "" {
			endMonth = &trimmed
		}
	}

	cdiAnnual := h.cdiAnnual(r.Context())
	now := time.Now()
	target := input.TargetAmount
	if endKind == models.SavingsEndNone {
		target = 0
		endMonth = nil
	}

	plan, err := planFromInput(endKind, target, saved, stringPtrValue(endMonth), len(memberIDs), now.Year(), int(now.Month()), cdiAnnual)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return models.SavingsGoal{}, false
	}

	return models.SavingsGoal{
		Name:          strings.TrimSpace(input.Name),
		TargetAmount:  plan.TargetAmount,
		MonthlyAmount: plan.MonthlyAmount,
		MemberIDs:     memberIDs,
		Notes:         strings.TrimSpace(input.Notes),
		EndKind:       endKind,
		EndMonth:      endMonth,
		CDIAnnual:     plan.CDIAnnual,
		YieldAnnual:   plan.YieldAnnual,
	}, true
}

func (h *Savings) cdiAnnual(ctx context.Context) float64 {
	if h.CDI == nil {
		return models.FallbackCDIAnnual
	}
	rate, err := h.CDI.AnnualRate(ctx)
	if err != nil || rate <= 0 {
		if err != nil {
			log.Printf("cdi: %v", err)
		}
		return models.FallbackCDIAnnual
	}
	return rate
}

func planFromInput(endKind string, target, saved float64, endMonth string, memberCount, year, month int, cdiAnnual float64) (models.SavingsPlan, error) {
	if endKind == models.SavingsEndNone {
		return models.SavingsPlan{
			Months:        0,
			MonthlyAmount: 0,
			CDIAnnual:     round2(cdiAnnual),
			YieldFactor:   models.CDIYieldFactor,
			YieldAnnual:   round2(cdiAnnual * models.CDIYieldFactor),
			MemberCount:   memberCount,
		}, nil
	}
	if target <= 0 {
		return models.SavingsPlan{}, errors.New("target_amount must be greater than zero")
	}

	usedDefault := false
	months := models.DefaultPlanMonths
	if endKind == models.SavingsEndDate {
		if strings.TrimSpace(endMonth) == "" {
			return models.SavingsPlan{}, errors.New("end_month is required for a date goal")
		}
		n, err := models.MonthsInclusive(year, month, endMonth)
		if err != nil {
			return models.SavingsPlan{}, err
		}
		months = n
	} else if strings.TrimSpace(endMonth) != "" {
		n, err := models.MonthsInclusive(year, month, endMonth)
		if err != nil {
			return models.SavingsPlan{}, err
		}
		months = n
	} else {
		usedDefault = true
	}

	return models.BuildSavingsPlan(target, saved, cdiAnnual, months, memberCount, usedDefault), nil
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
