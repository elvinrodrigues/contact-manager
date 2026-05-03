package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

// ConnectDB builds a Postgres DSN and connects with retry.
//
// The DSN is constructed from individual env vars so that credentials are
// defined once (in .env.db) and shared with the backend container:
//
//   POSTGRES_USER     — DB username  (required, same var Postgres uses)
//   POSTGRES_PASSWORD — DB password  (required, same var Postgres uses)
//   POSTGRES_DB       — DB name      (required, same var Postgres uses)
//   DB_HOST           — hostname     (default: "localhost"; set to "db" in Docker)
//   DB_SSLMODE        — SSL mode     (default: "disable")
//
// If DATABASE_URL is set explicitly, it takes priority (backward compat).
func ConnectDB() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		log.Println("[BOOT] Using explicit DATABASE_URL")
	} else {
		dsn = buildDSN()
	}

	return connectWithRetry(dsn)
}

// buildDSN constructs a lib/pq connection string from individual env vars.
func buildDSN() string {
	user := getEnvOrDefault("POSTGRES_USER", "contacts_app")
	pass := os.Getenv("POSTGRES_PASSWORD")
	dbName := getEnvOrDefault("POSTGRES_DB", "contacts_manager")
	host := getEnvOrDefault("DB_HOST", "localhost")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s", host, user, dbName, sslmode)
	if pass != "" {
		dsn += fmt.Sprintf(" password=%s", pass)
	}

	log.Printf("[BOOT] Built DSN from env vars (host=%s, db=%s, user=%s)", host, dbName, user)
	return dsn
}

// getEnvOrDefault reads an env var with a fallback default.
func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func connectWithRetry(dsn string) *sql.DB {
	var db *sql.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Println("[BOOT] Connected to database")
				return db
			}
		}

		log.Printf("[BOOT] DB not ready (attempt %d/10), retrying... %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatal("[BOOT] Database not reachable after retries:", err)
	return nil
}

// RunMigrations reads all .sql files from the migrations directory and executes
// them in alphabetical order. Each file should be idempotent (safe to re-run).
func RunMigrations(db *sql.DB, migrationsDir string) {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		log.Fatalf("Failed to read migrations directory: %v", err)
	}

	if len(files) == 0 {
		log.Println("[BOOT] No migration files found in", migrationsDir)
		return
	}

	sort.Strings(files) // ensures 001_*.sql runs before 002_*.sql

	log.Printf("[BOOT] Found %d migration file(s)", len(files))
	for _, file := range files {
		name := filepath.Base(file)
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Failed to read migration %s: %v", name, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Migration %s failed: %v", name, err)
		}

		log.Printf("  ✓ %s", name)
	}
	log.Println("[BOOT] Migrations complete")
}
