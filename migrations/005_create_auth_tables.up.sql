-- 005_create_auth_tables.up.sql
-- Users table with role-based access control.
-- Sessions table for refresh token tracking and JWT blacklisting.

CREATE TYPE user_role AS ENUM ('admin', 'editor', 'viewer');

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL       PRIMARY KEY,
    email         TEXT            NOT NULL UNIQUE,
    password_hash TEXT            NOT NULL,
    full_name     TEXT            NOT NULL DEFAULT '',
    role          user_role       NOT NULL DEFAULT 'viewer',
    is_active     BOOLEAN         NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email     ON users (email);
CREATE INDEX idx_users_role      ON users (role);
CREATE INDEX idx_users_is_active ON users (is_active);

-- Sessions tracks issued refresh tokens.
-- When a user logs out, the session row is deleted (token revoked).
-- Access tokens are stateless JWTs; only refresh tokens are tracked here.
CREATE TABLE IF NOT EXISTS sessions (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         BIGINT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token   TEXT            NOT NULL UNIQUE,   -- hashed
    user_agent      TEXT            NOT NULL DEFAULT '',
    ip_address      TEXT            NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ     NOT NULL,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id       ON sessions (user_id);
CREATE INDEX idx_sessions_refresh_token ON sessions (refresh_token);
CREATE INDEX idx_sessions_expires_at    ON sessions (expires_at);

-- Seed: default admin user (password: Admin@1234  — CHANGE IN PRODUCTION)
-- Hash generated with bcrypt cost=12 for "Admin@1234"
INSERT INTO users (email, password_hash, full_name, role)
VALUES (
    'admin@fits.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/o4BdLrG6.',
    'System Administrator',
    'admin'
) ON CONFLICT (email) DO NOTHING;
