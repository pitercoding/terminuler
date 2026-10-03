package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubPinger struct {
	err error
}

func (s stubPinger) PingContext(ctx context.Context) error {
	return s.err
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		pingErr        error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "database reachable",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			// The database error must not reach the response.
			name:           "database unreachable",
			pingErr:        errors.New("dial tcp 10.0.0.5:5432: connection refused"),
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody:   `{"status":"unavailable"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			NewHealthHandler(stubPinger{err: tt.pingErr}).ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, "/health", nil),
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}

			if body := strings.TrimSpace(recorder.Body.String()); body != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, body)
			}
		})
	}
}
