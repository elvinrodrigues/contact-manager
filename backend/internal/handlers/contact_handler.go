package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"contact-manager/internal/models"
	"contact-manager/internal/services"
	"contact-manager/internal/utils"

	"github.com/go-chi/chi/v5"
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

	var req CreateContactRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Phone == "" {
		utils.WriteError(w, http.StatusBadRequest, "name and phone number are required")
		return
	}
	var email *string
	if req.Email != "" {
		email = &req.Email
	}

	contact := models.Contact{
		Name:       req.Name,
		Phone:      req.Phone,
		Email:      email,
		CategoryID: req.CategoryID,
	}

	result, err := h.Service.CreateContact(contact)
	if err != nil {
		if err == utils.ErrInvalidPhone {
			utils.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, result, "Contact created successfully")
}

func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	result, err := h.Service.ListContacts(page, limit)

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}
func (h *ContactHandler) ListDeletedContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page <= 0 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	result, err := h.Service.ListDeletedContacts(page, limit)

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, result, "Contacts fetched successfully")
}
func (h *ContactHandler) GetContactByID(w http.ResponseWriter, r *http.Request) {

	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	result, err := h.Service.GetContactByID(id)

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
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)

	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	err = h.Service.DeleteContactByID(id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ContactHandler) RestoreContactByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)

	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	err = h.Service.RestoreContactByID(id)

	if err != nil {
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
	idStr := chi.URLParam(r, "id")
	var req models.UpdateContactRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	err = h.Service.UpdateContactByID(id, req)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "contact not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	utils.WriteJSON(w, http.StatusOK, nil, "Contact updated successfully")
}
func (h *ContactHandler) SearchContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("q")

	if len(strings.TrimSpace(query)) < 2 {
		utils.WriteError(w, http.StatusBadRequest, "search query is required")
		return
	}

	contacts, err := h.Service.SearchContacts(r.Context(), query)
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
