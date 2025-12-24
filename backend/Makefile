.PHONY: run migrate-up migrate-down migrate-force sqlc test test-short fmt lint cover

GOCACHE ?= $(PWD)/.gocache
GOMODCACHE ?= $(PWD)/.gomodcache
GOENV = GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

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

test:
	$(GOENV) go test ./...

test-short:
	$(GOENV) go test -short ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

cover:
	$(GOENV) go test ./... -cover
