// Command e2edb prepares the database used by the E2E tests: it creates the
// database in DATABASE_URL when it does not exist, applies the migrations
// and removes every appointment, so each E2E run starts from an empty table.
//
// Because it deletes data, it refuses to run unless the database name ends
// with "_test", which keeps it away from the development database.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/pitercoding/terminuler/internal/config"
	"github.com/pitercoding/terminuler/internal/database"
)

// maintenanceDatabase always exists in PostgreSQL and is used to create the test database.
const maintenanceDatabase = "postgres"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := config.Load(); err != nil {
		return err
	}

	databaseURL, err := config.DatabaseURL()
	if err != nil {
		return err
	}

	name, err := testDatabaseName(databaseURL)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if err := ensureDatabase(ctx, databaseURL, name); err != nil {
		return err
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	if _, err := db.ExecContext(
		ctx,
		"TRUNCATE appointments RESTART IDENTITY",
	); err != nil {
		return fmt.Errorf("failed to clean appointments table: %w", err)
	}

	log.Printf("E2E database %q is ready", name)

	return nil
}

// testDatabaseName returns the database name in databaseURL, which must end with "_test".
func testDatabaseName(databaseURL string) (string, error) {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	name := strings.TrimPrefix(parsedURL.Path, "/")

	if !strings.HasSuffix(name, "_test") {
		return "", fmt.Errorf(
			"refusing to prepare database %q: the E2E database name must end with _test",
			name,
		)
	}

	return name, nil
}

// ensureDatabase creates the database when it does not exist yet.
func ensureDatabase(ctx context.Context, databaseURL string, name string) error {
	maintenanceURL, err := withDatabase(databaseURL, maintenanceDatabase)
	if err != nil {
		return err
	}

	db, err := database.Connect(maintenanceURL)
	if err != nil {
		return err
	}
	defer db.Close()

	var exists bool

	if err := db.QueryRowContext(
		ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)",
		name,
	).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database %q: %w", name, err)
	}

	if exists {
		return nil
	}

	// CREATE DATABASE does not accept parameters, so the name is quoted as an identifier.
	if _, err := db.ExecContext(
		ctx,
		"CREATE DATABASE "+pgx.Identifier{name}.Sanitize(),
	); err != nil {
		return fmt.Errorf("failed to create database %q: %w", name, err)
	}

	log.Printf("Created database %q", name)

	return nil
}

// withDatabase returns databaseURL pointing to another database on the same server.
func withDatabase(databaseURL string, name string) (string, error) {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	parsedURL.Path = "/" + name

	return parsedURL.String(), nil
}
