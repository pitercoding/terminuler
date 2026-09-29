package email

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestLogSender_LogsConfirmation(t *testing.T) {
	var output bytes.Buffer

	sender := NewLogSender(log.New(&output, "", 0))

	err := sender.SendConfirmation(
		"jane@example.com",
		"Jane Doe",
		"2026-10-05",
		"09:00",
		"10:00",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	logged := output.String()

	for _, expected := range []string{
		`to="jane@example.com"`,
		`name="Jane Doe"`,
		"date=2026-10-05",
		"time=09:00-10:00",
	} {
		if !strings.Contains(logged, expected) {
			t.Errorf("expected log to contain %q, got %q", expected, logged)
		}
	}
}

func TestLogSender_ImplementsSender(t *testing.T) {
	var _ Sender = NewLogSender(nil)
}
