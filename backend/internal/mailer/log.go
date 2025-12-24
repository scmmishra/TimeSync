package mailer

import (
	"context"
	"log/slog"
)

type LogMailer struct{}

func (m *LogMailer) SendVerificationCode(_ context.Context, email, code string) error {
	slog.Info("verification code issued", slog.String("email", email), slog.String("code", code))
	return nil
}
