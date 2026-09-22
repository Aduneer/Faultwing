.PHONY: run run-worker db-up db-down test test-integration test-python generate-errors frontend-install frontend-dev frontend-build frontend-test

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

db-up:
	docker compose up -d db

db-down:
	docker compose down

test:
	go test ./...

test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-postgres://flytrap:flytrap@localhost:5432/flytrap?sslmode=disable}" \
		go test ./internal/database -run '^Test(Monitoring|UserSession)Flow$$' -v

test-python:
	PYTHONPATH=sdk/python python3 -m unittest discover -s sdk/python/tests -v

generate-errors:
	PYTHONPATH=sdk/python python3 scripts/generate-errors.py $(ARGS)

frontend-install:
	npm --prefix web install

frontend-dev:
	npm --prefix web run dev

frontend-build:
	npm --prefix web run build

frontend-test:
	npm --prefix web run test:e2e
