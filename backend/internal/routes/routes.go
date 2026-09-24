package routes

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/handlers"
)

func RegisterRoutes(
	mux *http.ServeMux,
	appointmentHandler *handlers.AppointmentHandler,
) {
	mux.HandleFunc("/health", handlers.HealthHandler)

	mux.HandleFunc(
		"/appointments/availability",
		appointmentHandler.GetAvailability,
	)
}
