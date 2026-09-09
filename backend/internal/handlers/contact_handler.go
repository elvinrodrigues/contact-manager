package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"contact-manager/internal/middleware"
	"contact-manager/internal/models"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"

	"github.com/go-chi/chi/v5"
)

type ContactHandler struct {
	Service *services.ContactService
}

func NewContactHandler(service *services.ContactService) *ContactHandler {
	return &ContactHandler{Service: service}
}

type CreateContactRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	CategoryID int    `json:"category_id"`
}

// writeServiceError maps domain errors onto HTTP status codes in one place, so
// every endpoint agrees about what a validation failure or a missing record
// looks like — and so no driver or database text ever reaches a client.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound):
		utils.WriteError(w, http.StatusNotFound, utils.CodeNotFound, "contact not found")

	case errors.Is(err, services.ErrNotDeleted):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeBadRequest, "contact is not deleted")

	case errors.Is(err, services.ErrEmptyUpdate):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "no fields to update")

	case errors.Is(err, services.ErrSearchTooShort):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, services.ErrSearchTooShort.Error())

	case errors.Is(err, utils.ErrDuplicatePhone):
		utils.WriteError(w, http.StatusConflict, utils.CodeConflict,
			"a contact with this phone number already exists")

	case errors.Is(err, repository.ErrInvalidCategory):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid category")

	case errors.Is(err, utils.ErrInvalidPhone),
		errors.Is(err, models.ErrNameRequired),
		errors.Is(err, models.ErrNameTooLong),
		errors.Is(err, models.ErrEmailInvalid),
		errors.Is(err, models.ErrEmailTooLong):
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, err.Error())

	default:
		utils.WriteError(w, http.StatusInternalServerError, utils.CodeInternal, "internal server error")
	}
}

// pathID parses a positive integer path parameter.
func pathID(r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, name))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// paging reads page and limit. A parameter that is present but unparseable is an
// error rather than a silent fall back to page 1, which previously made a
// nonsense page number look like a successful request for the first page.
func paging(r *http.Request) (page, limit int, err error) {
	q := r.URL.Query()

	page, err = intParam(q.Get("page"), 1)
	if err != nil {
		return 0, 0, apiProblem{http.StatusBadRequest, utils.CodeValidation, "page must be a positive integer"}
	}
	limit, err = intParam(q.Get("limit"), services.DefaultLimit)
	if err != nil {
		return 0, 0, apiProblem{http.StatusBadRequest, utils.CodeValidation, "limit must be a positive integer"}
	}
	return page, limit, nil
}

func intParam(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, errors.New("invalid")
	}
	return v, nil
}

// categoryParam parses the optional category filter. It is an int column, so a
// non-numeric value is the client's mistake (400) rather than a failed query (500).
func categoryParam(r *http.Request) (*int, error) {
	raw := r.URL.Query().Get("category")
	if raw == "" || raw == "all" {
		return nil, nil
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return nil, apiProblem{http.StatusBadRequest, utils.CodeValidation,
			"category must be a numeric category id, or \"all\""}
	}
	return &id, nil
}

func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req CreateContactRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeProblem(w, err)
		return
	}

	contact := models.Contact{
		Name:       req.Name,
		Phone:      req.Phone,
		CategoryID: req.CategoryID,
	}
	if req.Email != "" {
		contact.Email = &req.Email
	}

	result, err := h.Service.CreateContact(r.Context(), contact, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	// Only the "created" branch actually wrote a row; the duplicate branches are
	// conflicts and must not claim 201.
	if result.Status != models.StatusCreated {
		utils.WriteJSON(w, http.StatusConflict, result, "A contact with this phone number already exists")
		return
	}
	utils.WriteJSON(w, http.StatusCreated, result, "Contact created successfully")
}

func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	page, limit, err := paging(r)
	if err != nil {
		writeProblem(w, err)
		return
	}
	categoryID, err := categoryParam(r)
	if err != nil {
		writeProblem(w, err)
		return
	}

	result, err := h.Service.ListContacts(r.Context(), page, limit, categoryID, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}

func (h *ContactHandler) ListDeletedContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	page, limit, err := paging(r)
	if err != nil {
		writeProblem(w, err)
		return
	}

	result, err := h.Service.ListDeletedContacts(r.Context(), page, limit, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, result, "Deleted contacts fetched successfully")
}

func (h *ContactHandler) GetContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid contact id")
		return
	}

	contact, err := h.Service.GetContactByID(r.Context(), id, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, contact, "Contact fetched successfully")
}

func (h *ContactHandler) UpdateContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid contact id")
		return
	}

	var in models.UpdateContactInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, err)
		return
	}

	updated, err := h.Service.UpdateContactByID(r.Context(), id, userID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	// Return the stored row so a client can see exactly what was applied.
	utils.WriteJSON(w, http.StatusOK, updated, "Contact updated successfully")
}

func (h *ContactHandler) DeleteContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid contact id")
		return
	}

	if err := h.Service.DeleteContactByID(r.Context(), id, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact moved to trash")
}

func (h *ContactHandler) RestoreContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid contact id")
		return
	}

	if err := h.Service.RestoreContactByID(r.Context(), id, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact restored successfully")
}

func (h *ContactHandler) PermanentDeleteContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, ok := pathID(r, "id")
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, utils.CodeValidation, "invalid contact id")
		return
	}

	if err := h.Service.PermanentDeleteContactByID(r.Context(), id, userID); err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact permanently deleted")
}

func (h *ContactHandler) SearchContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	page, limit, err := paging(r)
	if err != nil {
		writeProblem(w, err)
		return
	}

	result, err := h.Service.SearchContacts(r.Context(), r.URL.Query().Get("q"), page, limit, userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}

func (h *ContactHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	stats, err := h.Service.GetStats(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, stats, "Stats fetched successfully")
}
