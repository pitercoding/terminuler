package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/services"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

type AppointmentHandler struct {
	service *services.AppointmentService
}

type createAppointmentRequest struct {
	AppointmentDate string `json:"appointment_date"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
}

// appointmentResponse is the JSON shape returned to clients. The date is a
// plain calendar date (YYYY-MM-DD) so browsers do not shift it to the
// previous day when converting from UTC to the local timezone.
type appointmentResponse struct {
	ID              int64     `json:"id"`
	AppointmentDate string    `json:"appointment_date"`
	StartTime       string    `json:"start_time"`
	EndTime         string    `json:"end_time"`
	CustomerName    string    `json:"customer_name"`
	CustomerPhone   string    `json:"customer_phone"`
	CustomerEmail   string    `json:"customer_email"`
	CreatedAt       time.Time `json:"created_at"`
}

func newAppointmentResponse(
	appointment *models.Appointment,
) appointmentResponse {
	return appointmentResponse{
		ID:              appointment.ID,
		AppointmentDate: appointment.AppointmentDate.Format("2006-01-02"),
		StartTime:       appointment.StartTime,
		EndTime:         appointment.EndTime,
		CustomerName:    appointment.CustomerName,
		CustomerPhone:   appointment.CustomerPhone,
		CustomerEmail:   appointment.CustomerEmail,
		CreatedAt:       appointment.CreatedAt,
	}
}

func NewAppointmentHandler(
	service *services.AppointmentService,
) *AppointmentHandler {
	return &AppointmentHandler{
		service: service,
	}
}

// writeServiceError maps service errors to HTTP responses without
// leaking internal error details to the client.
func writeServiceError(
	w http.ResponseWriter,
	err error,
) {
	var validationError *services.ValidationError

	switch {
	case errors.As(err, &validationError):
		writeError(
			w,
			http.StatusBadRequest,
			validationError.Message,
		)
	case errors.Is(err, repositories.ErrAppointmentConflict):
		writeError(
			w,
			http.StatusConflict,
			"appointment slot is already booked",
		)
	default:
		log.Printf("internal server error: %v", err)

		writeError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
	}
}

func (h *AppointmentHandler) GetAvailability(
	w http.ResponseWriter,
	r *http.Request,
) {
	date := r.URL.Query().Get("date")

	if date == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"date query parameter is required",
		)
		return
	}

	slots, err := h.service.GetAvailableSlots(
		r.Context(),
		date,
	)

	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := struct {
		Date           string                   `json:"date"`
		AvailableSlots []services.AvailableSlot `json:"available_slots"`
	}{
		Date:           date,
		AvailableSlots: slots,
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *AppointmentHandler) CreateAppointment(
	w http.ResponseWriter,
	r *http.Request,
) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var request createAppointmentRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	input := services.CreateAppointmentInput{
		AppointmentDate: request.AppointmentDate,
		StartTime:       request.StartTime,
		EndTime:         request.EndTime,
		CustomerName:    request.CustomerName,
		CustomerPhone:   request.CustomerPhone,
		CustomerEmail:   request.CustomerEmail,
	}

	appointment, err := h.service.Create(
		r.Context(),
		input,
	)

	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		newAppointmentResponse(appointment),
	)
}
