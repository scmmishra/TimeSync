package mailer

import (
	"context"
	"fmt"

	"github.com/wneessen/go-mail"
)

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type SMTPMailer struct {
	client *mail.Client
	from   string
}

func NewSMTP(cfg SMTPConfig) (*SMTPMailer, error) {
	client, err := mail.NewClient(
		cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithUsername(cfg.User),
		mail.WithPassword(cfg.Pass),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
	)
	if err != nil {
		return nil, err
	}

	return &SMTPMailer{
		client: client,
		from:   cfg.From,
	}, nil
}

func (m *SMTPMailer) SendVerificationCode(ctx context.Context, email, code string) error {
	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return err
	}
	if err := msg.To(email); err != nil {
		return err
	}
	msg.Subject("Your TimeSync verification code")
	body := fmt.Sprintf("Your TimeSync code is %s. It expires in 10 minutes.", code)
	msg.SetBodyString(mail.TypeTextPlain, body)
	return m.client.DialAndSendWithContext(ctx, msg)
}
