package routes

import (
	"net/http"

	"github.com/pitercoding/terminuler/internal/handlers"
)

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", handlers.HealthHandler)
}
