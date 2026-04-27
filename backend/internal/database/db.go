package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=contacts_app dbname=contacts_manager sslmode=disable"
		log.Println("[BOOT] DATABASE_URL not set — using local default")
	}

	return connectWithRetry(dbURL)
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
