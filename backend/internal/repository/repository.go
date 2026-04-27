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

// ─── Insert ────────────────────────────────────────────────────────────────────

func (r *ContactRepository) InsertContact(contact models.Contact) (int, error) {
	query := `
		INSERT INTO contacts (name, phone, email, category_id, user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	if contact.CategoryID == 0 {
		contact.CategoryID = 1
	}

	var id int
	err := r.DB.QueryRow(query, contact.Name, contact.Phone, contact.Email, contact.CategoryID, contact.UserID).Scan(&id)
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

// ─── Duplicate detection (scoped by user) ──────────────────────────────────────

func (r *ContactRepository) FindContactsByPhone(phone string, userID int) ([]models.Contact, error) {
	query := `
		SELECT id, name, phone, email, category_id, deleted_at
		FROM contacts
		WHERE phone = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
	`
	rows, err := r.DB.Query(query, phone, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var email sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID, &c.DeletedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = email.String
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) FindDeletedByPhone(phone string, userID int) (*models.Contact, error) {
	query := `
		SELECT id, name, phone, email, category_id, deleted_at
		FROM contacts
		WHERE phone = $1
		  AND user_id = $2
		  AND deleted_at IS NOT NULL
		LIMIT 1
	`
	var c models.Contact
	var email sql.NullString
	err := r.DB.QueryRow(query, phone, userID).Scan(
		&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID, &c.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	return &c, nil
}

// ─── List / Count (active) ─────────────────────────────────────────────────────

func (r *ContactRepository) ListContacts(limit, offset int, category string, userID int) ([]models.Contact, error) {
	query := `
		SELECT c.id, c.name, c.phone, c.email, c.category_id
		FROM contacts c
		WHERE c.deleted_at IS NULL
		  AND c.user_id = $1
	`
	args := []interface{}{userID}
	nextParam := 2

	if category != "" && category != "all" {
		query += fmt.Sprintf(` AND c.category_id = $%d`, nextParam)
		args = append(args, category)
		nextParam++
	}

	query += fmt.Sprintf(` ORDER BY c.name, c.id LIMIT $%d OFFSET $%d`, nextParam, nextParam+1)
	args = append(args, limit, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var email sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = email.String
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) CountContacts(category string, userID int) (int, error) {
	query := `
		SELECT COUNT(*) FROM contacts c
		WHERE c.deleted_at IS NULL
		  AND c.user_id = $1
	`
	args := []interface{}{userID}

	if category != "" && category != "all" {
		query += ` AND c.category_id = $2`
		args = append(args, category)
	}

	var total int
	if err := r.DB.QueryRow(query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ─── List / Count (deleted) ────────────────────────────────────────────────────

func (r *ContactRepository) ListDeletedContacts(limit, offset, userID int) ([]models.Contact, error) {
	query := `
		SELECT id, name, phone, email, category_id,
		       GREATEST(30 - (current_date - deleted_at::date), 0) AS days_remaining
		FROM contacts
		WHERE deleted_at IS NOT NULL
		  AND user_id = $1
		ORDER BY deleted_at DESC, id
		LIMIT $2 OFFSET $3
	`
	rows, err := r.DB.Query(query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var email sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID, &c.DaysRemaining); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = email.String
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) CountDeletedContacts(userID int) (int, error) {
	query := `SELECT COUNT(*) FROM contacts WHERE deleted_at IS NOT NULL AND user_id = $1`

	var total int
	if err := r.DB.QueryRow(query, userID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ─── Single contact operations ─────────────────────────────────────────────────

func (r *ContactRepository) GetContactByID(id, userID int) (*models.Contact, error) {
	query := `
		SELECT id, name, phone, email, category_id
		FROM contacts
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
	`
	var c models.Contact
	var email sql.NullString
	err := r.DB.QueryRow(query, id, userID).Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID)
	if err != nil {
		return nil, err
	}
	if email.Valid {
		c.Email = email.String
	}
	return &c, nil
}

func (r *ContactRepository) DeleteContactByID(id, userID int) error {
	query := `
		UPDATE contacts
		SET deleted_at = now(),
		    purge_at = now() + interval '30 days'
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NULL
	`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ContactRepository) RestoreContactByID(id, userID int) error {
	query := `
		UPDATE contacts
		SET deleted_at = NULL,
		    purge_at = NULL
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NOT NULL
	`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ContactRepository) PermanentDeleteContactByID(id, userID int) (int64, error) {
	query := `
		DELETE FROM contacts
		WHERE id = $1
		  AND user_id = $2
		  AND deleted_at IS NOT NULL
	`
	result, err := r.DB.Exec(query, id, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *ContactRepository) GetDeletedAtByID(id, userID int) (*time.Time, error) {
	query := `SELECT deleted_at FROM contacts WHERE id = $1 AND user_id = $2`

	var deletedAt sql.NullTime
	err := r.DB.QueryRow(query, id, userID).Scan(&deletedAt)
	if err != nil {
		return nil, err
	}
	if !deletedAt.Valid {
		return nil, nil
	}
	return &deletedAt.Time, nil
}

func (r *ContactRepository) UpdateContactByID(id int, name string, email *string, category, userID int) error {
	query := `
		UPDATE contacts
		SET name = $1,
		    email = $2,
		    category_id = $3,
		    updated_at = now()
		WHERE id = $4
		  AND user_id = $5
		  AND deleted_at IS NULL
	`
	result, err := r.DB.Exec(query, name, email, category, id, userID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ─── Search ────────────────────────────────────────────────────────────────────

func (r *ContactRepository) SearchContacts(ctx context.Context, query string, userID int) ([]models.Contact, error) {
	searchTerm := "%" + query + "%"

	sqlQuery := `
		SELECT id, name, phone, email, category_id
		FROM contacts
		WHERE deleted_at IS NULL
		  AND user_id = $1
		  AND (name ILIKE $2 OR phone ILIKE $2)
	`

	rows, err := r.DB.QueryContext(ctx, sqlQuery, userID, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		var email sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = email.String
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// ─── Stats (all sub-queries scoped to user) ────────────────────────────────────

func (r *ContactRepository) GetStats(userID int) (total int, deleted int, addedThisWeek int, recent []models.Contact, categories []models.CategoryStat, err error) {
	err = r.DB.QueryRow(`
		SELECT COUNT(*) FROM contacts WHERE deleted_at IS NULL AND user_id = $1
	`, userID).Scan(&total)
	if err != nil {
		return
	}

	err = r.DB.QueryRow(`
		SELECT COUNT(*) FROM contacts WHERE deleted_at IS NOT NULL AND user_id = $1
	`, userID).Scan(&deleted)
	if err != nil {
		return
	}

	err = r.DB.QueryRow(`
		SELECT COUNT(*)
		FROM contacts
		WHERE created_at >= date_trunc('week', now())
		  AND deleted_at IS NULL
		  AND user_id = $1
	`, userID).Scan(&addedThisWeek)
	if err != nil {
		return
	}

	rows, err := r.DB.Query(`
		SELECT id, name, category_id
		FROM contacts
		WHERE deleted_at IS NULL
		  AND user_id = $1
		ORDER BY created_at DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return
	}
	defer rows.Close()

	recent = make([]models.Contact, 0)
	for rows.Next() {
		var c models.Contact
		if errScan := rows.Scan(&c.ID, &c.Name, &c.CategoryID); errScan != nil {
			err = errScan
			return
		}
		recent = append(recent, c)
	}
	if err = rows.Err(); err != nil {
		return
	}

	catRows, err := r.DB.Query(`
		SELECT cat.name, COUNT(*) AS count
		FROM contacts c
		JOIN categories cat ON c.category_id = cat.id
		WHERE c.deleted_at IS NULL
		  AND c.user_id = $1
		GROUP BY cat.name
	`, userID)
	if err != nil {
		return
	}
	defer catRows.Close()

	categories = make([]models.CategoryStat, 0)
	for catRows.Next() {
		var cs models.CategoryStat
		if errScan := catRows.Scan(&cs.Name, &cs.Count); errScan != nil {
			err = errScan
			return
		}
		categories = append(categories, cs)
	}
	err = catRows.Err()
	return
}
