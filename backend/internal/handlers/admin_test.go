package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/services"
)

func TestListAppointments_Success(t *testing.T) {
	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				ID:              1,
				AppointmentDate: time.Date(2026, time.October, 7, 0, 0, 0, 0, time.UTC),
				StartTime:       "08:00:00",
				EndTime:         "09:00:00",
				CustomerName:    "Jane Doe",
				CustomerPhone:   "+49123456789",
				CustomerEmail:   "jane@example.com",
				CreatedAt:       time.Date(2026, time.September, 20, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	handler := NewAppointmentHandler(
		services.NewAppointmentService(repository, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.ListAppointments(
		recorder,
		httptest.NewRequest(http.MethodGet, "/admin/appointments", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var response []map[string]any

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf("expected 1 appointment, got %d", len(response))
	}

	expected := map[string]any{
		"id":               float64(1),
		"appointment_date": "2026-10-07",
		"start_time":       "08:00",
		"end_time":         "09:00",
		"customer_name":    "Jane Doe",
		"customer_phone":   "+49123456789",
		"customer_email":   "jane@example.com",
		"created_at":       "2026-09-20T10:00:00Z",
	}

	for key, value := range expected {
		if response[0][key] != value {
			t.Errorf("expected %s %v, got %v", key, value, response[0][key])
		}
	}
}

func TestListAppointments_EmptyReturnsEmptyList(t *testing.T) {
	handler := NewAppointmentHandler(
		services.NewAppointmentService(&mockAppointmentRepository{}, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.ListAppointments(
		recorder,
		httptest.NewRequest(http.MethodGet, "/admin/appointments", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	if body := recorder.Body.String(); body != "[]\n" {
		t.Fatalf("expected an empty JSON list, got %q", body)
	}
}

func TestListAppointments_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		listErr: errors.New("database connection failed"),
	}

	handler := NewAppointmentHandler(
		services.NewAppointmentService(repository, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.ListAppointments(
		recorder,
		httptest.NewRequest(http.MethodGet, "/admin/appointments", nil),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "internal server error")
}

// newDeleteRequest builds the request the mux would route to
// DeleteAppointment, with id as the {id} path value.
func newDeleteRequest(id string) *http.Request {
	request := httptest.NewRequest(
		http.MethodDelete,
		"/admin/appointments/"+id,
		nil,
	)
	request.SetPathValue("id", id)

	return request
}

func TestDeleteAppointment_Success(t *testing.T) {
	repository := &mockAppointmentRepository{}

	handler := NewAppointmentHandler(
		services.NewAppointmentService(repository, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.DeleteAppointment(recorder, newDeleteRequest("42"))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	if recorder.Body.Len() != 0 {
		t.Errorf("expected an empty body, got %q", recorder.Body.String())
	}

	if repository.deletedID != 42 {
		t.Errorf("expected appointment 42 to be deleted, got %d", repository.deletedID)
	}
}

func TestDeleteAppointment_NotFound(t *testing.T) {
	repository := &mockAppointmentRepository{
		deleteErr: repositories.ErrAppointmentNotFound,
	}

	handler := NewAppointmentHandler(
		services.NewAppointmentService(repository, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.DeleteAppointment(recorder, newDeleteRequest("999"))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	assertErrorResponse(t, recorder, "appointment not found")
}

func TestDeleteAppointment_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		deleteErr: errors.New("database connection failed"),
	}

	handler := NewAppointmentHandler(
		services.NewAppointmentService(repository, fixedClock),
	)

	recorder := httptest.NewRecorder()

	handler.DeleteAppointment(recorder, newDeleteRequest("42"))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}

	assertErrorResponse(t, recorder, "internal server error")
}

func TestDeleteAppointment_InvalidID(t *testing.T) {
	for _, id := range []string{"abc", "0", "-1", "1.5", "99999999999999999999"} {
		t.Run(id, func(t *testing.T) {
			repository := &mockAppointmentRepository{}

			handler := NewAppointmentHandler(
				services.NewAppointmentService(repository, fixedClock),
			)

			recorder := httptest.NewRecorder()

			handler.DeleteAppointment(recorder, newDeleteRequest(id))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
			}

			assertErrorResponse(t, recorder, "invalid appointment id")

			if repository.deletedID != 0 {
				t.Error("expected no appointment to be deleted")
			}
		})
	}
}
