package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllowEnforcesLimit(t *testing.T) {
	rl := NewRateLimiter(3, time.Minute)
	defer rl.Stop()

	for i := 1; i <= 3; i++ {
		if !rl.Allow("client") {
			t.Fatalf("request %d was rejected but is within the budget of 3", i)
		}
	}
	if rl.Allow("client") {
		t.Error("the 4th request was allowed past a limit of 3")
	}
}

func TestAllowIsPerKey(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	defer rl.Stop()

	if !rl.Allow("a") || !rl.Allow("b") {
		t.Fatal("separate keys should have separate budgets")
	}
	if rl.Allow("a") {
		t.Error("key \"a\" exceeded its budget")
	}
}

func TestWindowExpires(t *testing.T) {
	rl := NewRateLimiter(1, 25*time.Millisecond)
	defer rl.Stop()

	if !rl.Allow("client") {
		t.Fatal("first request rejected")
	}
	if rl.Allow("client") {
		t.Fatal("second request allowed inside the window")
	}

	time.Sleep(40 * time.Millisecond)
	if !rl.Allow("client") {
		t.Error("the budget did not reset after the window elapsed")
	}
}

// The limiter's map is shared across every in-flight request, so it must be
// safe to hammer from many goroutines at once.
func TestAllowIsConcurrencySafe(t *testing.T) {
	const (
		limit      = 100
		goroutines = 50
		perRoutine = 20
	)

	rl := NewRateLimiter(limit, time.Minute)
	defer rl.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perRoutine; j++ {
				if rl.Allow("shared") {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	// Exactly `limit` requests must get through: no more (the cap held) and no
	// fewer (no update was lost to a race).
	if allowed != limit {
		t.Errorf("allowed %d requests, want exactly %d", allowed, limit)
	}
}

func TestStopIsIdempotent(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.Stop()
	rl.Stop() // must not panic on a closed channel
}
