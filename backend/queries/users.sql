-- name: GetUserByID :one
SELECT id, email, email_domain, email_verified_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, email_domain, email_verified_at, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (
    email,
    email_domain,
    email_verified_at,
    created_at,
    updated_at
)
VALUES ($1, $2, $3, now(), now())
RETURNING id, email, email_domain, email_verified_at, created_at, updated_at;
