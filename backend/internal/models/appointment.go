package models

import "time"

// AppointmentStatus is the lifecycle state of an appointment. Its values
// match the status column of the appointments table.
type AppointmentStatus string

const (
	// AppointmentStatusConfirmed is the status of a booked appointment,
	// which holds its slot.
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"

	// AppointmentStatusCancelled is the status of a cancelled appointment,
	// which stays stored but no longer holds its slot.
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
)

type Appointment struct {
	ID              int64             `json:"id"`
	AppointmentDate time.Time         `json:"appointment_date"`
	StartTime       string            `json:"start_time"`
	EndTime         string            `json:"end_time"`
	CustomerName    string            `json:"customer_name"`
	CustomerPhone   string            `json:"customer_phone"`
	CustomerEmail   string            `json:"customer_email"`
	Status          AppointmentStatus `json:"status"`
	CreatedAt       time.Time         `json:"created_at"`
	// CancelledAt is nil unless Status is AppointmentStatusCancelled.
	CancelledAt *time.Time `json:"cancelled_at"`
}
