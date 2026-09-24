package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pitercoding/terminuler/internal/services"
)

type AppointmentHandler struct {
	service *services.AppointmentService
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
