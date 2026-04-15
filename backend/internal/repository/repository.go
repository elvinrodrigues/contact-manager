package repository

import (
	"contact-manager/internal/models"
	"contact-manager/internal/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
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
	fmt.Printf("RAW ERROR: %+v\n", err)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		fmt.Println("PQ ERROR CODE:", pqErr.Code)
	}
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code == "23505" {
				return 0, utils.ErrDuplicatePhone
			}
		}
		return 0, err
	}
	return id, nil
}

func (r *ContactRepository) FindContactsByPhone(phone string) ([]models.Contact, error) {
	query := `
		select id,name,phone,email,category_id,deleted_at
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
		var email sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Phone,
			&email,
			&contact.CategoryID,
			&contact.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		if email.Valid {
			contact.Email = email.String
		}
		contacts = append(contacts, contact)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return contacts, nil
}

func (r *ContactRepository) FindDeletedByPhone(phone string) (*models.Contact, error) {
	query := `
		select id,name,phone,email,category_id,deleted_at
		from contacts
		where phone=$1
		and deleted_at is not null
		limit 1
	`
	var contact models.Contact
	var email sql.NullString
	err := r.DB.QueryRow(query, phone).Scan(
		&contact.ID,
		&contact.Name,
		&contact.Phone,
		&email,
		&contact.CategoryID,
		&contact.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if email.Valid {
		contact.Email = email.String
	}

	return &contact, nil
}

func (r *ContactRepository) ListContacts(limit int, offset int) ([]models.Contact, error) {
	query := `
		select id,name,phone,email, category_id
		from contacts
		where  deleted_at is null
		order by name,id
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
		var email sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Phone,
			&email,
			&contact.CategoryID,
		)

		if err != nil {
			return nil, err
		}

		if email.Valid {
			contact.Email = email.String
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
		order by deleted_at desc,id
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
		var email sql.NullString

		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.Phone,
			&email,
			&contact.CategoryID,
		)

		if err != nil {
			return nil, err
		}

		if email.Valid {
			contact.Email = email.String
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
	var email sql.NullString

	err := r.DB.QueryRow(query, id).Scan(&contact.ID, &contact.Name, &contact.Phone, &email, &contact.CategoryID)

	if err != nil {
		return nil, err
	}
	if email.Valid {
		contact.Email = email.String
	}
	return &contact, nil
}

func (r *ContactRepository) DeleteContactByID(id int) error {
	query := `
		update contacts
		set deleted_at = now(),
		    purge_at = now() + interval '30 days'
		where id = $1
		and deleted_at is null
		`
	result, err := r.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
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
		set deleted_at = null,
		    purge_at = null
		where id = $1
		and deleted_at is not null
		`
	result, err := r.DB.Exec(query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}

func (r *ContactRepository) PermanentDeleteContactByID(id int) (int64, error) {
	query := `
		DELETE FROM contacts
		WHERE id = $1
		AND deleted_at IS NOT NULL;
		`
	result, err := r.DB.Exec(query, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *ContactRepository) GetDeletedAtByID(id int) (*time.Time, error) {
	query := `SELECT deleted_at FROM contacts WHERE id = $1`

	var deletedAt sql.NullTime

	err := r.DB.QueryRow(query, id).Scan(&deletedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	if !deletedAt.Valid {
		return nil, nil
	}

	return &deletedAt.Time, nil
}
func (r *ContactRepository) UpdateContactByID(id int, name string, email *string, category int) error {
	query := `
		UPDATE contacts
		SET name = $1,
		    email = $2,
		    category_id = $3,
		    updated_at = now()
		WHERE id = $4
		AND deleted_at IS NULL
		`
	result, err := r.DB.Exec(query, name, email, category, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}
func (r *ContactRepository) SearchContacts(ctx context.Context, query string) ([]models.Contact, error) {
	searchTerm := "%" + query + "%"

	sqlQuery := `
	SELECT id, name, phone, email, category_id
	FROM contacts
	WHERE deleted_at IS NULL
	AND (name ILIKE $1 OR phone ILIKE $1);
	`

	rows, err := r.DB.QueryContext(ctx, sqlQuery, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact

	for rows.Next() {
		var c models.Contact
		var email sql.NullString

		err := rows.Scan(
			&c.ID,
			&c.Name,
			&c.Phone,
			&email,
			&c.CategoryID,
		)
		if err != nil {
			return nil, err
		}

		if email.Valid {
			c.Email = email.String
		}
		contacts = append(contacts, c)
	}

	return contacts, rows.Err()
}
