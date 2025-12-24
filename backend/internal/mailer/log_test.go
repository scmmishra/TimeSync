package mailer

import (
	"context"
	"testing"
)

func TestLogMailerSendVerificationCode(t *testing.T) {
	m := &LogMailer{}
	if err := m.SendVerificationCode(context.Background(), "user@example.com", "ABC12345"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
