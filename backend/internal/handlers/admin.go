package handlers

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/auth"
)

type adminSessionResponse struct {
	UserID string `json:"user_id"`
}

// AdminSessionHandler returns the authenticated admin, letting the admin
// panel confirm its session token is accepted by the API. It must be
// wrapped by the admin authorization middleware.
func AdminSessionHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, adminSessionResponse{
		UserID: userID,
	})
}

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
