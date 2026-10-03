package ratelimit

import (
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIPResolver(t *testing.T) {
	trusted := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}

	tests := []struct {
		name       string
		trusted    []netip.Prefix
		remoteAddr string
		realIP     string
		expected   string
	}{
		{
			name:       "no trusted proxies ignores header",
			remoteAddr: "127.0.0.1:50000",
			realIP:     "203.0.113.7",
			expected:   "127.0.0.1",
		},
		{
			name:       "untrusted connection ignores header",
			trusted:    trusted,
			remoteAddr: "198.51.100.1:50000",
			realIP:     "203.0.113.7",
			expected:   "198.51.100.1",
		},
		{
			name:       "trusted IPv4 proxy uses header",
			trusted:    trusted,
			remoteAddr: "127.0.0.1:50000",
			realIP:     "203.0.113.7",
			expected:   "203.0.113.7",
		},
		{
			name:       "trusted IPv6 proxy uses header",
			trusted:    trusted,
			remoteAddr: "[::1]:50000",
			realIP:     "2001:db8::1",
			expected:   "2001:db8::1",
		},
		{
			name:       "trusted proxy network uses header",
			trusted:    trusted,
			remoteAddr: "10.1.2.3:50000",
			realIP:     " 203.0.113.7 ",
			expected:   "203.0.113.7",
		},
		{
			name:       "IPv4-mapped proxy address is trusted",
			trusted:    trusted,
			remoteAddr: "[::ffff:127.0.0.1]:50000",
			realIP:     "::ffff:203.0.113.7",
			expected:   "203.0.113.7",
		},
		{
			name:       "trusted proxy without header falls back to proxy",
			trusted:    trusted,
			remoteAddr: "127.0.0.1:50000",
			expected:   "127.0.0.1",
		},
		{
			name:       "trusted proxy with invalid header falls back to proxy",
			trusted:    trusted,
			remoteAddr: "127.0.0.1:50000",
			realIP:     "203.0.113.7, 198.51.100.1",
			expected:   "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/appointments", nil)
			request.RemoteAddr = tt.remoteAddr

			if tt.realIP != "" {
				request.Header.Set("X-Real-IP", tt.realIP)
			}

			addr, ok := NewClientIPResolver(tt.trusted, "").ClientIP(request)
			if !ok {
				t.Fatal("expected client IP to be resolved")
			}

			if addr.String() != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, addr)
			}
		})
	}
}

func TestClientIPResolver_InvalidRemoteAddr(t *testing.T) {
	request := httptest.NewRequest("POST", "/appointments", nil)
	request.RemoteAddr = "not-an-ip"

	if _, ok := NewClientIPResolver(nil, "").ClientIP(request); ok {
		t.Fatal("expected invalid remote address not to resolve")
	}
}

// In production the proxy connects from addresses that cannot be listed in
// TRUSTED_PROXIES, so it proves itself with the shared secret instead.
func TestClientIPResolver_ProxySecret(t *testing.T) {
	const secret = "0123456789abcdef0123456789abcdef"

	tests := []struct {
		name        string
		proxySecret string
		sentSecret  string
		expected    string
	}{
		{
			name:        "matching secret uses header",
			proxySecret: secret,
			sentSecret:  secret,
			expected:    "203.0.113.7",
		},
		{
			name:        "wrong secret ignores header",
			proxySecret: secret,
			sentSecret:  "0123456789abcdef0123456789abcdeX",
			expected:    "198.51.100.1",
		},
		{
			name:        "missing secret ignores header",
			proxySecret: secret,
			expected:    "198.51.100.1",
		},
		{
			// An unset secret must never match an empty header.
			name:     "secret disabled ignores header",
			expected: "198.51.100.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/appointments", nil)
			request.RemoteAddr = "198.51.100.1:50000"
			request.Header.Set("X-Real-IP", "203.0.113.7")

			if tt.sentSecret != "" {
				request.Header.Set(ProxySecretHeader, tt.sentSecret)
			}

			addr, ok := NewClientIPResolver(nil, tt.proxySecret).ClientIP(request)
			if !ok {
				t.Fatal("expected client IP to be resolved")
			}

			if addr.String() != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, addr)
			}
		})
	}
}
