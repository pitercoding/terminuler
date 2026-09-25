package routes

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/handlers"
)

// RegisterRoutes uses method-aware patterns (Go 1.22+): requests with any
// other method get 405 Method Not Allowed with an Allow header, and GET
// routes also answer HEAD.
func RegisterRoutes(
	mux *http.ServeMux,
	appointmentHandler *handlers.AppointmentHandler,
) {
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	mux.HandleFunc(
		"GET /appointments/availability",
		appointmentHandler.GetAvailability,
	)

	mux.HandleFunc(
		"POST /appointments",
		appointmentHandler.CreateAppointment,
	)
}
