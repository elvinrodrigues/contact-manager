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

// Function for creating contacts

func (s *ContactService) CreateContact(contact models.Contact) (models.CreateContactResult, error) {
	var err error
	contact.Phone, err = utils.NormalizePhone(contact.Phone)
	var result models.CreateContactResult
	if err != nil {
		return result, err
	}
	active, err := s.Repo.FindContactsByPhone(contact.Phone)
	if err != nil {
		return result, err
	}

	if len(active) > 0 {
		result.Status = "duplicate"
		result.Duplicates = active
		result.Incoming = &contact
		return result, nil
	}

	deleted, _ := s.Repo.FindDeletedByPhone(contact.Phone)
	if deleted != nil {
		result.Status = "deleted_duplicate"
		result.Duplicates = []models.Contact{*deleted}
		result.Incoming = &contact
		return result, nil
	}
	var id int
	id, err = s.Repo.InsertContact(contact)
	if err != nil {
		return result, err
	}
	contact.ID = id
	result.Status = "created"
	result.Contact = &contact
	return result, nil
}

func (s *ContactService) ListContacts(page int, limit int) (models.ListContactsResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	contacts, err := s.Repo.ListContacts(limit, offset)

	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountContacts()

	if err != nil {
		return models.ListContactsResult{}, err
	}

	result := models.ListContactsResult{
		Contacts: contacts,
		Page:     page,
		Limit:    limit,
		Total:    total,
	}
	return result, nil
}
func (s *ContactService) ListDeletedContacts(page int, limit int) (models.ListContactsResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	offset := (page - 1) * limit

	contacts, err := s.Repo.ListDeletedContacts(limit, offset)

	if err != nil {
		return models.ListContactsResult{}, err
	}
	total, err := s.Repo.CountDeletedContacts()

	if err != nil {
		return models.ListContactsResult{}, err
	}

	result := models.ListContactsResult{
		Contacts: contacts,
		Page:     page,
		Limit:    limit,
		Total:    total,
	}
	return result, nil
}

func (s *ContactService) GetContactByID(id int) (*models.Contact, error) {
	contact, err := s.Repo.GetContactByID(id)

	if err != nil {
		return &models.Contact{}, err
	}
	return contact, nil

}

func (s *ContactService) DeleteContactByID(id int) error {
	err := s.Repo.DeleteContactByID(id)

	if err != nil {
		return err
	}
	return nil
}

func (s *ContactService) PermanentDeleteContactByID(id int) error {
	rowsAffected, err := s.Repo.PermanentDeleteContactByID(id)
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		deletedAt, err := s.Repo.GetDeletedAtByID(id)
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

func (s *ContactService) RestoreContactByID(id int) error {
	err := s.Repo.RestoreContactByID(id)

	if err != nil {
		return err
	}
	return nil
}
func (s *ContactService) UpdateContactByID(id int, name string, email *string, category int) error {
	return s.Repo.UpdateContactByID(id, name, email, category)
}

func (s *ContactService) SearchContacts(ctx context.Context, query string) ([]models.Contact, error) {
	query = strings.TrimSpace(query)

	// if len(query) < 2 {
	// 	return nil, errors.New("query too short")
	// }

	return s.Repo.SearchContacts(ctx, query)
}
