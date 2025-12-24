package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.SMTPPort != 587 {
		t.Fatalf("expected default SMTP port 587, got %d", cfg.SMTPPort)
	}
	if cfg.SMTPFrom != "no-reply@timesync" {
		t.Fatalf("expected default SMTP from, got %q", cfg.SMTPFrom)
	}
	if cfg.TeamSizeLimit != 30 {
		t.Fatalf("expected default team size limit 30, got %d", cfg.TeamSizeLimit)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "9090")
	t.Setenv("SMTP_HOST", "smtp.example")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("ACCESS_TTL_MINUTES", "15")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Port)
	}
	if cfg.SMTPHost != "smtp.example" {
		t.Fatalf("expected SMTP host to be set")
	}
	if cfg.SMTPPort != 2525 {
		t.Fatalf("expected SMTP port 2525, got %d", cfg.SMTPPort)
	}
	if cfg.AccessTTLMinutes != 15 {
		t.Fatalf("expected access ttl minutes 15, got %d", cfg.AccessTTLMinutes)
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail without DATABASE_URL")
	}
}

func TestLoadRejectsInvalidNumbers(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "not-a-number")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to fail with invalid PORT")
	}
}
