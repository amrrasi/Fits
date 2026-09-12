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
