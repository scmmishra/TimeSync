.PHONY: run migrate-up migrate-down migrate-force sqlc

run:
	go run ./cmd/server

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down

migrate-force:
	migrate -path migrations -database "$$DATABASE_URL" force $$VERSION

sqlc:
	sqlc generate
