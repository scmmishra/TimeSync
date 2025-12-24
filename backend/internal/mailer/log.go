package mailer

import (
	"context"
	"log"
)

type LogMailer struct{}

func (m *LogMailer) SendVerificationCode(_ context.Context, email, code string) error {
	log.Printf("verification code for %s: %s", email, code)
	return nil
}
