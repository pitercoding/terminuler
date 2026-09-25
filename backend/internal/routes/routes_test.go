package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/handlers"
	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/services"
)

type stubAppointmentRepository struct{}

func (s *stubAppointmentRepository) GetByDate(
	ctx context.Context,
	date string,
) ([]models.Appointment, error) {
	return nil, nil
}

func (s *stubAppointmentRepository) Create(
	ctx context.Context,
	appointment *models.Appointment,
) error {
	appointment.ID = 1

	return nil
}

func newTestMux() *http.ServeMux {
	service := services.NewAppointmentService(
		&stubAppointmentRepository{},
		func() time.Time {
			return time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
		},
	)

	mux := http.NewServeMux()

	RegisterRoutes(mux, handlers.NewAppointmentHandler(service))

	return mux
}

func TestRoutes(t *testing.T) {
	validBody := `{
		"appointment_date": "2026-09-28",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedAllow  string
	}{
		{
			name:           "health check",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "health check HEAD",
			method:         http.MethodHead,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "availability",
			method:         http.MethodGet,
			path:           "/appointments/availability?date=2026-09-28",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "availability with wrong method",
			method:         http.MethodPost,
			path:           "/appointments/availability?date=2026-09-28",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedAllow:  "GET, HEAD",
		},
		{
			name:           "create appointment",
			method:         http.MethodPost,
			path:           "/appointments",
			body:           validBody,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "create appointment with wrong method",
			method:         http.MethodGet,
			path:           "/appointments",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedAllow:  "POST",
		},
		{
			name:           "health with wrong method",
			method:         http.MethodDelete,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedAllow:  "GET, HEAD",
		},
		{
			name:           "unknown route",
			method:         http.MethodGet,
			path:           "/unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "trailing slash is a different route",
			method:         http.MethodPost,
			path:           "/appointments/",
			body:           validBody,
			expectedStatus: http.StatusNotFound,
		},
	}

	mux := newTestMux()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				tt.method,
				tt.path,
				strings.NewReader(tt.body),
			)

			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d (%s)",
					tt.expectedStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if tt.expectedAllow != "" &&
				recorder.Header().Get("Allow") != tt.expectedAllow {
				t.Errorf(
					"expected Allow header %q, got %q",
					tt.expectedAllow,
					recorder.Header().Get("Allow"),
				)
			}
		})
	}
}
