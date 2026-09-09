package worker

import (
	"context"
	"database/sql"
	"log"
	"os"
	"sync"
	"time"

	"contact-manager/internal/repository"
)

const (
	defaultInterval = 24 * time.Hour
	// batchSize bounds one pass so a large backlog cannot hold locks or bloat
	// the WAL in a single statement.
	batchSize    = 1000
	queryTimeout = 30 * time.Second
)

// StartCleanupWorker permanently deletes contacts whose retention window has
// expired. It returns a WaitGroup the caller can wait on during shutdown so the
// process does not exit mid-batch.
func StartCleanupWorker(ctx context.Context, db *sql.DB) *sync.WaitGroup {
	interval := defaultInterval
	if raw := os.Getenv("CLEANUP_INTERVAL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			interval = parsed
		} else {
			log.Printf("[Worker] Ignoring invalid CLEANUP_INTERVAL %q; using %v", raw, interval)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		log.Printf("[Worker] Cleanup job started; running every %v (retention %d days)",
			interval, repository.RetentionDays)

		for {
			select {
			case <-ctx.Done():
				log.Println("[Worker] Cleanup job stopped")
				return
			case <-ticker.C:
				purgeExpired(ctx, db)
			}
		}
	}()

	return &wg
}

// purgeExpired deletes one batch of expired contacts. It relies on purge_at,
// which is set at soft-delete time, so changing the retention constant does not
// retroactively purge contacts deleted under the previous window.
func purgeExpired(ctx context.Context, db *sql.DB) {
	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	res, err := db.ExecContext(queryCtx, `
		DELETE FROM contacts
		WHERE id IN (
		    SELECT id FROM contacts
		    WHERE deleted_at IS NOT NULL
		      AND purge_at IS NOT NULL
		      AND purge_at <= now()
		    ORDER BY purge_at
		    LIMIT $1
		)
	`, batchSize)
	if err != nil {
		if ctx.Err() != nil {
			return // shutting down, not a failure
		}
		log.Printf("[Worker] Cleanup query failed: %v", err)
		return
	}

	if n, err := res.RowsAffected(); err == nil && n > 0 {
		log.Printf("[Worker] Purged %d expired contact(s)", n)
	}
}
