# FITS Processor

A full-stack system for scanning, storing, and querying astronomical FITS file headers.
Built with Go (backend API) and React (frontend).

---

## What it does

- Recursively scans a directory for `.fits` / `.fit` / `.fts` files
- Parses every HDU header keyword and stores them in PostgreSQL
- Extracts ~40 well-known typed fields (RA, DEC, EXPTIME, FILTER, temperatures, WCS…) into a fast-query metadata table
- Provides a REST API with JWT authentication and role-based access (admin / editor / viewer)
- Provides a React frontend with search, filtering, inline metadata editing, job monitoring, and user management

---

## Project structure

```
Fits/
├── cmd/fits-processor/main.go     — application entrypoint
├── internal/
│   ├── api/          helpers.go   — shared HTTP response writers, pagination
│   ├── auth/                      — JWT tokens, bcrypt, middleware, login/logout/refresh
│   ├── config/       config.go    — all env-var loading + production validation
│   ├── database/     database.go  — pgxpool connection + golang-migrate runner
│   ├── fits/                      — scanner, parser (astrogo/fitsio), processor (worker pool)
│   ├── fitshandler/  handler.go   — HTTP handlers for FITS data endpoints
│   ├── fitsservice/  service.go   — FITS business logic (files, metadata, jobs, stats)
│   ├── logger/       logger.go    — zap logger (file + error file + console)
│   ├── middleware/   middleware.go — RequestID, Logger, SecurityHeaders, CORS, RateLimiter, MaxBodySize
│   ├── models/                    — domain structs (FITSFile, FITSHeader, FITSMetadata, User, Job…)
│   ├── repository/                — split DB access: file, header, metadata, job, user repositories
│   ├── userhandler/  handler.go   — HTTP handlers for user management + audit log
│   └── userservice/  service.go   — user management business logic
├── migrations/                    — 006 versioned SQL migration files (up + down)
├── frontend/                      — React 18 + TypeScript + Vite + Tailwind
│   └── src/
│       ├── api/       — typed Axios client with auto-refresh interceptor
│       ├── components/— layout (sidebar, topbar, protected route), UI primitives
│       ├── context/   — AuthContext, ToastContext
│       ├── hooks/     — useDebounce
│       ├── pages/     — Dashboard, Files, FileDetail, Jobs, JobDetail, Profile, Users, AuditLog
│       └── types/     — TypeScript types mirroring Go models
├── Dockerfile                     — 3-stage build (Go + Node + alpine)
├── docker-compose.yml             — full stack (app + postgres)
├── docker-compose.dev.yml         — dev override
├── Makefile
└── .env.example
```

---

## Running locally (Windows / Linux / macOS)

### Prerequisites

Install these first:

| Tool | Download | Verify |
|---|---|---|
| Go 1.22+ | https://go.dev/dl/ | `go version` |
| Node.js 20+ | https://nodejs.org | `node --version` |
| PostgreSQL 14+ | https://postgresql.org/download | `psql --version` |
| Git | https://git-scm.com | `git --version` |

### Step 1 — Clone

```bash
git clone https://github.com/amrrasi/Fits.git
cd Fits
```

### Step 2 — Create the database

```bash
# Linux / macOS
sudo -u postgres psql -c "CREATE DATABASE fits_db;"
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';"

# Windows — open pgAdmin or run in Command Prompt:
psql -U postgres -c "CREATE DATABASE fits_db;"
```

### Step 3 — Configure

```bash
cp .env.example .env
```

Open `.env` and set at minimum:

```env
DB_PASSWORD=postgres          # your PostgreSQL password
DB_NAME=fits_db
FITS_SCAN_DIR=./testdata      # directory with .fits files
JWT_ACCESS_SECRET=any-string-at-least-32-chars-long!!
```

### Step 4 — Download Go dependencies

```bash
go mod tidy
```

### Step 5 — Install frontend dependencies

```bash
cd frontend
npm install
cd ..
```

### Step 6 — Start the backend

Open **Terminal 1**:

```bash
go run ./cmd/fits-processor --env .env --serve-only
```

Expected output:
```
database: connected to PostgreSQL
database: migrations applied  version=6
HTTP server listening  addr=:8080
```

### Step 7 — Start the frontend

Open **Terminal 2**:

```bash
cd frontend
npm run dev
```

Expected output:
```
VITE ready
Local: http://localhost:3000
```

### Step 8 — Open the app

Go to **http://localhost:3000** and log in with:

| Field | Value |
|---|---|
| Email | `ADMIN_EMAIL` (default `admin@fits.local`) |
| Password | `ADMIN_PASSWORD`, or the random password printed **once** in the console on first start |

**Change this password immediately after first login.**

### Step 9 — Scan your FITS files

Copy `.fits` files into the `testdata/` folder, then open **Terminal 3**:

```bash
go run ./cmd/fits-processor --env .env
```

This scans, parses headers, and inserts everything into the database.
You can also trigger a scan from the UI: Dashboard → شروع اسکن (admin only).

---

## Make commands

```bash
make serve           # backend API only, no scan
make run             # backend + scan on startup
make run-dir DIR=/path/to/fits    # scan a specific directory
make frontend-dev    # Vite dev server (port 3000)
make test            # go test -race ./...
make tidy            # go mod tidy + verify
make docker-up       # full stack via docker-compose
make docker-dev      # postgres only (app runs locally)
make prod-build      # cross-compile Linux amd64 binary
make clean           # remove bin/, logs/, frontend/dist/
```

---

## Docker deployment

```bash
cp .env.example .env
# Set DB_PASSWORD, JWT_ACCESS_SECRET, CORS_ALLOWED_ORIGINS

docker-compose up -d
```

The Docker image:
- Builds the Go binary and React app in one multi-stage Dockerfile
- Serves the frontend from `/app/static` (SPA fallback to index.html)
- Runs migrations automatically on startup
- Runs as a non-root user
- Health checks every 30 seconds

---

## API endpoints

### Public

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Liveness check |
| `GET` | `/ready` | Readiness check |
| `POST` | `/api/auth/login` | Login → returns token pair |
| `POST` | `/api/auth/logout` | Revoke refresh token |
| `POST` | `/api/auth/refresh` | Rotate token pair |

### FITS data (any authenticated role)

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/files` | List files — search, filter, sort, paginate |
| `GET` | `/api/files/{id}` | Single file detail |
| `GET` | `/api/files/{id}/headers` | Paginated raw headers |
| `GET` | `/api/files/{id}/metadata` | Typed metadata |
| `GET` | `/api/files/{id}/metadata/history` | Edit history |
| `GET` | `/api/jobs` | List processing jobs |
| `GET` | `/api/jobs/{id}` | Job detail |
| `GET` | `/api/jobs/{id}/errors` | Job error list |
| `GET` | `/api/jobs/{id}/status` | Lightweight status poll |
| `GET` | `/api/stats` | Dashboard summary counts |

### Editor + Admin

| Method | Path | Description |
|---|---|---|
| `PUT` | `/api/files/{id}/metadata` | Edit one metadata field |

### Admin only

| Method | Path | Description |
|---|---|---|
| `DELETE` | `/api/files/{id}` | Delete file and all data |
| `POST` | `/api/scan` | Trigger a new scan |
| `GET` | `/api/users` | List users |
| `POST` | `/api/users` | Create user |
| `GET` | `/api/users/{id}` | Get user |
| `PUT` | `/api/users/{id}` | Edit user |
| `DELETE` | `/api/users/{id}` | Delete user |
| `PUT` | `/api/users/{id}/password` | Reset user password |
| `GET` | `/api/users/me` | Own profile |
| `PUT` | `/api/users/me/password` | Change own password |
| `GET` | `/api/audit-logs` | System audit log |

---

## User roles

| Role | View data | Edit metadata | Manage users | Delete files | Trigger scan |
|---|---|---|---|---|---|
| viewer | ✅ | ❌ | ❌ | ❌ | ❌ |
| editor | ✅ | ✅ | ❌ | ❌ | ❌ |
| admin  | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | _(required in prod)_ | Database password |
| `DB_NAME` | `fits_db` | Database name |
| `DB_SSLMODE` | `disable` | `disable` / `require` |
| `JWT_ACCESS_SECRET` | _(insecure default)_ | Min 32 chars in production |
| `ADMIN_EMAIL` / `ADMIN_PASSWORD` | `admin@fits.local` / _(random, printed once)_ | First admin, created only when no active admin exists |
| `TRUSTED_PROXIES` | _(none)_ | CIDRs allowed to set `X-Forwarded-For` |
| `COOKIE_SECURE` | `true` in production | `Secure` flag of the refresh cookie |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `SERVER_ADDR` | `:8080` | HTTP listen address |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |
| `RATE_LIMIT_AUTH_RPS` | `5` | Auth endpoint rate limit (req/s per IP) |
| `RATE_LIMIT_API_RPS` | `60` | API rate limit (req/s per IP) |
| `MAX_BODY_BYTES` | `1048576` | Request body size limit (1 MB) |
| `STATIC_DIR` | _(empty)_ | Serve frontend from this path (production) |
| `FITS_SCAN_DIR` | `./testdata` | Directory to scan for FITS files |
| `FITS_WORKERS` | `4` | Parallel processing goroutines |
| `APP_ENV` | `development` | `development` / `production` |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `LOG_DIR` | `logs` | Log file directory |
| `MIGRATIONS_DIR` | `migrations` | Path to SQL migration files |

---

## Database tables

| Table | Purpose |
|---|---|
| `fits_files` | One row per FITS file (path, checksum, status, timing) |
| `fits_headers` | EAV — every raw keyword from every HDU |
| `fits_metadata` | Typed flat table (~40 columns) for fast science queries |
| `metadata_overrides` | Audit trail of every metadata field edit |
| `processing_jobs` | One row per scan run with progress counters |
| `processing_errors` | Per-file failure records linked to a job |
| `users` | User accounts with role (admin/editor/viewer) |
| `sessions` | Refresh token store (stored as SHA-256 hash) |
| `audit_logs` | System-wide audit log for all actions |

---

## Logs

| File | Contents |
|---|---|
| `logs/fits-processor-YYYY-MM-DD.log` | All levels — JSON |
| `logs/fits-processor-YYYY-MM-DD.error.log` | Errors only — JSON |

Console output is human-readable. Set `LOG_LEVEL=debug` for per-file and per-keyword detail.
