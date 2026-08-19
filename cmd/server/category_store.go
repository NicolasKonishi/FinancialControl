package main

import (
	"strings"
	"sync"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// categoryStore keeps categories in memory until we add PostgreSQL in Phase 4.
type categoryStore struct {
	mu         sync.Mutex
	categories []models.Category
	nextID     int
}

func newCategoryStore() *categoryStore {
	return &categoryStore{nextID: 1}
}

func (s *categoryStore) Create(input models.CreateCategoryInput) models.Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	category := models.Category{
		ID:          s.nextID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		CreatedAt:   time.Now().UTC(),
	}
	s.nextID++
	s.categories = append(s.categories, category)
	return category
}

func (s *categoryStore) List() []models.Category {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.categories) == 0 {
		return []models.Category{}
	}

	result := make([]models.Category, len(s.categories))
	copy(result, s.categories)
	return result
}
