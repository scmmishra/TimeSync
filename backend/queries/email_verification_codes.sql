-- name: CreateEmailVerificationCode :one
INSERT INTO email_verification_codes (
    email,
    code,
    expires_at,
    created_at
)
VALUES ($1, $2, $3, now())
RETURNING id, email, code, expires_at, used_at, created_at;

-- name: GetEmailVerificationCode :one
SELECT id, email, code, expires_at, used_at, created_at
FROM email_verification_codes
WHERE email = $1
  AND code = $2
  AND expires_at > $3
  AND used_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkEmailVerificationCodeUsed :exec
UPDATE email_verification_codes
SET used_at = $2
WHERE id = $1;
