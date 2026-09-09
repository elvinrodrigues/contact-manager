package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

// Connection pool settings. Go's defaults are unlimited open connections and two
// idle ones, which both exhausts the server's max_connections under load and
// churns through connection setup the rest of the time.
const (
	maxOpenConns    = 25
	maxIdleConns    = 25
	connMaxLifetime = 5 * time.Minute
	connMaxIdleTime = 2 * time.Minute

	connectAttempts = 10
	connectBackoff  = 2 * time.Second
	pingTimeout     = 5 * time.Second
)

// Connect builds a Postgres DSN and connects with retry.
//
// The DSN is constructed from individual env vars so that credentials are
// defined once (in .env.db) and shared with the backend container:
//
//	POSTGRES_USER     — DB username  (default: "contacts_app")
//	POSTGRES_PASSWORD — DB password
//	POSTGRES_DB       — DB name      (default: "contacts_manager")
//	DB_HOST           — hostname     (default: "localhost"; "db" in Docker)
//	DB_PORT           — port         (default: "5432")
//	DB_SSLMODE        — SSL mode     (default: "disable")
//
// If DATABASE_URL is set it takes priority.
func Connect(ctx context.Context) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		log.Println("[BOOT] Using explicit DATABASE_URL")
	} else {
		dsn = buildDSN()
	}
	return connectWithRetry(ctx, dsn)
}

func buildDSN() string {
	user := getEnvOrDefault("POSTGRES_USER", "contacts_app")
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbName := getEnvOrDefault("POSTGRES_DB", "contacts_manager")
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s", host, port, user, dbName, sslmode)
	if pass != "" {
		dsn += fmt.Sprintf(" password=%s", pass)
	}

	// Never log the password or the assembled DSN.
	log.Printf("[BOOT] Built DSN from env (host=%s port=%s db=%s user=%s sslmode=%s)",
		host, port, dbName, user, sslmode)
	return dsn
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func connectWithRetry(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	var lastErr error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		lastErr = db.PingContext(pingCtx)
		cancel()

		if lastErr == nil {
			log.Printf("[BOOT] Connected to database (pool: %d open / %d idle)", maxOpenConns, maxIdleConns)
			return db, nil
		}

		log.Printf("[BOOT] Database not ready (attempt %d/%d): %v", attempt, connectAttempts, lastErr)

		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(connectBackoff):
		}
	}

	db.Close()
	return nil, fmt.Errorf("database unreachable after %d attempts: %w", connectAttempts, lastErr)
}

// RunMigrations applies every .sql file in migrationsDir that has not been
// applied before, in filename order.
//
// Two properties this deliberately provides, both of which were previously
// missing: each file is recorded in schema_migrations once it succeeds, so a
// migration runs exactly once rather than on every boot; and each file runs
// inside a transaction, so a failure rolls back instead of leaving the schema
// half-changed.
func RunMigrations(ctx context.Context, db *sql.DB, migrationsDir string) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
		    filename   TEXT        PRIMARY KEY,
		    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedMigrations(ctx, db)
	if err != nil {
		return err
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}
	if len(files) == 0 {
		log.Printf("[BOOT] No migration files found in %s", migrationsDir)
		return nil
	}
	sort.Strings(files) // 000_*.sql before 001_*.sql

	pending := 0
	for _, file := range files {
		name := filepath.Base(file)
		if applied[name] {
			continue
		}
		if err := applyMigration(ctx, db, file, name); err != nil {
			return err
		}
		log.Printf("  applied %s", name)
		pending++
	}

	if pending == 0 {
		log.Printf("[BOOT] Schema up to date (%d migration(s) already applied)", len(applied))
	} else {
		log.Printf("[BOOT] Applied %d new migration(s)", pending)
	}
	return nil
}

func appliedMigrations(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT filename FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func applyMigration(ctx context.Context, db *sql.DB, path, name string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	// Runs on every path; a no-op once the transaction has been committed.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("migration %s failed: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

// ErrNoMigrations is returned when a migrations directory is missing entirely.
var ErrNoMigrations = errors.New("migrations directory not found")
