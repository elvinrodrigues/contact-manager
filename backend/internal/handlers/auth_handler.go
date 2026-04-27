package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"contact-manager/internal/middleware"
	"contact-manager/internal/models"
	"contact-manager/internal/ratelimit"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"
)

type AuthHandler struct {
	Service         *services.AuthService
	ipRateLimiter   *ratelimit.RateLimiter
	emailRateLimiter *ratelimit.RateLimiter
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{
		Service:         service,
		ipRateLimiter:   ratelimit.NewRateLimiter(5, 15*time.Minute),
		emailRateLimiter: ratelimit.NewRateLimiter(3, 15*time.Minute),
	}
}

// GetMe returns the current authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	user, err := h.Service.Repo.GetUserByID(userID)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "user not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
		"role":  user.Role,
	}, "current user")
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req models.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.Service.Signup(req)
	if err != nil {
		if errors.Is(err, services.ErrEmailTaken) {
			utils.WriteError(w, http.StatusBadRequest, "email already registered")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
	}, "user created. Please check your email to verify your account")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.Service.Login(req)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if errors.Is(err, services.ErrNotVerified) {
			utils.WriteError(w, http.StatusForbidden, "please verify your email")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
	}, "login successful")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.WriteError(w, http.StatusBadRequest, "missing token")
		return
	}

	if err := h.Service.VerifyEmail(token); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil, "email verified successfully")
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	// Small artificial delay to mitigate timing attacks on account probing
	time.Sleep(150 * time.Millisecond)

	var req models.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := utils.GetClientIP(r)
	if !h.ipRateLimiter.Allow(ip) {
		utils.WriteError(w, http.StatusTooManyRequests, "too many requests, please try again later")
		return
	}

	if req.Email != "" && !h.emailRateLimiter.Allow(req.Email) {
		utils.WriteError(w, http.StatusTooManyRequests, "too many requests for this email")
		return
	}

	_ = h.Service.ForgotPassword(req.Email)

	// Consistently generic response, never leaking whether the account exists
	utils.WriteJSON(w, http.StatusOK, nil, "If the account exists, a reset link has been sent")
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.Service.ResetPassword(req.Token, req.NewPassword); err != nil {
		if errors.Is(err, services.ErrInvalidToken) {
			utils.WriteError(w, http.StatusBadRequest, "invalid or expired token")
			return
		}
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil, "password reset successfully")
}
