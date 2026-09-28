// Package ratelimit limits how often a single client may call an endpoint.
// State is kept in memory, which is enough for a single API instance.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter allows up to limit requests per key within a sliding window.
// It remembers the time of each accepted request, so a client cannot send
// twice the limit around a window boundary as with a fixed window.
type Limiter struct {
	mu sync.Mutex

	limit  int
	window time.Duration
	now    func() time.Time

	// hits holds, for each key, the times of the accepted requests still
	// inside the window, oldest first.
	hits map[string][]time.Time

	lastSweep time.Time
}

func NewLimiter(
	limit int,
	window time.Duration,
	now func() time.Time,
) *Limiter {
	return &Limiter{
		limit:     limit,
		window:    window,
		now:       now,
		hits:      make(map[string][]time.Time),
		lastSweep: now(),
	}
}

// Allow records a request for key and reports whether it is within the
// limit. When it is not, it also returns how long until the oldest request
// leaves the window and the next one is accepted. Rejected requests are not
// recorded, so retrying too early does not extend the wait.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()

	l.sweep(now)

	hits := l.recentHits(key, now)

	if len(hits) >= l.limit {
		l.hits[key] = hits

		return false, hits[0].Add(l.window).Sub(now)
	}

	l.hits[key] = append(hits, now)

	return true, 0
}

// recentHits returns the requests of key that are still inside the window.
func (l *Limiter) recentHits(key string, now time.Time) []time.Time {
	hits := l.hits[key]
	cutoff := now.Add(-l.window)

	expired := 0
	for expired < len(hits) && !hits[expired].After(cutoff) {
		expired++
	}

	return hits[expired:]
}

// sweep removes keys with no requests inside the window, so the map does
// not grow with every client ever seen. It runs at most once per window,
// which keeps its cost low without a background goroutine.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.window {
		return
	}

	cutoff := now.Add(-l.window)

	for key, hits := range l.hits {
		if !hits[len(hits)-1].After(cutoff) {
			delete(l.hits, key)
		}
	}

	l.lastSweep = now
}
