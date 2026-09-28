package config

import (
	"os"
	"path/filepath"
	"testing"
)

// unsetEnv removes a variable for the duration of the test, restoring
// the original value afterwards.
func unsetEnv(t *testing.T, key string) {
	t.Helper()

	original, existed := os.LookupEnv(key)

	os.Unsetenv(key)

	t.Cleanup(func() {
		if existed {
			os.Setenv(key, original)
		} else {
			os.Unsetenv(key)
		}
	})
}

func writeEnvFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}
}

func TestLoad_WithoutEnvFile(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := Load(); err != nil {
		t.Fatalf("expected missing .env to be ignored, got %v", err)
	}
}

func TestLoad_EnvFileInCurrentDirectory(t *testing.T) {
	unsetEnv(t, "TERMINULER_TEST_VALUE")

	directory := t.TempDir()
	writeEnvFile(
		t,
		filepath.Join(directory, ".env"),
		"TERMINULER_TEST_VALUE=current\n",
	)

	t.Chdir(directory)

	if err := Load(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value := os.Getenv("TERMINULER_TEST_VALUE"); value != "current" {
		t.Fatalf("expected value 'current', got %q", value)
	}
}

func TestLoad_EnvFileInParentDirectory(t *testing.T) {
	unsetEnv(t, "TERMINULER_TEST_VALUE")

	root := t.TempDir()
	backend := filepath.Join(root, "backend")

	if err := os.Mkdir(backend, 0o755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	writeEnvFile(
		t,
		filepath.Join(root, ".env"),
		"TERMINULER_TEST_VALUE=parent\n",
	)

	t.Chdir(backend)

	if err := Load(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value := os.Getenv("TERMINULER_TEST_VALUE"); value != "parent" {
		t.Fatalf("expected value 'parent', got %q", value)
	}
}

func TestLoad_EnvironmentTakesPrecedence(t *testing.T) {
	t.Setenv("TERMINULER_TEST_VALUE", "from-environment")

	directory := t.TempDir()
	writeEnvFile(
		t,
		filepath.Join(directory, ".env"),
		"TERMINULER_TEST_VALUE=from-file\n",
	)

	t.Chdir(directory)

	if err := Load(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value := os.Getenv("TERMINULER_TEST_VALUE"); value != "from-environment" {
		t.Fatalf("expected value 'from-environment', got %q", value)
	}
}

func TestDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := DatabaseURL(); err == nil {
		t.Fatal("expected error when DATABASE_URL is empty")
	}

	t.Setenv("DATABASE_URL", "postgres://localhost/test")

	databaseURL, err := DatabaseURL()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if databaseURL != "postgres://localhost/test" {
		t.Fatalf("unexpected DATABASE_URL %q", databaseURL)
	}
}

func TestPort(t *testing.T) {
	t.Setenv("PORT", "")

	if port := Port(); port != "8080" {
		t.Fatalf("expected default port 8080, got %s", port)
	}

	t.Setenv("PORT", "3001")

	if port := Port(); port != "3001" {
		t.Fatalf("expected port 3001, got %s", port)
	}
}

func TestLocation(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "")

	location, err := Location()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if location.String() != "UTC" {
		t.Fatalf("expected default location UTC, got %s", location)
	}

	t.Setenv("APP_TIMEZONE", "Europe/Berlin")

	location, err = Location()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if location.String() != "Europe/Berlin" {
		t.Fatalf("expected Europe/Berlin, got %s", location)
	}

	t.Setenv("APP_TIMEZONE", "Invalid/Zone")

	if _, err := Location(); err == nil {
		t.Fatal("expected error for invalid timezone")
	}
}

func TestResendAPIKey(t *testing.T) {
	t.Setenv("RESEND_API_KEY", "")

	if _, err := ResendAPIKey(); err == nil {
		t.Fatal("expected error when RESEND_API_KEY is empty")
	}

	t.Setenv("RESEND_API_KEY", "re_test_key")

	apiKey, err := ResendAPIKey()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if apiKey != "re_test_key" {
		t.Fatalf("unexpected RESEND_API_KEY %q", apiKey)
	}
}

func TestResendFromEmail(t *testing.T) {
	t.Setenv("RESEND_FROM_EMAIL", "")

	if from := ResendFromEmail(); from != defaultResendFromEmail {
		t.Fatalf("expected default sender %q, got %q", defaultResendFromEmail, from)
	}

	t.Setenv("RESEND_FROM_EMAIL", "Terminuler <booking@example.com>")

	if from := ResendFromEmail(); from != "Terminuler <booking@example.com>" {
		t.Fatalf("unexpected RESEND_FROM_EMAIL %q", from)
	}
}

func TestAppointmentRateLimit(t *testing.T) {
	t.Setenv("APPOINTMENT_RATE_LIMIT", "")

	limit, err := AppointmentRateLimit()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if limit != defaultAppointmentRateLimit {
		t.Fatalf("expected default limit %d, got %d", defaultAppointmentRateLimit, limit)
	}

	t.Setenv("APPOINTMENT_RATE_LIMIT", "10")

	limit, err = AppointmentRateLimit()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if limit != 10 {
		t.Fatalf("expected limit 10, got %d", limit)
	}

	for _, invalid := range []string{"0", "-1", "abc", "1.5"} {
		t.Setenv("APPOINTMENT_RATE_LIMIT", invalid)

		if _, err := AppointmentRateLimit(); err == nil {
			t.Fatalf("expected error for APPOINTMENT_RATE_LIMIT %q", invalid)
		}
	}
}

func TestTrustedProxies(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")

	proxies, err := TrustedProxies()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(proxies) != 0 {
		t.Fatalf("expected no trusted proxies by default, got %v", proxies)
	}

	t.Setenv("TRUSTED_PROXIES", "127.0.0.1/32, ::1/128,10.1.2.3/8,, 192.168.0.10")

	proxies, err = TrustedProxies()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"127.0.0.1/32", "::1/128", "10.0.0.0/8", "192.168.0.10/32"}

	if len(proxies) != len(expected) {
		t.Fatalf("expected %d proxies, got %v", len(expected), proxies)
	}

	for i, prefix := range proxies {
		if prefix.String() != expected[i] {
			t.Errorf("expected proxy %d to be %s, got %s", i, expected[i], prefix)
		}
	}

	for _, invalid := range []string{"localhost", "10.0.0.0/33", "300.0.0.1"} {
		t.Setenv("TRUSTED_PROXIES", invalid)

		if _, err := TrustedProxies(); err == nil {
			t.Fatalf("expected error for TRUSTED_PROXIES %q", invalid)
		}
	}
}
