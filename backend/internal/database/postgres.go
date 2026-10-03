package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	// maxOpenConns caps the connections of one process. database/sql has no
	// limit by default, so a burst of requests could exhaust the connection
	// limit of a small managed PostgreSQL plan, failing every other client.
	maxOpenConns = 10

	maxIdleConns = 5

	// connMaxLifetime and connMaxIdleTime recycle connections, so ones
	// silently dropped by the database host or a network hop are replaced.
	connMaxLifetime = 30 * time.Minute
	connMaxIdleTime = 5 * time.Minute

	// pingTimeout bounds the first connection, so an unreachable database
	// fails the startup instead of hanging it.
	pingTimeout = 10 * time.Second
)

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
