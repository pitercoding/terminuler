package services

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

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
	if strings.TrimSpace(input.CustomerName) == "" {
		return fmt.Errorf("customer name is required")
	}

	if strings.TrimSpace(input.CustomerPhone) == "" {
		return fmt.Errorf("customer phone is required")
	}

	if strings.TrimSpace(input.CustomerEmail) == "" {
		return fmt.Errorf("customer email is required")
	}

	if _, err := mail.ParseAddress(input.CustomerEmail); err != nil {
		return fmt.Errorf("invalid customer email")
	}

	parsedDate, err := time.Parse("2006-01-02", input.AppointmentDate)
	if err != nil {
		return fmt.Errorf("invalid appointment date")
	}

	if parsedDate.Weekday() == time.Saturday ||
		parsedDate.Weekday() == time.Sunday {
		return fmt.Errorf("appointments are not available on weekends")
	}

	startTime, err := time.Parse("15:04", input.StartTime)
	if err != nil {
		return fmt.Errorf("invalid start time")
	}

	endTime, err := time.Parse("15:04", input.EndTime)
	if err != nil {
		return fmt.Errorf("invalid end time")
	}

	if startTime.Minute() != 0 || endTime.Minute() != 0 {
		return fmt.Errorf("appointments must start and end on the hour")
	}

	if startTime.Hour() < 8 || startTime.Hour() >= 16 {
		return fmt.Errorf("appointment start time must be between 08:00 and 15:00")
	}

	if endTime.Hour() < 9 || endTime.Hour() > 16 {
		return fmt.Errorf("appointment end time must be between 09:00 and 16:00")
	}

	if endTime.Sub(startTime) != time.Hour {
		return fmt.Errorf("appointment must last exactly one hour")
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

	startTime, err := time.Parse("15:04", input.StartTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	endTime, err := time.Parse("15:04", input.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse end time: %w", err)
	}

	appointment := &models.Appointment{
		AppointmentDate: appointmentDate,
		StartTime:       startTime,
		EndTime:         endTime,
		CustomerName:    strings.TrimSpace(input.CustomerName),
		CustomerPhone:   strings.TrimSpace(input.CustomerPhone),
		CustomerEmail:   strings.TrimSpace(input.CustomerEmail),
	}

	if err := s.repository.Create(ctx, appointment); err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}

	return appointment, nil
}

func (s *AppointmentService) GetAvailableSlots(
	ctx context.Context,
	date string,
) ([]AvailableSlot, error) {
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
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
		slot := appointment.StartTime.Format("15:04")
		bookedSlots[slot] = true
	}

	var availableSlots []AvailableSlot

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
