package models

import "time"

// Category represents a financial category (e.g. Food, Transport).
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateCategoryInput is the JSON body expected by POST /categories.
// It is separate from Category because the client does not send id or created_at.
type CreateCategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
