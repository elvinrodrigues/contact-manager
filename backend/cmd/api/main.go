package main

import (
	"contact-manager/internal/database"
	"contact-manager/internal/handlers"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"log"
	"net/http"
	"os"
	"time"

	"database/sql"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func StartCleanupJob(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			_, err := db.Exec(`
				DELETE FROM contacts
				WHERE deleted_at IS NOT NULL
				AND purge_at <= NOW()
			`)
			if err != nil {
				log.Println("cleanup error:", err)
			} else {
				log.Println("cleanup job executed")
			}
		}
	}()
}

func main() {
	db := database.ConnectDB()
	StartCleanupJob(db)

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
	router.Get("/contacts/search", handler.SearchContacts)
	router.Get("/contacts/deleted", handler.ListDeletedContacts)
	router.Get("/contacts/stats", handler.GetStats)
	router.Get("/contacts/{id}", handler.GetContactByID)
	router.Delete("/contacts/{id}", handler.DeleteContactByID)
	router.Patch("/contacts/{id}/restore", handler.RestoreContactByID)
	router.Put("/contacts/{id}", handler.UpdateContactByID)
	router.Delete("/contacts/{id}/permanent", handler.PermanentDeleteContactByID)

	log.Println("Server running on :8080")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err := http.ListenAndServe(":"+port, router)
	if err != nil {
		log.Fatal(err)
	}
}
