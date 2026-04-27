package services

import (
	"contact-manager/internal/models"
	"contact-manager/internal/utils"
	"context"
	"database/sql"
	"errors"
	"strings"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrNotDeleted = errors.New("not deleted")
)

// ─── Create ────────────────────────────────────────────────────────────────────

func (s *ContactService) CreateContact(contact models.Contact, userID int) (models.CreateContactResult, error) {
	var err error
	contact.Phone, err = utils.NormalizePhone(contact.Phone)
	var result models.CreateContactResult
	if err != nil {
		return result, err
	}

	contact.UserID = userID

	active, err := s.Repo.FindContactsByPhone(contact.Phone, userID)
	if err != nil {
		return result, err
	}
	if len(active) > 0 {
		result.Status = "duplicate"
		result.Duplicates = active
		result.Incoming = &contact
		return result, nil
	}

	deleted, _ := s.Repo.FindDeletedByPhone(contact.Phone, userID)
	if deleted != nil {
		result.Status = "deleted_duplicate"
		result.Duplicates = []models.Contact{*deleted}
		result.Incoming = &contact
		return result, nil
	}

	id, err := s.Repo.InsertContact(contact)
	if err != nil {
		return result, err
	}
	contact.ID = id
	result.Status = "created"
	result.Contact = &contact
	return result, nil
}

// ─── List ──────────────────────────────────────────────────────────────────────

func (s *ContactService) ListContacts(page, limit int, category string, userID int) (models.ListContactsResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	contacts, err := s.Repo.ListContacts(limit, offset, category, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountContacts(category, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}

	return models.ListContactsResult{
		Contacts: contacts,
		Page:     page,
		Limit:    limit,
		Total:    total,
	}, nil
}

func (s *ContactService) ListDeletedContacts(page, limit, userID int) (models.ListContactsResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	contacts, err := s.Repo.ListDeletedContacts(limit, offset, userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountDeletedContacts(userID)
	if err != nil {
		return models.ListContactsResult{}, err
	}

	return models.ListContactsResult{
		Contacts: contacts,
		Page:     page,
		Limit:    limit,
		Total:    total,
	}, nil
}

// ─── Single contact ────────────────────────────────────────────────────────────

func (s *ContactService) GetContactByID(id, userID int) (*models.Contact, error) {
	contact, err := s.Repo.GetContactByID(id, userID)
	if err != nil {
		return &models.Contact{}, err
	}
	return contact, nil
}

func (s *ContactService) DeleteContactByID(id, userID int) error {
	return s.Repo.DeleteContactByID(id, userID)
}

func (s *ContactService) PermanentDeleteContactByID(id, userID int) error {
	rowsAffected, err := s.Repo.PermanentDeleteContactByID(id, userID)
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		deletedAt, err := s.Repo.GetDeletedAtByID(id, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if deletedAt == nil {
			return ErrNotDeleted
		}
	}
	return nil
}

func (s *ContactService) RestoreContactByID(id, userID int) error {
	return s.Repo.RestoreContactByID(id, userID)
}

func (s *ContactService) UpdateContactByID(id int, name string, email *string, category, userID int) error {
	return s.Repo.UpdateContactByID(id, name, email, category, userID)
}

// ─── Search ────────────────────────────────────────────────────────────────────

func (s *ContactService) SearchContacts(ctx context.Context, query string, userID int) ([]models.Contact, error) {
	query = strings.TrimSpace(query)
	return s.Repo.SearchContacts(ctx, query, userID)
}

// ─── Stats ─────────────────────────────────────────────────────────────────────

type Stats struct {
	Total         int                   `json:"total"`
	Deleted       int                   `json:"deleted"`
	AddedThisWeek int                   `json:"added_this_week"`
	Recent        []models.Contact      `json:"recent"`
	Categories    []models.CategoryStat `json:"categories"`
}

func (s *ContactService) GetStats(userID int) (Stats, error) {
	total, deleted, added, recent, categories, err := s.Repo.GetStats(userID)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		Total:         total,
		Deleted:       deleted,
		AddedThisWeek: added,
		Recent:        recent,
		Categories:    categories,
	}, nil
}
