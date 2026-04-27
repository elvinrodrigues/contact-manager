package middleware

import (
	"contact-manager/internal/utils"
	"context"
	"database/sql"
	"net/http"
	"strings"
)

// contextKey is an unexported type to prevent context key collisions.
type contextKey string

const userIDKey contextKey = "user_id"

// RequireAuth validates the JWT from the Authorization header and injects
// the user_id into the request context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			utils.WriteError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			utils.WriteError(w, http.StatusUnauthorized, "invalid authorization format")
			return
		}

		userID, err := utils.ValidateToken(parts[1])
		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the authenticated user's ID from the request context.
func GetUserID(ctx context.Context) int {
	id, _ := ctx.Value(userIDKey).(int)
	return id
}

// AdminOnly is middleware that rejects non-admin users with 403.
// Must be used AFTER RequireAuth so user_id is already in context.
func AdminOnly(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == 0 {
				utils.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			var role string
			err := db.QueryRow("SELECT role FROM users WHERE id = $1", userID).Scan(&role)
			if err != nil || role != "admin" {
				utils.WriteError(w, http.StatusForbidden, "admin access required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
