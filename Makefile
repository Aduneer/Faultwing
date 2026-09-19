.PHONY: run db-up db-down test

run:
	go run ./cmd/api

db-up:
	docker compose up -d db

db-down:
	docker compose down

test:
	go test ./...
