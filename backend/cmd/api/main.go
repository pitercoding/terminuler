package main

import (
	"log"
	"net/http"

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

	mux := http.NewServeMux()

	appointmentRepository := repositories.NewAppointmentRepository(db)

	appointmentService := services.NewAppointmentService(
		appointmentRepository,
	)

	appointmentHandler := handlers.NewAppointmentHandler(
		appointmentService,
	)

	routes.RegisterRoutes(
		mux,
		appointmentHandler,
	)

	log.Println("Terminuler API running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
