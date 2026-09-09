package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"contact-manager/internal/models"
	"contact-manager/internal/utils"
)

// contextKey is an unexported type to prevent context key collisions.
type contextKey string

const userKey contextKey = "auth_user"

// UserLoader fetches the authenticated user so the middleware can confirm the
// account still exists and that the token has not been revoked.
type UserLoader interface {
	GetUserByID(ctx context.Context, id int) (*models.User, error)
}

// RequireAuth validates the bearer token, loads the user it names, and puts that
// user in the request context. Loading the row matters: a token stays
// cryptographically valid after its account is deleted or its password reset, so
// signature verification alone is not enough to authorise a request.
func RequireAuth(users UserLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "invalid authorization format")
				return
			}

			userID, tokenVersion, err := utils.ValidateToken(strings.TrimSpace(parts[1]))
			if err != nil {
				if errors.Is(err, utils.ErrNoJWTSecret) {
					log.Printf("[AUTH] JWT_SECRET is not configured; rejecting all requests")
					utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
					return
				}
				utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "invalid or expired token")
				return
			}

			user, err := users.GetUserByID(r.Context(), userID)
			if err != nil || user == nil {
				utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "invalid or expired token")
				return
			}

			// A password reset increments token_version, so tokens minted before
			// the reset no longer match and are rejected here.
			if user.TokenVersion != tokenVersion {
				utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "session expired, please sign in again")
				return
			}

			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFrom returns the authenticated user placed in the context by RequireAuth.
func UserFrom(ctx context.Context) (*models.User, bool) {
	user, ok := ctx.Value(userKey).(*models.User)
	return user, ok && user != nil
}

// GetUserID returns the authenticated user's id, or 0 when there is none.
func GetUserID(ctx context.Context) int {
	if user, ok := UserFrom(ctx); ok {
		return user.ID
	}
	return 0
}

// AdminOnly rejects non-admin users. It reads the role from the user RequireAuth
// already loaded, so it needs no database access of its own.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFrom(r.Context())
		if !ok {
			utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "unauthorized")
			return
		}
		if !user.IsAdmin() {
			utils.WriteError(w, http.StatusForbidden, utils.CodeForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// MaxBodyBytes caps request bodies. Without it a single request can make the
// server allocate without bound.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// ContextWithUser attaches an authenticated user to a context. It exists so
// handler tests can build an authenticated request without minting a JWT.
func ContextWithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}
