package worker

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"
)

// StartCleanupWorker runs an automated background job to permanently delete
// contacts that were soft-deleted over 30 days ago.
func StartCleanupWorker(ctx context.Context, db *sql.DB) {
	// Configurable interval (defaults to 24h)
	interval := 24 * time.Hour
	if envInterval := os.Getenv("CLEANUP_INTERVAL"); envInterval != "" {
		if parsed, err := time.ParseDuration(envInterval); err == nil {
			interval = parsed
		}
	}

	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		log.Printf("[Worker] Cleanup job started. Running every %v", interval)

		for {
			select {
			case <-ctx.Done():
				log.Println("[Worker] Cleanup job shutting down gracefully...")
				return
			case <-ticker.C:
				// Use batch deletion to avoid locking the table or blowing up transaction logs
				// CTE handles selecting the expired IDs and deleting them safely
				query := `
					WITH expired AS (
						SELECT id FROM contacts
						WHERE deleted_at IS NOT NULL
						  AND deleted_at < NOW() - interval '30 days'
						LIMIT 1000
					)
					DELETE FROM contacts
					WHERE id IN (SELECT id FROM expired);
				`

				res, err := db.ExecContext(ctx, query)
				if err != nil {
					log.Printf("[Worker] Error running cleanup query: %v", err)
					continue
				}

				rowsAffected, _ := res.RowsAffected()
				if rowsAffected > 0 {
					log.Printf("[Worker] Cleanup executed successfully: permanently deleted %d expired contacts", rowsAffected)
				}
			}
		}
	}()
}
