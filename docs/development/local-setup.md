## Local setup

### Requirements

- Go 1.21+
- Postgres (local or hosted)
- `sqlc`
- `golang-migrate`

### Backend setup

1) Create `backend/.env` from the template:
```env
DATABASE_URL=postgresql://USER:PASSWORD@HOST:PORT/DATABASE?sslmode=require
PORT=8080
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=no-reply@timesync
```

2) Install tools:
```bash
brew install sqlc golang-migrate
```

3) Generate sqlc code:
```bash
cd backend
make sqlc
```

4) Run migrations:
```bash
make migrate-up
```

5) Run the backend:
```bash
make run
```
