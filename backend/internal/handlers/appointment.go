package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/services"
)

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

func NewAppointmentHandler(
	service *services.AppointmentService,
) *AppointmentHandler {
	return &AppointmentHandler{
		service: service,
	}
}

func (h *AppointmentHandler) GetAvailability(
	w http.ResponseWriter,
	r *http.Request,
) {
	date := r.URL.Query().Get("date")

	if date == "" {
		http.Error(
			w,
			"date query parameter is required",
			http.StatusBadRequest,
		)
		return
	}

	slots, err := h.service.GetAvailableSlots(
		r.Context(),
		date,
	)

	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	response := struct {
		Date           string                   `json:"date"`
		AvailableSlots []services.AvailableSlot `json:"available_slots"`
	}{
		Date:           date,
		AvailableSlots: slots,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
	}
}

func (h *AppointmentHandler) CreateAppointment(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	var request createAppointmentRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
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
		if errors.Is(err, repositories.ErrAppointmentConflict) {
			http.Error(
				w,
				"appointment slot is already booked",
				http.StatusConflict,
			)
			return
		}

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(appointment); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
	}
}
