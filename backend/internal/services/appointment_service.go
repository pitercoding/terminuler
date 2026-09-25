package services

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pitercoding/terminuler/internal/models"
)

type AppointmentRepository interface {
	GetByDate(ctx context.Context, date string) ([]models.Appointment, error)
	Create(ctx context.Context, appointment *models.Appointment) error
}

type AvailableSlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type CreateAppointmentInput struct {
	AppointmentDate string
	StartTime       string
	EndTime         string
	CustomerName    string
	CustomerPhone   string
	CustomerEmail   string
}

// Maximum lengths mirror the VARCHAR columns of the appointments table.
const (
	maxCustomerNameLength  = 255
	maxCustomerPhoneLength = 50
	maxCustomerEmailLength = 255
)

// ValidationError indicates that the input provided by the client is invalid.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func newValidationError(message string) error {
	return &ValidationError{
		Message: message,
	}
}

type AppointmentService struct {
	repository AppointmentRepository
}

func NewAppointmentService(
	repository AppointmentRepository,
) *AppointmentService {
	return &AppointmentService{
		repository: repository,
	}
}

func (s *AppointmentService) ValidateCreateAppointment(
	input CreateAppointmentInput,
) error {
	customerName := strings.TrimSpace(input.CustomerName)
	customerPhone := strings.TrimSpace(input.CustomerPhone)
	customerEmail := strings.TrimSpace(input.CustomerEmail)

	if customerName == "" {
		return newValidationError("customer name is required")
	}

	if utf8.RuneCountInString(customerName) > maxCustomerNameLength {
		return newValidationError("customer name must have at most 255 characters")
	}

	if customerPhone == "" {
		return newValidationError("customer phone is required")
	}

	if utf8.RuneCountInString(customerPhone) > maxCustomerPhoneLength {
		return newValidationError("customer phone must have at most 50 characters")
	}

	if customerEmail == "" {
		return newValidationError("customer email is required")
	}

	if utf8.RuneCountInString(customerEmail) > maxCustomerEmailLength {
		return newValidationError("customer email must have at most 255 characters")
	}

	// mail.ParseAddress also accepts "Name <email>", so only a bare address is valid.
	address, err := mail.ParseAddress(customerEmail)
	if err != nil || address.Address != customerEmail {
		return newValidationError("invalid customer email")
	}

	parsedDate, err := time.Parse("2006-01-02", input.AppointmentDate)
	if err != nil {
		return newValidationError("invalid appointment date")
	}

	if parsedDate.Weekday() == time.Saturday ||
		parsedDate.Weekday() == time.Sunday {
		return newValidationError("appointments are not available on weekends")
	}

	startTime, err := time.Parse("15:04", input.StartTime)
	if err != nil {
		return newValidationError("invalid start time")
	}

	endTime, err := time.Parse("15:04", input.EndTime)
	if err != nil {
		return newValidationError("invalid end time")
	}

	if startTime.Minute() != 0 || endTime.Minute() != 0 {
		return newValidationError("appointments must start and end on the hour")
	}

	if startTime.Hour() < 8 || startTime.Hour() >= 16 {
		return newValidationError("appointment start time must be between 08:00 and 15:00")
	}

	if endTime.Hour() < 9 || endTime.Hour() > 16 {
		return newValidationError("appointment end time must be between 09:00 and 16:00")
	}

	if endTime.Sub(startTime) != time.Hour {
		return newValidationError("appointment must last exactly one hour")
	}

	return nil
}

func (s *AppointmentService) Create(
	ctx context.Context,
	input CreateAppointmentInput,
) (*models.Appointment, error) {
	if err := s.ValidateCreateAppointment(input); err != nil {
		return nil, err
	}

	appointmentDate, err := time.Parse(
		"2006-01-02",
		input.AppointmentDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse appointment date: %w", err)
	}

	appointment := &models.Appointment{
		AppointmentDate: appointmentDate,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		CustomerName:    strings.TrimSpace(input.CustomerName),
		CustomerPhone:   strings.TrimSpace(input.CustomerPhone),
		CustomerEmail:   strings.TrimSpace(input.CustomerEmail),
	}

	if err := s.repository.Create(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	return appointment, nil
}

func normalizeTime(value string) string {
	if len(value) >= 5 {
		return value[:5]
	}

	return value
}

func (s *AppointmentService) GetAvailableSlots(
	ctx context.Context,
	date string,
) ([]AvailableSlot, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, newValidationError("invalid date format, expected YYYY-MM-DD")
	}

	if parsedDate.Weekday() == time.Saturday ||
		parsedDate.Weekday() == time.Sunday {
		return []AvailableSlot{}, nil
	}

	appointments, err := s.repository.GetByDate(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %w", err)
	}

	bookedSlots := make(map[string]bool)

	for _, appointment := range appointments {
		startTime := normalizeTime(appointment.StartTime)
		bookedSlots[startTime] = true
	}

	// Non-nil so a fully booked day is encoded as [] instead of null.
	availableSlots := []AvailableSlot{}

	for hour := 8; hour < 16; hour++ {
		startTime := fmt.Sprintf("%02d:00", hour)
		endTime := fmt.Sprintf("%02d:00", hour+1)

		if bookedSlots[startTime] {
			continue
		}

		availableSlots = append(availableSlots, AvailableSlot{
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return availableSlots, nil
}
