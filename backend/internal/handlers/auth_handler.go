package handlers

import (
	"errors"
	"net/http"
	"time"

	"contact-manager/internal/middleware"
	"contact-manager/internal/models"
	"contact-manager/internal/ratelimit"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"
)

// Rate limits. Login and signup are unauthenticated endpoints that do expensive
// work (bcrypt, outbound mail), so they need their own budgets — previously only
// the password-reset flow was limited at all.
const (
	loginIPLimit      = 10
	loginEmailLimit   = 5
	signupIPLimit     = 5
	forgotIPLimit     = 5
	forgotEmailLimit  = 3
	rateLimitWindow   = 15 * time.Minute
	probeMitigationMs = 150
)

type AuthHandler struct {
	Service *services.AuthService

	loginIP     *ratelimit.RateLimiter
	loginEmail  *ratelimit.RateLimiter
	signupIP    *ratelimit.RateLimiter
	forgotIP    *ratelimit.RateLimiter
	forgotEmail *ratelimit.RateLimiter
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{
		Service:     service,
		loginIP:     ratelimit.NewRateLimiter(loginIPLimit, rateLimitWindow),
		loginEmail:  ratelimit.NewRateLimiter(loginEmailLimit, rateLimitWindow),
		signupIP:    ratelimit.NewRateLimiter(signupIPLimit, rateLimitWindow),
		forgotIP:    ratelimit.NewRateLimiter(forgotIPLimit, rateLimitWindow),
		forgotEmail: ratelimit.NewRateLimiter(forgotEmailLimit, rateLimitWindow),
	}
}

// Close stops the limiters' background cleanup goroutines.
func (h *AuthHandler) Close() {
	for _, rl := range []*ratelimit.RateLimiter{
		h.loginIP, h.loginEmail, h.signupIP, h.forgotIP, h.forgotEmail,
	} {
		rl.Stop()
	}
}

func tooManyRequests(w http.ResponseWriter) {
	utils.WriteError(w, http.StatusTooManyRequests, utils.CodeRateLimited,
		"too many requests, please try again later")
}

// GetMe returns the current authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFrom(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "unauthorized")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"role":        user.Role,
		"is_verified": user.IsVerified,
	}, "Current user")
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if !h.signupIP.Allow(utils.GetClientIP(r)) {
		tooManyRequests(w)
		return
	}

	var req models.SignupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, err)
		return
	}

	user, err := h.Service.Signup(r.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			// Signup necessarily reveals that an address is taken; the reset
			// flow is the one that must stay silent about it.
			utils.WriteError(w, http.StatusConflict, utils.CodeConflict, "email already registered")
			return
		}
		writeAuthValidationError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
	}, "Account created. Check your email to verify your account")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.loginIP.Allow(utils.GetClientIP(r)) {
		tooManyRequests(w)
		return
	}

	var req models.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, err)
		return
	}

	// A second budget keyed on the account stops an attacker who has many source
	// addresses from grinding a single password.
	if req.Email != "" && !h.loginEmail.Allow(req.Email) {
		tooManyRequests(w)
		return
	}

	token, err := h.Service.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidCredentials):
			utils.WriteError(w, http.StatusUnauthorized, utils.CodeUnauthorized, "invalid credentials")
		case errors.Is(err, services.ErrNotVerified):
			utils.WriteError(w, http.StatusForbidden, utils.CodeForbidden, "please verify your email")
		default:
			utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
		}
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{"token": token}, "Login successful")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "missing token")
		return
	}

	if err := h.Service.VerifyEmail(r.Context(), token); err != nil {
		if errors.Is(err, services.ErrInvalidToken) {
			utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid or expired token")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Email verified successfully")
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Even out response time so the endpoint cannot be used to probe for
	// registered addresses.
	time.Sleep(probeMitigationMs * time.Millisecond)

	if !h.forgotIP.Allow(utils.GetClientIP(r)) {
		tooManyRequests(w)
		return
	}

	var req models.ForgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, err)
		return
	}

	if req.Email != "" && !h.forgotEmail.Allow(req.Email) {
		tooManyRequests(w)
		return
	}

	_ = h.Service.ForgotPassword(r.Context(), req.Email)

	utils.WriteJSON(w, http.StatusOK, nil, "If the account exists, a reset link has been sent")
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, err)
		return
	}

	if err := h.Service.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		if errors.Is(err, services.ErrInvalidToken) {
			utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid or expired token")
			return
		}
		writeAuthValidationError(w, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil, "Password reset successfully")
}

// writeAuthValidationError reports known validation failures verbatim and
// everything else as a generic 500, so an unexpected internal error is never
// echoed back to the caller.
func writeAuthValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrWeakPassword),
		errors.Is(err, models.ErrEmailInvalid),
		errors.Is(err, models.ErrEmailTooLong),
		errors.Is(err, models.ErrNameRequired),
		errors.Is(err, models.ErrNameTooLong):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, err.Error())
	default:
		if isSimpleValidation(err) {
			utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
	}
}

// isSimpleValidation recognises the "<field> is required" errors the auth
// service raises for missing input.
func isSimpleValidation(err error) bool {
	switch err.Error() {
	case "name is required", "email is required":
		return true
	}
	return false
}
