package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/utils"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrNotDeleted = errors.New("contact is not deleted")
	// ErrEmptyUpdate is returned when an update request would change nothing.
	ErrEmptyUpdate = errors.New("no fields to update")
	// ErrSearchTooShort guards against a query that would match everything.
	ErrSearchTooShort = errors.New("search query must be at least 2 characters")
)

// Paging bounds. Without a ceiling on limit, one request can ask the database
// for the caller's entire table and hold all of it in memory.
const (
	DefaultLimit = 20
	MaxLimit     = 100
	MinSearchLen = 2
)

// ContactRepo is the persistence surface the service needs. Declaring it here,
// in the consuming package, is what lets a test substitute a fake — the service
// depends on this interface, not on a concrete *repository.ContactRepository.
type ContactRepo interface {
	InsertContact(ctx context.Context, c models.Contact) (*models.Contact, error)
	FindContactsByPhone(ctx context.Context, phone string, userID int) ([]models.Contact, error)
	FindDeletedByPhone(ctx context.Context, phone string, userID int) (*models.Contact, error)

	ListContacts(ctx context.Context, limit, offset int, categoryID *int, userID int) ([]models.Contact, error)
	CountContacts(ctx context.Context, categoryID *int, userID int) (int, error)
	ListDeletedContacts(ctx context.Context, limit, offset, userID int) ([]models.Contact, error)
	CountDeletedContacts(ctx context.Context, userID int) (int, error)

	GetContactByID(ctx context.Context, id, userID int) (*models.Contact, error)
	UpdateContact(ctx context.Context, id, userID int, in models.UpdateContactInput) (*models.Contact, error)
	DeleteContactByID(ctx context.Context, id, userID int) error
	RestoreContactByID(ctx context.Context, id, userID int) error
	PermanentDeleteContactByID(ctx context.Context, id, userID int) (int64, error)
	GetDeletedAtByID(ctx context.Context, id, userID int) (*time.Time, error)

	SearchContacts(ctx context.Context, term string, limit, offset, userID int) ([]models.Contact, error)
	CountSearchContacts(ctx context.Context, term string, userID int) (int, error)

	GetStats(ctx context.Context, userID int) (repository.Stats, error)
}

type ContactService struct {
	Repo ContactRepo
}

func NewContactService(repo ContactRepo) *ContactService {
	return &ContactService{Repo: repo}
}

// clampPaging applies the defaults and the ceiling in one place, so every list
// endpoint agrees about what page 0 or limit 1000000 means.
func clampPaging(page, limit int) (int, int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return page, limit, (page - 1) * limit
}

// ─── Create ────────────────────────────────────────────────────────────────────

// CreateContact normalizes and validates the incoming contact, then checks for
// an existing active or soft-deleted contact on the same number. The pre-check
// exists to produce a useful "here is your duplicate" response; correctness
// under concurrency comes from the partial unique index on (user_id, phone),
// whose violation is translated to ErrDuplicatePhone by the repository.
func (s *ContactService) CreateContact(ctx context.Context, contact models.Contact, userID int) (models.CreateContactResult, error) {
	var result models.CreateContactResult

	if err := contact.NormalizeAndValidate(); err != nil {
		return result, err
	}

	phone, err := utils.NormalizePhone(contact.Phone)
	if err != nil {
		return result, err
	}
	contact.Phone = phone
	contact.UserID = userID

	active, err := s.Repo.FindContactsByPhone(ctx, contact.Phone, userID)
	if err != nil {
		return result, err
	}
	if len(active) > 0 {
		result.Status = models.StatusDuplicate
		result.Duplicates = active
		result.Incoming = &contact
		return result, nil
	}

	deleted, err := s.Repo.FindDeletedByPhone(ctx, contact.Phone, userID)
	if err != nil {
		return result, err
	}
	if deleted != nil {
		result.Status = models.StatusDeletedDuplicate
		result.Duplicates = []models.Contact{*deleted}
		result.Incoming = &contact
		return result, nil
	}

	saved, err := s.Repo.InsertContact(ctx, contact)
	if err != nil {
		return result, err
	}

	result.Status = models.StatusCreated
	result.Contact = saved
	return result, nil
}

// ─── List ──────────────────────────────────────────────────────────────────────

func (s *ContactService) ListContacts(ctx context.Context, page, limit int, categoryID *int, userID int) (models.ListContactsResult, error) {
	page, limit, offset := clampPaging(page, limit)

	contacts, err := s.Repo.ListContacts(ctx, limit, offset, categoryID, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountContacts(ctx, categoryID, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}

	return models.ListContactsResult{Contacts: contacts, Page: page, Limit: limit, Total: total}, nil
}

func (s *ContactService) ListDeletedContacts(ctx context.Context, page, limit, userID int) (models.ListContactsResult, error) {
	page, limit, offset := clampPaging(page, limit)

	contacts, err := s.Repo.ListDeletedContacts(ctx, limit, offset, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountDeletedContacts(ctx, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}

	return models.ListContactsResult{Contacts: contacts, Page: page, Limit: limit, Total: total}, nil
}

// ─── Single contact ────────────────────────────────────────────────────────────

func (s *ContactService) GetContactByID(ctx context.Context, id, userID int) (*models.Contact, error) {
	contact, err := s.Repo.GetContactByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return contact, nil
}

// UpdateContactByID applies a partial update. Every supplied field is validated
// with the same rules the create path uses, so an update cannot produce a record
// that could not have been created.
func (s *ContactService) UpdateContactByID(ctx context.Context, id, userID int, in models.UpdateContactInput) (*models.Contact, error) {
	if in.IsEmpty() {
		return nil, ErrEmptyUpdate
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, models.ErrNameRequired
		}
		if len(name) > models.MaxNameLen {
			return nil, models.ErrNameTooLong
		}
		in.Name = &name
	}

	if in.Phone != nil {
		phone, err := utils.NormalizePhone(*in.Phone)
		if err != nil {
			return nil, err
		}
		in.Phone = &phone
	}

	// A pointer to the empty string means "clear this field"; anything else has
	// to survive the same validation a new address would.
	if in.Email != nil && *in.Email != "" {
		email, err := models.NormalizeEmail(*in.Email)
		if err != nil {
			return nil, err
		}
		in.Email = &email
	}

	updated, err := s.Repo.UpdateContact(ctx, id, userID, in)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func (s *ContactService) DeleteContactByID(ctx context.Context, id, userID int) error {
	err := s.Repo.DeleteContactByID(ctx, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *ContactService) RestoreContactByID(ctx context.Context, id, userID int) error {
	err := s.Repo.RestoreContactByID(ctx, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// PermanentDeleteContactByID distinguishes "no such contact" (404) from "that
// contact exists but is still active" (400), which the delete statement alone
// cannot tell apart.
func (s *ContactService) PermanentDeleteContactByID(ctx context.Context, id, userID int) error {
	rowsAffected, err := s.Repo.PermanentDeleteContactByID(ctx, id, userID)
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		return nil
	}

	deletedAt, err := s.Repo.GetDeletedAtByID(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if deletedAt == nil {
		return ErrNotDeleted
	}
	return ErrNotFound
}

// ─── Search ────────────────────────────────────────────────────────────────────

func (s *ContactService) SearchContacts(ctx context.Context, term string, page, limit, userID int) (models.ListContactsResult, error) {
	term = strings.TrimSpace(term)
	if len([]rune(term)) < MinSearchLen {
		return models.ListContactsResult{}, ErrSearchTooShort
	}

	page, limit, offset := clampPaging(page, limit)

	contacts, err := s.Repo.SearchContacts(ctx, term, limit, offset, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountSearchContacts(ctx, term, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}

	return models.ListContactsResult{Contacts: contacts, Page: page, Limit: limit, Total: total}, nil
}

// ─── Stats ─────────────────────────────────────────────────────────────────────

type Stats struct {
	Total         int                   `json:"total"`
	Deleted       int                   `json:"deleted"`
	AddedThisWeek int                   `json:"added_this_week"`
	Recent        []models.Contact      `json:"recent"`
	Categories    []models.CategoryStat `json:"categories"`
}

func (s *ContactService) GetStats(ctx context.Context, userID int) (Stats, error) {
	raw, err := s.Repo.GetStats(ctx, userID)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		Total:         raw.Total,
		Deleted:       raw.Deleted,
		AddedThisWeek: raw.AddedThisWeek,
		Recent:        raw.Recent,
		Categories:    raw.Categories,
	}, nil
}
