package ratelimit

import (
	"sync"
	"time"
)

const cleanupInterval = 5 * time.Minute

type clientRecord struct {
	count     int
	expiresAt time.Time
}

// RateLimiter is a fixed-window counter keyed by an arbitrary string (a client
// IP or an account identifier). It is process-local: with more than one replica
// each holds its own counters, so a shared store is required before scaling out.
type RateLimiter struct {
	mu       sync.Mutex
	records  map[string]*clientRecord
	limit    int
	duration time.Duration

	stopOnce sync.Once
	stop     chan struct{}
}

func NewRateLimiter(limit int, duration time.Duration) *RateLimiter {
	rl := &RateLimiter{
		records:  make(map[string]*clientRecord),
		limit:    limit,
		duration: duration,
		stop:     make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Allow records an attempt and reports whether it is within budget.
func (rl *RateLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.records[key]
	if !exists || now.After(record.expiresAt) {
		rl.records[key] = &clientRecord{count: 1, expiresAt: now.Add(rl.duration)}
		return true
	}
	if record.count >= rl.limit {
		return false
	}
	record.count++
	return true
}

// Stop ends the cleanup goroutine. Safe to call more than once.
func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case now := <-ticker.C:
			rl.mu.Lock()
			for key, record := range rl.records {
				if now.After(record.expiresAt) {
					delete(rl.records, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}
