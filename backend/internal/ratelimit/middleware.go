package ratelimit

import (
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

// ipv6PrefixBits groups IPv6 clients by their /64 network: a single
// connection usually owns a whole /64, so limiting each address separately
// would let a client bypass the limit by rotating addresses.
const ipv6PrefixBits = 64

// unknownClientKey groups the requests whose client IP cannot be found, so
// they share one limit instead of bypassing it.
const unknownClientKey = "unknown"

// rateLimitExceededBody matches the {"error": message} format used by the
// handlers.
const rateLimitExceededBody = `{"error":"rate limit exceeded"}` + "\n"

// Middleware rejects requests over the limit of their client with
// 429 Too Many Requests and a Retry-After header in seconds.
func Middleware(
	limiter *Limiter,
	resolver *ClientIPResolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := limiter.Allow(clientKey(resolver, r))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", retryAfterSeconds(retryAfter))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(rateLimitExceededBody))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(resolver *ClientIPResolver, r *http.Request) string {
	addr, ok := resolver.ClientIP(r)
	if !ok {
		return unknownClientKey
	}

	if addr.Is6() {
		return netip.PrefixFrom(addr, ipv6PrefixBits).Masked().String()
	}

	return addr.String()
}

// retryAfterSeconds rounds up, so a client that waits the advertised time
// is never rejected again for having retried a fraction of a second early.
func retryAfterSeconds(retryAfter time.Duration) string {
	seconds := int(math.Ceil(retryAfter.Seconds()))

	return strconv.Itoa(max(seconds, 1))
}
