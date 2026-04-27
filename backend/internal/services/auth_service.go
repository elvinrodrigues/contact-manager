package services

import (
	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/utils"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotVerified        = errors.New("please verify your email")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService struct {
	Repo *repository.UserRepository
}

func NewAuthService(repo *repository.UserRepository) *AuthService {
	return &AuthService{Repo: repo}
}

// Signup validates input, hashes the password, creates a new user, and issues a verification email.
func (s *AuthService) Signup(req models.SignupRequest) (*models.User, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return nil, errors.New("name, email, and password are required")
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		return nil, errors.New("invalid email format")
	}

	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	id, err := s.Repo.CreateUser(req.Name, req.Email, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	// Email Verification Flow
	rawToken, hashedToken, err := utils.GenerateSecureToken()
	if err == nil {
		expiry := time.Now().Add(24 * time.Hour)
		_ = s.Repo.StoreVerificationToken(id, hashedToken, expiry)

		link := fmt.Sprintf("%s/verify?token=%s", utils.GetBaseURL(), rawToken)

		if utils.IsDebugEmail() {
			log.Printf("[DEBUG] Verification link for %s: %s", req.Email, link)
		}

		html := fmt.Sprintf(`<h1>Welcome to Contact Manager</h1><p>Please click <a href="%s">here</a> to verify your email.</p>`, link)
		go utils.SendEmail(req.Email, "Verify your email", html)
	}

	return &models.User{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	}, nil
}

// Login verifies credentials and returns a signed JWT. Blocks unverified accounts.
func (s *AuthService) Login(req models.LoginRequest) (string, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Email == "" || req.Password == "" {
		return "", ErrInvalidCredentials
	}

	user, err := s.Repo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", ErrInvalidCredentials
	}

	if !user.IsVerified {
		return "", ErrNotVerified
	}

	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) VerifyEmail(rawToken string) error {
	if rawToken == "" {
		return ErrInvalidToken
	}

	hash := utils.HashToken(rawToken)
	user, expiry, err := s.Repo.GetUserByVerificationToken(hash)
	if err != nil || user == nil {
		return ErrInvalidToken
	}

	// strict expiry block
	if time.Now().After(expiry) {
		return ErrInvalidToken
	}

	return s.Repo.MarkUserVerified(user.ID)
}

func (s *AuthService) ForgotPassword(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil // generic success per hardening
	}

	user, err := s.Repo.GetUserByEmail(email)
	if err != nil || user == nil {
		return nil // generic success
	}

	rawToken, hashedToken, err := utils.GenerateSecureToken()
	if err != nil {
		return err
	}

	expiry := time.Now().Add(15 * time.Minute)
	if err := s.Repo.StoreResetToken(user.ID, hashedToken, expiry); err != nil {
		return err
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", utils.GetBaseURL(), rawToken)

	if utils.IsDebugEmail() {
		log.Printf("[DEBUG] Reset link for %s: %s", email, link)
	}

	html := fmt.Sprintf(`<h1>Password Reset</h1><p>Reset your password by clicking <a href="%s">this link</a>.</p><p>This link expires in 15 minutes.</p>`, link)
	go utils.SendEmail(user.Email, "Password Reset", html)

	return nil
}

func (s *AuthService) ResetPassword(rawToken, newPassword string) error {
	if rawToken == "" {
		return ErrInvalidToken
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hash := utils.HashToken(rawToken)
	user, expiry, err := s.Repo.GetUserByResetToken(hash)
	if err != nil || user == nil {
		return ErrInvalidToken
	}

	if time.Now().After(expiry) {
		return ErrInvalidToken
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// this also clears the tokens inside the query
	return s.Repo.UpdatePassword(user.ID, string(newHash))
}

func validatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}
