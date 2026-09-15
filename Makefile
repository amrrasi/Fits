.PHONY: build run serve run-dir test lint tidy clean \
        docker-build docker-up docker-down docker-logs \
        frontend-install frontend-dev frontend-build migrate-up migrate-down help

BINARY    := fits-processor
CMD_PATH  := ./cmd/fits-processor
ENV_FILE  := .env
DOCKER_COMPOSE := docker-compose

# ── Development ───────────────────────────────────────────────────────────────

## build: compile the binary to ./bin/fits-processor
build:
	@mkdir -p bin
	go build -o bin/$(BINARY) $(CMD_PATH)

## run: build and run (scans FITS files + serves API)
run: build
	./bin/$(BINARY) --env $(ENV_FILE)

## serve: build and run API server only (no scan)
serve: build
	./bin/$(BINARY) --env $(ENV_FILE) --serve-only

## run-dir: scan a specific directory (make run-dir DIR=/path/to/fits)
run-dir: build
	./bin/$(BINARY) --env $(ENV_FILE) --scan-dir $(DIR)

## frontend-install: install frontend npm dependencies
frontend-install:
	cd frontend && npm install

## frontend-dev: start the Vite dev server on :3000
frontend-dev:
	cd frontend && npm run dev

## frontend-build: build the React app to frontend/dist/
frontend-build:
	cd frontend && npm run build

## dev: start backend (serve-only) and frontend dev server in parallel
dev:
	@echo "Starting backend on :8080 and frontend on :3000..."
	@$(MAKE) serve &
	@$(MAKE) frontend-dev

# ── Testing & Quality ─────────────────────────────────────────────────────────

## test: run all Go tests with race detector
test:
	go test ./... -v -race -count=1

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

# ── Migrations ────────────────────────────────────────────────────────────────

## migrate-up: run all pending migrations
migrate-up:
	@export $$(grep -v '^#' $(ENV_FILE) | xargs) && \
	  migrate -database "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" \
	          -path migrations up

## migrate-down: rollback the last migration
migrate-down:
	@export $$(grep -v '^#' $(ENV_FILE) | xargs) && \
	  migrate -database "postgres://$$DB_USER:$$DB_PASSWORD@$$DB_HOST:$$DB_PORT/$$DB_NAME?sslmode=$$DB_SSLMODE" \
	          -path migrations down 1

# ── Docker ────────────────────────────────────────────────────────────────────

## docker-build: build the production Docker image
docker-build: frontend-build
	docker build -t fits-processor:latest .

## docker-up: start full stack with docker-compose
docker-up:
	$(DOCKER_COMPOSE) up -d

## docker-down: stop and remove containers
docker-down:
	$(DOCKER_COMPOSE) down

## docker-logs: tail logs from all containers
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-dev: start development stack (postgres only, app runs locally)
docker-dev:
	$(DOCKER_COMPOSE) up -d db

# ── Production ────────────────────────────────────────────────────────────────

## prod-build: build production binary (Linux amd64)
prod-build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	  go build -ldflags="-w -s" -o bin/$(BINARY)-linux-amd64 $(CMD_PATH)

## clean: remove build artifacts
clean:
	rm -rf bin/ frontend/dist/ logs/*.log logs/*.error.log

## help: show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'
