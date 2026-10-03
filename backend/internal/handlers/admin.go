package handlers

import (
	"net/http"
	"strconv"
)

// ListAppointments returns the upcoming appointments for the admin panel.
// It must be wrapped by the admin authorization middleware.
func (h *AppointmentHandler) ListAppointments(
	w http.ResponseWriter,
	r *http.Request,
) {
	appointments, err := h.service.ListAppointments(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	response := make([]appointmentResponse, 0, len(appointments))

	for i := range appointments {
		response = append(response, newAppointmentResponse(&appointments[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

// DeleteAppointment cancels the appointment whose ID is the {id} path
// value, responding 204 No Content. It must be wrapped by the admin
// authorization middleware.
func (h *AppointmentHandler) DeleteAppointment(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid appointment id")
		return
	}

	if err := h.service.CancelAppointment(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
