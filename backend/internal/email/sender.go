// Package email sends transactional emails to customers.
package email

// Sender sends appointment emails. Implementations must be safe to call
// after the change has been persisted: failures are reported to the caller
// but must not undo the booking or the cancellation.
//
// In every method appointmentDate is formatted as YYYY-MM-DD and startTime
// and endTime as HH:MM.
type Sender interface {
	// SendConfirmation notifies the customer that the appointment was
	// booked.
	SendConfirmation(
		to string,
		customerName string,
		appointmentDate string,
		startTime string,
		endTime string,
	) error

	// SendCancellation notifies the customer that the appointment was
	// cancelled and its time slot is available again.
	SendCancellation(
		to string,
		customerName string,
		appointmentDate string,
		startTime string,
		endTime string,
	) error
}
