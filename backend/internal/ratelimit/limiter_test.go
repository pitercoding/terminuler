package ratelimit

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// fakeClock is a controllable time source for the limiter.
type fakeClock struct {
	current time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{
		current: time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC),
	}
}

func (c *fakeClock) now() time.Time {
	return c.current
}

func (c *fakeClock) advance(d time.Duration) {
	c.current = c.current.Add(d)
}

func TestLimiter_AllowsUpToLimit(t *testing.T) {
	clock := newFakeClock()
	limiter := NewLimiter(3, time.Minute, clock.now)

	for i := range 3 {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatalf("expected request %d to be allowed", i+1)
		}
	}

	allowed, retryAfter := limiter.Allow("client")
	if allowed {
		t.Fatal("expected request over the limit to be rejected")
	}

	if retryAfter != time.Minute {
		t.Fatalf("expected retry after 1m, got %s", retryAfter)
	}
}

func TestLimiter_KeysAreIndependent(t *testing.T) {
	limiter := NewLimiter(1, time.Minute, newFakeClock().now)

	if allowed, _ := limiter.Allow("a"); !allowed {
		t.Fatal("expected first request of a to be allowed")
	}

	if allowed, _ := limiter.Allow("b"); !allowed {
		t.Fatal("expected first request of b to be allowed")
	}

	if allowed, _ := limiter.Allow("a"); allowed {
		t.Fatal("expected second request of a to be rejected")
	}
}

func TestLimiter_SlidingWindow(t *testing.T) {
	clock := newFakeClock()
	limiter := NewLimiter(2, time.Minute, clock.now)

	limiter.Allow("client")
	clock.advance(40 * time.Second)
	limiter.Allow("client")

	// Just after the boundary a fixed window would reset. Here both
	// requests are still inside the last minute.
	clock.advance(10 * time.Second)

	allowed, retryAfter := limiter.Allow("client")
	if allowed {
		t.Fatal("expected request to be rejected")
	}

	if retryAfter != 10*time.Second {
		t.Fatalf("expected retry after 10s, got %s", retryAfter)
	}

	// Once the first request leaves the window, one slot opens.
	clock.advance(10 * time.Second)

	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected request to be allowed after the oldest expired")
	}

	if allowed, _ := limiter.Allow("client"); allowed {
		t.Fatal("expected only one slot to open")
	}
}

func TestLimiter_RejectedRequestsAreNotRecorded(t *testing.T) {
	clock := newFakeClock()
	limiter := NewLimiter(1, time.Minute, clock.now)

	limiter.Allow("client")

	for range 5 {
		clock.advance(10 * time.Second)
		limiter.Allow("client")
	}

	clock.advance(10 * time.Second)

	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("expected retries while limited not to extend the wait")
	}
}

func TestLimiter_SweepRemovesIdleKeys(t *testing.T) {
	clock := newFakeClock()
	limiter := NewLimiter(5, time.Minute, clock.now)

	for i := range 100 {
		limiter.Allow(fmt.Sprintf("client-%d", i))
	}

	clock.advance(time.Minute)
	limiter.Allow("active")

	if len(limiter.hits) != 1 {
		t.Fatalf("expected only the active key to remain, got %d keys", len(limiter.hits))
	}
}

func TestLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewLimiter(50, time.Minute, time.Now)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)

	for range 200 {
		wg.Go(func() {
			if ok, _ := limiter.Allow("client"); ok {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		})
	}

	wg.Wait()

	if allowed != 50 {
		t.Fatalf("expected exactly 50 allowed requests, got %d", allowed)
	}
}
