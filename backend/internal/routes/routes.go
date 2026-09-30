package routes

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/handlers"
)

// RegisterRoutes uses method-aware patterns (Go 1.22+): requests with any
// other method get 405 Method Not Allowed with an Allow header, and GET
// routes also answer HEAD.
//
// createAppointmentLimit wraps only POST /appointments, the one route that
// writes data and sends an email; reads are left unlimited.
//
// requireAdmin wraps every /admin route, so only the admin can reach them.
func RegisterRoutes(
	mux *http.ServeMux,
	appointmentHandler *handlers.AppointmentHandler,
	createAppointmentLimit func(http.Handler) http.Handler,
	requireAdmin func(http.Handler) http.Handler,
) {
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	mux.HandleFunc(
		"GET /appointments/availability",
		appointmentHandler.GetAvailability,
	)

	mux.Handle(
		"POST /appointments",
		createAppointmentLimit(
			http.HandlerFunc(appointmentHandler.CreateAppointment),
		),
	)

	mux.Handle(
		"GET /admin/session",
		requireAdmin(
			http.HandlerFunc(handlers.AdminSessionHandler),
		),
	)
}
