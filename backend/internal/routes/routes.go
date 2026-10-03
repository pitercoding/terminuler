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
// requireAdmin wraps every /admin route, reads and deletes alike, so only
// the admin can reach them.
func RegisterRoutes(
	mux *http.ServeMux,
	healthHandler http.Handler,
	appointmentHandler *handlers.AppointmentHandler,
	createAppointmentLimit func(http.Handler) http.Handler,
	requireAdmin func(http.Handler) http.Handler,
) {
	mux.Handle("GET /health", healthHandler)

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
		"GET /admin/appointments",
		requireAdmin(
			http.HandlerFunc(appointmentHandler.ListAppointments),
		),
	)

	mux.Handle(
		"DELETE /admin/appointments/{id}",
		requireAdmin(
			http.HandlerFunc(appointmentHandler.DeleteAppointment),
		),
	)
}
