package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pitercoding/terminuler/internal/models"
)

var ErrAppointmentConflict = errors.New("appointment slot is already booked")

var ErrAppointmentNotFound = errors.New("appointment not found")

type AppointmentRepository struct {
	db *sql.DB
}

func NewAppointmentRepository(db *sql.DB) *AppointmentRepository {
	return &AppointmentRepository{
		db: db,
	}
}

func (r *AppointmentRepository) GetByDate(
	ctx context.Context,
	date string,
) ([]models.Appointment, error) {
	query := `
		SELECT
			id,
			appointment_date,
			start_time,
			end_time,
			customer_name,
			customer_phone,
			customer_email,
			created_at
		FROM appointments
		WHERE appointment_date = $1
		ORDER BY start_time
	`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments by date: %w", err)
	}

	return scanAppointments(rows)
}

// GetAppointments returns the appointments on or after fromDate
// (YYYY-MM-DD), in chronological order.
func (r *AppointmentRepository) GetAppointments(
	ctx context.Context,
	fromDate string,
) ([]models.Appointment, error) {
	query := `
		SELECT
			id,
			appointment_date,
			start_time,
			end_time,
			customer_name,
			customer_phone,
			customer_email,
			created_at
		FROM appointments
		WHERE appointment_date >= $1
		ORDER BY appointment_date, start_time
	`

	rows, err := r.db.QueryContext(ctx, query, fromDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get appointments: %w", err)
	}

	return scanAppointments(rows)
}

// scanAppointments reads every row selected by the appointment queries and
// closes rows.
func scanAppointments(rows *sql.Rows) ([]models.Appointment, error) {
	defer rows.Close()

	var appointments []models.Appointment

	for rows.Next() {
		var appointment models.Appointment

		if err := rows.Scan(
			&appointment.ID,
			&appointment.AppointmentDate,
			&appointment.StartTime,
			&appointment.EndTime,
			&appointment.CustomerName,
			&appointment.CustomerPhone,
			&appointment.CustomerEmail,
			&appointment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan appointment: %w", err)
		}

		appointments = append(appointments, appointment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate appointments: %w", err)
	}

	return appointments, nil
}

func (r *AppointmentRepository) Create(
	ctx context.Context,
	appointment *models.Appointment,
) error {
	query := `
		INSERT INTO appointments (
			appointment_date,
			start_time,
			end_time,
			customer_name,
			customer_phone,
			customer_email
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		appointment.AppointmentDate,
		appointment.StartTime,
		appointment.EndTime,
		appointment.CustomerName,
		appointment.CustomerPhone,
		appointment.CustomerEmail,
	).Scan(
		&appointment.ID,
		&appointment.CreatedAt,
	)

	if err != nil {
		var pgError *pgconn.PgError

		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return ErrAppointmentConflict
		}

		return fmt.Errorf("failed to create appointment: %w", err)
	}

	return nil
}

// Delete removes the appointment with the given ID, freeing its slot. It returns ErrAppointmentNotFound when no appointment has that ID.
func (r *AppointmentRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	query := `
		DELETE FROM appointments
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete appointment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check deleted appointment: %w", err)
	}

	if rowsAffected == 0 {
		return ErrAppointmentNotFound
	}

	return nil
}
