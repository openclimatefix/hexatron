.PHONY: run build test clean tidy

## run: start the development server
# Nothing in the Go app parses .env — config reads the process environment via
# os.Getenv, and .env is otherwise only consumed by docker-compose's env_file.
# Sourcing it here means a refreshed AIRFLOW_SESSION_COOKIE takes effect on the
# next `make run` instead of silently going nowhere.
run:
	set -a; . ./.env; set +a; go run ./cmd/server/main.go

## build: compile the binary
build:
	go build -o bin/hexatron ./cmd/server/main.go

## test: run all tests
test:
	go test ./...

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf bin/
