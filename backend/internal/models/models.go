package models

import (
	"time"
)

type Contact struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id,omitempty"`
	Name       string     `json:"name"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email,omitempty"`
	CategoryID int        `json:"category_id"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	DaysRemaining *int       `json:"daysRemaining,omitempty"`
}

type CreateContactResult struct {
	Status     string     `json:"status"`
	Contact    *Contact   `json:"contact,omitempty"`
	Duplicates []Contact  `json:"duplicates,omitempty"`
	Incoming   *Contact   `json:"incoming,omitempty"`
}
type ListContactsResult struct {
	Contacts []Contact `json:"contacts"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	Total    int       `json:"total"`
}
type UpdateContactRequest struct {
	Name       string `json:"name"`
	Email      string `json:"email"`
	CategoryID int    `json:"category_id"`
}

type CategoryStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
