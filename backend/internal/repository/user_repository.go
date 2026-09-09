package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"contact-manager/internal/models"

	"github.com/lib/pq"
)

// RetentionDays is how long a soft-deleted contact stays recoverable before the
// cleanup worker purges it. The soft-delete query, the "days remaining" figure
// shown to users, and the worker all read this one constant so they cannot
// disagree about the window.
const RetentionDays = 30

var (
	ErrDuplicateEmail = errors.New("email already registered")
	// ErrUserNotFound is returned when a write matched no user row.
	ErrUserNotFound = errors.New("user not found")
)

// userColumns keeps every user projection and its Scan target in step.
const userColumns = `id, name, email, password_hash, is_verified, role, token_version, created_at`

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func scanUser(s interface{ Scan(...any) error }) (models.User, error) {
	var u models.User
	err := s.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash,
		&u.IsVerified, &u.Role, &u.TokenVersion, &u.CreatedAt)
	return u, err
}

func (r *UserRepository) CreateUser(ctx context.Context, name, email, passwordHash string) (*models.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING ` + userColumns

	u, err := scanUser(r.DB.QueryRowContext(ctx, query, name, email, passwordHash))
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ── Verification ─────────────────────────────────────────────────────────────

func (r *UserRepository) StoreVerificationToken(ctx context.Context, userID int, hash string, expiry time.Time) error {
	return r.execExpectingOneRow(ctx, `
		UPDATE users
		SET verification_token_hash = $2, verification_token_expiry = $3
		WHERE id = $1
	`, userID, hash, expiry)
}

// GetUserByVerificationToken looks a user up by token hash. The expiry is
// compared in SQL so an expired token is simply not found.
func (r *UserRepository) GetUserByVerificationToken(ctx context.Context, hash string) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRowContext(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE verification_token_hash = $1
		  AND verification_token_expiry IS NOT NULL
		  AND verification_token_expiry > now()
	`, hash))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) MarkUserVerified(ctx context.Context, userID int) error {
	return r.execExpectingOneRow(ctx, `
		UPDATE users
		SET is_verified = true,
		    verification_token_hash = NULL,
		    verification_token_expiry = NULL
		WHERE id = $1
	`, userID)
}

// ── Password reset ───────────────────────────────────────────────────────────

func (r *UserRepository) StoreResetToken(ctx context.Context, userID int, hash string, expiry time.Time) error {
	return r.execExpectingOneRow(ctx, `
		UPDATE users
		SET reset_token_hash = $2, reset_token_expiry = $3
		WHERE id = $1
	`, userID, hash, expiry)
}

func (r *UserRepository) GetUserByResetToken(ctx context.Context, hash string) (*models.User, error) {
	u, err := scanUser(r.DB.QueryRowContext(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE reset_token_hash = $1
		  AND reset_token_expiry IS NOT NULL
		  AND reset_token_expiry > now()
	`, hash))
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpdatePassword sets a new hash, clears the reset token and bumps token_version
// so every JWT issued before the reset stops validating.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID int, newPasswordHash string) error {
	return r.execExpectingOneRow(ctx, `
		UPDATE users
		SET password_hash      = $2,
		    reset_token_hash   = NULL,
		    reset_token_expiry = NULL,
		    token_version      = token_version + 1
		WHERE id = $1
	`, userID, newPasswordHash)
}

// ── Admin ────────────────────────────────────────────────────────────────────

// ListUsers returns one page of users, newest first. Passwords are not selected.
func (r *UserRepository) ListUsers(ctx context.Context, limit, offset int) ([]models.User, error) {
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, name, email, is_verified, role, created_at
		FROM users
		ORDER BY created_at DESC, id DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsVerified, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) CountUsers(ctx context.Context) (int, error) {
	var total int
	if err := r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *UserRepository) AdminVerifyUser(ctx context.Context, userID int) error {
	return r.execExpectingOneRow(ctx, `
		UPDATE users
		SET is_verified = true,
		    verification_token_hash = NULL,
		    verification_token_expiry = NULL
		WHERE id = $1
	`, userID)
}

// DeleteUser removes a user. contacts.user_id is declared ON DELETE CASCADE, so
// the user's contacts go with them in the same statement rather than in a second
// unguarded query.
func (r *UserRepository) DeleteUser(ctx context.Context, userID int) error {
	return r.execExpectingOneRow(ctx, `DELETE FROM users WHERE id = $1`, userID)
}

// PromoteToAdmin grants the admin role to an existing account. It is called at
// boot from ADMIN_EMAIL so that administrators are configured per-deployment
// instead of being hardcoded into a migration.
func (r *UserRepository) PromoteToAdmin(ctx context.Context, email string) error {
	return r.execExpectingOneRow(ctx,
		`UPDATE users SET role = $2 WHERE email = $1`, email, models.RoleAdmin)
}

func (r *UserRepository) execExpectingOneRow(ctx context.Context, query string, args ...interface{}) error {
	res, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("user repository: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}
