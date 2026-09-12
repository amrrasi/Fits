.PHONY: build run test lint migrate-up migrate-down tidy clean

BINARY   := fits-processor
CMD_PATH := ./cmd/fits-processor
ENV_FILE := .env

## build: compile the binary to ./bin/fits-processor
build:
	@mkdir -p bin
	go build -o bin/$(BINARY) $(CMD_PATH)

## run: build and run with .env settings
run: build
	./bin/$(BINARY) --env $(ENV_FILE)

## run-dir: scan a specific directory (make run-dir DIR=/path/to/fits)
run-dir: build
	./bin/$(BINARY) --env $(ENV_FILE) --scan-dir $(DIR)

## test: run all tests
test:
	go test ./... -v -race -count=1

## lint: run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## migrate-up: run all pending migrations (requires DB_* env vars)
migrate-up:
	@source $(ENV_FILE) && \
	  migrate -database "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" \
	          -path migrations up

## migrate-down: rollback all migrations
migrate-down:
	@source $(ENV_FILE) && \
	  migrate -database "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" \
	          -path migrations down

## clean: remove build artifacts and log files
clean:
	rm -rf bin/ logs/*.log logs/*.error.log

## help: show this message
help:
	@grep -E '^## ' Makefile | sed 's/## //'
