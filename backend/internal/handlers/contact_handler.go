package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"contact-manager/internal/models"
	"contact-manager/internal/services"

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
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Phone == "" {
		http.Error(w, "name and phone number are required", http.StatusBadRequest)
		return
	}

	contact := models.Contact{
		Name:       req.Name,
		Phone:      req.Phone,
		Email:      &req.Email,
		CategoryID: req.CategoryID,
	}

	result, err := h.Service.CreateContact(contact)
	if err != nil {
		if err.Error() == "Invalid Phone Number" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *ContactHandler) ListContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	result, err := h.Service.ListContacts(page, limit)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}
func (h *ContactHandler) ListDeletedContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	result, err := h.Service.ListDeletedContacts(page, limit)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}
func (h *ContactHandler) GetContactByID(w http.ResponseWriter, r *http.Request) {

	idStr := chi.URLParam(r, "id")

	id, _ := strconv.Atoi(idStr)

	result, err := h.Service.GetContactByID(id)

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(result)
}

func (h *ContactHandler) DeleteContactByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)

	if err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
	}

	err = h.Service.DeleteContactByID(id)

	if err != nil {
		http.Error(w, "contact not found", http.StatusNotFound)
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *ContactHandler) RestoreContactByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := strconv.Atoi(idStr)

	if err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
	}

	err = h.Service.RestoreContactByID(id)

	if err != nil {
		http.Error(w, "contact not found", http.StatusNotFound)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Contact restored successfully",
	})
}
