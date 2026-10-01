package email

import "log"

// LogSender writes emails to a logger instead of sending them.
// It is meant for local testing, such as the E2E tests, where no email must
// leave the machine and no Resend API key is needed.
type LogSender struct {
	logger *log.Logger
}

// NewLogSender creates a sender that writes to logger, or to the standard
// logger when logger is nil.
func NewLogSender(logger *log.Logger) *LogSender {
	if logger == nil {
		logger = log.Default()
	}

	return &LogSender{
		logger: logger,
	}
}

// SendConfirmation logs the confirmation email and never fails.
func (s *LogSender) SendConfirmation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	s.logger.Printf(
		"confirmation email not sent (EMAIL_PROVIDER=log): to=%q name=%q date=%s time=%s-%s",
		to,
		customerName,
		appointmentDate,
		startTime,
		endTime,
	)

	return nil
}

// SendCancellation logs the cancellation email and never fails.
func (s *LogSender) SendCancellation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	s.logger.Printf(
		"cancellation email not sent (EMAIL_PROVIDER=log): to=%q name=%q date=%s time=%s-%s",
		to,
		customerName,
		appointmentDate,
		startTime,
		endTime,
	)

	return nil
}
