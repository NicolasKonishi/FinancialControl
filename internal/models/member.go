package models

import "time"

// Member is a family member who can have a monthly salary and own transactions.
type Member struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	MonthlySalary float64   `json:"monthly_salary"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateMemberInput is the JSON body for POST /members.
type CreateMemberInput struct {
	Name          string  `json:"name"`
	MonthlySalary float64 `json:"monthly_salary"`
}

// UpdateMemberInput is the JSON body for PUT /members/{id}.
type UpdateMemberInput struct {
	Name          string  `json:"name"`
	MonthlySalary float64 `json:"monthly_salary"`
}

// MemberSaveTarget is how much a person plans to set aside in a month.
type MemberSaveTarget struct {
	MemberID int     `json:"member_id"`
	Year     int     `json:"year"`
	Month    int     `json:"month"`
	Amount   float64 `json:"amount"`
}

// SetMemberSaveTargetInput is the JSON body for PUT /members/{id}/save-target.
type SetMemberSaveTargetInput struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Amount float64 `json:"amount"`
}
