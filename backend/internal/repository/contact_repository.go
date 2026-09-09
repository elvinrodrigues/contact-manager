package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"contact-manager/internal/models"
	"contact-manager/internal/utils"

	"github.com/lib/pq"
)

// PostgreSQL error codes we translate into domain errors rather than leaking.
const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// ErrInvalidCategory is returned when a category_id does not exist.
var ErrInvalidCategory = errors.New("invalid category")

// contactColumns is the single source of truth for the contact projection, so a
// SELECT list and its Scan target cannot drift apart.
const contactColumns = `id, name, phone, email, category_id, created_at, updated_at`

type ContactRepository struct {
	DB *sql.DB
}

func NewContactRepository(db *sql.DB) *ContactRepository {
	return &ContactRepository{DB: db}
}

// scanContact reads one row in contactColumns order.
func scanContact(s interface{ Scan(...any) error }) (models.Contact, error) {
	var c models.Contact
	var email sql.NullString
	err := s.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return models.Contact{}, err
	}
	if email.Valid {
		c.Email = &email.String
	}
	return c, nil
}

// translateWriteErr maps driver-level constraint violations onto domain errors
// so that no layer above the repository has to import lib/pq.
func translateWriteErr(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}
	switch pqErr.Code {
	case pgUniqueViolation:
		return utils.ErrDuplicatePhone
	case pgForeignKeyViolation:
		return ErrInvalidCategory
	}
	return err
}

// ─── Insert ────────────────────────────────────────────────────────────────────

// InsertContact writes the row and returns it as stored, so the caller never has
// to guess at database-assigned values (id, defaults, timestamps).
func (r *ContactRepository) InsertContact(ctx context.Context, contact models.Contact) (*models.Contact, error) {
	query := `
		INSERT INTO contacts (name, phone, email, category_id, user_id)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, 0), 1), $5)
		RETURNING ` + contactColumns

	row := r.DB.QueryRowContext(ctx, query,
		contact.Name, contact.Phone, contact.Email, contact.CategoryID, contact.UserID)

	saved, err := scanContact(row)
	if err != nil {
		return nil, translateWriteErr(err)
	}
	return &saved, nil
}

// ─── Duplicate detection (scoped by user) ──────────────────────────────────────

func (r *ContactRepository) FindContactsByPhone(ctx context.Context, phone string, userID int) ([]models.Contact, error) {
	query := `
		SELECT ` + contactColumns + `
		FROM contacts
		WHERE phone = $1 AND user_id = $2 AND deleted_at IS NULL
		ORDER BY id
	`
	rows, err := r.DB.QueryContext(ctx, query, phone, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]models.Contact, 0)
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// FindDeletedByPhone returns a soft-deleted contact holding this phone, or
// (nil, nil) when there is none.
func (r *ContactRepository) FindDeletedByPhone(ctx context.Context, phone string, userID int) (*models.Contact, error) {
	query := `
		SELECT ` + contactColumns + `, deleted_at
		FROM contacts
		WHERE phone = $1 AND user_id = $2 AND deleted_at IS NOT NULL
		ORDER BY deleted_at DESC
		LIMIT 1
	`
	var c models.Contact
	var email sql.NullString
	var deletedAt sql.NullTime

	err := r.DB.QueryRowContext(ctx, query, phone, userID).Scan(
		&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID, &c.CreatedAt, &c.UpdatedAt, &deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if email.Valid {
		c.Email = &email.String
	}
	if deletedAt.Valid {
		c.DeletedAt = &deletedAt.Time
	}
	return &c, nil
}

// ─── List / Count (active) ─────────────────────────────────────────────────────

// ListContacts returns one page of a user's active contacts. The ORDER BY
// carries an id tiebreaker so paging is deterministic when names collide.
func (r *ContactRepository) ListContacts(ctx context.Context, limit, offset int, categoryID *int, userID int) ([]models.Contact, error) {
	query := `
		SELECT ` + contactColumns + `
		FROM contacts
		WHERE deleted_at IS NULL AND user_id = $1
	`
	args := []interface{}{userID}

	if categoryID != nil {
		args = append(args, *categoryID)
		query += fmt.Sprintf(" AND category_id = $%d", len(args))
	}

	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY name, id LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]models.Contact, 0)
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) CountContacts(ctx context.Context, categoryID *int, userID int) (int, error) {
	query := `SELECT COUNT(*) FROM contacts WHERE deleted_at IS NULL AND user_id = $1`
	args := []interface{}{userID}

	if categoryID != nil {
		args = append(args, *categoryID)
		query += fmt.Sprintf(" AND category_id = $%d", len(args))
	}

	var total int
	if err := r.DB.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ─── List / Count (deleted) ────────────────────────────────────────────────────

func (r *ContactRepository) ListDeletedContacts(ctx context.Context, limit, offset, userID int) ([]models.Contact, error) {
	query := `
		SELECT ` + contactColumns + `, deleted_at,
		       GREATEST(` + fmt.Sprint(RetentionDays) + ` - (current_date - deleted_at::date), 0) AS days_remaining
		FROM contacts
		WHERE deleted_at IS NOT NULL AND user_id = $1
		ORDER BY deleted_at DESC, id
		LIMIT $2 OFFSET $3
	`
	rows, err := r.DB.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]models.Contact, 0)
	for rows.Next() {
		var c models.Contact
		var email sql.NullString
		var deletedAt sql.NullTime
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &email, &c.CategoryID,
			&c.CreatedAt, &c.UpdatedAt, &deletedAt, &c.DaysRemaining); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = &email.String
		}
		if deletedAt.Valid {
			c.DeletedAt = &deletedAt.Time
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) CountDeletedContacts(ctx context.Context, userID int) (int, error) {
	var total int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM contacts WHERE deleted_at IS NOT NULL AND user_id = $1`,
		userID).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// ─── Single contact operations ─────────────────────────────────────────────────

// GetContactByID returns sql.ErrNoRows when the id does not exist, is deleted,
// or belongs to another user — the three cases are deliberately indistinguishable
// so the endpoint cannot be used to probe for other users' contact ids.
func (r *ContactRepository) GetContactByID(ctx context.Context, id, userID int) (*models.Contact, error) {
	query := `
		SELECT ` + contactColumns + `
		FROM contacts
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	c, err := scanContact(r.DB.QueryRowContext(ctx, query, id, userID))
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContactRepository) DeleteContactByID(ctx context.Context, id, userID int) error {
	query := `
		UPDATE contacts
		SET deleted_at = now(),
		    purge_at   = now() + ($3 || ' days')::interval,
		    updated_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
	`
	return r.execExpectingOneRow(ctx, query, id, userID, RetentionDays)
}

func (r *ContactRepository) RestoreContactByID(ctx context.Context, id, userID int) error {
	query := `
		UPDATE contacts
		SET deleted_at = NULL, purge_at = NULL, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL
	`
	if err := r.execExpectingOneRow(ctx, query, id, userID); err != nil {
		return translateWriteErr(err)
	}
	return nil
}

func (r *ContactRepository) PermanentDeleteContactByID(ctx context.Context, id, userID int) (int64, error) {
	res, err := r.DB.ExecContext(ctx, `
		DELETE FROM contacts
		WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL
	`, id, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetDeletedAtByID reports the soft-delete timestamp for a contact the user
// owns. It returns sql.ErrNoRows when no such contact exists, and (nil, nil)
// when the contact exists but is active.
func (r *ContactRepository) GetDeletedAtByID(ctx context.Context, id, userID int) (*time.Time, error) {
	var deletedAt sql.NullTime
	err := r.DB.QueryRowContext(ctx,
		`SELECT deleted_at FROM contacts WHERE id = $1 AND user_id = $2`,
		id, userID).Scan(&deletedAt)
	if err != nil {
		return nil, err
	}
	if !deletedAt.Valid {
		return nil, nil
	}
	return &deletedAt.Time, nil
}

// UpdateContact applies only the fields the caller supplied, in a single
// statement. Doing it as one atomic UPDATE (rather than read-modify-write)
// removes the lost-update race and keeps an untouched NULL column NULL.
func (r *ContactRepository) UpdateContact(ctx context.Context, id, userID int, in models.UpdateContactInput) (*models.Contact, error) {
	sets := []string{"updated_at = now()"}
	args := []interface{}{}

	add := func(column string, value interface{}) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	if in.Name != nil {
		add("name", *in.Name)
	}
	if in.Phone != nil {
		add("phone", *in.Phone)
	}
	if in.Email != nil {
		if *in.Email == "" {
			sets = append(sets, "email = NULL")
		} else {
			add("email", *in.Email)
		}
	}
	if in.CategoryID != nil {
		add("category_id", *in.CategoryID)
	}

	args = append(args, id, userID)
	query := fmt.Sprintf(`
		UPDATE contacts SET %s
		WHERE id = $%d AND user_id = $%d AND deleted_at IS NULL
		RETURNING %s
	`, strings.Join(sets, ", "), len(args)-1, len(args), contactColumns)

	updated, err := scanContact(r.DB.QueryRowContext(ctx, query, args...))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, translateWriteErr(err)
	}
	return &updated, nil
}

// execExpectingOneRow runs a statement and reports sql.ErrNoRows when it matched
// nothing, which is how "not found or not yours" reaches the handler as a 404.
func (r *ContactRepository) execExpectingOneRow(ctx context.Context, query string, args ...interface{}) error {
	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ─── Search ────────────────────────────────────────────────────────────────────

// escapeLike neutralises LIKE/ILIKE metacharacters in user input. Without this a
// two-character search for "%%" matches every row, which both defeats the
// minimum-length guard and turns one small request into a full scan.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// SearchContacts matches name, phone and email, case-insensitively, with the
// same ordering and paging guarantees as ListContacts.
func (r *ContactRepository) SearchContacts(ctx context.Context, term string, limit, offset, userID int) ([]models.Contact, error) {
	pattern := "%" + escapeLike(term) + "%"

	query := `
		SELECT ` + contactColumns + `
		FROM contacts
		WHERE deleted_at IS NULL
		  AND user_id = $1
		  AND (name ILIKE $2 ESCAPE '\' OR phone ILIKE $2 ESCAPE '\' OR email ILIKE $2 ESCAPE '\')
		ORDER BY name, id
		LIMIT $3 OFFSET $4
	`
	rows, err := r.DB.QueryContext(ctx, query, userID, pattern, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contacts := make([]models.Contact, 0)
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func (r *ContactRepository) CountSearchContacts(ctx context.Context, term string, userID int) (int, error) {
	pattern := "%" + escapeLike(term) + "%"

	var total int
	err := r.DB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM contacts
		WHERE deleted_at IS NULL
		  AND user_id = $1
		  AND (name ILIKE $2 ESCAPE '\' OR phone ILIKE $2 ESCAPE '\' OR email ILIKE $2 ESCAPE '\')
	`, userID, pattern).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// ─── Stats ─────────────────────────────────────────────────────────────────────

// Stats is the aggregate dashboard payload, all of it scoped to one user.
type Stats struct {
	Total         int
	Deleted       int
	AddedThisWeek int
	Recent        []models.Contact
	Categories    []models.CategoryStat
}

func (r *ContactRepository) GetStats(ctx context.Context, userID int) (Stats, error) {
	stats := Stats{
		Recent:     make([]models.Contact, 0),
		Categories: make([]models.CategoryStat, 0),
	}

	// One round trip for the three counters instead of three.
	err := r.DB.QueryRowContext(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE deleted_at IS NULL),
		    COUNT(*) FILTER (WHERE deleted_at IS NOT NULL),
		    COUNT(*) FILTER (WHERE deleted_at IS NULL
		                       AND created_at >= date_trunc('week', now()))
		FROM contacts
		WHERE user_id = $1
	`, userID).Scan(&stats.Total, &stats.Deleted, &stats.AddedThisWeek)
	if err != nil {
		return Stats{}, err
	}

	// id breaks ties so "recent" is stable across identical timestamps.
	rows, err := r.DB.QueryContext(ctx, `
		SELECT `+contactColumns+`
		FROM contacts
		WHERE deleted_at IS NULL AND user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()

	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return Stats{}, err
		}
		stats.Recent = append(stats.Recent, c)
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}

	catRows, err := r.DB.QueryContext(ctx, `
		SELECT cat.name, COUNT(*) AS count
		FROM contacts c
		JOIN categories cat ON c.category_id = cat.id
		WHERE c.deleted_at IS NULL AND c.user_id = $1
		GROUP BY cat.name
		ORDER BY count DESC, cat.name
	`, userID)
	if err != nil {
		return Stats{}, err
	}
	defer catRows.Close()

	for catRows.Next() {
		var cs models.CategoryStat
		if err := catRows.Scan(&cs.Name, &cs.Count); err != nil {
			return Stats{}, err
		}
		stats.Categories = append(stats.Categories, cs)
	}
	return stats, catRows.Err()
}
