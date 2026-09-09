package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"contact-manager/internal/config"
	"contact-manager/internal/database"
	"contact-manager/internal/handlers"
	"contact-manager/internal/middleware"
	"contact-manager/internal/repository"
	"contact-manager/internal/services"
	"contact-manager/internal/worker"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("[BOOT] %v", err)
	}
}

func run() error {
	log.Println("[BOOT] Starting server")

	// Load .env for local development; absent in Docker, which injects env vars.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	for _, warning := range cfg.Warnings() {
		log.Printf("[BOOT] WARNING: %s", warning)
	}

	// Cancelled on SIGINT/SIGTERM; every long-lived component derives from it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("[BOOT] Running migrations")
	if err := database.RunMigrations(ctx, db, migrationsDir()); err != nil {
		return err
	}

	// ── Wiring ───────────────────────────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	contactRepo := repository.NewContactRepository(db)

	authService := services.NewAuthService(userRepo, nil)
	contactService := services.NewContactService(contactRepo)
	adminService := services.NewAdminService(userRepo)

	authHandler := handlers.NewAuthHandler(authService)
	defer authHandler.Close()
	contactHandler := handlers.NewContactHandler(contactService)
	adminHandler := handlers.NewAdminHandler(adminService)
	healthHandler := handlers.NewHealthHandler(db)

	// Administrators are granted per-deployment from ADMIN_EMAIL rather than
	// being hardcoded into a migration.
	grantAdmin(ctx, userRepo, cfg.AdminEmail)

	workerWG := worker.StartCleanupWorker(ctx, db)

	router := buildRouter(cfg, userRepo, authHandler, contactHandler, adminHandler, healthHandler)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          log.Default(),
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("[BOOT] Listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		log.Println("[SHUTDOWN] Signal received; draining connections")
	}

	// Stop listening for signals so a second Ctrl-C can still force an exit.
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[SHUTDOWN] Graceful shutdown failed: %v", err)
		_ = srv.Close()
	}
	workerWG.Wait()

	log.Println("[SHUTDOWN] Stopped cleanly")
	return nil
}

func buildRouter(
	cfg *config.Config,
	users middleware.UserLoader,
	auth *handlers.AuthHandler,
	contacts *handlers.ContactHandler,
	admin *handlers.AdminHandler,
	health *handlers.HealthHandler,
) http.Handler {
	router := chi.NewRouter()

	// Deliberately NOT chimw.RealIP: it overwrites RemoteAddr from
	// X-Forwarded-For with no trust check, which would hand every client
	// control of the identity the rate limiters key on. utils.GetClientIP
	// consults that header only for peers in TRUSTED_PROXY_CIDRS.
	router.Use(chimw.RequestID)
	router.Use(chimw.Recoverer) // a panic becomes a 500, not a dead process
	router.Use(chimw.Timeout(30 * time.Second))
	router.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	router.Get("/health", health.Health)

	// ── Public ───────────────────────────────────────────────────────────────
	router.Route("/auth", func(r chi.Router) {
		r.Post("/signup", auth.Signup)
		r.Post("/login", auth.Login)
		r.Get("/verify", auth.VerifyEmail)
		r.Post("/forgot-password", auth.ForgotPassword)
		r.Post("/reset-password", auth.ResetPassword)
	})

	// ── Authenticated ────────────────────────────────────────────────────────
	requireAuth := middleware.RequireAuth(users)

	router.Group(func(r chi.Router) {
		r.Use(requireAuth)

		r.Get("/auth/me", auth.GetMe)

		r.Route("/contacts", func(r chi.Router) {
			r.Post("/", contacts.CreateContact)
			r.Get("/", contacts.ListContacts)
			// Literal segments are registered before "/{id}" so they are not
			// swallowed by the parameter route.
			r.Get("/search", contacts.SearchContacts)
			r.Get("/deleted", contacts.ListDeletedContacts)
			r.Get("/stats", contacts.GetStats)

			r.Get("/{id}", contacts.GetContactByID)
			r.Put("/{id}", contacts.UpdateContactByID)
			r.Delete("/{id}", contacts.DeleteContactByID)
			r.Patch("/{id}/restore", contacts.RestoreContactByID)
			r.Delete("/{id}/permanent", contacts.PermanentDeleteContactByID)
		})
	})

	// ── Admin ────────────────────────────────────────────────────────────────
	router.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Use(middleware.AdminOnly)

		r.Get("/admin/users", admin.ListUsers)
		r.Patch("/admin/users/{id}/verify", admin.VerifyUser)
		r.Delete("/admin/users/{id}", admin.DeleteUser)
	})

	return router
}

// grantAdmin promotes the configured account, if it exists. A missing account is
// only a warning: the address may be registered after first boot.
func grantAdmin(ctx context.Context, users *repository.UserRepository, email string) {
	if email == "" {
		return
	}
	if err := users.PromoteToAdmin(ctx, email); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			log.Printf("[BOOT] ADMIN_EMAIL %q is not registered yet; no role granted", email)
			return
		}
		log.Printf("[BOOT] Could not grant admin role: %v", err)
		return
	}
	log.Printf("[BOOT] Granted admin role to %s", email)
}

// migrationsDir allows the path to be overridden; the Docker image copies
// migrations next to the binary.
func migrationsDir() string {
	if dir := os.Getenv("MIGRATIONS_DIR"); dir != "" {
		return dir
	}
	return "migrations"
}
