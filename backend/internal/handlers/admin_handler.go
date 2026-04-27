package handlers

import (
	"net/http"
	"strconv"

	"contact-manager/internal/repository"
	"contact-manager/internal/utils"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	Repo *repository.UserRepository
}

func NewAdminHandler(repo *repository.UserRepository) *AdminHandler {
	return &AdminHandler{Repo: repo}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Repo.ListAllUsers()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	utils.WriteJSON(w, http.StatusOK, users, "users fetched")
}

func (h *AdminHandler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.Repo.AdminVerifyUser(id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil, "user verified")
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.Repo.DeleteUser(id); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	utils.WriteJSON(w, http.StatusOK, nil, "user deleted")
}
