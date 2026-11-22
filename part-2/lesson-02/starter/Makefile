# Shortcuts. Each target is one or two plain commands you can also type yourself.
DATABASE_URL      ?= postgres://app:app@127.0.0.1:5432/taskdb?sslmode=disable
TEST_DATABASE_URL ?= postgres://app:app@127.0.0.1:5432/taskdb_test?sslmode=disable

.PHONY: run test db-up db-down test-integration

run:
	DATABASE_URL='$(DATABASE_URL)' go run ./cmd/api

# Unit tests only: the Postgres tests skip themselves.
test:
	go vet ./...
	go test -race ./...

# Start Postgres and wait until it is healthy; create the test database once.
db-up:
	docker compose up -d --wait
	docker compose exec -T db createdb -U app taskdb_test 2>/dev/null || true

db-down:
	docker compose down

# Everything, against a real database.
test-integration: db-up
	TEST_DATABASE_URL='$(TEST_DATABASE_URL)' go test -race -count=1 ./...
