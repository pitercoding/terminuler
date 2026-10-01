package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/repositories"
)

type mockAppointmentRepository struct {
	appointments []models.Appointment
	err          error
	createCalled bool
	fromDate     string
	deletedID    int64
}

type mockEmailSender struct {
	sendCalled      bool
	to              string
	customerName    string
	appointmentDate string
	startTime       string
	endTime         string
	err             error
}

func (m *mockEmailSender) SendConfirmation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	m.sendCalled = true
	m.to = to
	m.customerName = customerName
	m.appointmentDate = appointmentDate
	m.startTime = startTime
	m.endTime = endTime

	return m.err
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

func (m *mockAppointmentRepository) GetAppointments(
	ctx context.Context,
	fromDate string,
) ([]models.Appointment, error) {
	m.fromDate = fromDate

	if m.err != nil {
		return nil, m.err
	}

	return m.appointments, nil
}

func (m *mockAppointmentRepository) Create(
	ctx context.Context,
	appointment *models.Appointment,
) error {
	m.createCalled = true

	if m.err != nil {
		return m.err
	}

	appointment.ID = 1

	return nil
}

func (m *mockAppointmentRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	m.deletedID = id

	return m.err
}

// testNow is the fixed "current time" used by the tests (Friday, 2026-09-25 12:00 UTC), so they do not depend on the real date.
var testNow = time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)

func fixedClock() time.Time {
	return testNow
}

func clockAt(now time.Time) func() time.Time {
	return func() time.Time {
		return now
	}
}

func validInputFor(date string, startTime string, endTime string) CreateAppointmentInput {
	return CreateAppointmentInput{
		AppointmentDate: date,
		StartTime:       startTime,
		EndTime:         endTime,
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}
}

func TestValidateCreateAppointment_PastAndPresent(t *testing.T) {
	// Thursday, 2026-09-24 10:30 UTC
	now := time.Date(2026, time.September, 24, 10, 30, 0, 0, time.UTC)

	service := NewAppointmentService(nil, clockAt(now))

	tests := []struct {
		name        string
		input       CreateAppointmentInput
		expectError bool
	}{
		{
			name:        "date in the past",
			input:       validInputFor("2020-01-06", "10:00", "11:00"),
			expectError: true,
		},
		{
			name:        "yesterday",
			input:       validInputFor("2026-09-23", "15:00", "16:00"),
			expectError: true,
		},
		{
			name:        "today, slot already finished",
			input:       validInputFor("2026-09-24", "08:00", "09:00"),
			expectError: true,
		},
		{
			name:        "today, slot in progress",
			input:       validInputFor("2026-09-24", "10:00", "11:00"),
			expectError: true,
		},
		{
			name:        "today, next slot",
			input:       validInputFor("2026-09-24", "11:00", "12:00"),
			expectError: false,
		},
		{
			name:        "tomorrow, first slot",
			input:       validInputFor("2026-09-25", "08:00", "09:00"),
			expectError: false,
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

func TestValidateCreateAppointment_UsesClockTimezone(t *testing.T) {
	saoPaulo, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// 12:30 UTC is 09:30 in São Paulo (UTC-3).
	now := time.Date(2026, time.September, 24, 12, 30, 0, 0, time.UTC).
		In(saoPaulo)

	service := NewAppointmentService(nil, clockAt(now))

	// 10:00 is in the future in São Paulo, although 10:00 UTC has passed.
	if err := service.ValidateCreateAppointment(
		validInputFor("2026-09-24", "10:00", "11:00"),
	); err != nil {
		t.Fatalf("expected 10:00 to be bookable, got %v", err)
	}

	if err := service.ValidateCreateAppointment(
		validInputFor("2026-09-24", "09:00", "10:00"),
	); err == nil {
		t.Fatal("expected 09:00 to be in the past")
	}
}

func TestValidateCreateAppointment_BookingHorizon(t *testing.T) {
	// testNow is Friday, 2026-09-25, so the last bookable date is Tuesday, 2026-11-24.
	service := NewAppointmentService(nil, fixedClock)

	tests := []struct {
		name        string
		date        string
		expectError bool
	}{
		{
			name:        "last bookable date",
			date:        "2026-11-24",
			expectError: false,
		},
		{
			name:        "day after the last bookable date",
			date:        "2026-11-25",
			expectError: true,
		},
		{
			name:        "far future",
			date:        "2099-06-01",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateCreateAppointment(
				validInputFor(tt.date, "10:00", "11:00"),
			)

			if !tt.expectError {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}

				return
			}

			var validationError *ValidationError

			if !errors.As(err, &validationError) {
				t.Fatalf("expected a ValidationError, got %v", err)
			}

			expected := "appointment date must be within the next 60 days"

			if validationError.Message != expected {
				t.Fatalf("expected %q, got %q", expected, validationError.Message)
			}
		})
	}
}

func TestValidateCreateAppointment_BookingHorizonUsesClockTimezone(t *testing.T) {
	saoPaulo, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	// 01:00 UTC on 2026-09-25 is still 22:00 on 2026-09-24 in São Paulo, so
	// the horizon counts from 2026-09-24 and ends on 2026-11-23.
	now := time.Date(2026, time.September, 25, 1, 0, 0, 0, time.UTC).
		In(saoPaulo)

	service := NewAppointmentService(nil, clockAt(now))

	if err := service.ValidateCreateAppointment(
		validInputFor("2026-11-23", "10:00", "11:00"),
	); err != nil {
		t.Fatalf("expected 2026-11-23 to be bookable, got %v", err)
	}

	if err := service.ValidateCreateAppointment(
		validInputFor("2026-11-24", "10:00", "11:00"),
	); err == nil {
		t.Fatal("expected 2026-11-24 to be beyond the booking horizon")
	}
}

func TestGetAvailableSlots_BookingHorizon(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-11-24",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 8 {
		t.Fatalf("expected 8 slots on the last bookable date, got %d", len(slots))
	}

	repository.err = errors.New("repository should not be called")

	slots, err = service.GetAvailableSlots(
		context.Background(),
		"2026-11-25",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if slots == nil || len(slots) != 0 {
		t.Fatalf("expected empty non-nil slice beyond the horizon, got %v", slots)
	}
}

func TestGetAvailableSlots_PastDate(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("repository should not be called"),
	}

	service := NewAppointmentService(repository, fixedClock)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-24",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 0 {
		t.Fatalf("expected 0 available slots, got %d", len(slots))
	}
}

func TestGetAvailableSlots_TodayHidesStartedSlots(t *testing.T) {
	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				StartTime: "14:00:00",
				EndTime:   "15:00:00",
			},
		},
	}

	// Friday, 2026-09-25 12:00: 08:00-12:00 have started, 14:00 is booked.
	service := NewAppointmentService(repository, fixedClock)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-25",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"13:00", "15:00"}

	if len(slots) != len(expected) {
		t.Fatalf("expected slots %v, got %v", expected, slots)
	}

	for i, slot := range slots {
		if slot.StartTime != expected[i] {
			t.Errorf(
				"expected slot %d to start at %s, got %s",
				i,
				expected[i],
				slot.StartTime,
			)
		}
	}
}

func TestGetAvailableSlots_TodayAfterLastSlotStarted(t *testing.T) {
	now := time.Date(2026, time.September, 25, 15, 0, 0, 0, time.UTC)

	service := NewAppointmentService(
		&mockAppointmentRepository{},
		clockAt(now),
	)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-25",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if slots == nil || len(slots) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", slots)
	}
}

func TestNewAppointmentService_NilClockUsesTimeNow(t *testing.T) {
	service := NewAppointmentService(nil, nil)

	if service.now == nil {
		t.Fatal("expected default clock")
	}

	if time.Since(service.now()) > time.Minute {
		t.Fatal("expected default clock to return the current time")
	}
}

func TestGetAvailableSlots_WeekdayWithoutAppointments(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

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
		t.Errorf(
			"expected first slot to start at 08:00, got %s",
			slots[0].StartTime,
		)
	}

	if slots[7].StartTime != "15:00" {
		t.Errorf(
			"expected last slot to start at 15:00, got %s",
			slots[7].StartTime,
		)
	}
}

func TestGetAvailableSlots_Saturday(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

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

	service := NewAppointmentService(repository, fixedClock)

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

	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				ID:              1,
				AppointmentDate: appointmentDate,
				StartTime:       "10:00:00",
				EndTime:         "11:00:00",
				CustomerName:    "Test Customer",
				CustomerPhone:   "+4915112345678",
				CustomerEmail:   "test@example.com",
			},
		},
	}

	service := NewAppointmentService(repository, fixedClock)

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

	service := NewAppointmentService(repository, fixedClock)

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

	service := NewAppointmentService(repository, fixedClock)

	_, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err == nil {
		t.Fatal("expected an error from repository")
	}
}

func TestValidateCreateAppointment(t *testing.T) {
	service := NewAppointmentService(nil, fixedClock)

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
		{
			name: "valid first slot of the day",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "08:00",
				EndTime:         "09:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: false,
		},
		{
			name: "valid last slot of the day",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "15:00",
				EndTime:         "16:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: false,
		},
		{
			name: "end time before start time",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "11:00",
				EndTime:         "10:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "invalid start time format",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10h",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "invalid end time format",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "non-existent date",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-02-30",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "whitespace-only customer name",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "   ",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "email with display name",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "Racha Cuca <rc@exemple.com>",
			},
			expectError: true,
		},
		{
			name: "customer name too long",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    strings.Repeat("a", 256),
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "customer name with multibyte characters at limit",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    strings.Repeat("ã", 255),
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: false,
		},
		{
			name: "customer phone too long",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   strings.Repeat("1", 51),
				CustomerEmail:   "rc@exemple.com",
			},
			expectError: true,
		},
		{
			name: "customer email too long",
			input: CreateAppointmentInput{
				AppointmentDate: "2026-09-28",
				StartTime:       "10:00",
				EndTime:         "11:00",
				CustomerName:    "Racha Cuca",
				CustomerPhone:   "+5511999999999",
				CustomerEmail:   strings.Repeat("a", 250) + "@x.com",
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

			var validationError *ValidationError

			if tt.expectError && !errors.As(err, &validationError) {
				t.Fatalf("expected a ValidationError, got %T", err)
			}
		})
	}
}

func TestCreateAppointment_Success(t *testing.T) {
	repository := &mockAppointmentRepository{}
	emailSender := &mockEmailSender{}

	service := NewAppointmentServiceWithEmail(
		repository,
		emailSender,
		fixedClock,
	)

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

	if appointment.StartTime != "10:00" {
		t.Errorf(
			"expected start time 10:00, got %s",
			appointment.StartTime,
		)
	}

	if appointment.EndTime != "11:00" {
		t.Errorf(
			"expected end time 11:00, got %s",
			appointment.EndTime,
		)
	}

	if !emailSender.sendCalled {
		t.Fatal("expected confirmation email to be sent")
	}

	if emailSender.to != "rc@exemple.com" {
		t.Errorf(
			"expected email recipient 'rc@exemple.com', got %q",
			emailSender.to,
		)
	}

	if emailSender.customerName != "Racha Cuca" {
		t.Errorf(
			"expected email customer name 'Racha Cuca', got %q",
			emailSender.customerName,
		)
	}

	if emailSender.appointmentDate != "2026-09-28" {
		t.Errorf(
			"expected email appointment date '2026-09-28', got %q",
			emailSender.appointmentDate,
		)
	}

	if emailSender.startTime != "10:00" {
		t.Errorf(
			"expected email start time '10:00', got %q",
			emailSender.startTime,
		)
	}

	if emailSender.endTime != "11:00" {
		t.Errorf(
			"expected email end time '11:00', got %q",
			emailSender.endTime,
		)
	}
}

func TestCreateAppointment_NormalizesSingleDigitHours(t *testing.T) {
	repository := &mockAppointmentRepository{}
	emailSender := &mockEmailSender{}

	service := NewAppointmentServiceWithEmail(
		repository,
		emailSender,
		fixedClock,
	)

	appointment, err := service.Create(
		context.Background(),
		validInputFor("2026-09-28", "8:00", "9:00"),
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if appointment.StartTime != "08:00" || appointment.EndTime != "09:00" {
		t.Errorf(
			"expected stored times 08:00-09:00, got %s-%s",
			appointment.StartTime,
			appointment.EndTime,
		)
	}

	if emailSender.startTime != "08:00" || emailSender.endTime != "09:00" {
		t.Errorf(
			"expected email times 08:00-09:00, got %s-%s",
			emailSender.startTime,
			emailSender.endTime,
		)
	}
}

func TestCreateAppointment_ValidationError(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

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

	if repository.createCalled {
		t.Fatal("expected repository not to be called on validation error")
	}
}

func TestCreateAppointment_ConflictError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: repositories.ErrAppointmentConflict,
	}

	service := NewAppointmentService(repository, fixedClock)

	input := CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       "10:00",
		EndTime:         "11:00",
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}

	_, err := service.Create(
		context.Background(),
		input,
	)

	if !errors.Is(err, repositories.ErrAppointmentConflict) {
		t.Fatalf("expected ErrAppointmentConflict, got %v", err)
	}
}

func TestCreateAppointment_ParsesAppointmentDate(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

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

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedDate := time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)

	if !appointment.AppointmentDate.Equal(expectedDate) {
		t.Errorf(
			"expected appointment date %v, got %v",
			expectedDate,
			appointment.AppointmentDate,
		)
	}
}

func TestGetAvailableSlots_FullyBooked(t *testing.T) {
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

	service := NewAppointmentService(repository, fixedClock)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if slots == nil {
		t.Fatal("expected empty slice, got nil")
	}

	if len(slots) != 0 {
		t.Fatalf("expected 0 available slots, got %d", len(slots))
	}
}

func TestGetAvailableSlots_SlotEndTimes(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

	slots, err := service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, slot := range slots {
		start, _ := time.Parse("15:04", slot.StartTime)
		end, _ := time.Parse("15:04", slot.EndTime)

		if end.Sub(start) != time.Hour {
			t.Errorf(
				"expected slot %s-%s to last one hour",
				slot.StartTime,
				slot.EndTime,
			)
		}
	}
}

func TestGetAvailableSlots_ErrorTypes(t *testing.T) {
	var validationError *ValidationError

	service := NewAppointmentService(&mockAppointmentRepository{}, fixedClock)

	_, err := service.GetAvailableSlots(
		context.Background(),
		"invalid",
	)

	if !errors.As(err, &validationError) {
		t.Fatalf("expected ValidationError for invalid date, got %T", err)
	}

	service = NewAppointmentService(&mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}, fixedClock)

	_, err = service.GetAvailableSlots(
		context.Background(),
		"2026-09-28",
	)

	if errors.As(err, &validationError) {
		t.Fatal("expected repository error not to be a ValidationError")
	}
}

func TestCreateAppointment_RepositoryError(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	service := NewAppointmentService(repository, fixedClock)

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

func TestCreateAppointment_EmailErrorDoesNotFailAppointment(t *testing.T) {
	repository := &mockAppointmentRepository{}

	emailSender := &mockEmailSender{
		err: errors.New("email provider unavailable"),
	}

	service := NewAppointmentServiceWithEmail(
		repository,
		emailSender,
		fixedClock,
	)

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

	if err != nil {
		t.Fatalf(
			"expected appointment creation to succeed despite email error, got %v",
			err,
		)
	}

	if appointment == nil {
		t.Fatal("expected appointment, got nil")
	}

	if appointment.ID != 1 {
		t.Errorf(
			"expected appointment ID 1, got %d",
			appointment.ID,
		)
	}

	if !emailSender.sendCalled {
		t.Fatal("expected confirmation email to be attempted")
	}
}

func TestCreateAppointment_DatabaseErrorDoesNotSendEmail(t *testing.T) {
	repository := &mockAppointmentRepository{
		err: errors.New("database connection failed"),
	}

	emailSender := &mockEmailSender{}

	service := NewAppointmentServiceWithEmail(
		repository,
		emailSender,
		fixedClock,
	)

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
		t.Fatal("expected database error, got nil")
	}

	if appointment != nil {
		t.Fatal("expected nil appointment, got an appointment")
	}

	if emailSender.sendCalled {
		t.Fatal("expected email not to be sent when appointment creation fails")
	}
}

func TestListAppointments_ReturnsAppointmentsFromToday(t *testing.T) {
	repository := &mockAppointmentRepository{
		appointments: []models.Appointment{
			{
				ID:              1,
				AppointmentDate: time.Date(2026, time.September, 25, 0, 0, 0, 0, time.UTC),
				StartTime:       "08:00:00",
				EndTime:         "09:00:00",
				CustomerName:    "Racha Cuca",
			},
			{
				ID:              2,
				AppointmentDate: time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC),
				StartTime:       "14:00:00",
				EndTime:         "15:00:00",
				CustomerName:    "Jane Doe",
			},
		},
	}

	service := NewAppointmentService(repository, fixedClock)

	appointments, err := service.ListAppointments(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.fromDate != "2026-09-25" {
		t.Errorf("expected appointments from 2026-09-25, got %q", repository.fromDate)
	}

	if len(appointments) != 2 {
		t.Fatalf("expected 2 appointments, got %d", len(appointments))
	}

	// PostgreSQL returns TIME values with seconds; the service trims them.
	expectedTimes := [][2]string{{"08:00", "09:00"}, {"14:00", "15:00"}}

	for i, appointment := range appointments {
		if appointment.ID != repository.appointments[i].ID {
			t.Errorf("expected appointment %d to keep its order", i)
		}

		if appointment.StartTime != expectedTimes[i][0] ||
			appointment.EndTime != expectedTimes[i][1] {
			t.Errorf(
				"expected appointment %d at %s-%s, got %s-%s",
				i,
				expectedTimes[i][0],
				expectedTimes[i][1],
				appointment.StartTime,
				appointment.EndTime,
			)
		}
	}
}

func TestListAppointments_UsesClockTimezone(t *testing.T) {
	// 2026-09-25 23:30 UTC is already 2026-09-26 in Berlin.
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load location: %v", err)
	}

	now := time.Date(2026, time.September, 25, 23, 30, 0, 0, time.UTC).In(berlin)

	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, clockAt(now))

	if _, err := service.ListAppointments(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.fromDate != "2026-09-26" {
		t.Errorf("expected appointments from 2026-09-26, got %q", repository.fromDate)
	}
}

func TestListAppointments_EmptyIsNotNil(t *testing.T) {
	service := NewAppointmentService(&mockAppointmentRepository{}, fixedClock)

	appointments, err := service.ListAppointments(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if appointments == nil {
		t.Fatal("expected an empty slice, got nil")
	}
}

func TestListAppointments_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	service := NewAppointmentService(
		&mockAppointmentRepository{err: repositoryErr},
		fixedClock,
	)

	_, err := service.ListAppointments(context.Background())

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error to be wrapped, got %v", err)
	}
}

func TestDeleteAppointment_DeletesGivenID(t *testing.T) {
	repository := &mockAppointmentRepository{}

	service := NewAppointmentService(repository, fixedClock)

	if err := service.DeleteAppointment(context.Background(), 42); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repository.deletedID != 42 {
		t.Errorf("expected appointment 42 to be deleted, got %d", repository.deletedID)
	}
}

func TestDeleteAppointment_NotFound(t *testing.T) {
	service := NewAppointmentService(
		&mockAppointmentRepository{err: repositories.ErrAppointmentNotFound},
		fixedClock,
	)

	err := service.DeleteAppointment(context.Background(), 999)

	if !errors.Is(err, repositories.ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound, got %v", err)
	}
}

func TestDeleteAppointment_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	service := NewAppointmentService(
		&mockAppointmentRepository{err: repositoryErr},
		fixedClock,
	)

	err := service.DeleteAppointment(context.Background(), 42)

	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error to be wrapped, got %v", err)
	}
}
