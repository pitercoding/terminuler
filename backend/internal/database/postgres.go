package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func Connect(databaseURL string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		conn.Close(context.Background())
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return conn, nil
}
