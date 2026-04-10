package services

import (
	"contact-manager/internal/models"
	"contact-manager/internal/utils"
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Function for creating contacts

func (s *ContactService) CreateContact(contact models.Contact) (models.CreateContactResult, error) {
	var err error
	contact.Phone, err = utils.NormalizePhone(contact.Phone)
	var result models.CreateContactResult
	if err != nil {
		return result, err
	}
	duplicates, err := s.Repo.FindContactsByPhone(contact.Phone)

	if err != nil {
		return result, err
	}
	if len(duplicates) > 0 {
		result.Status = "duplicate"
		result.Duplicates = duplicates
		return result, nil
	}
	var id int
	id, err = s.Repo.InsertContact(contact)

	if err != nil {
		if err == utils.ErrDuplicatePhone {
			duplicates, err := s.Repo.FindContactsByPhone(contact.Phone)
			if err != nil {
				return result, err
			}

			result.Status = "duplicate"
			result.Duplicates = duplicates
			return result, nil
		}

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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
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
func (s *ContactService) RestoreContactByID(id int) error {
	err := s.Repo.RestoreContactByID(id)

	if err != nil {
		return err
	}
	return nil
}
func (s *ContactService) UpdateContactByID(id int, req models.UpdateContactRequest) error {
	existing, err := s.Repo.GetContactByID(id)

	if err != nil {
		return err
	}

	name := existing.Name
	email := existing.Email
	categoryID := existing.CategoryID

	if req.Name != nil {
		name = *req.Name
	}

	if req.Email != "" {
		email = req.Email
	}
	if req.CategoryID != nil {
		categoryID = *req.CategoryID
	}

	err = s.Repo.UpdateContactByID(id, name, *email, categoryID)
	if err != nil {
		return err
	}
	return nil
}
func (s *ContactService) SearchContacts(ctx context.Context, query string) ([]models.Contact, error) {
	query = strings.TrimSpace(query)

	return s.Repo.SearchContacts(ctx, query)
}
