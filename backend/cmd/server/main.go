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

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", slog.Any("err", err))
		os.Exit(1)
	}
	defer st.Close()

	var mailerSvc mailer.Mailer
	mailerSvc, err = newMailer(cfg)
	if err != nil {
		slog.Error("failed to init smtp mailer", slog.Any("err", err))
		os.Exit(1)
	}

	settings := buildSettings(cfg)

	api := httpapi.New(st, mailerSvc, settings, logger)

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      api.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("backend listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", slog.Any("err", err))
	}
}

func newMailer(cfg config.Config) (mailer.Mailer, error) {
	if cfg.SMTPHost == "" {
		return &mailer.LogMailer{}, nil
	}
	return mailer.NewSMTP(mailer.SMTPConfig{
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
