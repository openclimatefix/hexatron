# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/hexatron ./cmd/server/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/hexatron .
COPY --from=builder /app/data ./data

EXPOSE 8080

CMD ["./hexatron"]
