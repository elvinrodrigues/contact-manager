package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
	// dbURL := os.Getenv("DATABASE_URL")
	dbURL := "host=localhost user=contacts_app dbname=contacts_manager sslmode=disable"

	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Database not reachable:", err)
	}

	log.Println("Connected to PostgreSQL")
	return db
}
