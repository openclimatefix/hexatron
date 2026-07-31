.PHONY: run build test clean tidy

## run: start the development server
run:
	go run ./cmd/server/main.go

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
