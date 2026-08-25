package models

import "time"

// ValidCategoryIcons are the supported icon keys for the UI.
var ValidCategoryIcons = map[string]struct{}{
	"food": {}, "market": {}, "transport": {}, "home": {},
	"health": {}, "leisure": {}, "salary": {}, "freelance": {},
	"education": {}, "pets": {}, "clothing": {}, "shopping": {},
	"travel": {}, "phone": {}, "cafe": {}, "kids": {},
	"car": {}, "utilities": {}, "subscriptions": {}, "gift": {},
	"investment": {}, "other": {},
}

// Category represents a financial category (e.g. Food, Transport).
type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateCategoryInput is the JSON body expected by POST /categories.
type CreateCategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// UpdateCategoryInput is the JSON body expected by PUT /categories/{id}.
type UpdateCategoryInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}
