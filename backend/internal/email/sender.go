// Package email sends transactional emails to customers.
package email

// Sender sends appointment confirmation emails. Implementations must be
// safe to call after the appointment has been persisted: failures are
// reported to the caller but must not undo the booking.
type Sender interface {
	// SendConfirmation notifies the customer that the appointment was
	// booked. appointmentDate is formatted as YYYY-MM-DD and startTime and
	// endTime as HH:MM.
	SendConfirmation(
		to string,
		customerName string,
		appointmentDate string,
		startTime string,
		endTime string,
	) error
}
