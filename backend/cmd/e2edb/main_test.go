package main

import "testing"

func TestTestDatabaseName(t *testing.T) {
	name, err := testDatabaseName(
		"postgres://user:secret@localhost:5433/terminuler_test?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if name != "terminuler_test" {
		t.Fatalf("expected terminuler_test, got %q", name)
	}

	for _, databaseURL := range []string{
		"postgres://user:secret@localhost:5433/terminuler",
		"postgres://user:secret@localhost:5433/terminuler_testing",
		"postgres://user:secret@localhost:5433/",
	} {
		if _, err := testDatabaseName(databaseURL); err == nil {
			t.Errorf("expected %q to be refused", databaseURL)
		}
	}
}

func TestWithDatabase(t *testing.T) {
	maintenanceURL, err := withDatabase(
		"postgres://user:secret@localhost:5433/terminuler_test?sslmode=disable",
		"postgres",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "postgres://user:secret@localhost:5433/postgres?sslmode=disable"

	if maintenanceURL != expected {
		t.Fatalf("expected %q, got %q", expected, maintenanceURL)
	}
}
