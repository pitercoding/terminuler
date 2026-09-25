package models

import "time"

type Appointment struct {
	ID              int64     `json:"id"`
	AppointmentDate time.Time `json:"appointment_date"`
	StartTime       string    `json:"start_time"`
	EndTime         string    `json:"end_time"`
	CustomerName    string    `json:"customer_name"`
	CustomerPhone   string    `json:"customer_phone"`
	CustomerEmail   string    `json:"customer_email"`
	CreatedAt       time.Time `json:"created_at"`
}
