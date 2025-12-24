package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"timesync/backend/internal/config"
	"timesync/backend/internal/mailer"
	"timesync/backend/internal/store"
)

// stubHelper saves and restores function pointers for testing
type stubHelper struct {
	t *testing.T
	// saved originals
	origLoadConfig   func() (config.Config, error)
	origOpenStore    func(context.Context, string) (*store.Store, error)
	origNewSMTP      func(mailer.SMTPConfig) (*mailer.SMTPMailer, error)
	origListenServe  func(*http.Server) error
	origShutdownSrv  func(*http.Server, context.Context) error
}

func newStubHelper(t *testing.T) *stubHelper {
	return &stubHelper{
		t:               t,
		origLoadConfig:  loadConfig,
		origOpenStore:   openStore,
		origNewSMTP:     newSMTP,
		origListenServe: listenAndServe,
		origShutdownSrv: shutdownServer,
	}
}

func (sh *stubHelper) onLoadConfig(fn func() (config.Config, error)) *stubHelper {
	loadConfig = fn
	return sh
}

func (sh *stubHelper) onOpenStore(fn func(context.Context, string) (*store.Store, error)) *stubHelper {
	openStore = fn
	return sh
}

func (sh *stubHelper) onNewSMTP(fn func(mailer.SMTPConfig) (*mailer.SMTPMailer, error)) *stubHelper {
	newSMTP = fn
	return sh
}

func (sh *stubHelper) onListenAndServe(fn func(*http.Server) error) *stubHelper {
	listenAndServe = fn
	return sh
}

func (sh *stubHelper) onShutdownServer(fn func(*http.Server, context.Context) error) *stubHelper {
	shutdownServer = fn
	return sh
}

func (sh *stubHelper) restore() {
	loadConfig = sh.origLoadConfig
	openStore = sh.origOpenStore
	newSMTP = sh.origNewSMTP
	listenAndServe = sh.origListenServe
	shutdownServer = sh.origShutdownSrv
}

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
	stub := newStubHelper(t).onNewSMTP(func(mailer.SMTPConfig) (*mailer.SMTPMailer, error) {
		return nil, errors.New("boom")
	})
	t.Cleanup(stub.restore)

	_, err := newMailer(config.Config{
		SMTPHost: "smtp.example.com",
		SMTPPort: 587,
		SMTPFrom: "no-reply@example.com",
	})
	if err == nil {
		t.Fatal("expected error from newMailer")
	}
}

func TestRunLoadConfigError(t *testing.T) {
	stub := newStubHelper(t).onLoadConfig(func() (config.Config, error) {
		return config.Config{}, errors.New("load failed")
	})
	t.Cleanup(stub.restore)

	if err := run(context.Background(), slog.Default()); err == nil {
		t.Fatal("expected error from run")
	}
}

func TestRunOpenStoreError(t *testing.T) {
	stub := newStubHelper(t).
		onLoadConfig(func() (config.Config, error) {
			return config.Config{DatabaseURL: "postgres://example"}, nil
		}).
		onOpenStore(func(context.Context, string) (*store.Store, error) {
			return nil, errors.New("open failed")
		})
	t.Cleanup(stub.restore)

	if err := run(context.Background(), slog.Default()); err == nil {
		t.Fatal("expected error from run")
	}
}

func TestRunMailerError(t *testing.T) {
	stub := newStubHelper(t).
		onLoadConfig(func() (config.Config, error) {
			return config.Config{
				DatabaseURL: "postgres://example",
				SMTPHost:    "smtp.example.com",
				SMTPPort:    587,
				SMTPFrom:    "no-reply@example.com",
			}, nil
		}).
		onOpenStore(func(context.Context, string) (*store.Store, error) {
			return &store.Store{}, nil
		}).
		onNewSMTP(func(mailer.SMTPConfig) (*mailer.SMTPMailer, error) {
			return nil, errors.New("smtp failed")
		})
	t.Cleanup(stub.restore)

	if err := run(context.Background(), slog.Default()); err == nil {
		t.Fatal("expected error from run")
	}
}

func TestRunListenError(t *testing.T) {
	stub := newStubHelper(t).
		onLoadConfig(func() (config.Config, error) {
			return config.Config{DatabaseURL: "postgres://example"}, nil
		}).
		onOpenStore(func(context.Context, string) (*store.Store, error) {
			return &store.Store{}, nil
		}).
		onListenAndServe(func(*http.Server) error {
			return errors.New("listen failed")
		})
	t.Cleanup(stub.restore)

	if err := run(context.Background(), slog.Default()); err == nil {
		t.Fatal("expected error from run")
	}
}

func TestRunShutdownError(t *testing.T) {
	stop := make(chan struct{})
	stub := newStubHelper(t).
		onLoadConfig(func() (config.Config, error) {
			return config.Config{DatabaseURL: "postgres://example"}, nil
		}).
		onOpenStore(func(context.Context, string) (*store.Store, error) {
			return &store.Store{}, nil
		}).
		onListenAndServe(func(*http.Server) error {
			<-stop
			return http.ErrServerClosed
		}).
		onShutdownServer(func(*http.Server, context.Context) error {
			close(stop)
			return errors.New("shutdown failed")
		})
	t.Cleanup(stub.restore)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := run(ctx, slog.Default()); err == nil {
		t.Fatal("expected shutdown error from run")
	}
}

func TestBuildServer(t *testing.T) {
	srv := buildServer(":8080", http.NewServeMux())
	if srv.Addr != ":8080" {
		t.Fatalf("unexpected addr: %s", srv.Addr)
	}
	if srv.ReadTimeout != 10*time.Second || srv.WriteTimeout != 10*time.Second || srv.IdleTimeout != 60*time.Second {
		t.Fatal("unexpected timeouts")
	}
	if srv.Handler == nil {
		t.Fatal("expected handler to be set")
	}
}

func TestBuildSettingsUsesValues(t *testing.T) {
	cfg := config.Config{
		AccessTTLMinutes:       1,
		RefreshTTLHours:        2,
		CodeTTLMinutes:         3,
		RefreshGraceSeconds:    4,
		TeamSizeLimit:          5,
		RequestCodeEmailLimit:  6,
		RequestCodeEmailWindow: 7,
		RequestCodeIPLimit:     8,
		RequestCodeIPWindow:    9,
		VerifyCodeEmailLimit:   10,
		VerifyCodeEmailWindow:  11,
		VerifyCodeLockMinutes:  12,
		VerifyCodeIPLimit:      13,
		VerifyCodeIPWindow:     14,
		RefreshDeviceLimit:     15,
		RefreshDeviceWindow:    16,
	}
	settings := buildSettings(cfg)
	if settings.TeamSizeLimit != 5 || settings.RequestCodeIPLimit != 8 || settings.RefreshDeviceLimit != 15 {
		t.Fatal("settings not mapped correctly")
	}
	if settings.AccessTTL != time.Minute || settings.RefreshTTL != 2*time.Hour || settings.CodeTTL != 3*time.Minute {
		t.Fatal("settings durations not mapped correctly")
	}
}
