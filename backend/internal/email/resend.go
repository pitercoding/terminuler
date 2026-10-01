package email

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/resend/resend-go/v4"
)

// sendTimeout bounds each request to Resend. The email is sent while the
// booking request is still open, so a slow provider must not hold the
// response for the client's default of one minute.
const sendTimeout = 10 * time.Second

// ResendSender sends emails through the Resend API.
type ResendSender struct {
	client *resend.Client
	from   string
}

// NewResendSender creates a sender authenticated with apiKey that uses
// from as the sender address.
func NewResendSender(apiKey string, from string) *ResendSender {
	httpClient := &http.Client{
		Timeout: sendTimeout,
	}

	return &ResendSender{
		client: resend.NewCustomClient(httpClient, apiKey),
		from:   from,
	}
}

// SendConfirmation sends the appointment confirmation email. The customer
// name is user input, so it is HTML-escaped before being placed in the body.
func (s *ResendSender) SendConfirmation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	safeCustomerName := html.EscapeString(customerName)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Appointment confirmation",
		Html: fmt.Sprintf(
			`
			<h1>Appointment confirmed</h1>
			<p>Hello %s,</p>
			<p>Your appointment has been successfully confirmed.</p>

			<p>
				<strong>Date:</strong> %s<br>
				<strong>Time:</strong> %s - %s
			</p>

			<p>Thank you!</p>
			`,
			safeCustomerName,
			appointmentDate,
			startTime,
			endTime,
		),
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send confirmation email: %w", err)
	}

	return nil
}

// SendCancellation sends the appointment cancellation email. Like
// SendConfirmation, it HTML-escapes the customer name.
func (s *ResendSender) SendCancellation(
	to string,
	customerName string,
	appointmentDate string,
	startTime string,
	endTime string,
) error {
	safeCustomerName := html.EscapeString(customerName)

	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: "Appointment cancelled",
		Html: fmt.Sprintf(
			`
			<h1>Your appointment has been cancelled</h1>
			<p>Hello %s,</p>
			<p>Your appointment has been cancelled.</p>

			<p>
				<strong>Date:</strong> %s<br>
				<strong>Time:</strong> %s - %s
			</p>

			<p>The time slot is now available for a new booking.</p>

			<p>If you would like to schedule another appointment, you can book one on our booking page.</p>
			`,
			safeCustomerName,
			appointmentDate,
			startTime,
			endTime,
		),
	}

	_, err := s.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send cancellation email: %w", err)
	}

	return nil
}
