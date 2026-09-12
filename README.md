# FITS Processor

A robust Go service that recursively scans directories for FITS (Flexible Image Transport System) files, parses every HDU header, and persists all data to PostgreSQL.

---

## Features

- **Full header extraction** — every keyword from every HDU, stored in an EAV table (`fits_headers`)
- **Typed fast-query table** — well-known keywords (RA, DEC, EXPTIME, WCS, temperatures …) stored as typed columns in `fits_metadata`
- **Duplicate detection** — SHA-256 checksum prevents re-processing unchanged files
- **Concurrent processing** — configurable worker pool
- **Structured logging** — JSON logs to file + human-readable to stdout, separate error log file, daily rotation
- **Database migrations** — versioned SQL files via `golang-migrate`
- **12-factor config** — all settings via environment variables or `.env` file

---

## Project Structure

```
fits/
├── cmd/
│   └── fits-processor/
│       └── main.go              # Entrypoint
├── internal/
│   ├── config/      config.go   # All env-var config loading
│   ├── database/    database.go # PgxPool + migration runner
│   ├── fits/
│   │   ├── parser.go            # FITS file → ParseResult (headers + metadata)
│   │   ├── scanner.go           # Recursive directory scanner
│   │   └── processor.go         # Orchestrator: scan → parse → persist
│   ├── logger/      logger.go   # Zap structured logger (file + console + errors)
│   ├── models/      models.go   # Domain structs
│   └── repository/  fits_repository.go  # All DB writes
├── migrations/                  # Versioned SQL up/down files
├── testdata/                    # Drop .fits files here for testing
├── logs/                        # Created at runtime
├── .env.example
├── Makefile
└── go.mod
```

---

## Quick Start

### 1. Prerequisites

- Go 1.22+
- PostgreSQL 14+

### 2. Configure

```bash
cp .env.example .env
# Edit .env — set DB_PASSWORD and FITS_SCAN_DIR at minimum
```

### 3. Install dependencies

```bash
go mod tidy
```

### 4. Create the database

```sql
CREATE DATABASE fits_db;
```

### 5. Run

```bash
make run
# or scan a specific directory:
make run-dir DIR=/path/to/fits/files
# or directly:
go run ./cmd/fits-processor --env .env --scan-dir /data/fits
```

Migrations are applied automatically on every startup.

---

## Configuration Reference

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | _(required in prod)_ | Database password |
| `DB_NAME` | `fits_db` | Database name |
| `DB_SSLMODE` | `disable` | `disable` / `require` / `verify-full` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_DIR` | `logs` | Directory for log files |
| `FITS_SCAN_DIR` | `./testdata` | Root directory to scan |
| `FITS_WORKERS` | `4` | Parallel processing goroutines |
| `MIGRATIONS_DIR` | `migrations` | Path to SQL migration files |

---

## Database Schema

| Table | Purpose |
|---|---|
| `fits_files` | One row per file — path, checksum, status |
| `fits_headers` | EAV — every raw keyword from every HDU |
| `fits_metadata` | Typed flat table for fast range queries |
| `processing_jobs` | One row per scan run with progress counters |
| `processing_errors` | Per-file failure records |

---

## Adding New Keywords

1. Add column to a new migration SQL file
2. Add field to `internal/models/models.go` → `FITSMetadata`
3. Add `case "KEYWORD":` in `internal/fits/parser.go` → `populateMetadata()`
4. Add column to INSERT/UPDATE in `internal/repository/fits_repository.go` → `UpsertMetadata()`

---

## Logs

| File | Contents |
|---|---|
| `logs/fits-processor-YYYY-MM-DD.log` | All levels — JSON |
| `logs/fits-processor-YYYY-MM-DD.error.log` | Errors only — JSON |

Console output is human-readable. Set `LOG_LEVEL=debug` for per-keyword detail.

---

## Phase 2 — Authentication & User System

### Default admin account

| Field | Value |
|---|---|
| Email | `admin@fits.local` |
| Password | `Admin@1234` |
| Role | `admin` |

**Change the password immediately after first login.**

### Auth endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/api/auth/login` | Public | Returns access + refresh token |
| `POST` | `/api/auth/logout` | Public | Revokes refresh token |
| `POST` | `/api/auth/refresh` | Public | Issues new token pair (rotates refresh token) |
| `GET` | `/health` | Public | Health check |
| `GET` | `/api/me` | Bearer token | Returns current user info |

### Login example

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@fits.local","password":"Admin@1234"}' | jq
```

Response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "550e8400-...",
  "expires_at": "2026-01-01T00:15:00Z",
  "user": { "id": 1, "email": "admin@fits.local", "role": "admin" }
}
```

### Using the access token

```bash
curl -H "Authorization: Bearer eyJ..." http://localhost:8080/api/me
```

### Token lifecycle

- **Access token** — short-lived (15 min), stateless JWT, validated on every request
- **Refresh token** — long-lived (7 days), stored as SHA-256 hash in `sessions` table
- **Refresh rotation** — every refresh call deletes the old session and issues a new pair

### JWT environment variables

| Variable | Default | Description |
|---|---|---|
| `JWT_ACCESS_SECRET` | _(insecure default)_ | HMAC signing key for access tokens |
| `JWT_REFRESH_SECRET` | _(insecure default)_ | HMAC signing key for refresh tokens |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `SERVER_ADDR` | `:8080` | HTTP listen address |

### Running the server

```bash
# Start HTTP server + run a FITS scan on startup
make run

# Start HTTP server only (no scan)
make serve

# Start with a specific scan directory
make run-dir DIR=/data/fits
```

---

## Phase 3 — User & Role Management API

### Roles

| Role | Can view data | Can edit FITS metadata | Can manage users |
|---|---|---|---|
| `viewer` | ✅ | ❌ | ❌ |
| `editor` | ✅ | ✅ | ❌ |
| `admin`  | ✅ | ✅ | ✅ |

### User endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/users/me` | Any role | Own profile |
| `PUT` | `/api/users/me/password` | Any role | Change own password |
| `GET` | `/api/users` | Admin | List users (paginated + search) |
| `POST` | `/api/users` | Admin | Create user |
| `GET` | `/api/users/{id}` | Admin | Get user by ID |
| `PUT` | `/api/users/{id}` | Admin | Edit name, role, active status |
| `DELETE` | `/api/users/{id}` | Admin | Delete user |
| `PUT` | `/api/users/{id}/password` | Admin | Reset any user's password |

### Query params for `GET /api/users`

| Param | Example | Description |
|---|---|---|
| `page` | `1` | Page number |
| `page_size` | `20` | Items per page (max 100) |
| `search` | `reza` | Partial match on email or name |
| `role` | `editor` | Filter by role |
| `active` | `true` | Filter active/inactive |

### Examples

```bash
# List all users
curl -H "Authorization: Bearer TOKEN" http://localhost:8080/api/users

# Search users
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:8080/api/users?search=reza&role=editor&page=1&page_size=10"

# Create a new editor
curl -X POST http://localhost:8080/api/users \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"email":"reza@fits.local","password":"Secret@123","full_name":"Reza","role":"editor"}'

# Edit user role
curl -X PUT http://localhost:8080/api/users/2 \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"full_name":"Reza Updated","role":"viewer","is_active":true}'

# Change own password
curl -X PUT http://localhost:8080/api/users/me/password \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"old_password":"Admin@1234","new_password":"NewPass@456"}'

# Admin reset another user password
curl -X PUT http://localhost:8080/api/users/2/password \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"new_password":"Reset@789"}'

# Delete user
curl -X DELETE http://localhost:8080/api/users/2 \
  -H "Authorization: Bearer TOKEN"
```

### Safety rules
- You cannot delete your own account
- You cannot delete or demote the last active admin
- Passwords must be at least 8 characters
- Emails must be unique across the system
