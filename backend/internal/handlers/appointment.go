package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
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

// appointmentResponse is the JSON shape returned to clients. The date is a plain calendar date (YYYY-MM-DD) so browsers do not shift it to the previous day when converting from UTC to the local timezone.
type appointmentResponse struct {
	ID              int64                    `json:"id"`
	AppointmentDate string                   `json:"appointment_date"`
	StartTime       string                   `json:"start_time"`
	EndTime         string                   `json:"end_time"`
	CustomerName    string                   `json:"customer_name"`
	CustomerPhone   string                   `json:"customer_phone"`
	CustomerEmail   string                   `json:"customer_email"`
	Status          models.AppointmentStatus `json:"status"`
	CreatedAt       time.Time                `json:"created_at"`
	// CancelledAt is null unless Status is cancelled.
	CancelledAt *time.Time `json:"cancelled_at"`
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
		Status:          appointment.Status,
		CreatedAt:       appointment.CreatedAt,
		CancelledAt:     appointment.CancelledAt,
	}
}

func NewAppointmentHandler(
	service *services.AppointmentService,
) *AppointmentHandler {
	return &AppointmentHandler{
		service: service,
	}
}

// writeServiceError maps service errors to HTTP responses without leaking internal error details to the client.
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
	case errors.Is(err, repositories.ErrAppointmentNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"appointment not found",
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

// isJSONContentType reports whether contentType is application/json, with
// or without parameters such as charset. Browsers can send a cross-site POST
// without a CORS preflight only as text/plain, a form or with no content
// type at all, so requiring JSON keeps other sites from booking through the
// browsers of their visitors.
func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)

	return err == nil && mediaType == "application/json"
}

// writeDecodeError responds to a request body that could not be decoded. err may be nil when the body holds more than one JSON value.
func writeDecodeError(
	w http.ResponseWriter,
	err error,
) {
	var maxBytesError *http.MaxBytesError

	if errors.As(err, &maxBytesError) {
		writeError(
			w,
			http.StatusRequestEntityTooLarge,
			"request body too large",
		)
		return
	}

	writeError(
		w,
		http.StatusBadRequest,
		"invalid request body",
	)
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
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeError(
			w,
			http.StatusUnsupportedMediaType,
			"content type must be application/json",
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var request createAppointmentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeDecodeError(w, err)
		return
	}

	// The body must hold a single JSON object: anything after it, such as a second object, is rejected.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeDecodeError(w, err)
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
