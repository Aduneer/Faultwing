.PHONY: run db-up db-down test test-integration test-python

run:
	go run ./cmd/api

db-up:
	docker compose up -d db

db-down:
	docker compose down

test:
	go test ./...

test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-postgres://flytrap:flytrap@localhost:5432/flytrap?sslmode=disable}" \
		go test ./internal/database -run '^TestMonitoringFlow$$' -v

test-python:
	PYTHONPATH=sdk/python python3 -m unittest discover -s sdk/python/tests -v
