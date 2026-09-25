package main

import (
	"context"
	"log"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
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

	if err := database.Migrate(context.Background(), db); err != nil {
		log.Fatal(err)
	}

	log.Println("Database migrations applied successfully")
}
