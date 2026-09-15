# ── Stage 1: Build Go backend ────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /fits-processor ./cmd/fits-processor

# ── Stage 2: Build React frontend ────────────────────────────────────────────
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci --silent

COPY frontend/ .
RUN npm run build

# ── Stage 3: Final minimal image ─────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S fits && adduser -S fits -G fits

WORKDIR /app

# Copy binary
COPY --from=builder  /fits-processor          ./fits-processor

# Copy migrations
COPY --from=builder  /app/migrations          ./migrations

# Copy built frontend
COPY --from=frontend-builder /app/frontend/dist ./static

# Create log directory owned by app user
RUN mkdir -p /app/logs && chown -R fits:fits /app

USER fits

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./fits-processor"]
CMD ["--env", "/app/.env", "--serve-only"]
