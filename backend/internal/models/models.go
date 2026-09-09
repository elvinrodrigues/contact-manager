package models

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// Field limits enforced before any value reaches the database. The database
// columns are TEXT, so these are the only thing standing between a request
// body and an unbounded write.
const (
	MaxNameLen  = 200
	MaxEmailLen = 320 // RFC 5321 maximum addr-spec length
)

var (
	ErrNameRequired = errors.New("name is required")
	ErrNameTooLong  = errors.New("name must be 200 characters or fewer")
	ErrEmailInvalid = errors.New("email is not a valid address")
	ErrEmailTooLong = errors.New("email must be 320 characters or fewer")
)

type Contact struct {
	ID         int     `json:"id"`
	UserID     int     `json:"-"` // ownership is implicit in the authenticated request
	Name       string  `json:"name"`
	Phone      string  `json:"phone"`
	Email      *string `json:"email,omitempty"` // nil == no email recorded (SQL NULL)
	CategoryID int     `json:"category_id"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	DaysRemaining *int       `json:"daysRemaining,omitempty"`
}

// NormalizeAndValidate trims and lowercases user-supplied fields in place, then
// checks them. It lives on the domain type so the create and update paths cannot
// drift apart the way they previously did.
func (c *Contact) NormalizeAndValidate() error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" {
		return ErrNameRequired
	}
	if len(c.Name) > MaxNameLen {
		return ErrNameTooLong
	}

	if c.Email != nil {
		email, err := NormalizeEmail(*c.Email)
		if err != nil {
			return err
		}
		if email == "" {
			c.Email = nil // an all-whitespace email means "no email"
		} else {
			c.Email = &email
		}
	}
	return nil
}

// NormalizeEmail trims and lowercases an address and validates its syntax.
// An empty input is returned as empty with no error — callers decide whether
// "absent" is acceptable for their field.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", nil
	}
	if len(email) > MaxEmailLen {
		return "", ErrEmailTooLong
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrEmailInvalid
	}
	return email, nil
}

// CreateContactResult reports which branch the create path took. Handlers map
// Status to an HTTP code: "created" -> 201, everything else -> 409.
type CreateContactResult struct {
	Status     string    `json:"status"`
	Contact    *Contact  `json:"contact,omitempty"`
	Duplicates []Contact `json:"duplicates,omitempty"`
	Incoming   *Contact  `json:"incoming,omitempty"`
}

const (
	StatusCreated          = "created"
	StatusDuplicate        = "duplicate"
	StatusDeletedDuplicate = "deleted_duplicate"
)

type ListContactsResult struct {
	Contacts []Contact `json:"contacts"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
	Total    int       `json:"total"`
}

// UpdateContactInput carries partial-update semantics: a nil field is left
// untouched, and a pointer to the empty string clears a nullable field.
type UpdateContactInput struct {
	Name       *string `json:"name"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	CategoryID *int    `json:"category_id"`
}

// IsEmpty reports whether the request would change nothing.
func (u UpdateContactInput) IsEmpty() bool {
	return u.Name == nil && u.Phone == nil && u.Email == nil && u.CategoryID == nil
}

type CategoryStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
