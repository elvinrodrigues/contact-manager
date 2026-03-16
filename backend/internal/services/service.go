package services

import (
	"contact-manager/internal/models"
	"contact-manager/internal/utils"
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
func (s *ContactService) RestoreContactByID(id int) error {
	err := s.Repo.RestoreContactByID(id)

	if err != nil {
		return err
	}
	return nil
}
