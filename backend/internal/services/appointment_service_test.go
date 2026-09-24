package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
)

type mockAppointmentRepository struct {
	appointments []models.Appointment
	err          error
}

func (m *mockAppointmentRepository) GetByDate(
	ctx context.Context,
	date string,
) ([]models.Appointment, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.appointments, nil
}

func TestGetAvailableSlots_WeekdayWithoutAppointments(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 8 {
		t.Fatalf("expected 8 available slots, got %d", len(slots))
	}

	if slots[0].StartTime != "08:00" {
		t.Errorf("expected first slot to start at 08:00, got %s", slots[0].StartTime)
	}

	if slots[7].StartTime != "15:00" {
		t.Errorf("expected last slot to start at 15:00, got %s", slots[7].StartTime)
	}
}

func TestGetAvailableSlots_Saturday(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-26",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 0 {
		t.Fatalf("expected 0 available slots, got %d", len(slots))
	}
}

func TestGetAvailableSlots_Sunday(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-27",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 0 {
		t.Fatalf("expected 0 available slots, got %d", len(slots))
	}
}

func TestGetAvailableSlots_WithBookedAppointment(t *testing.T) {
	appointmentDate := time.Date(
		2026,
		time.September,
		28,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	startTime := time.Date(
		2026,
		time.September,
		28,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	endTime := time.Date(
		2026,
		time.September,
		28,
		11,
		0,
		0,
		0,
		time.UTC,
	)

	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				ID:              1,
				AppointmentDate: appointmentDate,
				StartTime:       startTime,
				EndTime:         endTime,
				CustomerName:    "John Doe",
				CustomerPhone:   "+49123456789",
				CustomerEmail:   "john@example.com",
			},
		},
	}

	service := NewAppointmentService(repository)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 7 {
		t.Fatalf("expected 7 available slots, got %d", len(slots))
	}

	for _, slot := range slots {
		if slot.StartTime == "10:00" {
			t.Error("expected 10:00 slot to be unavailable")
		}
	}
}

func TestGetAvailableSlots_InvalidDate(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	_, err := service.GetAvailableSlots(
		context.Background(),
		"28-09-2026",
	)

	if err == nil {
		t.Fatal("expected an error for invalid date")
	}
}

func TestGetAvailableSlots_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	service := NewAppointmentService(repository)

	_, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err == nil {
		t.Fatal("expected an error from repository")
	}
}
