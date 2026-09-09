package models

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialised
	IsVerified   bool      `json:"is_verified"`
	Role         string    `json:"role"`
	TokenVersion int       `json:"-"` // bumped on password reset to revoke issued JWTs
	CreatedAt    time.Time `json:"created_at"`
}

func (u *User) IsAdmin() bool { return u.Role == RoleAdmin }

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
