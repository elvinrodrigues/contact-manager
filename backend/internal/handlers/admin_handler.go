package handlers

import (
	"errors"
	"net/http"

	"contact-manager/internal/middleware"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"
)

type AdminHandler struct {
	Service *services.AdminService
}

func NewAdminHandler(service *services.AdminService) *AdminHandler {
	return &AdminHandler{Service: service}
}

func writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.WriteError(w, http.StatusNotFound, utils.CodeNotFound, "user not found")
	case errors.Is(err, services.ErrCannotDeleteSelf):
		utils.WriteError(w, http.StatusConflict, utils.CodeConflict,
			"you cannot delete your own account")
	default:
		utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
	}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page, limit, err := paging(r)
	if err != nil {
		writeProblem(w, err)
		return
	}

	result, err := h.Service.ListUsers(r.Context(), page, limit)
	if err != nil {
		writeAdminError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, result, "Users fetched successfully")
}

func (h *AdminHandler) VerifyUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid user id")
		return
	}

	if err := h.Service.VerifyUser(r.Context(), id); err != nil {
		writeAdminError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "User verified")
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid user id")
		return
	}

	// An admin deleting themselves can lock the last administrator out of the
	// deployment, so it is refused rather than silently allowed.
	if err := h.Service.DeleteUser(r.Context(), id, middleware.GetUserID(r.Context())); err != nil {
		writeAdminError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "User deleted")
}
