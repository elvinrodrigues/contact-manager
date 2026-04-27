package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"contact-manager/internal/middleware"
	"contact-manager/internal/models"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

type ContactHandler struct {
	Service *services.ContactService
}

func NewContactHandler(service *services.ContactService) *ContactHandler {
	return &ContactHandler{
		Service: service,
	}
}

type CreateContactRequest struct {
	Name       string `json:"name"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	CategoryID int    `json:"category_id"`
}

func (h *ContactHandler) CreateContact(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Phone == "" {
		utils.WriteError(w, http.StatusBadRequest, "name and phone number are required")
		return
	}

	contact := models.Contact{
		Name:       req.Name,
		Phone:      req.Phone,
		Email:      req.Email,
		CategoryID: req.CategoryID,
	}

	result, err := h.Service.CreateContact(contact, userID)
	if err != nil {
		if err == utils.ErrInvalidPhone {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == utils.ErrDuplicatePhone {
			utils.WriteError(w, http.StatusConflict, "contact with this phone number already exists")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, result, "Contact created successfully")
}

func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	query := r.URL.Query()

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}
	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	category := query.Get("category")

	result, err := h.Service.ListContacts(page, limit, category, userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}

func (h *ContactHandler) ListDeletedContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	query := r.URL.Query()

	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page <= 0 {
		page = 1
	}
	limit, err := strconv.Atoi(query.Get("limit"))
	if err != nil || limit <= 0 {
		limit = 10
	}

	result, err := h.Service.ListDeletedContacts(page, limit, userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}

func (h *ContactHandler) GetContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	result, err := h.Service.GetContactByID(id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, result, "Contact fetched successfully")
}

func (h *ContactHandler) DeleteContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	if err := h.Service.DeleteContactByID(id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "contact deleted")
}

func (h *ContactHandler) PermanentDeleteContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	if err := h.Service.PermanentDeleteContactByID(id, userID); err != nil {
		if errors.Is(err, services.ErrNotFound) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		if errors.Is(err, services.ErrNotDeleted) {
			utils.WriteError(w, http.StatusBadRequest, "contact not deleted")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "contact permanently deleted")
}

func (h *ContactHandler) RestoreContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	if err := h.Service.RestoreContactByID(id, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact restored successfully")
}

func (h *ContactHandler) UpdateContactByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req struct {
		Name       *string `json:"name"`
		Email      *string `json:"email"`
		CategoryID *int    `json:"category_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	existing, err := h.Service.GetContactByID(id, userID)
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "contact not found")
		return
	}

	// Name: nil → keep existing, non-nil → use value
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	// Email: nil → keep existing, "" → clear (NULL), "value" → set
	var emailPtr *string
	if req.Email == nil {
		emailPtr = &existing.Email
	} else if *req.Email == "" {
		emailPtr = nil
	} else {
		emailPtr = req.Email
	}

	// CategoryID: nil → keep existing, non-nil → use value
	categoryID := existing.CategoryID
	if req.CategoryID != nil {
		categoryID = *req.CategoryID
	}

	if err := h.Service.UpdateContactByID(id, name, emailPtr, categoryID, userID); err != nil {
		log.Printf("update error: %T %+v\n", err, err)

		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			utils.WriteError(w, http.StatusBadRequest, "invalid category")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact updated successfully")
}

func (h *ContactHandler) SearchContacts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	query := r.URL.Query().Get("q")

	if len(strings.TrimSpace(query)) < 2 {
		utils.WriteError(w, http.StatusBadRequest, "search query is required")
		return
	}

	contacts, err := h.Service.SearchContacts(r.Context(), query, userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "failed to search contacts")
		return
	}

	utils.WriteJSON(w, http.StatusOK, models.ListContactsResult{
		Contacts: contacts,
		Page:     0,
		Limit:    0,
		Total:    len(contacts),
	}, "Contacts fetched successfully")
}

func (h *ContactHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	stats, err := h.Service.GetStats(userID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, stats, "")
}
