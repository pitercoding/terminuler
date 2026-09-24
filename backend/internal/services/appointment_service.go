package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
)

type AppointmentRepository interface {
	GetByDate(ctx context.Context, date string) ([]models.Appointment, error)
}

type AvailableSlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
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
