package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// CreateSavingsGoal inserts a savings goal and its contributing members.
func (s *Store) CreateSavingsGoal(ctx context.Context, goal models.SavingsGoal) (models.SavingsGoal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("begin create savings goal: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO savings_goals (name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at
	`

	now := formatDateTime(time.Now())
	created, err := scanSavingsGoal(tx.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(goal.Name),
		goal.TargetAmount,
		goal.MonthlyAmount,
		strings.TrimSpace(goal.Notes),
		models.NormalizeEndKind(goal.EndKind),
		nullString(goal.EndMonth),
		goal.CDIAnnual,
		goal.OpeningAmount,
		now,
	))
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("create savings goal: %w", err)
	}

	if err := replaceGoalMembersTx(ctx, tx, created.ID, goal.MemberIDs); err != nil {
		return models.SavingsGoal{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.SavingsGoal{}, fmt.Errorf("commit create savings goal: %w", err)
	}

	created.MemberIDs = normalizeMemberIDs(goal.MemberIDs)
	created.SavedAmount = created.OpeningAmount
	created.ApplyYield()
	return created, nil
}

// ListSavingsGoals returns all goals with saved totals.
func (s *Store) ListSavingsGoals(ctx context.Context) ([]models.SavingsGoal, error) {
	const query = `
		SELECT id, name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at
		FROM savings_goals
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list savings goals: %w", err)
	}
	defer rows.Close()

	goals := make([]models.SavingsGoal, 0)
	for rows.Next() {
		goal, err := scanSavingsGoal(rows)
		if err != nil {
			return nil, err
		}
		goals = append(goals, goal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate savings goals: %w", err)
	}

	for i := range goals {
		hydrated, err := s.hydrateSavingsGoal(ctx, goals[i])
		if err != nil {
			return nil, err
		}
		goals[i] = hydrated
	}
	return goals, nil
}

// GetSavingsGoalByID returns one goal or ErrNotFound.
func (s *Store) GetSavingsGoalByID(ctx context.Context, id int) (models.SavingsGoal, error) {
	const query = `
		SELECT id, name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at
		FROM savings_goals
		WHERE id = ?
	`

	goal, err := scanSavingsGoal(s.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.SavingsGoal{}, ErrNotFound
	}
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("get savings goal: %w", err)
	}
	return s.hydrateSavingsGoal(ctx, goal)
}

// UpdateSavingsGoal updates a goal and its members.
func (s *Store) UpdateSavingsGoal(ctx context.Context, id int, goal models.SavingsGoal) (models.SavingsGoal, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("begin update savings goal: %w", err)
	}
	defer tx.Rollback()

	const query = `
		UPDATE savings_goals
		SET name = ?, target_amount = ?, monthly_amount = ?, notes = ?, end_kind = ?, end_month = ?, cdi_annual = ?
		WHERE id = ?
		RETURNING id, name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at
	`

	updated, err := scanSavingsGoal(tx.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(goal.Name),
		goal.TargetAmount,
		goal.MonthlyAmount,
		strings.TrimSpace(goal.Notes),
		models.NormalizeEndKind(goal.EndKind),
		nullString(goal.EndMonth),
		goal.CDIAnnual,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return models.SavingsGoal{}, ErrNotFound
	}
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("update savings goal: %w", err)
	}

	if err := replaceGoalMembersTx(ctx, tx, id, goal.MemberIDs); err != nil {
		return models.SavingsGoal{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.SavingsGoal{}, fmt.Errorf("commit update savings goal: %w", err)
	}

	updated.MemberIDs = normalizeMemberIDs(goal.MemberIDs)
	saved, err := s.savedAmountForGoal(ctx, id)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	updated.SavedAmount = saved
	updated.ApplyYield()
	return updated, nil
}

// DeleteSavingsGoal removes a goal and its related rows.
func (s *Store) DeleteSavingsGoal(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM savings_goals WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete savings goal: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete savings goal rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListSavingsMonthAmounts returns amounts saved in year/month.
func (s *Store) ListSavingsMonthAmounts(ctx context.Context, year, month int) ([]models.SavingsMonthAmount, error) {
	const query = `
		SELECT goal_id, year, month, amount, saved_at
		FROM savings_month_amounts
		WHERE year = ? AND month = ?
		ORDER BY goal_id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, year, month)
	if err != nil {
		return nil, fmt.Errorf("list savings month amounts: %w", err)
	}
	defer rows.Close()

	items := make([]models.SavingsMonthAmount, 0)
	for rows.Next() {
		var (
			item    models.SavingsMonthAmount
			savedAt string
		)
		if err := rows.Scan(&item.GoalID, &item.Year, &item.Month, &item.Amount, &savedAt); err != nil {
			return nil, fmt.Errorf("scan savings month amount: %w", err)
		}
		parsed, err := parseDateTime(savedAt)
		if err != nil {
			return nil, err
		}
		item.SavedAt = parsed
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate savings month amounts: %w", err)
	}
	return items, nil
}

// SetSavingsMonthAmount records or clears how much was saved for a goal in a month.
func (s *Store) SetSavingsMonthAmount(ctx context.Context, goalID, year, month int, amount float64) error {
	if _, err := s.GetSavingsGoalByID(ctx, goalID); err != nil {
		return err
	}

	if amount <= 0 {
		if _, err := s.db.ExecContext(
			ctx,
			`DELETE FROM savings_month_amounts WHERE goal_id = ? AND year = ? AND month = ?`,
			goalID,
			year,
			month,
		); err != nil {
			return fmt.Errorf("clear savings month amount: %w", err)
		}
		return nil
	}

	now := formatDateTime(time.Now())
	if _, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO savings_month_amounts (goal_id, year, month, amount, saved_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (goal_id, year, month) DO UPDATE SET amount = excluded.amount, saved_at = excluded.saved_at
		`,
		goalID,
		year,
		month,
		amount,
		now,
	); err != nil {
		return fmt.Errorf("set savings month amount: %w", err)
	}
	return nil
}

// AdjustSavings adds or removes money from a box and optionally moves wallet cash.
func (s *Store) AdjustSavings(ctx context.Context, goalID int, amount float64, walletID *int) (models.SavingsGoal, error) {
	if amount == 0 {
		return s.GetSavingsGoalByID(ctx, goalID)
	}

	goal, err := s.GetSavingsGoalByID(ctx, goalID)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	if goal.SavedAmount+amount < -0.005 {
		return models.SavingsGoal{}, ErrInsufficient
	}

	if walletID != nil && *walletID > 0 {
		wallet, err := s.GetWalletByID(ctx, *walletID)
		if err != nil {
			return models.SavingsGoal{}, err
		}
		if !models.WalletCanFundGoal(wallet, goal.MemberIDs) {
			return models.SavingsGoal{}, ErrWalletOwner
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.SavingsGoal{}, fmt.Errorf("begin adjust savings: %w", err)
	}
	defer tx.Rollback()

	nextOpening := math.Round((goal.OpeningAmount+amount)*100) / 100
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE savings_goals SET opening_amount = ? WHERE id = ?`,
		nextOpening,
		goalID,
	); err != nil {
		return models.SavingsGoal{}, fmt.Errorf("adjust savings opening: %w", err)
	}

	if walletID != nil && *walletID > 0 {
		if err := adjustWalletTx(ctx, tx, *walletID, -amount); err != nil {
			return models.SavingsGoal{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.SavingsGoal{}, fmt.Errorf("commit adjust savings: %w", err)
	}
	return s.GetSavingsGoalByID(ctx, goalID)
}

func (s *Store) hydrateSavingsGoal(ctx context.Context, goal models.SavingsGoal) (models.SavingsGoal, error) {
	members, err := s.listGoalMemberIDs(ctx, goal.ID)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	goal.MemberIDs = members
	saved, err := s.savedAmountForGoal(ctx, goal.ID)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	goal.SavedAmount = saved
	goal.ApplyYield()
	return goal, nil
}

func (s *Store) listGoalMemberIDs(ctx context.Context, goalID int) ([]int, error) {
	const query = `
		SELECT member_id
		FROM savings_goal_members
		WHERE goal_id = ?
		ORDER BY member_id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, goalID)
	if err != nil {
		return nil, fmt.Errorf("list savings goal members: %w", err)
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan savings goal member: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate savings goal members: %w", err)
	}
	return ids, nil
}

func (s *Store) savedAmountForGoal(ctx context.Context, goalID int) (float64, error) {
	const query = `
		SELECT
			COALESCE((SELECT opening_amount FROM savings_goals WHERE id = ?), 0)
			+ COALESCE((SELECT SUM(amount) FROM savings_month_amounts WHERE goal_id = ?), 0)
	`

	var total float64
	if err := s.db.QueryRowContext(ctx, query, goalID, goalID).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum savings for goal: %w", err)
	}
	return total, nil
}

func replaceGoalMembersTx(ctx context.Context, tx *sql.Tx, goalID int, memberIDs []int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM savings_goal_members WHERE goal_id = ?`, goalID); err != nil {
		return fmt.Errorf("clear savings goal members: %w", err)
	}
	for _, memberID := range normalizeMemberIDs(memberIDs) {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO savings_goal_members (goal_id, member_id) VALUES (?, ?)`,
			goalID,
			memberID,
		); err != nil {
			return fmt.Errorf("insert savings goal member: %w", err)
		}
	}
	return nil
}

func scanSavingsGoal(row interface {
	Scan(dest ...any) error
}) (models.SavingsGoal, error) {
	var (
		goal      models.SavingsGoal
		endMonth  sql.NullString
		createdAt string
	)
	err := row.Scan(
		&goal.ID,
		&goal.Name,
		&goal.TargetAmount,
		&goal.MonthlyAmount,
		&goal.Notes,
		&goal.EndKind,
		&endMonth,
		&goal.CDIAnnual,
		&goal.OpeningAmount,
		&createdAt,
	)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	goal.CreatedAt = parsed
	goal.MemberIDs = []int{}
	if endMonth.Valid {
		value := endMonth.String
		goal.EndMonth = &value
	}
	goal.ApplyYield()
	return goal, nil
}
