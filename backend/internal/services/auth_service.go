package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNotVerified        = errors.New("please verify your email")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrWeakPassword       = fmt.Errorf("password must be at least %d characters", MinPasswordLen)
)

const (
	// MinPasswordLen is deliberately well above the previous 6: the login
	// endpoint is now rate limited, but a short floor still makes offline
	// cracking of a leaked hash cheap.
	MinPasswordLen = 12
	// MaxPasswordLen bounds bcrypt's input. bcrypt silently truncates past 72
	// bytes, so rejecting longer input is clearer than accepting a prefix.
	MaxPasswordLen = 72

	verificationTokenTTL = 24 * time.Hour
	resetTokenTTL        = 15 * time.Minute
)

// UserRepo is the persistence surface the auth service needs.
type UserRepo interface {
	CreateUser(ctx context.Context, name, email, passwordHash string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)

	StoreVerificationToken(ctx context.Context, userID int, hash string, expiry time.Time) error
	GetUserByVerificationToken(ctx context.Context, hash string) (*models.User, error)
	MarkUserVerified(ctx context.Context, userID int) error

	StoreResetToken(ctx context.Context, userID int, hash string, expiry time.Time) error
	GetUserByResetToken(ctx context.Context, hash string) (*models.User, error)
	UpdatePassword(ctx context.Context, userID int, newPasswordHash string) error
}

// Mailer sends transactional mail. It is an interface so tests do not make
// outbound HTTP calls.
type Mailer interface {
	Send(ctx context.Context, to, subject, html string)
}

// ResendMailer is the production implementation.
type ResendMailer struct{}

func (ResendMailer) Send(ctx context.Context, to, subject, html string) {
	utils.SendEmail(ctx, to, subject, html)
}

type AuthService struct {
	Repo   UserRepo
	Mailer Mailer
}

func NewAuthService(repo UserRepo, mailer Mailer) *AuthService {
	if mailer == nil {
		mailer = ResendMailer{}
	}
	return &AuthService{Repo: repo, Mailer: mailer}
}

// Signup validates input, hashes the password, creates the user and issues a
// verification email. The account cannot log in until the link is followed.
func (s *AuthService) Signup(ctx context.Context, req models.SignupRequest) (*models.User, error) {
	name, err := validateName(req.Name)
	if err != nil {
		return nil, err
	}

	email, err := models.NormalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.Repo.CreateUser(ctx, name, email, string(hash))
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}

	s.sendVerificationEmail(ctx, user)
	return user, nil
}

// sendVerificationEmail issues a fresh token and mails the link. A failure here
// is logged but not fatal — the account exists and the link can be re-requested.
func (s *AuthService) sendVerificationEmail(ctx context.Context, user *models.User) {
	rawToken, hashedToken, err := utils.GenerateSecureToken()
	if err != nil {
		log.Printf("[AUTH] could not generate verification token for user %d: %v", user.ID, err)
		return
	}

	if err := s.Repo.StoreVerificationToken(ctx, user.ID, hashedToken, time.Now().Add(verificationTokenTTL)); err != nil {
		log.Printf("[AUTH] could not store verification token for user %d: %v", user.ID, err)
		return
	}

	link := fmt.Sprintf("%s/verify?token=%s", utils.GetBaseURL(), rawToken)
	html := fmt.Sprintf(
		`<h1>Welcome to Contact Manager</h1><p>Please click <a href="%s">here</a> to verify your email.</p>`, link)

	// Detached from the request context so the send is not cancelled when the
	// HTTP response is written, but still bounded by its own timeout.
	go s.Mailer.Send(context.WithoutCancel(ctx), user.Email, "Verify your email", html)
}

// Login verifies credentials and returns a signed JWT. The password is compared
// even when the email is unknown, so response time does not reveal whether an
// account exists.
func (s *AuthService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	email, _ := models.NormalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return "", ErrInvalidCredentials
	}

	user, err := s.Repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	storedHash := dummyBcryptHash
	if user != nil {
		storedHash = user.PasswordHash
	}
	passwordOK := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)) == nil

	if user == nil || !passwordOK {
		return "", ErrInvalidCredentials
	}
	if !user.IsVerified {
		return "", ErrNotVerified
	}

	return utils.GenerateToken(user.ID, user.TokenVersion)
}

// dummyBcryptHash is a valid bcrypt hash of a random value. Comparing against it
// for unknown accounts keeps the work factor identical on both paths.
const dummyBcryptHash = `$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy`

func (s *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return ErrInvalidToken
	}

	// Expiry is enforced in the query, so an expired token simply does not match.
	user, err := s.Repo.GetUserByVerificationToken(ctx, utils.HashToken(rawToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}
	return s.Repo.MarkUserVerified(ctx, user.ID)
}

// ForgotPassword always reports success to its caller: whether the address is
// registered must not be observable from the response.
func (s *AuthService) ForgotPassword(ctx context.Context, rawEmail string) error {
	email, err := models.NormalizeEmail(rawEmail)
	if err != nil || email == "" {
		return nil
	}

	user, err := s.Repo.GetUserByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[AUTH] forgot-password lookup failed: %v", err)
		}
		return nil
	}

	rawToken, hashedToken, err := utils.GenerateSecureToken()
	if err != nil {
		log.Printf("[AUTH] could not generate reset token for user %d: %v", user.ID, err)
		return nil
	}
	if err := s.Repo.StoreResetToken(ctx, user.ID, hashedToken, time.Now().Add(resetTokenTTL)); err != nil {
		log.Printf("[AUTH] could not store reset token for user %d: %v", user.ID, err)
		return nil
	}

	link := fmt.Sprintf("%s/reset-password?token=%s", utils.GetBaseURL(), rawToken)
	html := fmt.Sprintf(
		`<h1>Password Reset</h1><p>Reset your password by clicking <a href="%s">this link</a>.</p>`+
			`<p>This link expires in %d minutes.</p>`, link, int(resetTokenTTL.Minutes()))

	go s.Mailer.Send(context.WithoutCancel(ctx), user.Email, "Password Reset", html)
	return nil
}

// ResetPassword consumes the token and sets a new password. UpdatePassword also
// bumps token_version, which invalidates every JWT issued before this point.
func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if rawToken == "" {
		return ErrInvalidToken
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.Repo.GetUserByResetToken(ctx, utils.HashToken(rawToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.Repo.UpdatePassword(ctx, user.ID, string(newHash))
}

// GetUserByID backs the /auth/me endpoint so handlers do not reach through the
// service into the repository.
func (s *AuthService) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	user, err := s.Repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return user, nil
}

func ValidatePassword(password string) error {
	if len(password) < MinPasswordLen || len(password) > MaxPasswordLen {
		return ErrWeakPassword
	}
	return nil
}

func validateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", errors.New("name is required")
	}
	if len(name) > models.MaxNameLen {
		return "", models.ErrNameTooLong
	}
	return name, nil
}
