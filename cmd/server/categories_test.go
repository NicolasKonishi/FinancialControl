package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

func TestCreateCategoryHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantErrMsg string
	}{
		{
			name:       "POST creates category",
			body:       `{"name":"Food","description":"Groceries and restaurants"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "POST without name returns 400",
			body:       `{"name":"   "}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "name is required\n",
		},
		{
			name:       "invalid JSON returns 400",
			body:       `{invalid`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid JSON body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newCategoryStore()
			handler := categoriesHandler(store)

			req := httptest.NewRequest(http.MethodPost, "/categories", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantErrMsg != "" {
				if rec.Body.String() != tt.wantErrMsg {
					t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantErrMsg)
				}
				return
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var category models.Category
			if err := json.NewDecoder(rec.Body).Decode(&category); err != nil {
				t.Fatalf("failed to decode category: %v", err)
			}

			if category.ID != 1 {
				t.Errorf("id = %d, want 1", category.ID)
			}
			if category.Name != "Food" {
				t.Errorf("name = %q, want Food", category.Name)
			}
			if category.Description != "Groceries and restaurants" {
				t.Errorf("description = %q, want Groceries and restaurants", category.Description)
			}
			if category.CreatedAt.IsZero() {
				t.Error("created_at should not be zero")
			}
		})
	}
}

func TestListCategoriesHandler(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*categoryStore)
		wantStatus    int
		wantCount     int
		wantFirstName string
	}{
		{
			name:       "GET returns empty list",
			setup:      func(s *categoryStore) {},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "GET returns all categories",
			setup: func(s *categoryStore) {
				s.Create(models.CreateCategoryInput{Name: "Food"})
				s.Create(models.CreateCategoryInput{Name: "Transport"})
			},
			wantStatus:    http.StatusOK,
			wantCount:     2,
			wantFirstName: "Food",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newCategoryStore()
			tt.setup(store)
			handler := categoriesHandler(store)

			req := httptest.NewRequest(http.MethodGet, "/categories", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}

			var categories []models.Category
			if err := json.NewDecoder(rec.Body).Decode(&categories); err != nil {
				t.Fatalf("failed to decode categories: %v", err)
			}

			if len(categories) != tt.wantCount {
				t.Errorf("count = %d, want %d", len(categories), tt.wantCount)
			}

			if tt.wantCount > 0 && categories[0].Name != tt.wantFirstName {
				t.Errorf("first name = %q, want %q", categories[0].Name, tt.wantFirstName)
			}
		})
	}
}

func TestCategoriesHandlerMethodNotAllowed(t *testing.T) {
	store := newCategoryStore()
	handler := categoriesHandler(store)

	req := httptest.NewRequest(http.MethodDelete, "/categories", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCategoryStoreCreate(t *testing.T) {
	store := newCategoryStore()

	first := store.Create(models.CreateCategoryInput{Name: "Food"})
	second := store.Create(models.CreateCategoryInput{Name: "Transport"})

	if first.ID != 1 {
		t.Errorf("first id = %d, want 1", first.ID)
	}
	if second.ID != 2 {
		t.Errorf("second id = %d, want 2", second.ID)
	}
}

func TestCategoryStoreList(t *testing.T) {
	store := newCategoryStore()

	empty := store.List()
	if len(empty) != 0 {
		t.Errorf("empty list count = %d, want 0", len(empty))
	}

	store.Create(models.CreateCategoryInput{Name: "Food"})
	list := store.List()
	if len(list) != 1 {
		t.Errorf("list count = %d, want 1", len(list))
	}
}
