package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/services"
)

type mockAppointmentRepository struct {
	appointments []models.Appointment
	getByDateErr error
	err          error
}

func (m *mockAppointmentRepository) GetByDate(
	ctx context.Context,
	date string,
) ([]models.Appointment, error) {
	if m.getByDateErr != nil {
		return nil, m.getByDateErr
	}

	return m.appointments, nil
}

func (m *mockAppointmentRepository) Create(
	ctx context.Context,
	appointment *models.Appointment,
) error {
	if m.err != nil {
		return m.err
	}

	appointment.ID = 1

	return nil
}

// fixedClock returns a fixed "current time" (Friday, 2026-09-25 12:00 UTC),
// so the tests do not depend on the real date.
func fixedClock() time.Time {
	return time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
}

// assertErrorResponse checks that the response is a JSON error body in the
// format {"error": expectedMessage}.
func assertErrorResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	expectedMessage string,
) {
	t.Helper()

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var response map[string]any

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if len(response) != 1 || response["error"] != expectedMessage {
		t.Fatalf(
			`expected {"error": %q}, got %v`,
			expectedMessage,
			response,
		)
	}
}

func TestCreateAppointment_Success(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2026-09-28",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	var response map[string]any

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expected := map[string]any{
		"id":               float64(1),
		"appointment_date": "2026-09-28",
		"start_time":       "10:00",
		"end_time":         "11:00",
		"customer_name":    "Racha Cuca",
		"customer_phone":   "+5511999999999",
		"customer_email":   "rc@exemple.com",
	}

	for key, value := range expected {
		if response[key] != value {
			t.Errorf("expected %s to be %v, got %v", key, value, response[key])
		}
	}

	if _, ok := response["created_at"]; !ok {
		t.Error("expected response to contain created_at")
	}
}

func TestCreateAppointment_InvalidJSON(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{"appointment_date":`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "invalid request body")
}

func TestCreateAppointment_ValidationError(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2026-09-28",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "customer name is required")
}

func TestCreateAppointment_Conflict(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: repositories.ErrAppointmentConflict,
	}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2026-09-28",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "appointment slot is already booked")
}

func TestCreateAppointment_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2026-09-28",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	if strings.Contains(
		recorder.Body.String(),
		"database connection failed",
	) {
		t.Fatalf(
			"expected internal error details to be hidden, got %s",
			recorder.Body.String(),
		)
	}

	assertErrorResponse(t, recorder, "internal server error")
}

func TestCreateAppointment_ValidationErrorMessage(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2026-09-26",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "appointments are not available on weekends")
}

func TestCreateAppointment_BodyTooLarge(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{"customer_name": "` +
		strings.Repeat("a", maxRequestBodyBytes) +
		`"}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "invalid request body")
}

func TestGetAvailability_Success(t *testing.T) {
	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				StartTime: "10:00:00",
				EndTime:   "11:00:00",
			},
		},
	}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability?date=2026-09-28",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response struct {
		Date           string                   `json:"date"`
		AvailableSlots []services.AvailableSlot `json:"available_slots"`
	}

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Date != "2026-09-28" {
		t.Errorf("expected date 2026-09-28, got %s", response.Date)
	}

	if len(response.AvailableSlots) != 7 {
		t.Fatalf(
			"expected 7 available slots, got %d",
			len(response.AvailableSlots),
		)
	}

	for _, slot := range response.AvailableSlots {
		if slot.StartTime == "10:00" {
			t.Error("expected 10:00 slot to be unavailable")
		}
	}
}

func TestGetAvailability_FullyBookedReturnsEmptyList(t *testing.T) {
	var appointments []models.Appointment

	for hour := 8; hour < 16; hour++ {
		appointments = append(appointments, models.Appointment{
			StartTime: fmt.Sprintf("%02d:00:00", hour),
			EndTime:   fmt.Sprintf("%02d:00:00", hour+1),
		})
	}

	repository := &mockAppointmentRepository{
		appointments: appointments,
	}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability?date=2026-09-28",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		`"available_slots":[]`,
	) {
		t.Fatalf(
			"expected empty available slots list, got %s",
			recorder.Body.String(),
		)
	}
}

func TestGetAvailability_MissingDate(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "date query parameter is required")
}

func TestGetAvailability_InvalidDate(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability?date=28-09-2026",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "invalid date format, expected YYYY-MM-DD")
}

func TestGetAvailability_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		getByDateErr: errors.New("database connection failed"),
	}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability?date=2026-09-28",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	if strings.Contains(
		recorder.Body.String(),
		"database connection failed",
	) {
		t.Fatalf(
			"expected internal error details to be hidden, got %s",
			recorder.Body.String(),
		)
	}

	assertErrorResponse(t, recorder, "internal server error")
}

func TestCreateAppointment_PastDate(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	body := `{
		"appointment_date": "2020-01-06",
		"start_time": "10:00",
		"end_time": "11:00",
		"customer_name": "Racha Cuca",
		"customer_phone": "+5511999999999",
		"customer_email": "rc@exemple.com"
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/appointments",
		strings.NewReader(body),
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "appointment must be scheduled in the future")
}

func TestGetAvailability_PastDateReturnsEmptyList(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository, fixedClock)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments/availability?date=2020-01-06",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.GetAvailability(
		recorder,
		request,
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if !strings.Contains(
		recorder.Body.String(),
		`"available_slots":[]`,
	) {
		t.Fatalf(
			"expected empty available slots list, got %s",
			recorder.Body.String(),
		)
	}
}
