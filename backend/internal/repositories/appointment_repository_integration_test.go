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

	"github.com/jackc/pgx/v5/pgconn"

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

	if appointment.Status != models.AppointmentStatusConfirmed {
		t.Errorf("expected status %q, got %q", models.AppointmentStatusConfirmed, appointment.Status)
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

	if stored.Status != models.AppointmentStatusConfirmed {
		t.Errorf("expected status %q, got %q", models.AppointmentStatusConfirmed, stored.Status)
	}

	if stored.CancelledAt != nil {
		t.Errorf("expected no CancelledAt, got %v", stored.CancelledAt)
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

func TestGetByDate_LeavesOutCancelledAppointments(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	cancelled := newAppointment("2026-09-28", "10:00", "11:00")

	for _, appointment := range []*models.Appointment{
		newAppointment("2026-09-28", "09:00", "10:00"),
		cancelled,
		newAppointment("2026-09-28", "11:00", "12:00"),
	} {
		if err := repository.Create(ctx, appointment); err != nil {
			t.Fatalf("failed to create appointment: %v", err)
		}
	}

	if _, err := repository.Cancel(ctx, cancelled.ID); err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

	appointments, err := repository.GetByDate(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{"09:00:00", "11:00:00"}

	if len(appointments) != len(expected) {
		t.Fatalf(
			"expected %d appointments, got %d",
			len(expected),
			len(appointments),
		)
	}

	for i, appointment := range appointments {
		if appointment.StartTime != expected[i] ||
			appointment.Status != models.AppointmentStatusConfirmed {
			t.Errorf(
				"expected confirmed appointment %d at %s, got %q at %s",
				i,
				expected[i],
				appointment.Status,
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

func TestGetAppointments_FiltersFromDateAndOrdersChronologically(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	for _, appointment := range []*models.Appointment{
		newAppointment("2026-09-29", "09:00", "10:00"),
		newAppointment("2026-09-25", "08:00", "09:00"),
		newAppointment("2026-09-28", "14:00", "15:00"),
		newAppointment("2026-09-28", "08:00", "09:00"),
		newAppointment("2026-09-24", "10:00", "11:00"),
	} {
		if err := repository.Create(ctx, appointment); err != nil {
			t.Fatalf("failed to create appointment: %v", err)
		}
	}

	appointments, err := repository.GetAppointments(ctx, "2026-09-25")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []string{
		"2026-09-25 08:00:00",
		"2026-09-28 08:00:00",
		"2026-09-28 14:00:00",
		"2026-09-29 09:00:00",
	}

	if len(appointments) != len(expected) {
		t.Fatalf(
			"expected %d appointments, got %d",
			len(expected),
			len(appointments),
		)
	}

	for i, appointment := range appointments {
		got := appointment.AppointmentDate.Format("2006-01-02") +
			" " + appointment.StartTime

		if got != expected[i] {
			t.Errorf("expected appointment %d at %s, got %s", i, expected[i], got)
		}
	}
}

func TestGetAppointments_NoAppointments(t *testing.T) {
	repository := newTestRepository(t)

	appointments, err := repository.GetAppointments(
		context.Background(),
		"2026-09-25",
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

func TestCancel_ReturnsCancelledAppointment(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	appointment := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, appointment); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	before := time.Now().Add(-time.Minute)

	cancelled, err := repository.Cancel(ctx, appointment.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cancelled.ID != appointment.ID ||
		cancelled.AppointmentDate.Format("2006-01-02") != "2026-09-28" ||
		cancelled.CustomerName != appointment.CustomerName ||
		cancelled.CustomerPhone != appointment.CustomerPhone ||
		cancelled.CustomerEmail != appointment.CustomerEmail ||
		!cancelled.CreatedAt.Equal(appointment.CreatedAt) {
		t.Fatalf("expected the cancelled appointment %+v, got %+v", appointment, cancelled)
	}

	// PostgreSQL returns TIME values with seconds.
	if cancelled.StartTime != "10:00:00" || cancelled.EndTime != "11:00:00" {
		t.Errorf(
			"expected times 10:00:00-11:00:00, got %s-%s",
			cancelled.StartTime,
			cancelled.EndTime,
		)
	}

	if cancelled.Status != models.AppointmentStatusCancelled {
		t.Errorf("expected status %q, got %q", models.AppointmentStatusCancelled, cancelled.Status)
	}

	if cancelled.CancelledAt == nil || cancelled.CancelledAt.Before(before) {
		t.Errorf("expected CancelledAt to be recent, got %v", cancelled.CancelledAt)
	}
}

func TestCancel_KeepsAppointmentStored(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	appointment := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, appointment); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	cancelled, err := repository.Cancel(ctx, appointment.ID)
	if err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

	appointments, err := repository.GetAppointments(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(appointments) != 1 || appointments[0].ID != appointment.ID {
		t.Fatalf("expected appointment %d to remain stored, got %+v", appointment.ID, appointments)
	}

	stored := appointments[0]

	if stored.Status != models.AppointmentStatusCancelled {
		t.Errorf("expected status %q, got %q", models.AppointmentStatusCancelled, stored.Status)
	}

	if stored.CancelledAt == nil || !stored.CancelledAt.Equal(*cancelled.CancelledAt) {
		t.Errorf("expected CancelledAt %v, got %v", cancelled.CancelledAt, stored.CancelledAt)
	}
}

func TestCancel_TwiceReturnsNotFound(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	appointment := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, appointment); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	first, err := repository.Cancel(ctx, appointment.ID)
	if err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

	second, err := repository.Cancel(ctx, appointment.ID)

	if !errors.Is(err, ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound, got %v", err)
	}

	if second != nil {
		t.Fatalf("expected no appointment, got %+v", second)
	}

	// The failed attempt must not move the original cancellation time.
	appointments, err := repository.GetAppointments(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(appointments) != 1 ||
		appointments[0].CancelledAt == nil ||
		!appointments[0].CancelledAt.Equal(*first.CancelledAt) {
		t.Fatalf("expected CancelledAt %v to be kept, got %+v", first.CancelledAt, appointments)
	}
}

func TestCancel_NonexistentReturnsNotFound(t *testing.T) {
	repository := newTestRepository(t)

	cancelled, err := repository.Cancel(context.Background(), 999)

	if !errors.Is(err, ErrAppointmentNotFound) {
		t.Fatalf("expected ErrAppointmentNotFound, got %v", err)
	}

	if cancelled != nil {
		t.Fatalf("expected no appointment, got %+v", cancelled)
	}
}

func TestCancel_AffectsOnlyThatAppointment(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	before := newAppointment("2026-09-28", "09:00", "10:00")
	cancelled := newAppointment("2026-09-28", "10:00", "11:00")
	after := newAppointment("2026-09-28", "11:00", "12:00")

	for _, appointment := range []*models.Appointment{before, cancelled, after} {
		if err := repository.Create(ctx, appointment); err != nil {
			t.Fatalf("failed to create appointment: %v", err)
		}
	}

	if _, err := repository.Cancel(ctx, cancelled.ID); err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

	appointments, err := repository.GetAppointments(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(appointments) != 3 {
		t.Fatalf("expected 3 appointments, got %d", len(appointments))
	}

	for _, appointment := range appointments {
		isCancelled := appointment.ID == cancelled.ID

		expected := models.AppointmentStatusConfirmed

		if isCancelled {
			expected = models.AppointmentStatusCancelled
		}

		if appointment.Status != expected {
			t.Errorf(
				"expected appointment %d to be %q, got %q",
				appointment.ID,
				expected,
				appointment.Status,
			)
		}

		if (appointment.CancelledAt != nil) != isCancelled {
			t.Errorf(
				"expected CancelledAt to be set only on appointment %d, got %v on %d",
				cancelled.ID,
				appointment.CancelledAt,
				appointment.ID,
			)
		}
	}
}

func TestCancel_FreesSlotForRebooking(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	cancelled := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, cancelled); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	if _, err := repository.Cancel(ctx, cancelled.ID); err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

	rebooked := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, rebooked); err != nil {
		t.Fatalf("expected the cancelled slot to be bookable, got %v", err)
	}

	if rebooked.ID == cancelled.ID ||
		rebooked.Status != models.AppointmentStatusConfirmed {
		t.Fatalf("expected a new confirmed appointment, got %+v", rebooked)
	}

	// The new booking is confirmed, so it holds the slot again.
	err := repository.Create(
		ctx,
		newAppointment("2026-09-28", "10:00", "11:00"),
	)

	if !errors.Is(err, ErrAppointmentConflict) {
		t.Fatalf("expected ErrAppointmentConflict, got %v", err)
	}
}

func TestCancel_ConcurrentCancellationsOnlyOneSucceeds(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()

	appointment := newAppointment("2026-09-28", "10:00", "11:00")

	if err := repository.Create(ctx, appointment); err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	const attempts = 10

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
		notFound  int
		others    []error
	)

	for range attempts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_, err := repository.Cancel(ctx, appointment.ID)

			mu.Lock()
			defer mu.Unlock()

			switch {
			case err == nil:
				succeeded++
			case errors.Is(err, ErrAppointmentNotFound):
				notFound++
			default:
				others = append(others, err)
			}
		}()
	}

	wg.Wait()

	if len(others) > 0 {
		t.Fatalf("unexpected errors: %v", others)
	}

	if succeeded != 1 || notFound != attempts-1 {
		t.Fatalf(
			"expected 1 success and %d not found, got %d and %d",
			attempts-1,
			succeeded,
			notFound,
		)
	}
}

func TestAppointmentsTable_RejectsInconsistentStatus(t *testing.T) {
	tests := []struct {
		name   string
		update string
	}{
		{
			name:   "unknown status",
			update: "UPDATE appointments SET status = 'pending' WHERE id = $1",
		},
		{
			name:   "cancelled without cancelled_at",
			update: "UPDATE appointments SET status = 'cancelled' WHERE id = $1",
		},
		{
			name:   "confirmed with cancelled_at",
			update: "UPDATE appointments SET cancelled_at = NOW() WHERE id = $1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := newTestRepository(t)

			appointment := newAppointment("2026-09-28", "10:00", "11:00")

			if err := repository.Create(context.Background(), appointment); err != nil {
				t.Fatalf("failed to create appointment: %v", err)
			}

			_, err := testDB.Exec(tt.update, appointment.ID)

			var pgError *pgconn.PgError

			// 23514 is check_violation.
			if !errors.As(err, &pgError) || pgError.Code != "23514" {
				t.Fatalf("expected a check violation, got %v", err)
			}
		})
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

// countingEmailSender counts the emails sent through it. It is safe for
// concurrent use.
type countingEmailSender struct {
	mu            sync.Mutex
	confirmations int
	cancellations int
}

func (s *countingEmailSender) SendConfirmation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.confirmations++

	return nil
}

func (s *countingEmailSender) SendCancellation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancellations++

	return nil
}

func (s *countingEmailSender) cancellationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.cancellations
}

// newFlowService returns a service backed by the real repository, with a
// clock fixed before 2026-09-28 so every slot of that day is bookable.
func newFlowService(
	repository *AppointmentRepository,
	emailSender *countingEmailSender,
) *services.AppointmentService {
	return services.NewAppointmentServiceWithEmail(
		repository,
		emailSender,
		func() time.Time {
			return time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
		},
	)
}

func flowInput(startTime string, endTime string) services.CreateAppointmentInput {
	return services.CreateAppointmentInput{
		AppointmentDate: "2026-09-28",
		StartTime:       startTime,
		EndTime:         endTime,
		CustomerName:    "Racha Cuca",
		CustomerPhone:   "+5511999999999",
		CustomerEmail:   "rc@exemple.com",
	}
}

// availableStartTimes returns the start times listed as available on
// 2026-09-28.
func availableStartTimes(
	t *testing.T,
	service *services.AppointmentService,
) map[string]bool {
	t.Helper()

	slots, err := service.GetAvailableSlots(context.Background(), "2026-09-28")
	if err != nil {
		t.Fatalf("failed to get available slots: %v", err)
	}

	available := make(map[string]bool, len(slots))

	for _, slot := range slots {
		available[slot.StartTime] = true
	}

	return available
}

// TestCancelAppointmentFlow cancels one of three appointments through the
// service and checks that it stays stored as cancelled, that the customer
// gets exactly one email, and that only its slot becomes available again.
func TestCancelAppointmentFlow(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	emailSender := &countingEmailSender{}
	service := newFlowService(repository, emailSender)

	var cancelled *models.Appointment

	for _, times := range [][2]string{
		{"09:00", "10:00"},
		{"10:00", "11:00"},
		{"11:00", "12:00"},
	} {
		appointment, err := service.Create(ctx, flowInput(times[0], times[1]))
		if err != nil {
			t.Fatalf("failed to create appointment at %s: %v", times[0], err)
		}

		if times[0] == "10:00" {
			cancelled = appointment
		}
	}

	if err := service.CancelAppointment(ctx, cancelled.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if count := emailSender.cancellationCount(); count != 1 {
		t.Fatalf("expected 1 cancellation email, got %d", count)
	}

	appointments, err := repository.GetAppointments(ctx, "2026-09-28")
	if err != nil {
		t.Fatalf("failed to list appointments: %v", err)
	}

	if len(appointments) != 3 {
		t.Fatalf("expected all 3 appointments to stay stored, got %d", len(appointments))
	}

	for _, appointment := range appointments {
		if appointment.ID == cancelled.ID &&
			(appointment.Status != models.AppointmentStatusCancelled ||
				appointment.CancelledAt == nil) {
			t.Errorf("expected appointment %d to be cancelled, got %+v", cancelled.ID, appointment)
		}
	}

	// Cancelling again finds no confirmed appointment and sends no email.
	if err := service.CancelAppointment(ctx, cancelled.ID); !errors.Is(
		err,
		ErrAppointmentNotFound,
	) {
		t.Fatalf("expected ErrAppointmentNotFound on second cancellation, got %v", err)
	}

	if count := emailSender.cancellationCount(); count != 1 {
		t.Fatalf("expected still 1 cancellation email, got %d", count)
	}

	available := availableStartTimes(t, service)

	if available["09:00"] || available["11:00"] {
		t.Errorf("expected 09:00 and 11:00 to stay booked, got %v", available)
	}

	if !available["10:00"] {
		t.Errorf("expected the cancelled 10:00 slot to be available, got %v", available)
	}

	if len(available) != 6 {
		t.Errorf("expected 6 available slots, got %d", len(available))
	}
}

// TestCancelAppointmentFlow_RebookingKeepsDoubleBookingProtection books a
// cancelled slot again with concurrent requests: exactly one succeeds, and
// the slot is booked again.
func TestCancelAppointmentFlow_RebookingKeepsDoubleBookingProtection(t *testing.T) {
	repository := newTestRepository(t)
	ctx := context.Background()
	service := newFlowService(repository, &countingEmailSender{})

	appointment, err := service.Create(ctx, flowInput("10:00", "11:00"))
	if err != nil {
		t.Fatalf("failed to create appointment: %v", err)
	}

	if err := service.CancelAppointment(ctx, appointment.ID); err != nil {
		t.Fatalf("failed to cancel appointment: %v", err)
	}

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

			_, err := service.Create(ctx, flowInput("10:00", "11:00"))

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

	if availableStartTimes(t, service)["10:00"] {
		t.Error("expected the rebooked 10:00 slot to be unavailable")
	}
}
