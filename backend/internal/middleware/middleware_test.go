package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	expected := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "no-store",
	}

	// Errors, such as this 404, must carry the headers too.
	for name, value := range expected {
		if got := recorder.Header().Get(name); got != value {
			t.Errorf("expected %s %q, got %q", name, value, got)
		}
	}
}

func TestAccessLog(t *testing.T) {
	var output bytes.Buffer

	logger := log.New(&output, "", 0)

	handler := AccessLog(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Write([]byte("ok"))
			return
		}

		w.WriteHeader(http.StatusForbidden)
	}))

	request := httptest.NewRequest(
		http.MethodGet,
		"/admin/appointments?email=rc@exemple.com",
		nil,
	)
	request.Header.Set("Authorization", "Bearer secret-session-token")

	handler.ServeHTTP(httptest.NewRecorder(), request)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))

	logged := output.String()

	if !strings.HasPrefix(logged, "GET /admin/appointments 403 ") {
		t.Errorf("expected method, path and status to be logged, got %q", logged)
	}

	for _, secret := range []string{"secret-session-token", "rc@exemple.com"} {
		if strings.Contains(logged, secret) {
			t.Errorf("expected %q not to be logged, got %q", secret, logged)
		}
	}

	if strings.Contains(logged, "/health") {
		t.Errorf("expected successful health checks not to be logged, got %q", logged)
	}
}
