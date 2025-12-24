package mailer

import "context"

type Mailer interface {
	SendVerificationCode(ctx context.Context, email, code string) error
}
