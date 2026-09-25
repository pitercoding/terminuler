//go:build integration

// Integration tests run against a real PostgreSQL database:
//
//	TEST_DATABASE_URL=postgres://... go test -tags=integration ./...
//
// Each run creates an isolated schema, applies the migrations to it and
// drops it at the end, so existing data is never touched.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pitercoding/terminuler/internal/database"
	"github.com/pitercoding/terminuler/internal/models"
	"github.com/pitercoding/terminuler/internal/services"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	os.Exit(runIntegrationTests(m))
}

func runIntegrationTests(m *testing.M) int {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		log.Println("TEST_DATABASE_URL is not set")
		return 1
	}

	adminDB, err := database.Connect(databaseURL)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer adminDB.Close()

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())

	if _, err := adminDB.Exec("CREATE SCHEMA " + schema); err != nil {
		log.Printf("failed to create schema: %v", err)
		return 1
	}
	defer func() {
		if _, err := adminDB.Exec(
			"DROP SCHEMA " + schema + " CASCADE",
		); err != nil {
			log.Printf("failed to drop schema %s: %v", schema, err)
		}
	}()

	schemaURL, err := withSearchPath(databaseURL, schema)
	if err != nil {
		log.Println(err)
		return 1
	}

	testDB, err = database.Connect(schemaURL)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer testDB.Close()

	if err := database.Migrate(context.Background(), testDB); err != nil {
		log.Println(err)
		return 1
	}

	return m.Run()
}

// withSearchPath makes every connection of the pool use the given schema.
func withSearchPath(databaseURL string, schema string) (string, error) {
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("invalid TEST_DATABASE_URL: %w", err)
	}

	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String(), nil
}

// newTestRepository returns a repository backed by an empty table.
func newTestRepository(t *testing.T) *AppointmentRepository {
	t.Helper()

	if _, err := testDB.Exec(
		"TRUNCATE appointments RESTART IDENTITY",
	); err != nil {
		t.Fatalf("failed to clean appointments table: %v", err)
	}

	return NewAppointmentRepository(testDB)
}

func newAppointment(date string, startTime string, endTime string) *models.Appointment {
	appointmentDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		panic(err)
	}

	return &models.Appointment{
		AppointmentDate: appointmentDate,
		StartTime:       startTime,
		EndTime:         endTime,
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}
}

func TestMigrate_IsIdempotent(t *testing.T) {
	if err := database.Migrate(context.Background(), testDB); err != nil {
		t.Fatalf("expected second migration run to succeed, got %v", err)
	}

	// The migrator must not close the shared connection pool.
	if err := testDB.Ping(); err != nil {
		t.Fatalf("expected database to remain open, got %v", err)
	}
}

func TestCreate_SetsGeneratedFields(t *testing.T) {
	repository := newTestRepository(t)

	appointment := newAppointment("2026-09-28", "10:00", "11:00")

	before := time.Now().Add(-time.Minute)

	if err := repository.Create(context.Background(), appointment); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if appointment.ID == 0 {
		t.Error("expected ID to be set")
	}

	if appointment.CreatedAt.Before(before) {
		t.Errorf("expected CreatedAt to be recent, got %v", appointment.CreatedAt)
	}
}

func TestGetByDate_ReturnsStoredAppointment(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	created := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, created); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	appointments, err := repository.GetByDate(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(appointments) != 1 {
		t.Fatalf("expected 1 appointment, got %d", len(appointments))
	}

	stored := appointments[0]

	if stored.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, stored.ID)
	}

	if stored.AppointmentDate.Format("2006-01-02") != "2026-09-28" {
		t.Errorf("expected date 2026-09-28, got %v", stored.AppointmentDate)
	}

	// PostgreSQL returns TIME values with seconds.
	if stored.StartTime != "10:00:00" {
		t.Errorf("expected start time 10:00:00, got %q", stored.StartTime)
	}

	if stored.EndTime != "11:00:00" {
		t.Errorf("expected end time 11:00:00, got %q", stored.EndTime)
	}

	if stored.CustomerName != created.CustomerName ||
		stored.CustomerPhone != created.CustomerPhone ||
		stored.CustomerEmail != created.CustomerEmail {
		t.Errorf("expected customer data %+v, got %+v", created, stored)
	}

	if !stored.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf(
			"expected CreatedAt %v, got %v",
			created.CreatedAt,
			stored.CreatedAt,
		)
	}
}

func TestGetByDate_FiltersByDateAndOrdersByStartTime(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	for _, appointment := range []*models.Appointment{
		newAppointment("2026-09-28", "14:00", "15:00"),
		newAppointment("2026-09-29", "09:00", "10:00"),
		newAppointment("2026-09-28", "08:00", "09:00"),
		newAppointment("2026-09-28", "11:00", "12:00"),
	} {
		if err := repository.Create(ctx, appointment); err != nil {
			t.Fatalf("failed to create appointment: %v", err)
		}
	}

	appointments, err := repository.GetByDate(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"08:00:00", "11:00:00", "14:00:00"}

	if len(appointments) != len(expected) {
		t.Fatalf(
			"expected %d appointments, got %d",
			len(expected),
			len(appointments),
		)
	}

	for i, appointment := range appointments {
		if appointment.StartTime != expected[i] {
			t.Errorf(
				"expected appointment %d at %s, got %s",
				i,
				expected[i],
				appointment.StartTime,
			)
		}
	}
}

func TestGetByDate_NoAppointments(t *testing.T) {
	repository := newTestRepository(t)

	appointments, err := repository.GetByDate(
		context.Background(),
		"2026-09-28",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(appointments) != 0 {
		t.Fatalf("expected 0 appointments, got %d", len(appointments))
	}
}

func TestCreate_DuplicateSlotReturnsConflict(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	if err := repository.Create(
		ctx,
		newAppointment("2026-09-28", "10:00", "11:00"),
	); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	err := repository.Create(
		ctx,
		newAppointment("2026-09-28", "10:00", "11:00"),
	)

	if !errors.Is(err, ErrAppointmentConflict) {
		t.Fatalf("expected ErrAppointmentConflict, got %v", err)
	}
}

func TestCreate_SameTimeOnDifferentDates(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	for _, date := range []string{"2026-09-28", "2026-09-29"} {
		if err := repository.Create(
			ctx,
			newAppointment(date, "10:00", "11:00"),
		); err != nil {
			t.Fatalf("expected no error for %s, got %v", date, err)
		}
	}
}

func TestCreate_ConcurrentBookingsOnlyOneSucceeds(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	const attempts = 10

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		conflicts int
		others    []error
	)

	for range attempts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			err := repository.Create(
				ctx,
				newAppointment("2026-09-28", "10:00", "11:00"),
			)

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, ErrAppointmentConflict):
				conflicts++
			default:
				others = append(others, err)
			}
		}()
	}

	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("unexpected errors: %v", others)
	}

	if succeeded != 1 || conflicts != attempts-1 {
		t.Fatalf(
			"expected 1 success and %d conflicts, got %d and %d",
			attempts-1,
			succeeded,
			conflicts,
		)
	}
}

// TestAppointmentFlow exercises the service with the real repository to
// make sure booked slots read back from PostgreSQL are hidden from the
// availability list.
func TestAppointmentFlow(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	service := services.NewAppointmentService(
		repository,
		func() time.Time {
			return time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
		},
	)

	input := services.CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       "10:00",
		EndTime:         "11:00",
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}

	if _, err := service.Create(ctx, input); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	slots, err := service.GetAvailableSlots(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(slots) != 7 {
		t.Fatalf("expected 7 available slots, got %d", len(slots))
	}

	for _, slot := range slots {
		if slot.StartTime == "10:00" {
			t.Fatal("expected 10:00 slot to be unavailable")
		}
	}

	if _, err := service.Create(ctx, input); !errors.Is(
		err,
		ErrAppointmentConflict,
	) {
		t.Fatalf("expected ErrAppointmentConflict on rebooking, got %v", err)
	}
}
