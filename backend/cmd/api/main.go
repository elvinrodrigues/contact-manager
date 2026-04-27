package main

import (
	"contact-manager/internal/database"
	"contact-manager/internal/handlers"
	"contact-manager/internal/middleware"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"contact-manager/internal/worker"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("[BOOT] Starting server...")

	// Load .env file for local development (silently ignored if missing)
	_ = godotenv.Load()

	log.Println("[BOOT] Connecting to database...")
	db := database.ConnectDB()

	// Run migrations on startup (idempotent — safe to re-run)
	log.Println("[BOOT] Running migrations...")
	database.RunMigrations(db, "migrations")

	// ── Env validation ───────────────────────────────────────────────────────
	log.Println("[BOOT] Validating environment...")
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("[BOOT] FATAL: JWT_SECRET environment variable is required")
	}
	if os.Getenv("RESEND_API_KEY") == "" {
		log.Println("[BOOT] WARNING: RESEND_API_KEY not set — emails will not be sent")
	}
	if strings.EqualFold(os.Getenv("DEBUG_EMAIL"), "true") {
		log.Println("[BOOT] DEBUG_EMAIL=true — emails will be printed to console instead of sent")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensures cleanup on shutdown

	worker.StartCleanupWorker(ctx, db)

	log.Println("[BOOT] Wiring up handlers...")

	// ── Auth wiring ──────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo)
	authHandler := handlers.NewAuthHandler(authService)

	// ── Contact wiring ───────────────────────────────────────────────────────
	contactRepo := repository.NewContactRepository(db)
	contactService := services.NewContactSerivce(contactRepo)
	contactHandler := handlers.NewContactHandler(contactService)

	// ── Admin wiring ─────────────────────────────────────────────────────────
	adminHandler := handlers.NewAdminHandler(userRepo)

	// ── Health wiring ────────────────────────────────────────────────────────
	healthHandler := handlers.NewHealthHandler(db)

	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	// ── Health route ─────────────────────────────────────────────────────────
	router.Get("/health", healthHandler.Health)

	// ── Public routes ────────────────────────────────────────────────────────
	router.Post("/auth/signup", authHandler.Signup)
	router.Post("/auth/login", authHandler.Login)
	router.Get("/auth/verify", authHandler.VerifyEmail)
	router.Post("/auth/forgot-password", authHandler.ForgotPassword)
	router.Post("/auth/reset-password", authHandler.ResetPassword)

	// ── Protected routes ─────────────────────────────────────────────────────
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)

		r.Get("/auth/me", authHandler.GetMe)

		r.Post("/contacts", contactHandler.CreateContact)
		r.Get("/contacts", contactHandler.ListContacts)
		r.Get("/contacts/search", contactHandler.SearchContacts)
		r.Get("/contacts/deleted", contactHandler.ListDeletedContacts)
		r.Get("/contacts/stats", contactHandler.GetStats)
		r.Get("/contacts/{id}", contactHandler.GetContactByID)
		r.Delete("/contacts/{id}", contactHandler.DeleteContactByID)
		r.Patch("/contacts/{id}/restore", contactHandler.RestoreContactByID)
		r.Put("/contacts/{id}", contactHandler.UpdateContactByID)
		r.Delete("/contacts/{id}/permanent", contactHandler.PermanentDeleteContactByID)
	})

	// ── Admin routes ─────────────────────────────────────────────────────────
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)
		r.Use(middleware.AdminOnly(db))

		r.Get("/admin/users", adminHandler.ListUsers)
		r.Patch("/admin/users/{id}/verify", adminHandler.VerifyUser)
		r.Delete("/admin/users/{id}", adminHandler.DeleteUser)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[BOOT] Server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}
