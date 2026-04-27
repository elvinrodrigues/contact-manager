package repository

import (
	"contact-manager/internal/models"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

var ErrDuplicateEmail = errors.New("email already registered")

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(name, email, passwordHash string) (int, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	var id int
	err := r.DB.QueryRow(query, name, email, passwordHash).Scan(&id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return 0, ErrDuplicateEmail
		}
		return 0, err
	}
	return id, nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, is_verified, role, created_at
		FROM users
		WHERE email = $1
	`
	var user models.User
	err := r.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) StoreVerificationToken(userID int, hash string, expiry time.Time) error {
	query := `
		UPDATE users 
		SET verification_token_hash = $2, verification_token_expiry = $3
		WHERE id = $1
	`
	_, err := r.DB.Exec(query, userID, hash, expiry)
	return err
}

func (r *UserRepository) GetUserByVerificationToken(hash string) (*models.User, time.Time, error) {
	query := `
		SELECT id, name, email, password_hash, is_verified, role, created_at, verification_token_expiry
		FROM users
		WHERE verification_token_hash = $1
	`
	var user models.User
	var expiry sql.NullTime
	err := r.DB.QueryRow(query, hash).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Role,
		&user.CreatedAt,
		&expiry,
	)
	if err != nil {
		return nil, time.Time{}, err
	}
	return &user, expiry.Time, nil
}

func (r *UserRepository) MarkUserVerified(userID int) error {
	query := `
		UPDATE users 
		SET is_verified = true,
		    verification_token_hash = NULL,
		    verification_token_expiry = NULL
		WHERE id = $1
	`
	_, err := r.DB.Exec(query, userID)
	return err
}

func (r *UserRepository) StoreResetToken(userID int, hash string, expiry time.Time) error {
	query := `
		UPDATE users 
		SET reset_token_hash = $2, reset_token_expiry = $3
		WHERE id = $1
	`
	_, err := r.DB.Exec(query, userID, hash, expiry)
	return err
}

func (r *UserRepository) GetUserByResetToken(hash string) (*models.User, time.Time, error) {
	query := `
		SELECT id, name, email, password_hash, is_verified, role, created_at, reset_token_expiry
		FROM users
		WHERE reset_token_hash = $1
	`
	var user models.User
	var expiry sql.NullTime
	err := r.DB.QueryRow(query, hash).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Role,
		&user.CreatedAt,
		&expiry,
	)
	if err != nil {
		return nil, time.Time{}, err
	}
	return &user, expiry.Time, nil
}

func (r *UserRepository) UpdatePassword(userID int, newPasswordHash string) error {
	query := `
		UPDATE users 
		SET password_hash = $2,
		    reset_token_hash = NULL,
		    reset_token_expiry = NULL
		WHERE id = $1
	`
	_, err := r.DB.Exec(query, userID, newPasswordHash)
	return err
}

// ── Admin methods ────────────────────────────────────────────────────────────

func (r *UserRepository) GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, name, email, password_hash, is_verified, role, created_at
		FROM users
		WHERE id = $1
	`
	var user models.User
	err := r.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ListAllUsers() ([]models.User, error) {
	query := `
		SELECT id, name, email, is_verified, role, created_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.IsVerified, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) AdminVerifyUser(userID int) error {
	query := `UPDATE users SET is_verified = true WHERE id = $1`
	_, err := r.DB.Exec(query, userID)
	return err
}

func (r *UserRepository) DeleteUser(userID int) error {
	// First remove their contacts to avoid FK violations
	_, _ = r.DB.Exec(`DELETE FROM contacts WHERE user_id = $1`, userID)
	_, err := r.DB.Exec(`DELETE FROM users WHERE id = $1`, userID)
	return err
}

