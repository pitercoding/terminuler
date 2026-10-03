package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"
)

func newTestHandler(limit int, clock *fakeClock) http.Handler {
	middleware := Middleware(
		NewLimiter(limit, time.Minute, clock.now),
		NewClientIPResolver([]netip.Prefix{
			netip.MustParsePrefix("127.0.0.1/32"),
		}, ""),
	)

	return middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
}

func sendFrom(handler http.Handler, remoteAddr, realIP string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/appointments", nil)
	request.RemoteAddr = remoteAddr

	if realIP != "" {
		request.Header.Set("X-Real-IP", realIP)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestMiddleware_RejectsOverLimit(t *testing.T) {
	clock := newFakeClock()
	handler := newTestHandler(5, clock)

	for i := range 5 {
		if code := sendFrom(handler, "198.51.100.1:1234", "").Code; code != http.StatusCreated {
			t.Fatalf("expected request %d to pass, got %d", i+1, code)
		}
	}

	clock.advance(500 * time.Millisecond)

	recorder := sendFrom(handler, "198.51.100.1:1234", "")

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}

	if retryAfter := recorder.Header().Get("Retry-After"); retryAfter != "60" {
		t.Errorf("expected Retry-After 60, got %q", retryAfter)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected JSON content type, got %q", contentType)
	}

	if body := recorder.Body.String(); body != `{"error":"rate limit exceeded"}`+"\n" {
		t.Errorf("unexpected body %q", body)
	}
}

func TestMiddleware_ClientsBehindProxyHaveSeparateLimits(t *testing.T) {
	handler := newTestHandler(1, newFakeClock())

	if code := sendFrom(handler, "127.0.0.1:1234", "203.0.113.1").Code; code != http.StatusCreated {
		t.Fatalf("expected first client to pass, got %d", code)
	}

	if code := sendFrom(handler, "127.0.0.1:1234", "203.0.113.2").Code; code != http.StatusCreated {
		t.Fatalf("expected second client to pass, got %d", code)
	}

	if code := sendFrom(handler, "127.0.0.1:1234", "203.0.113.1").Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected first client to be limited, got %d", code)
	}
}

func TestMiddleware_SpoofedHeaderFromUntrustedClientIsIgnored(t *testing.T) {
	handler := newTestHandler(1, newFakeClock())

	sendFrom(handler, "198.51.100.1:1234", "203.0.113.1")

	if code := sendFrom(handler, "198.51.100.1:1234", "203.0.113.2").Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected spoofed header not to bypass the limit, got %d", code)
	}
}

func TestMiddleware_IPv6ClientsShareTheirSlash64(t *testing.T) {
	handler := newTestHandler(1, newFakeClock())

	sendFrom(handler, "[2001:db8:1:2::1]:1234", "")

	if code := sendFrom(handler, "[2001:db8:1:2::ffff]:1234", "").Code; code != http.StatusTooManyRequests {
		t.Fatalf("expected addresses in the same /64 to share a limit, got %d", code)
	}

	if code := sendFrom(handler, "[2001:db8:1:3::1]:1234", "").Code; code != http.StatusCreated {
		t.Fatalf("expected another /64 to have its own limit, got %d", code)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	tests := []struct {
		retryAfter time.Duration
		expected   string
	}{
		{time.Minute, "60"},
		{59*time.Second + time.Millisecond, "60"},
		{time.Millisecond, "1"},
		{0, "1"},
	}

	for _, tt := range tests {
		if got := retryAfterSeconds(tt.retryAfter); got != tt.expected {
			t.Errorf("retryAfterSeconds(%s) = %q, expected %q", tt.retryAfter, got, tt.expected)
		}
	}
}
