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

// Appointment is a row of the appointments table. It is never encoded to
// JSON directly: the handlers convert it to their own response type, which
// formats the date as YYYY-MM-DD.
type Appointment struct {
	ID              int64
	AppointmentDate time.Time
	StartTime       string
	EndTime         string
	CustomerName    string
	CustomerPhone   string
	CustomerEmail   string
	Status          AppointmentStatus
	CreatedAt       time.Time
	// CancelledAt is nil unless Status is AppointmentStatusCancelled.
	CancelledAt *time.Time
}
