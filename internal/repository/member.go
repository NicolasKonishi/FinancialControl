package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// CreateMember inserts a family member.
func (s *Store) CreateMember(ctx context.Context, input models.CreateMemberInput) (models.Member, error) {
	const query = `
		INSERT INTO members (name, monthly_salary, created_at)
		VALUES (?, ?, ?)
		RETURNING id, name, monthly_salary, created_at
	`

	now := formatDateTime(time.Now())
	var (
		member    models.Member
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		input.MonthlySalary,
		now,
	).Scan(&member.ID, &member.Name, &member.MonthlySalary, &createdAt)
	if err != nil {
		return models.Member{}, fmt.Errorf("create member: %w", err)
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Member{}, err
	}
	member.CreatedAt = parsed
	return member, nil
}

// ListMembers returns all family members.
func (s *Store) ListMembers(ctx context.Context) ([]models.Member, error) {
	const query = `
		SELECT id, name, monthly_salary, created_at
		FROM members
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	members := make([]models.Member, 0)
	for rows.Next() {
		var (
			member    models.Member
			createdAt string
		)
		if err := rows.Scan(&member.ID, &member.Name, &member.MonthlySalary, &createdAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		parsed, err := parseDateTime(createdAt)
		if err != nil {
			return nil, err
		}
		member.CreatedAt = parsed
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return members, nil
}

// GetMemberByID returns one member or ErrNotFound.
func (s *Store) GetMemberByID(ctx context.Context, id int) (models.Member, error) {
	const query = `
		SELECT id, name, monthly_salary, created_at
		FROM members
		WHERE id = ?
	`

	var (
		member    models.Member
		createdAt string
	)
	err := s.db.QueryRowContext(ctx, query, id).Scan(&member.ID, &member.Name, &member.MonthlySalary, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Member{}, ErrNotFound
	}
	if err != nil {
		return models.Member{}, fmt.Errorf("get member: %w", err)
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Member{}, err
	}
	member.CreatedAt = parsed
	return member, nil
}

// UpdateMember updates a family member.
func (s *Store) UpdateMember(ctx context.Context, id int, input models.UpdateMemberInput) (models.Member, error) {
	const query = `
		UPDATE members
		SET name = ?, monthly_salary = ?
		WHERE id = ?
		RETURNING id, name, monthly_salary, created_at
	`

	var (
		member    models.Member
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		input.MonthlySalary,
		id,
	).Scan(&member.ID, &member.Name, &member.MonthlySalary, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Member{}, ErrNotFound
	}
	if err != nil {
		return models.Member{}, fmt.Errorf("update member: %w", err)
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Member{}, err
	}
	member.CreatedAt = parsed
	return member, nil
}

// DeleteMember removes a family member.
func (s *Store) DeleteMember(ctx context.Context, id int) error {
	const query = `DELETE FROM members WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete member rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListMemberSaveTargets returns save targets for year/month.
func (s *Store) ListMemberSaveTargets(ctx context.Context, year, month int) ([]models.MemberSaveTarget, error) {
	const query = `
		SELECT member_id, year, month, amount
		FROM member_save_targets
		WHERE year = ? AND month = ?
		ORDER BY member_id
	`

	rows, err := s.db.QueryContext(ctx, query, year, month)
	if err != nil {
		return nil, fmt.Errorf("list member save targets: %w", err)
	}
	defer rows.Close()

	items := make([]models.MemberSaveTarget, 0)
	for rows.Next() {
		var item models.MemberSaveTarget
		if err := rows.Scan(&item.MemberID, &item.Year, &item.Month, &item.Amount); err != nil {
			return nil, fmt.Errorf("scan member save target: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate member save targets: %w", err)
	}
	return items, nil
}

// SetMemberSaveTarget records how much a person plans to save in year/month.
func (s *Store) SetMemberSaveTarget(ctx context.Context, memberID, year, month int, amount float64) error {
	if _, err := s.GetMemberByID(ctx, memberID); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO member_save_targets (member_id, year, month, amount)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (member_id, year, month) DO UPDATE SET amount = excluded.amount
		`,
		memberID,
		year,
		month,
		amount,
	); err != nil {
		return fmt.Errorf("set member save target: %w", err)
	}
	return nil
}

// DeleteMemberSaveTarget clears a custom save target so the forecast falls back to goals.
func (s *Store) DeleteMemberSaveTarget(ctx context.Context, memberID, year, month int) error {
	if _, err := s.GetMemberByID(ctx, memberID); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM member_save_targets WHERE member_id = ? AND year = ? AND month = ?`,
		memberID,
		year,
		month,
	); err != nil {
		return fmt.Errorf("delete member save target: %w", err)
	}
	return nil
}

// SumMonthlySalaries returns the household planned salary total.
func (s *Store) SumMonthlySalaries(ctx context.Context) (float64, error) {
	const query = `SELECT COALESCE(SUM(monthly_salary), 0) FROM members`

	var total float64
	if err := s.db.QueryRowContext(ctx, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum monthly salaries: %w", err)
	}
	return total, nil
}
