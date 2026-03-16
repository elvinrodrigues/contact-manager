package repository

import (
	"contact-manager/internal/models"
	"database/sql"
)

func (r *ContactRepository) InsertContact(contact models.Contact) (int, error) {
	query := `
		insert into contacts (name,phone,email,category_id)
		values ($1,$2,$3,$4)
		returning id
		`
	var id int
	if contact.CategoryID == 0 {
		contact.CategoryID = 1
	}
	err := r.DB.QueryRow(query, contact.Name, contact.Phone, contact.Email, contact.CategoryID).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *ContactRepository) FindContactsByPhone(phone string) ([]models.Contact, error) {
	query := `
		select id,name,email,category_id
		from contacts
		where phone=$1
		and deleted_at is null
	`
	var contacts []models.Contact

	rows, err := r.DB.Query(query, phone)

	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var contact models.Contact

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Email,
			&contact.CategoryID,
		)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return contacts, nil
}

func (r *ContactRepository) ListContacts(limit int, offset int) ([]models.Contact, error) {
	query := `
		select id,name,phone,email, category_id
		from contacts
		where  deleted_at is null
		order by name
		limit $1 offset $2
	`

	rows, err := r.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var contacts []models.Contact

	for rows.Next() {

		var contact models.Contact

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Phone,
			&contact.Email,
			&contact.CategoryID,
		)

		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func (r *ContactRepository) CountContacts() (int, error) {
	query := `
		select count(*) from contacts
		where deleted_at is null
	`
	var total int

	err := r.DB.QueryRow(query).Scan(&total)

	if err != nil {
		return 0, err
	}
	return total, nil
}
func (r *ContactRepository) ListDeletedContacts(limit int, offset int) ([]models.Contact, error) {
	query := `
		select id,name,phone,email, category_id
		from contacts
		where  deleted_at is not null
		order by deleted_at desc
		limit $1 offset $2
	`

	rows, err := r.DB.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var contacts []models.Contact

	for rows.Next() {

		var contact models.Contact

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Phone,
			&contact.Email,
			&contact.CategoryID,
		)

		if err != nil {
			return nil, err
		}
		contacts = append(contacts, contact)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

func (r *ContactRepository) CountDeletedContacts() (int, error) {
	query := `
		select count(*) from contacts
		where deleted_at is not null
	`
	var total int

	err := r.DB.QueryRow(query).Scan(&total)

	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *ContactRepository) GetContactByID(id int) (*models.Contact, error) {
	query := `
		select id,name,phone,email,category_id
		from contacts
		where id = $1
		and deleted_at is null
	`
	var contact models.Contact

	err := r.DB.QueryRow(query, id).Scan(&contact.ID, &contact.Name, &contact.Phone, &contact.Email, &contact.CategoryID)

	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) DeleteContactByID(id int) error {
	query := `
		update contacts
		set deleted_at = now()
		where id = $1
		and deleted_at is null
		`
	restult, err := r.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := restult.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}
func (r *ContactRepository) RestoreContactByID(id int) error {
	query := `
		update contacts
		set deleted_at = null
		where id = $1
		and deleted_at is not null
		`
	restult, err := r.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := restult.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}
