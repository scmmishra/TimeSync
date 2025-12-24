package main

import (
	"errors"
	"testing"
	"time"

	"timesync/backend/internal/config"
	"timesync/backend/internal/mailer"
)

func TestBuildSettings(t *testing.T) {
	cfg := config.Config{
		AccessTTLMinutes:       5,
		RefreshTTLHours:        2,
		CodeTTLMinutes:         3,
		RefreshGraceSeconds:    9,
		TeamSizeLimit:          12,
		RequestCodeEmailLimit:  1,
		RequestCodeEmailWindow: 7,
		RequestCodeIPLimit:     2,
		RequestCodeIPWindow:    8,
		VerifyCodeEmailLimit:   3,
		VerifyCodeEmailWindow:  11,
		VerifyCodeLockMinutes:  13,
		VerifyCodeIPLimit:      4,
		VerifyCodeIPWindow:     14,
		RefreshDeviceLimit:     5,
		RefreshDeviceWindow:    6,
	}

	settings := buildSettings(cfg)
	if settings.AccessTTL != 5*time.Minute {
		t.Fatalf("unexpected access ttl: %v", settings.AccessTTL)
	}
	if settings.RefreshTTL != 2*time.Hour {
		t.Fatalf("unexpected refresh ttl: %v", settings.RefreshTTL)
	}
	if settings.CodeTTL != 3*time.Minute {
		t.Fatalf("unexpected code ttl: %v", settings.CodeTTL)
	}
	if settings.RefreshGrace != 9*time.Second {
		t.Fatalf("unexpected refresh grace: %v", settings.RefreshGrace)
	}
	if settings.TeamSizeLimit != 12 {
		t.Fatalf("unexpected team size limit: %d", settings.TeamSizeLimit)
	}
	if settings.RequestCodeIPWindow != 8*time.Minute {
		t.Fatalf("unexpected request code ip window: %v", settings.RequestCodeIPWindow)
	}
	if settings.VerifyCodeLock != 13*time.Minute {
		t.Fatalf("unexpected verify code lock: %v", settings.VerifyCodeLock)
	}
	if settings.RefreshDeviceWindow != 6*time.Minute {
		t.Fatalf("unexpected refresh device window: %v", settings.RefreshDeviceWindow)
	}
}

func TestNewMailerUsesLogMailer(t *testing.T) {
	m, err := newMailer(config.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := m.(*mailer.LogMailer); !ok {
		t.Fatalf("expected LogMailer, got %T", m)
	}
}

func TestNewMailerUsesSMTP(t *testing.T) {
	m, err := newMailer(config.Config{
		SMTPHost: "localhost",
		SMTPPort: 1025,
		SMTPFrom: "no-reply@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := m.(*mailer.SMTPMailer); !ok {
		t.Fatalf("expected SMTPMailer, got %T", m)
	}
}

func TestNewMailerSMTPError(t *testing.T) {
	orig := newSMTP
	newSMTP = func(mailer.SMTPConfig) (*mailer.SMTPMailer, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { newSMTP = orig })

	_, err := newMailer(config.Config{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		SMTPFrom: "no-reply@example.com",
	})
	if err == nil {
		t.Fatal("expected error from newMailer")
	}
}
