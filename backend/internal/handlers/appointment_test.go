package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/services"
)

type mockAppointmentRepository struct {
	err error
}

func (m *mockAppointmentRepository) GetByDate(
	ctx context.Context,
	date string,
) ([]models.Appointment, error) {
	return nil, nil
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

func TestCreateAppointment_Success(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository)

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

	if !strings.Contains(
		recorder.Body.String(),
		`"id":1`,
	) {
		t.Fatalf(
			"expected response to contain appointment ID, got %s",
			recorder.Body.String(),
		)
	}
}

func TestCreateAppointment_InvalidJSON(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository)

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
}

func TestCreateAppointment_ValidationError(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository)

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
}

func TestCreateAppointment_Conflict(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: repositories.ErrAppointmentConflict,
	}

	service := services.NewAppointmentService(repository)

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
}

func TestCreateAppointment_MethodNotAllowed(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := services.NewAppointmentService(repository)

	handler := NewAppointmentHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/appointments",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.CreateAppointment(
		recorder,
		request,
	)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusMethodNotAllowed,
			recorder.Code,
		)
	}
}

func TestCreateAppointment_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	service := services.NewAppointmentService(repository)

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

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}
