package ratelimit

import (
	"sync"
	"time"
)

type clientRecord struct {
	count     int
	expiresAt time.Time
}

type RateLimiter struct {
	mu       sync.Mutex
	records  map[string]*clientRecord
	limit    int
	duration time.Duration
}

func NewRateLimiter(limit int, duration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records:  make(map[string]*clientRecord),
		limit:    limit,
		duration: duration,
	}

	// Background goroutine cleans expired entries every 5 minutes
	go rl.startCleanup()

	return rl
}

// Allow returns true if the entity is within its limit. Returns false otherwise.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	record, exists := rl.records[key]
	if !exists || now.After(record.expiresAt) {
		// Expired record — delete and start fresh
		delete(rl.records, key)
		rl.records[key] = &clientRecord{
			count:     1,
			expiresAt: now.Add(rl.duration),
		}
		return true
	}

	if record.count >= rl.limit {
		return false
	}

	record.count++
	return true
}

func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, record := range rl.records {
			if now.After(record.expiresAt) {
				delete(rl.records, key)
			}
		}
		rl.mu.Unlock()
	}
}

