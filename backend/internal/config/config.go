package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func Load() error {
	if err := godotenv.Load("../.env"); err != nil {
		return fmt.Errorf("failed to load .env: %w", err)
	}

	return nil
}

func DatabaseURL() (string, error) {
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		return "", fmt.Errorf("DATABASE_URL is not set")
	}

	return databaseURL, nil
}
