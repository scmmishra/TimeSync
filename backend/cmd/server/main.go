package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"timesync/backend/internal/config"
	"timesync/backend/internal/httpapi"
	"timesync/backend/internal/mailer"
	"timesync/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil {
		slog.Error("server error", slog.Any("err", err))
		os.Exit(1)
	}
}

var (
	loadConfig     = config.Load
	openStore      = store.Open
	newServer      = buildServer
	listenAndServe = func(srv *http.Server) error { return srv.ListenAndServe() }
	shutdownServer = func(srv *http.Server, ctx context.Context) error { return srv.Shutdown(ctx) }
)

func run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	st, err := openStore(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer st.Close()

	mailerSvc, err := newMailer(cfg)
	if err != nil {
		return err
	}

	settings := buildSettings(cfg)

	api := httpapi.New(st, mailerSvc, settings, logger)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := newServer(addr, api.Handler())

	errCh := make(chan error, 1)
	go func() {
		logger.Info("backend listening", slog.String("addr", addr))
		if err := listenAndServe(srv); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := shutdownServer(srv, shutdownCtx); err != nil {
		return err
	}
	return nil
}

var newSMTP = mailer.NewSMTP

func newMailer(cfg config.Config) (mailer.Mailer, error) {
	if cfg.SMTPHost == "" {
		return &mailer.LogMailer{}, nil
	}
	return newSMTP(mailer.SMTPConfig{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		User: cfg.SMTPUser,
		Pass: cfg.SMTPPass,
		From: cfg.SMTPFrom,
	})
}

func buildSettings(cfg config.Config) httpapi.Settings {
	return httpapi.Settings{
		AccessTTL:              time.Duration(cfg.AccessTTLMinutes) * time.Minute,
		RefreshTTL:             time.Duration(cfg.RefreshTTLHours) * time.Hour,
		CodeTTL:                time.Duration(cfg.CodeTTLMinutes) * time.Minute,
		RefreshGrace:           time.Duration(cfg.RefreshGraceSeconds) * time.Second,
		TeamSizeLimit:          cfg.TeamSizeLimit,
		RequestCodeEmailLimit:  cfg.RequestCodeEmailLimit,
		RequestCodeEmailWindow: time.Duration(cfg.RequestCodeEmailWindow) * time.Minute,
		RequestCodeIPLimit:     cfg.RequestCodeIPLimit,
		RequestCodeIPWindow:    time.Duration(cfg.RequestCodeIPWindow) * time.Minute,
		VerifyCodeEmailLimit:   cfg.VerifyCodeEmailLimit,
		VerifyCodeEmailWindow:  time.Duration(cfg.VerifyCodeEmailWindow) * time.Minute,
		VerifyCodeLock:         time.Duration(cfg.VerifyCodeLockMinutes) * time.Minute,
		VerifyCodeIPLimit:      cfg.VerifyCodeIPLimit,
		VerifyCodeIPWindow:     time.Duration(cfg.VerifyCodeIPWindow) * time.Minute,
		RefreshDeviceLimit:     cfg.RefreshDeviceLimit,
		RefreshDeviceWindow:    time.Duration(cfg.RefreshDeviceWindow) * time.Minute,
	}
}

func buildServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
