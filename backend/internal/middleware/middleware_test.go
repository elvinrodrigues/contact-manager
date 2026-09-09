package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"contact-manager/internal/models"
	"contact-manager/internal/utils"
)

type stubLoader struct {
	users map[int]*models.User
}

func (s stubLoader) GetUserByID(_ context.Context, id int) (*models.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return u, nil
}

// okHandler records that the request made it through the middleware.
func okHandler(reached *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusOK)
	})
}

func serve(t *testing.T, h http.Handler, authHeader string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/contacts", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec, rec.Code == http.StatusOK
}

func TestRequireAuthAcceptsValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	user := &models.User{ID: 7, Email: "ada@example.com", Role: models.RoleUser}
	loader := stubLoader{users: map[int]*models.User{7: user}}

	token, err := utils.GenerateToken(7, 0)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	var reached bool
	var seenID int
	h := RequireAuth(loader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		seenID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec, _ := serve(t, h, "Bearer "+token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !reached {
		t.Fatal("the handler was never reached")
	}
	if seenID != 7 {
		t.Errorf("context user id = %d, want 7", seenID)
	}
}

func TestRequireAuthRejectsBadCredentials(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	loader := stubLoader{users: map[int]*models.User{7: {ID: 7, Role: models.RoleUser}}}
	valid, _ := utils.GenerateToken(7, 0)

	tests := []struct {
		name   string
		header string
	}{
		{"no header", ""},
		{"bearer with no token", "Bearer"},
		{"bearer with empty token", "Bearer "},
		{"wrong scheme", "Basic " + valid},
		{"garbage token", "Bearer not.a.token"},
		{"tampered token", "Bearer " + valid[:len(valid)-4] + "AAAA"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reached bool
			h := RequireAuth(loader)(okHandler(&reached))

			rec, _ := serve(t, h, tc.header)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", rec.Code)
			}
			if reached {
				t.Error("the protected handler ran despite a rejected token")
			}
		})
	}
}

// A signature stays valid after the account is gone, so the middleware has to
// confirm the user still exists rather than trusting the claim alone.
func TestRequireAuthRejectsTokenForDeletedUser(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	token, _ := utils.GenerateToken(999, 0)
	loader := stubLoader{users: map[int]*models.User{}} // user no longer exists

	var reached bool
	h := RequireAuth(loader)(okHandler(&reached))

	rec, _ := serve(t, h, "Bearer "+token)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if reached {
		t.Error("a token for a deleted account reached the handler")
	}
}

// After a password reset the stored token_version moves on, and tokens minted
// before it must stop working.
func TestRequireAuthRejectsStaleTokenVersion(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	token, _ := utils.GenerateToken(7, 0)
	loader := stubLoader{users: map[int]*models.User{7: {ID: 7, TokenVersion: 1}}}

	var reached bool
	h := RequireAuth(loader)(okHandler(&reached))

	rec, _ := serve(t, h, "Bearer "+token)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if reached {
		t.Error("a pre-reset token reached the handler")
	}
}

func TestAdminOnly(t *testing.T) {
	tests := []struct {
		name       string
		user       *models.User
		wantStatus int
	}{
		{"admin passes", &models.User{ID: 1, Role: models.RoleAdmin}, http.StatusOK},
		{"regular user is forbidden", &models.User{ID: 2, Role: models.RoleUser}, http.StatusForbidden},
		{"unknown role is forbidden", &models.User{ID: 3, Role: "superuser"}, http.StatusForbidden},
		{"no user is unauthorized", nil, http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reached bool
			h := AdminOnly(okHandler(&reached))

			req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
			if tc.user != nil {
				req = req.WithContext(context.WithValue(req.Context(), userKey, tc.user))
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if reached != (tc.wantStatus == http.StatusOK) {
				t.Errorf("handler reached = %v, want %v", reached, tc.wantStatus == http.StatusOK)
			}
		})
	}
}

// Error bodies must use the shared envelope, never a bare string.
func TestAuthErrorsUseTheStandardEnvelope(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	var reached bool
	h := RequireAuth(stubLoader{users: map[int]*models.User{}})(okHandler(&reached))

	rec, _ := serve(t, h, "")

	var body struct {
		Data    interface{}                     `json:"data"`
		Error   *struct{ Code, Message string } `json:"error"`
		Message string                          `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body.Error == nil {
		t.Fatal("error object missing from the response")
	}
	if body.Error.Code != utils.CodeUnauthorized {
		t.Errorf("code = %q, want %q", body.Error.Code, utils.CodeUnauthorized)
	}
}

func TestMaxBodyBytes(t *testing.T) {
	const limit = 32

	h := MaxBodyBytes(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		if _, err := r.Body.Read(buf); err != nil && err.Error() != "EOF" {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/contacts",
		strings.NewReader(strings.Repeat("x", limit*10)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 for a body past the limit", rec.Code)
	}
}
