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
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
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

func TestValidateCreateAppointment(t *testing.T) {
	service := NewAppointmentService(nil)

	tests := []struct {
		name        string
		input       CreateAppointmentInput
		expectError bool
	}{
		{
			name: "valid appointment",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: false,
		},
		{
			name: "missing customer name",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "missing customer phone",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "missing customer email",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
			},
			expectError: true,
		},
		{
			name: "invalid email",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "invalid-email",
			},
			expectError: true,
		},
		{
			name: "invalid date",
			input: CreateAppointmentInput{
				AppointmentDate: "28-09-2026",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "saturday",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-26",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "sunday",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-27",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "start time before business hours",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "07:00",
				EndTime:         "08:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "start time at 16:00",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "16:00",
				EndTime:         "17:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "end time after business hours",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "15:00",
				EndTime:         "17:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "appointment shorter than one hour",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "10:30",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "appointment longer than one hour",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "12:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "appointment does not start on the hour",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:30",
				EndTime:         "11:30",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateCreateAppointment(tt.input)

			if tt.expectError && err == nil {
				t.Fatal("expected an error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestCreateAppointment_Success(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	input := CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       "10:00",
		EndTime:         "11:00",
		CustomerName:    "  Racha Cuca  ",
		CustomerPhone:   " +5511999999999 ",
		CustomerEmail:   " rc@exemple.com ",
	}

	appointment, err := service.Create(
		context.Background(),
		input,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if appointment == nil {
		t.Fatal("expected appointment, got nil")
	}

	if appointment.ID != 1 {
		t.Errorf("expected appointment ID 1, got %d", appointment.ID)
	}

	if appointment.CustomerName != "Racha Cuca" {
		t.Errorf(
			"expected customer name 'Racha Cuca', got %q",
			appointment.CustomerName,
		)
	}

	if appointment.CustomerPhone != "+5511999999999" {
		t.Errorf(
			"expected customer phone '+5511999999999', got %q",
			appointment.CustomerPhone,
		)
	}

	if appointment.CustomerEmail != "rc@exemple.com" {
		t.Errorf(
			"expected customer email 'rc@exemple.com', got %q",
			appointment.CustomerEmail,
		)
	}

	if appointment.StartTime.Hour() != 10 {
		t.Errorf(
			"expected start time 10:00, got %s",
			appointment.StartTime.Format("15:04"),
		)
	}

	if appointment.EndTime.Hour() != 11 {
		t.Errorf(
			"expected end time 11:00, got %s",
			appointment.EndTime.Format("15:04"),
		)
	}
}

func TestCreateAppointment_ValidationError(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository)

	input := CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       "10:00",
		EndTime:         "11:00",
		CustomerName:    "",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}

	appointment, err := service.Create(
		context.Background(),
		input,
	)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	if appointment != nil {
		t.Fatal("expected nil appointment, got an appointment")
	}
}

func TestCreateAppointment_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	service := NewAppointmentService(repository)

	input := CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       "10:00",
		EndTime:         "11:00",
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}

	appointment, err := service.Create(
		context.Background(),
		input,
	)

	if err == nil {
		t.Fatal("expected repository error, got nil")
	}

	if appointment != nil {
		t.Fatal("expected nil appointment, got an appointment")
	}
}
