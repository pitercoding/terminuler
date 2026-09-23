package main

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
	dbmigrations "github.com/pitercoding/terminuler/internal/database/migrations"
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

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close()

	source, err := iofs.New(dbmigrations.FS, ".")
	if err != nil {
		log.Fatal(err)
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		source,
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal(err)
	}

	log.Println("Database migrations applied successfully")
}
