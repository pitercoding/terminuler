package main

import (
	"context"
	"log"
	"net/http"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
	"github.com/pitercoding/terminuler/internal/routes"
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
	defer db.Close(context.Background())

	mux := http.NewServeMux()

	routes.RegisterRoutes(mux)

	log.Println("Terminuler API running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
