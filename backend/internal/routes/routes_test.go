package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	clerk "github.com/clerk/clerk-sdk-go/v2"

	"github.com/pitercoding/terminuler/internal/auth"
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

func (s *stubAppointmentRepository) GetAppointments(
	ctx context.Context,
	fromDate string,
) ([]models.Appointment, error) {
	return []models.Appointment{
		{
			ID:              1,
			AppointmentDate: time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC),
			StartTime:       "10:00:00",
			EndTime:         "11:00:00",
		},
	}, nil
}

func (s *stubAppointmentRepository) Create(
	ctx context.Context,
	appointment *models.Appointment,
) error {
	appointment.ID = 1

	return nil
}

func (s *stubAppointmentRepository) Delete(
	ctx context.Context,
	id int64,
) (*models.Appointment, error) {
	return &models.Appointment{ID: id}, nil
}

func newTestMux(
	createAppointmentLimit func(http.Handler) http.Handler,
) *http.ServeMux {
	return newTestMuxWithAdmin(createAppointmentLimit, rejectNonAdmin)
}

func newTestMuxWithAdmin(
	createAppointmentLimit func(http.Handler) http.Handler,
	requireAdmin func(http.Handler) http.Handler,
) *http.ServeMux {
	service := services.NewAppointmentService(
		&stubAppointmentRepository{},
		func() time.Time {
			return time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
		},
	)

	mux := http.NewServeMux()

	RegisterRoutes(
		mux,
		handlers.NewAppointmentHandler(service),
		createAppointmentLimit,
		requireAdmin,
	)

	return mux
}

func noLimit(next http.Handler) http.Handler {
	return next
}

// rejectNonAdmin stands in for the admin authorization rejecting the
// request, so routes that skip it are caught.
func rejectNonAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
}

// rejectAll stands in for a rate limit that is already exceeded.
func rejectAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
}

func TestRoutes_RateLimitAppliesOnlyToCreateAppointment(t *testing.T) {
	mux := newTestMux(rejectAll)

	tests := []struct {
		method         string
		path           string
		expectedStatus int
	}{
		{http.MethodPost, "/appointments", http.StatusTooManyRequests},
		{http.MethodGet, "/appointments/availability?date=2026-09-28", http.StatusOK},
		{http.MethodGet, "/health", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d", tt.expectedStatus, recorder.Code)
			}
		})
	}
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
			name:           "admin session requires admin",
			method:         http.MethodGet,
			path:           "/admin/session",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin appointments requires admin",
			method:         http.MethodGet,
			path:           "/admin/appointments",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin appointments with wrong method",
			method:         http.MethodPost,
			path:           "/admin/appointments",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedAllow:  "GET, HEAD",
		},
		{
			name:           "delete appointment requires admin",
			method:         http.MethodDelete,
			path:           "/admin/appointments/1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "delete appointment with wrong method",
			method:         http.MethodGet,
			path:           "/admin/appointments/1",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedAllow:  "DELETE",
		},
		{
			name:           "delete appointments collection is not allowed",
			method:         http.MethodDelete,
			path:           "/admin/appointments",
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

	mux := newTestMux(noLimit)

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

const testAdminUserID = "user_admin"

// withSessionUser stands in for the Clerk middleware having verified a
// session token for userID, so the real admin check runs without calling
// Clerk. An empty userID leaves the request unauthenticated.
func withSessionUser(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		requireAdmin := auth.RequireAdmin(
			testAdminUserID,
			handlers.WriteError,
			next,
		)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID != "" {
				r = r.WithContext(clerk.ContextWithSessionClaims(
					r.Context(),
					&clerk.SessionClaims{
						RegisteredClaims: clerk.RegisteredClaims{
							Subject: userID,
						},
					},
				))
			}

			requireAdmin.ServeHTTP(w, r)
		})
	}
}

func TestRoutes_AdminAppointments(t *testing.T) {
	tests := []struct {
		name           string
		requireAdmin   func(http.Handler) http.Handler
		expectedStatus int
	}{
		{
			// The real middleware: without an Authorization header Clerk
			// verifies nothing, so no network call is made.
			name:           "missing session token",
			requireAdmin:   auth.AdminMiddleware(testAdminUserID, handlers.WriteError),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unauthenticated",
			requireAdmin:   withSessionUser(""),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "authenticated non-admin",
			requireAdmin:   withSessionUser("user_other"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin",
			requireAdmin:   withSessionUser(testAdminUserID),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestMuxWithAdmin(noLimit, tt.requireAdmin)

			recorder := httptest.NewRecorder()

			mux.ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, "/admin/appointments", nil),
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d (%s)",
					tt.expectedStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if tt.expectedStatus == http.StatusOK {
				body := recorder.Body.String()

				if !strings.Contains(body, `"appointment_date":"2026-09-28"`) ||
					!strings.Contains(body, `"start_time":"10:00"`) {
					t.Errorf("expected the stored appointment, got %s", body)
				}
			}
		})
	}
}

func TestRoutes_DeleteAdminAppointment(t *testing.T) {
	tests := []struct {
		name           string
		requireAdmin   func(http.Handler) http.Handler
		expectedStatus int
	}{
		{
			name:           "missing session token",
			requireAdmin:   auth.AdminMiddleware(testAdminUserID, handlers.WriteError),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "unauthenticated",
			requireAdmin:   withSessionUser(""),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "authenticated non-admin",
			requireAdmin:   withSessionUser("user_other"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "admin",
			requireAdmin:   withSessionUser(testAdminUserID),
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestMuxWithAdmin(noLimit, tt.requireAdmin)

			recorder := httptest.NewRecorder()

			mux.ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodDelete, "/admin/appointments/1", nil),
			)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d (%s)",
					tt.expectedStatus,
					recorder.Code,
					recorder.Body.String(),
				)
			}
		})
	}
}

// unverifiableSessionToken is a well-formed JWT without the kid header, so
// the Clerk middleware rejects it before fetching any key from Clerk, as it
// does for a token issued by another Clerk instance.
const unverifiableSessionToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." +
	"eyJzdWIiOiJ1c2VyX2FkbWluIn0." +
	"c2lnbmF0dXJl"

// The Clerk middleware answers a rejected token with an empty body by
// default. The Next.js proxy cannot parse it, so every admin route must
// answer with the JSON error body instead.
func TestRoutes_AdminRejectsUnverifiableTokenWithJSON(t *testing.T) {
	mux := newTestMuxWithAdmin(
		noLimit,
		auth.AdminMiddleware(testAdminUserID, handlers.WriteError),
	)

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/session"},
		{http.MethodGet, "/admin/appointments"},
		{http.MethodDelete, "/admin/appointments/1"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			request.Header.Set("Authorization", "Bearer "+unverifiableSessionToken)

			recorder := httptest.NewRecorder()

			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf(
					"expected status 401, got %d (%s)",
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %q", contentType)
			}

			if body := strings.TrimSpace(recorder.Body.String()); body != `{"error":"unauthorized"}` {
				t.Errorf(`expected body {"error":"unauthorized"}, got %q`, body)
			}
		})
	}
}
