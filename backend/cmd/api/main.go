package main

import (
	"contact-manager/internal/database"
	"contact-manager/internal/handlers"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func main() {
	db := database.ConnectDB()

	repo := repository.NewContactRepository(db)

	service := services.NewContactSerivce(repo)

	handler := handlers.NewContactHandler(service)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	}))

	router.Post("/contacts", handler.CreateContact)
	router.Get("/contacts", handler.ListContacts)
	router.Get("/contacts/{id}", handler.GetContactByID)
	router.Delete("/contacts/{id}", handler.DeleteContactByID)
	router.Get("/contacts/deleted", handler.ListDeletedContacts)
	router.Patch("/contacts/{id}/restore", handler.RestoreContactByID)

	log.Println("Server running on :8080")

	http.ListenAndServe(":8080", router)
}
