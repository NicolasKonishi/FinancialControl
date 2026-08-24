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

// Store is the SQLite-backed persistence layer.
type Store struct {
	db *sql.DB
}

// NewStore creates a repository Store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func normalizeIcon(icon string) string {
	icon = strings.ToLower(strings.TrimSpace(icon))
	if icon == "" {
		return "other"
	}
	if _, ok := models.ValidCategoryIcons[icon]; !ok {
		return "other"
	}
	return icon
}

// CreateCategory inserts a new category and returns the persisted row.
func (s *Store) CreateCategory(ctx context.Context, input models.CreateCategoryInput) (models.Category, error) {
	const query = `
		INSERT INTO categories (name, description, icon, created_at)
		VALUES (?, ?, ?, ?)
		RETURNING id, name, description, icon, created_at
	`

	now := formatDateTime(time.Now())
	var (
		category  models.Category
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Description),
		normalizeIcon(input.Icon),
		now,
	).Scan(&category.ID, &category.Name, &category.Description, &category.Icon, &createdAt)
	if err != nil {
		return models.Category{}, fmt.Errorf("create category: %w", err)
	}

	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Category{}, err
	}
	category.CreatedAt = parsed
	return category, nil
}

// ListCategories returns all categories ordered by id.
func (s *Store) ListCategories(ctx context.Context) ([]models.Category, error) {
	const query = `
		SELECT id, name, description, icon, created_at
		FROM categories
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]models.Category, 0)
	for rows.Next() {
		var (
			category  models.Category
			createdAt string
		)
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.Icon, &createdAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		parsed, err := parseDateTime(createdAt)
		if err != nil {
			return nil, err
		}
		category.CreatedAt = parsed
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

// GetCategoryByID returns one category or ErrNotFound.
func (s *Store) GetCategoryByID(ctx context.Context, id int) (models.Category, error) {
	const query = `
		SELECT id, name, description, icon, created_at
		FROM categories
		WHERE id = ?
	`

	var (
		category  models.Category
		createdAt string
	)
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.Icon,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Category{}, ErrNotFound
	}
	if err != nil {
		return models.Category{}, fmt.Errorf("get category: %w", err)
	}

	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Category{}, err
	}
	category.CreatedAt = parsed
	return category, nil
}

// UpdateCategory updates an existing category.
func (s *Store) UpdateCategory(ctx context.Context, id int, input models.UpdateCategoryInput) (models.Category, error) {
	const query = `
		UPDATE categories
		SET name = ?, description = ?, icon = ?
		WHERE id = ?
		RETURNING id, name, description, icon, created_at
	`

	var (
		category  models.Category
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		strings.TrimSpace(input.Description),
		normalizeIcon(input.Icon),
		id,
	).Scan(&category.ID, &category.Name, &category.Description, &category.Icon, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Category{}, ErrNotFound
	}
	if err != nil {
		return models.Category{}, fmt.Errorf("update category: %w", err)
	}

	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Category{}, err
	}
	category.CreatedAt = parsed
	return category, nil
}

// DeleteCategory removes a category by id.
func (s *Store) DeleteCategory(ctx context.Context, id int) error {
	const query = `DELETE FROM categories WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete category rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
