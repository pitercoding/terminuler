package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
	// Embeds the timezone database so APP_TIMEZONE works on hosts
	// without one (Windows, minimal Docker images).
	_ "time/tzdata"

	"github.com/joho/godotenv"
)

const (
	defaultPort     = "8080"
	defaultTimezone = "UTC"

	defaultResendFromEmail = "onboarding@resend.dev"
)

// envFiles are the locations checked for a .env file, relative to the
// working directory: the current directory and the repository root when
// running from backend/.
var envFiles = []string{
	".env",
	"../.env",
}

// Load reads the first .env file found. The file is optional so the
// application can run with environment variables provided by the host
// (Docker, CI, production). Variables already set in the environment
// always take precedence over values from the file.
func Load() error {
	for _, path := range envFiles {
		err := godotenv.Load(path)

		if err == nil {
			return nil
		}

		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}
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

// ResendAPIKey returns the API key used to send confirmation emails
// (RESEND_API_KEY). It is required so a misconfigured deployment fails at
// startup instead of silently skipping every confirmation email.
func ResendAPIKey() (string, error) {
	apiKey := os.Getenv("RESEND_API_KEY")

	if apiKey == "" {
		return "", fmt.Errorf("RESEND_API_KEY is not set")
	}

	return apiKey, nil
}

// ResendFromEmail returns the sender address used for confirmation emails
// (RESEND_FROM_EMAIL), defaulting to Resend's testing address, which can
// only deliver to the email of the Resend account owner.
func ResendFromEmail() string {
	if from := os.Getenv("RESEND_FROM_EMAIL"); from != "" {
		return from
	}

	return defaultResendFromEmail
}

// Port returns the HTTP port (PORT), defaulting to 8080.
func Port() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}

	return defaultPort
}

// Location returns the timezone used for business hours (APP_TIMEZONE),
// defaulting to UTC when it is not set.
func Location() (*time.Location, error) {
	timezone := os.Getenv("APP_TIMEZONE")

	if timezone == "" {
		timezone = defaultTimezone
	}

	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE %q: %w", timezone, err)
	}

	return location, nil
}
