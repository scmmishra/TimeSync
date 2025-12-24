## Development setup

- Local: [Local setup](local-setup.md)

## Stack overview

Backend (Go):

- `chi` for HTTP routing and middleware
- `pgx` + `pgxpool` for Postgres connectivity
- `sqlc` for typed query generation
- `golang-migrate` for schema migrations
- `httprate` for rate limiting on auth endpoints
- `caarlos0/env` + `godotenv` for configuration loading
- `go-mail` for SMTP email delivery (swap provider via SMTP)

## Auth flow (v1)

1) `POST /auth/request-code` with email
2) `POST /auth/verify-code` with email + code + `X-Device-Id`
3) Use `Authorization: Bearer <access_token>` for API calls
4) Refresh via `POST /auth/refresh` with `X-Device-Id` and `refresh_token`
5) Logout via `POST /auth/logout`

## API endpoints

- `GET /health`
- `POST /auth/request-code`
- `POST /auth/verify-code`
- `POST /auth/refresh`
- `POST /auth/logout`

## Troubleshooting

- `sqlc: command not found`: `brew install sqlc`
- `migrate: command not found`: `brew install golang-migrate`
- `DATABASE_URL is required`: ensure `backend/.env` exists or set the env var
