package main

import (
	"log"
	"net/http"
	"time"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
	"github.com/pitercoding/terminuler/internal/handlers"
	"github.com/pitercoding/terminuler/internal/repositories"
	"github.com/pitercoding/terminuler/internal/routes"
	"github.com/pitercoding/terminuler/internal/services"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatal(err)
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	location, err := config.Location()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()

	appointmentRepository := repositories.NewAppointmentRepository(db)

	appointmentService := services.NewAppointmentService(
		appointmentRepository,
		func() time.Time {
			return time.Now().In(location)
		},
	)

	appointmentHandler := handlers.NewAppointmentHandler(
		appointmentService,
	)

	routes.RegisterRoutes(
		mux,
		appointmentHandler,
	)

	port := config.Port()

	log.Printf(
		"Terminuler API running on http://localhost:%s (timezone: %s)",
		port,
		location,
	)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
