# FITS Processor — Full Project TODO & Improvement Checklist

> **Security hardening pass (done):** scan-path restriction, atomic single-use refresh tokens in an HttpOnly cookie,
> session revocation, per-request role/permission re-check, login throttling + uniform errors, password policy,
> forced password change, real audit log, last-admin guard (transactional), advisory-lock scan (crash-safe),
> parser panic recovery, CSP/HSTS/CORS fixes, LIKE-wildcard escaping, audit retention, permission-based UI,
> device/session management page, manual metadata edits preserved on re-scan, tests + `scripts/smoke_test.sh`.
>
> **Still open (by choice):** login throttle is in-memory (single instance); no 2FA; absolute file paths are shown to
> everyone with `files.view`; no automatic DB backup (use `pg_dump`).

This file tracks every known issue, improvement, and missing feature across the codebase.
Items are grouped by priority: 🔴 Critical (must fix before Phase 4) → 🟠 Important → 🟡 Nice to have → 🟢 Frontend.

Update this file as items are completed. Each item has a status:
- `[ ]` not started
- `[~]` in progress
- `[x]` done

---

## 🔴 CRITICAL — Fix Before Phase 4

### Ingestion / Transaction Safety
- [ ] Wrap the entire per-file ingestion (upsert file + delete old headers + bulk insert headers + upsert metadata + update status) in a single PostgreSQL transaction — currently a crash mid-way leaves partial data
- [ ] Fix duplicate detection / idempotency: currently checksum check and upsert are two separate non-atomic operations, creating a race condition under concurrent workers
- [ ] Add `UNIQUE` constraint on `fits_files.checksum` (separate migration) OR decide that `file_path` is the identity and checksum is just a change-detection signal
- [ ] Define clearly: is a file identified by `file_path` or `checksum`? (moving a file should be a new record; renaming should update; same content at two paths = two records or one?)
- [ ] Prevent concurrent scans running simultaneously — use PostgreSQL Advisory Lock on a fixed key

### Job Lifecycle
- [ ] Expand job status from `running/completed/failed` to:
  - `pending` → `running` → `completed` | `partially_failed` | `failed` | `cancelled`
- [ ] Set job to `partially_failed` when some files succeed and some fail
- [ ] Handle `context.Done()` (SIGINT) by marking running job as `cancelled`, not leaving it as `running` forever
- [ ] Define retry behaviour: should failed files be retried on next scan automatically?
- [ ] Add `processing_attempts` counter per file if retry is implemented

### FITS / Scientific Data
- [ ] Review `fits_files` model — add `observation_id` (nullable, for future linking)
- [ ] Decide: store header `value` as `TEXT` or `JSONB`? Current TEXT is fine but JSONB enables indexed queries on value content
- [ ] Store `value_type` — already in schema, verify parser always sets it correctly
- [ ] Handle **duplicate FITS keywords** (e.g. multiple `COMMENT` cards) — current schema allows it but parser behaviour needs verification
- [ ] Store **card order** (add `card_index INT` to `fits_headers`) — FITS header order is meaningful for some instruments
- [ ] Correctly store `COMMENT` cards (keyword=`COMMENT`, value=full comment text)
- [ ] Correctly store `HISTORY` cards (keyword=`HISTORY`, value=history text)
- [ ] Verify multi-HDU support — test with a file that has IMAGE + BINTABLE extensions
- [ ] Decide: does `fits_metadata` belong to the **file** or to a specific **HDU**? Currently it maps to file (primary HDU only) — make this explicit or add `hdu_index` column
- [ ] Fix `RA` parsing — FITS RA can be in degrees (float) or HH:MM:SS (string) depending on keyword and convention; handle both
- [ ] Fix `DEC` parsing — similarly can be DD:MM:SS or decimal degrees
- [ ] Normalise RA/DEC to decimal degrees as the stored representation
- [ ] Verify `DATE-OBS` parsing — can be `YYYY-MM-DD` or `YYYY-MM-DDTHH:MM:SS` or `YYYY-MM-DDTHH:MM:SS.sss`
- [ ] Verify `TIME-OBS` is correctly combined with `DATE-OBS` when both present
- [ ] Verify `MJD-OBS` stored correctly (double precision)
- [ ] Verify `MJD-END` stored correctly
- [ ] Document units for all stored metadata fields (degrees, seconds, Celsius, metres, etc.)
- [ ] Define Source of Truth: raw `fits_headers` is always canonical; `fits_metadata` is a derived cache — make this contract explicit in code comments

### Metadata Editing
- [ ] Decide whether `fits_metadata` fields are directly editable or only overridable
- [ ] If overridable: create `metadata_overrides` table (file_id, field_name, original_value, override_value, edited_by, edited_at, reason)
- [ ] Never overwrite raw `fits_headers` — edits always go to overrides table
- [ ] Record `edited_by` (user_id), `edited_at`, old value, new value on every change
- [ ] Add validation rules per field (e.g. RA must be 0–360, DEC must be −90–90)
- [ ] Add audit log entry for every metadata edit
- [ ] Decide if versioning / history of edits is needed

### Database
- [ ] Review all Foreign Key `ON DELETE` actions — current CASCADE on headers/metadata is correct; verify processing_errors
- [ ] Add missing `NOT NULL` constraints where NULL makes no semantic sense
- [ ] Add `CHECK` constraints: e.g. `exptime > 0`, `ra BETWEEN 0 AND 360`, `dec BETWEEN -90 AND 90`
- [ ] Add missing `UNIQUE` constraints (checksum policy, session refresh_token already done)
- [ ] Design and add composite indexes for common query patterns:
  - `fits_headers (file_id, keyword)` — already exists, verify covers lookup
  - `fits_metadata (date_obs, filter, object)` — for common science queries
  - `fits_metadata (ra, dec)` — for cone search queries
- [ ] Add partial indexes where useful: e.g. `fits_files (status) WHERE status != 'done'`
- [ ] Review query performance with `EXPLAIN ANALYZE` after loading real data
- [ ] All migrations must be versioned, idempotent (use `IF NOT EXISTS`), and have matching `.down.sql`
- [ ] Define backup strategy and test restore procedure

### Authentication / Authorization
- [ ] **Fail application startup** if `JWT_ACCESS_SECRET` is the default insecure value and `APP_ENV=production`
- [ ] Remove `JWT_REFRESH_SECRET` from config if it is not actually used for signing (currently only `JWT_ACCESS_SECRET` signs tokens — clarify or implement dual signing)
- [ ] Confirm refresh tokens are stored as SHA-256 hash — already implemented, add a test
- [ ] Implement `POST /api/auth/logout-all` — delete all sessions for a user
- [ ] Prevent stale role authorization: if a user's role is changed, their existing access tokens should either be invalidated or the role should be re-checked from DB on sensitive operations
- [ ] Add `POST /api/auth/me` or enrich `GET /api/users/me` to always read from DB (not just from JWT claims) so role changes take effect immediately
- [ ] Write explicit tests for each role's access to each endpoint

---

## 🟠 IMPORTANT — Before or During Phase 4

### Repository Architecture
- [ ] Split `fits_repository.go` into focused repositories:
  - `file_repository.go` — fits_files CRUD
  - `header_repository.go` — fits_headers bulk insert + query
  - `metadata_repository.go` — fits_metadata upsert + query
  - `job_repository.go` — processing_jobs + processing_errors
- [ ] Create dedicated service layer files:
  - `internal/fitsservice/service.go` — FITS file business logic
  - `internal/metaservice/service.go` — metadata read + override logic
  - `internal/jobservice/service.go` — job lifecycle management
  - `internal/auditservice/service.go` — audit log writes
- [ ] All transaction management must happen in one defined layer (service or repository transaction helper) — never split a transaction across two layers
- [ ] Zero business logic in HTTP handlers — handlers only parse request, call service, write response
- [ ] Zero raw SQL in service layer — services call repository methods only

### REST API Design
- [ ] Define full API contract (request/response shapes) before writing any handler
- [ ] All list endpoints must support: `page`, `page_size`, `sort`, `order` (asc/desc)
- [ ] Sort fields must be allowlisted — never pass user input directly to ORDER BY
- [ ] `GET /api/files` filters needed: `status`, `date_from`, `date_to`, `object`, `filter`, `instrume`, `telescope`, `exptime_min`, `exptime_max`
- [ ] `GET /api/files/:id/headers` must be paginated (a file can have hundreds of headers)
- [ ] RA/DEC filtering: cone search or bounding box
- [ ] Standardise all HTTP status codes across handlers
- [ ] Centralize error response format: `{ "error": "...", "code": 400, "request_id": "..." }`
- [ ] Add `Request-ID` header to every response (generate UUID per request in middleware)
- [ ] Add input validation on all POST/PUT endpoints with field-level error messages
- [ ] API error codes (machine-readable string codes alongside HTTP status)
- [ ] OpenAPI / Swagger documentation

### API Endpoints to Implement (Phase 4)
- [ ] `GET    /api/files` — paginated list with all filters
- [ ] `GET    /api/files/:id` — single file + metadata summary
- [ ] `GET    /api/files/:id/headers` — paginated raw headers
- [ ] `GET    /api/files/:id/metadata` — typed metadata
- [ ] `PUT    /api/files/:id/metadata` — editor/admin edit (override model)
- [ ] `DELETE /api/files/:id` — admin only, cascades
- [ ] `GET    /api/jobs` — paginated job list
- [ ] `GET    /api/jobs/:id` — job detail
- [ ] `GET    /api/jobs/:id/errors` — errors for a job
- [ ] `POST   /api/scan` — trigger new scan (admin only, returns job_id)
- [ ] `GET    /api/jobs/:id/status` — lightweight status poll endpoint
- [ ] `GET    /health` — liveness check (already done)
- [ ] `GET    /ready` — readiness check (DB ping + migration version)
- [ ] `GET    /metrics` — Prometheus metrics (optional, Phase 8)

### Scanner / Processor
- [ ] Prevent concurrent processing of the same file (per-file mutex or DB-level lock)
- [ ] Corrupted FITS file handling — `fitsio.Open` panics on some malformed files; add recover()
- [ ] File permission error handling — log clearly, mark as error, continue
- [ ] Partial scan handling — if scan is interrupted, mark job cancelled and leave completed files as done
- [ ] Graceful shutdown — on SIGINT, wait for in-flight file to finish, then exit cleanly
- [ ] Worker pool tuning — add `FITS_WORKER_QUEUE_SIZE` env var for channel buffer
- [ ] Limit DB write pressure — add configurable delay between batches if needed
- [ ] Add `FITS_PROCESSING_TIMEOUT` per file (default 60s) — kill hung parser
- [ ] Record processing duration per file in `fits_files` (`processing_ms BIGINT`)
- [ ] Record processing duration per job in `processing_jobs` (`duration_ms BIGINT`)

### Audit Log
- [ ] Create `audit_logs` table (migration 006):
  - `id`, `user_id` (nullable — system ops have no user), `action`, `entity_type`, `entity_id`, `old_value JSONB`, `new_value JSONB`, `ip_address`, `request_id`, `created_at`
- [ ] Write audit entry on: user create, user update, user delete, password change, role change, metadata edit, file delete, scan trigger
- [ ] System/cron operations (scanner) write audit with `user_id = NULL` and `action = 'system.scan'`

### Security
- [ ] Add input length limits on all string fields (email max 255, keyword max 8, value max 72 per FITS standard)
- [ ] SQL injection review — all queries use `$N` placeholders, verify no string concatenation in query bodies
- [ ] JWT security review — verify `alg` header is checked in `ValidateAccessToken` (already done, confirm)
- [ ] Add rate limiting middleware (e.g. 10 req/s per IP on auth endpoints, 100 req/s on API)
- [ ] CORS configuration — set allowed origins explicitly, not `*`
- [ ] Add security headers middleware: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`
- [ ] Request size limit — add `http.MaxBytesReader` on request body (1MB default)
- [ ] Never log passwords, tokens, or secrets — audit all logger calls
- [ ] Never return stack traces to client — `WriteInternalError` already hides them, confirm all paths
- [ ] Path traversal protection on `file_path` — validate it is within the configured `FITS_SCAN_DIR`
- [ ] Trusted proxy configuration for `X-Forwarded-For` IP extraction

---

## 🟡 NICE TO HAVE

### Testing
- [ ] `internal/fits/parser_test.go` — unit test `ParseFile` with a real small FITS file
- [ ] `internal/fits/scanner_test.go` — unit test `ScanDir` with temp directories
- [ ] `internal/fits/processor_test.go` — test worker pool, duplicate skip, error recording
- [ ] `internal/repository/*_test.go` — repository tests against a test PostgreSQL instance
- [ ] `internal/userservice/service_test.go` — test create/update/delete/last-admin guard
- [ ] `internal/auth/*_test.go` — test login, logout, refresh, token validation
- [ ] Handler integration tests — spin up test server, call endpoints, assert responses
- [ ] Authorization tests — verify each role can/cannot access each endpoint
- [ ] Migration test — run all up migrations on a clean DB, verify schema
- [ ] Transaction rollback test — simulate DB error mid-ingestion, verify no partial data
- [ ] Duplicate detection test — ingest same file twice, verify exactly one record
- [ ] Concurrent processing test — `go test -race ./...`
- [ ] Multi-HDU test — file with PRIMARY + BINTABLE + IMAGE extensions
- [ ] Invalid FITS test — corrupted file, empty file, non-FITS file
- [ ] Missing header test — file with no OBJECT, no DATE-OBS, etc. (all nulls in metadata)
- [ ] RA/DEC parsing test — HH:MM:SS input → decimal degrees output

### Observability
- [ ] Add `request_id` to every log line within a request (use middleware to inject into context)
- [ ] Add `job_id` and `file_id` to all processor log lines
- [ ] Log processing duration per file at INFO level
- [ ] Prometheus metrics endpoint `/metrics` (optional): requests/s, error rate, processing duration, DB pool stats
- [ ] Health check already done — add readiness check that pings DB and checks migration version
- [ ] Failed job alerting — log at ERROR level with enough context to alert on
- [ ] Log rotation already done (daily files) — add retention policy (delete logs older than N days)

---

## 🟢 FRONTEND (Phase 5–7)

### Foundation
- [ ] React + TypeScript + Vite project setup
- [ ] Tailwind CSS
- [ ] React Router v6
- [ ] Axios with interceptors (auto-attach token, auto-refresh on 401)
- [ ] API client layer (typed functions per endpoint, not raw axios calls in components)
- [ ] Auth context (current user, role, token storage in memory not localStorage)
- [ ] Login page with form validation
- [ ] Protected route wrapper
- [ ] Role-based route guard
- [ ] App shell: sidebar + topbar + logout button

### FITS Data Pages
- [ ] Files list — searchable, filterable, sortable, paginated table
- [ ] File detail page
- [ ] Raw headers tab (paginated keyword/value table with search)
- [ ] Metadata tab (typed fields displayed cleanly)
- [ ] Inline edit for metadata (editor/admin only, with confirmation)
- [ ] Processing jobs list
- [ ] Job detail + error list
- [ ] Scan trigger button with live status polling
- [ ] Loading states on all data fetches
- [ ] Empty states (no files yet, no results)
- [ ] Error states (API down, 403, 404)

### User Management (Admin only)
- [ ] Users list with search/filter/pagination
- [ ] Create user modal
- [ ] Edit user modal (name, role, active)
- [ ] Delete user with confirmation dialog
- [ ] Admin password reset
- [ ] My profile page + change password form

### Audit Log UI (Admin)
- [ ] Audit log list with filters (user, action, entity, date range)
- [ ] Audit log detail (old/new value diff)

---

## Completed
- [x] Project structure and Go module setup
- [x] Config loading from `.env` (Phase 1)
- [x] Structured logger — zap, file + error file + console (Phase 1)
- [x] PostgreSQL connection pool with pgx (Phase 1)
- [x] golang-migrate runner (Phase 1)
- [x] Domain models — FITSFile, FITSHeader, FITSMetadata, ProcessingJob, ProcessingError (Phase 1)
- [x] FITS directory scanner (Phase 1)
- [x] FITS header parser with astrogo/fitsio (Phase 1)
- [x] Worker pool processor (Phase 1)
- [x] Repository layer — bulk insert with CopyFrom, upserts (Phase 1)
- [x] Migrations 001–004 — fits_files, fits_headers, fits_metadata, processing tables (Phase 1)
- [x] bcrypt password hashing (Phase 2)
- [x] JWT access + refresh token issue/validate (Phase 2)
- [x] Auth middleware + RequireRole (Phase 2)
- [x] Login / Logout / Refresh endpoints (Phase 2)
- [x] Migration 005 — users + sessions tables, default admin seed (Phase 2)
- [x] Shared API helpers — WriteOK/Paged/Error, DecodeJSON, PathID, Pagination (Phase 3)
- [x] User service — create, update, delete, password change, last-admin guard (Phase 3)
- [x] User handlers — full CRUD + me + change password (Phase 3)
- [x] User repository — ListUsers with dynamic filter/pagination (Phase 3)
