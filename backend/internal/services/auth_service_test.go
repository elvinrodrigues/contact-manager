package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

// ─── Fake user repository ──────────────────────────────────────────────────────

type fakeUserRepo struct {
	mu     sync.Mutex
	byID   map[int]*models.User
	nextID int

	verificationTokens map[string]int // hash -> user id
	resetTokens        map[string]int
	expiries           map[string]time.Time
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:               map[int]*models.User{},
		nextID:             1,
		verificationTokens: map[string]int{},
		resetTokens:        map[string]int{},
		expiries:           map[string]time.Time{},
	}
}

func (f *fakeUserRepo) CreateUser(_ context.Context, name, email, hash string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.Email == email {
			return nil, repository.ErrDuplicateEmail
		}
	}
	u := &models.User{
		ID: f.nextID, Name: name, Email: email, PasswordHash: hash,
		Role: models.RoleUser, CreatedAt: time.Now(),
	}
	f.byID[u.ID] = u
	f.nextID++
	copied := *u
	return &copied, nil
}

func (f *fakeUserRepo) GetUserByEmail(_ context.Context, email string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.byID {
		if u.Email == email {
			copied := *u
			return &copied, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (f *fakeUserRepo) GetUserByID(_ context.Context, id int) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	copied := *u
	return &copied, nil
}

func (f *fakeUserRepo) StoreVerificationToken(_ context.Context, userID int, hash string, expiry time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.verificationTokens[hash] = userID
	f.expiries[hash] = expiry
	return nil
}

func (f *fakeUserRepo) GetUserByVerificationToken(_ context.Context, hash string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.verificationTokens[hash]
	// Expiry is enforced in SQL by the real repository, so an expired token is
	// simply not found — the fake mirrors that.
	if !ok || time.Now().After(f.expiries[hash]) {
		return nil, sql.ErrNoRows
	}
	copied := *f.byID[id]
	return &copied, nil
}

func (f *fakeUserRepo) MarkUserVerified(_ context.Context, userID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[userID]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.IsVerified = true
	return nil
}

func (f *fakeUserRepo) StoreResetToken(_ context.Context, userID int, hash string, expiry time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resetTokens[hash] = userID
	f.expiries[hash] = expiry
	return nil
}

func (f *fakeUserRepo) GetUserByResetToken(_ context.Context, hash string) (*models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.resetTokens[hash]
	if !ok || time.Now().After(f.expiries[hash]) {
		return nil, sql.ErrNoRows
	}
	copied := *f.byID[id]
	return &copied, nil
}

func (f *fakeUserRepo) UpdatePassword(_ context.Context, userID int, hash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[userID]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.PasswordHash = hash
	u.TokenVersion++ // revokes every JWT issued before the reset
	for h, id := range f.resetTokens {
		if id == userID {
			delete(f.resetTokens, h)
		}
	}
	return nil
}

// recordingMailer captures what would have been sent.
type recordingMailer struct {
	mu   sync.Mutex
	sent []struct{ To, Subject, HTML string }
}

func (m *recordingMailer) Send(_ context.Context, to, subject, html string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ To, Subject, HTML string }{to, subject, html})
}

func (m *recordingMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

func newAuthService(t *testing.T) (*AuthService, *fakeUserRepo, *recordingMailer) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret")
	repo := newFakeUserRepo()
	mailer := &recordingMailer{}
	return NewAuthService(repo, mailer), repo, mailer
}

const goodPassword = "correct-horse-battery"

// ─── Signup ────────────────────────────────────────────────────────────────────

func TestSignup(t *testing.T) {
	svc, repo, _ := newAuthService(t)

	user, err := svc.Signup(context.Background(), models.SignupRequest{
		Name: "  Ada  ", Email: " Ada@Example.COM ", Password: goodPassword,
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	if user.Name != "Ada" {
		t.Errorf("name = %q, want trimmed", user.Name)
	}
	if user.Email != "ada@example.com" {
		t.Errorf("email = %q, want normalized", user.Email)
	}
	// A new account must not be usable until the address is proven.
	if user.IsVerified {
		t.Error("a new account was created already verified")
	}

	stored, _ := repo.GetUserByID(context.Background(), user.ID)
	if stored.PasswordHash == goodPassword {
		t.Fatal("password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(goodPassword)); err != nil {
		t.Errorf("stored hash does not verify: %v", err)
	}
}

func TestSignupValidation(t *testing.T) {
	tests := []struct {
		name string
		req  models.SignupRequest
	}{
		{"missing name", models.SignupRequest{Email: "a@b.com", Password: goodPassword}},
		{"blank name", models.SignupRequest{Name: "   ", Email: "a@b.com", Password: goodPassword}},
		{"missing email", models.SignupRequest{Name: "Ada", Password: goodPassword}},
		{"malformed email", models.SignupRequest{Name: "Ada", Email: "nope", Password: goodPassword}},
		{"short password", models.SignupRequest{Name: "Ada", Email: "a@b.com", Password: "short"}},
		{"password past bcrypt's limit", models.SignupRequest{
			Name: "Ada", Email: "a@b.com", Password: strings.Repeat("x", 100)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, _ := newAuthService(t)
			if _, err := svc.Signup(context.Background(), tc.req); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestSignupRejectsDuplicateEmail(t *testing.T) {
	svc, _, _ := newAuthService(t)
	ctx := context.Background()
	req := models.SignupRequest{Name: "Ada", Email: "a@b.com", Password: goodPassword}

	if _, err := svc.Signup(ctx, req); err != nil {
		t.Fatalf("first signup: %v", err)
	}
	// Casing must not create a second account for the same address.
	req.Email = "A@B.com"
	if _, err := svc.Signup(ctx, req); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("error = %v, want ErrEmailTaken", err)
	}
}

// ─── Login ─────────────────────────────────────────────────────────────────────

func signupAndVerify(t *testing.T, svc *AuthService, repo *fakeUserRepo, email string) *models.User {
	t.Helper()
	ctx := context.Background()
	user, err := svc.Signup(ctx, models.SignupRequest{Name: "Ada", Email: email, Password: goodPassword})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}
	if err := repo.MarkUserVerified(ctx, user.ID); err != nil {
		t.Fatalf("verify: %v", err)
	}
	return user
}

func TestLogin(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	user := signupAndVerify(t, svc, repo, "ada@example.com")

	token, err := svc.Login(context.Background(), models.LoginRequest{
		Email: "ADA@example.com", Password: goodPassword,
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	gotID, gotVersion, err := utils.ValidateToken(token)
	if err != nil {
		t.Fatalf("issued token does not validate: %v", err)
	}
	if gotID != user.ID {
		t.Errorf("token user id = %d, want %d", gotID, user.ID)
	}
	if gotVersion != 0 {
		t.Errorf("token version = %d, want 0", gotVersion)
	}
}

func TestLoginFailures(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	signupAndVerify(t, svc, repo, "ada@example.com")
	ctx := context.Background()

	tests := []struct {
		name    string
		req     models.LoginRequest
		wantErr error
	}{
		{"wrong password", models.LoginRequest{Email: "ada@example.com", Password: "wrong-password"}, ErrInvalidCredentials},
		{"unknown account", models.LoginRequest{Email: "nobody@example.com", Password: goodPassword}, ErrInvalidCredentials},
		{"empty email", models.LoginRequest{Password: goodPassword}, ErrInvalidCredentials},
		{"empty password", models.LoginRequest{Email: "ada@example.com"}, ErrInvalidCredentials},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Login(ctx, tc.req); !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestLoginBlocksUnverifiedAccount(t *testing.T) {
	svc, _, _ := newAuthService(t)
	ctx := context.Background()

	if _, err := svc.Signup(ctx, models.SignupRequest{
		Name: "Ada", Email: "ada@example.com", Password: goodPassword,
	}); err != nil {
		t.Fatalf("signup: %v", err)
	}

	if _, err := svc.Login(ctx, models.LoginRequest{
		Email: "ada@example.com", Password: goodPassword,
	}); !errors.Is(err, ErrNotVerified) {
		t.Fatalf("error = %v, want ErrNotVerified", err)
	}
}

// ─── Verification ──────────────────────────────────────────────────────────────

func TestVerifyEmail(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	ctx := context.Background()

	user, err := svc.Signup(ctx, models.SignupRequest{
		Name: "Ada", Email: "ada@example.com", Password: goodPassword,
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	// Store a token whose plaintext this test knows.
	raw, hash, err := utils.GenerateSecureToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if err := repo.StoreVerificationToken(ctx, user.ID, hash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("store: %v", err)
	}

	if err := svc.VerifyEmail(ctx, raw); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	stored, _ := repo.GetUserByID(ctx, user.ID)
	if !stored.IsVerified {
		t.Error("account is still unverified after using a valid token")
	}
}

func TestVerifyEmailRejectsBadTokens(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	ctx := context.Background()

	user, _ := svc.Signup(ctx, models.SignupRequest{
		Name: "Ada", Email: "ada@example.com", Password: goodPassword,
	})

	expiredRaw, expiredHash, _ := utils.GenerateSecureToken()
	if err := repo.StoreVerificationToken(ctx, user.ID, expiredHash, time.Now().Add(-time.Minute)); err != nil {
		t.Fatalf("store: %v", err)
	}

	for _, tc := range []struct{ name, token string }{
		{"empty", ""},
		{"unknown", "0000000000000000000000000000000000000000000000000000000000000000"},
		{"expired", expiredRaw},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.VerifyEmail(ctx, tc.token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

// ─── Password reset ────────────────────────────────────────────────────────────

// The endpoint must behave identically for known and unknown addresses, or it
// becomes an account-enumeration oracle.
func TestForgotPasswordDoesNotRevealAccountExistence(t *testing.T) {
	svc, repo, mailer := newAuthService(t)
	ctx := context.Background()
	signupAndVerify(t, svc, repo, "ada@example.com")

	if err := svc.ForgotPassword(ctx, "ada@example.com"); err != nil {
		t.Fatalf("known address returned an error: %v", err)
	}
	if err := svc.ForgotPassword(ctx, "nobody@example.com"); err != nil {
		t.Fatalf("unknown address returned an error: %v", err)
	}
	if err := svc.ForgotPassword(ctx, "not-an-email"); err != nil {
		t.Fatalf("malformed address returned an error: %v", err)
	}

	// Only the real account should actually receive mail.
	time.Sleep(20 * time.Millisecond) // the send is dispatched in a goroutine
	if mailer.count() > 2 {           // signup email + at most one reset email
		t.Errorf("sent %d emails, want at most 2", mailer.count())
	}
}

func TestResetPassword(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	ctx := context.Background()
	user := signupAndVerify(t, svc, repo, "ada@example.com")

	raw, hash, _ := utils.GenerateSecureToken()
	if err := repo.StoreResetToken(ctx, user.ID, hash, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("store: %v", err)
	}

	const newPassword = "a-brand-new-passphrase"
	if err := svc.ResetPassword(ctx, raw, newPassword); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	if _, err := svc.Login(ctx, models.LoginRequest{Email: "ada@example.com", Password: newPassword}); err != nil {
		t.Errorf("cannot log in with the new password: %v", err)
	}
	if _, err := svc.Login(ctx, models.LoginRequest{Email: "ada@example.com", Password: goodPassword}); !errors.Is(err, ErrInvalidCredentials) {
		t.Error("the old password still works after a reset")
	}
	// Single use.
	if err := svc.ResetPassword(ctx, raw, "yet-another-passphrase"); !errors.Is(err, ErrInvalidToken) {
		t.Error("the reset token was accepted twice")
	}
}

// A reset must invalidate sessions that existed before it, or changing the
// password does not evict whoever prompted the change.
func TestResetPasswordRevokesExistingTokens(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	ctx := context.Background()
	user := signupAndVerify(t, svc, repo, "ada@example.com")

	oldToken, err := svc.Login(ctx, models.LoginRequest{Email: "ada@example.com", Password: goodPassword})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	_, oldVersion, err := utils.ValidateToken(oldToken)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	raw, hash, _ := utils.GenerateSecureToken()
	if err := repo.StoreResetToken(ctx, user.ID, hash, time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := svc.ResetPassword(ctx, raw, "a-brand-new-passphrase"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	stored, _ := repo.GetUserByID(ctx, user.ID)
	if stored.TokenVersion == oldVersion {
		t.Fatal("token_version did not change, so pre-reset tokens are still valid")
	}
}

func TestResetPasswordRejectsWeakPassword(t *testing.T) {
	svc, repo, _ := newAuthService(t)
	ctx := context.Background()
	user := signupAndVerify(t, svc, repo, "ada@example.com")

	raw, hash, _ := utils.GenerateSecureToken()
	_ = repo.StoreResetToken(ctx, user.ID, hash, time.Now().Add(time.Minute))

	if err := svc.ResetPassword(ctx, raw, "short"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("error = %v, want ErrWeakPassword", err)
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("x", MinPasswordLen)); err != nil {
		t.Errorf("a password at the minimum length was rejected: %v", err)
	}
	if err := ValidatePassword(strings.Repeat("x", MinPasswordLen-1)); err == nil {
		t.Error("a password below the minimum length was accepted")
	}
	if err := ValidatePassword(strings.Repeat("x", MaxPasswordLen+1)); err == nil {
		t.Error("a password past bcrypt's 72-byte limit was accepted")
	}
}
