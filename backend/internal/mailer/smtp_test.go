package mailer

import (
	"context"
	"testing"
)

func TestNewSMTP(t *testing.T) {
	cfg := SMTPConfig{
		Host: "localhost",
		Port: 1025,
		User: "user",
		Pass: "pass",
		From: "no-reply@example.com",
	}

	m, err := NewSMTP(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil || m.client == nil {
		t.Fatal("expected smtp mailer to have client")
	}
	if m.from != cfg.From {
		t.Fatalf("expected from %q, got %q", cfg.From, m.from)
	}
}

func TestSMTPMailerSendVerificationCodeInvalidFrom(t *testing.T) {
	m := &SMTPMailer{
		from: "invalid address",
	}

	if err := m.SendVerificationCode(context.Background(), "user@example.com", "ABC12345"); err == nil {
		t.Fatal("expected error for invalid from address")
	}
}

func TestSMTPMailerSendVerificationCodeInvalidTo(t *testing.T) {
	m := &SMTPMailer{
		from: "no-reply@example.com",
	}

	if err := m.SendVerificationCode(context.Background(), "bad address", "ABC12345"); err == nil {
		t.Fatal("expected error for invalid recipient")
	}
}
