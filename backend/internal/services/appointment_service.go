package services

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pitercoding/terminuler/internal/email"
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

const (
	openingHour = 8
	closingHour = 16
)

// maxBookingDays is how far ahead appointments can be booked, counting from today in the business timezone.
const maxBookingDays = 60

type AppointmentService struct {
	repository AppointmentRepository
	email      email.Sender
	now        func() time.Time
}

// NewAppointmentService creates the service. now returns the current time
// in the business timezone; appointment dates and times are interpreted in
// that timezone. Passing nil uses time.Now.
func NewAppointmentService(
	repository AppointmentRepository,
	now func() time.Time,
) *AppointmentService {
	return NewAppointmentServiceWithEmail(
		repository,
		nil,
		now,
	)
}

// NewAppointmentServiceWithEmail is like NewAppointmentService but also sends a confirmation email through emailSender after each successful booking. A nil emailSender disables confirmation emails.
func NewAppointmentServiceWithEmail(
	repository AppointmentRepository,
	emailSender email.Sender,
	now func() time.Time,
) *AppointmentService {
	if now == nil {
		now = time.Now
	}

	return &AppointmentService{
		repository: repository,
		email:      emailSender,
		now:        now,
	}
}

// slotStart combines a date and an hour into a point in time in the business timezone.
func slotStart(date time.Time, hour int, location *time.Location) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		hour,
		0,
		0,
		0,
		location,
	)
}

// lastBookableDate returns the latest date that can be booked. The result is a calendar date at midnight UTC, like dates parsed from "2006-01-02", so both can be compared directly.
func lastBookableDate(now time.Time) time.Time {
	today := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)

	return today.AddDate(0, 0, maxBookingDays)
}

func (s *AppointmentService) ValidateCreateAppointment(
	input CreateAppointmentInput,
) error {
	_, err := s.newAppointment(input)
	return err
}

// newAppointment validates input and returns the appointment to store, with customer fields trimmed and times normalized to HH:MM.
func (s *AppointmentService) newAppointment(
	input CreateAppointmentInput,
) (*models.Appointment, error) {
	customerName := strings.TrimSpace(input.CustomerName)
	customerPhone := strings.TrimSpace(input.CustomerPhone)
	customerEmail := strings.TrimSpace(input.CustomerEmail)

	if customerName == "" {
		return nil, newValidationError("customer name is required")
	}

	if utf8.RuneCountInString(customerName) > maxCustomerNameLength {
		return nil, newValidationError("customer name must have at most 255 characters")
	}

	if customerPhone == "" {
		return nil, newValidationError("customer phone is required")
	}

	if utf8.RuneCountInString(customerPhone) > maxCustomerPhoneLength {
		return nil, newValidationError("customer phone must have at most 50 characters")
	}

	if customerEmail == "" {
		return nil, newValidationError("customer email is required")
	}

	if utf8.RuneCountInString(customerEmail) > maxCustomerEmailLength {
		return nil, newValidationError("customer email must have at most 255 characters")
	}

	// mail.ParseAddress also accepts "Name <email>", so only a bare address is valid.
	address, err := mail.ParseAddress(customerEmail)
	if err != nil || address.Address != customerEmail {
		return nil, newValidationError("invalid customer email")
	}

	parsedDate, err := time.Parse("2006-01-02", input.AppointmentDate)
	if err != nil {
		return nil, newValidationError("invalid appointment date")
	}

	if parsedDate.Weekday() == time.Saturday ||
		parsedDate.Weekday() == time.Sunday {
		return nil, newValidationError("appointments are not available on weekends")
	}

	startTime, err := time.Parse("15:04", input.StartTime)
	if err != nil {
		return nil, newValidationError("invalid start time")
	}

	endTime, err := time.Parse("15:04", input.EndTime)
	if err != nil {
		return nil, newValidationError("invalid end time")
	}

	if startTime.Minute() != 0 || endTime.Minute() != 0 {
		return nil, newValidationError("appointments must start and end on the hour")
	}

	if startTime.Hour() < openingHour || startTime.Hour() >= closingHour {
		return nil, newValidationError("appointment start time must be between 08:00 and 15:00")
	}

	if endTime.Hour() < openingHour+1 || endTime.Hour() > closingHour {
		return nil, newValidationError("appointment end time must be between 09:00 and 16:00")
	}

	if endTime.Sub(startTime) != time.Hour {
		return nil, newValidationError("appointment must last exactly one hour")
	}

	now := s.now()

	if !slotStart(parsedDate, startTime.Hour(), now.Location()).After(now) {
		return nil, newValidationError("appointment must be scheduled in the future")
	}

	if parsedDate.After(lastBookableDate(now)) {
		return nil, newValidationError(fmt.Sprintf(
			"appointment date must be within the next %d days",
			maxBookingDays,
		))
	}

	// time.Parse accepts single-digit hours ("8:00"), so the stored and returned times are formatted again as HH:MM.
	return &models.Appointment{
		AppointmentDate: parsedDate,
		StartTime:       startTime.Format("15:04"),
		EndTime:         endTime.Format("15:04"),
		CustomerName:    customerName,
		CustomerPhone:   customerPhone,
		CustomerEmail:   customerEmail,
	}, nil
}

func (s *AppointmentService) Create(
	ctx context.Context,
	input CreateAppointmentInput,
) (*models.Appointment, error) {
	appointment, err := s.newAppointment(input)
	if err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	// The appointment is already stored, so an email failure is only logged: returning an error here would tell the client the booking failed.
	if s.email != nil {
		if err := s.email.SendConfirmation(
			appointment.CustomerEmail,
			appointment.CustomerName,
			appointment.AppointmentDate.Format("2006-01-02"),
			appointment.StartTime,
			appointment.EndTime,
		); err != nil {
			log.Printf(
				"failed to send confirmation email for appointment %d: %v",
				appointment.ID,
				err,
			)
		}
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

	now := s.now()

	if parsedDate.After(lastBookableDate(now)) {
		return []AvailableSlot{}, nil
	}

	// The last slot of the day has already started: nothing left to book.
	if !slotStart(parsedDate, closingHour-1, now.Location()).After(now) {
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

	for hour := openingHour; hour < closingHour; hour++ {
		startTime := fmt.Sprintf("%02d:00", hour)
		endTime := fmt.Sprintf("%02d:00", hour+1)

		if bookedSlots[startTime] {
			continue
		}

		if !slotStart(parsedDate, hour, now.Location()).After(now) {
			continue
		}

		availableSlots = append(availableSlots, AvailableSlot{
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return availableSlots, nil
}
