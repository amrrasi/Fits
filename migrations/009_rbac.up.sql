-- 009_rbac.up.sql
-- Replaces the flat users.role enum as the *authorization* source of truth
-- with a real RBAC schema: roles, permissions, role_permissions (M:N),
-- user_roles (M:N — a user can hold more than one role).
--
-- users.role is kept as a convenience "primary role" label (used for display,
-- and to seed a user's built-in role on create/update), but access-control
-- decisions are now driven entirely by these tables.

CREATE TABLE IF NOT EXISTS permissions (
    id          BIGSERIAL   PRIMARY KEY,
    code        TEXT        NOT NULL UNIQUE,   -- e.g. 'files.delete', 'users.create'
    description TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id          BIGSERIAL   PRIMARY KEY,
    name        TEXT        NOT NULL UNIQUE,   -- 'admin', 'editor', 'viewer', or custom
    description TEXT        NOT NULL DEFAULT '',
    is_builtin  BOOLEAN     NOT NULL DEFAULT FALSE, -- built-in roles cannot be deleted
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     BIGINT      NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_by BIGINT      REFERENCES users(id) ON DELETE SET NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions (permission_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role             ON user_roles (role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user              ON user_roles (user_id);

-- ── Seed permission catalog ───────────────────────────────────────────────────
INSERT INTO permissions (code, description) VALUES
    ('files.view',           'مشاهده‌ی لیست و جزئیات فایل‌های FITS'),
    ('files.scan',           'اجرای اسکن جدید روی یک مسیر یا فایل'),
    ('files.delete',         'حذف یک فایل FITS'),
    ('files.metadata.edit',  'ویرایش متادیتای فایل'),
    ('jobs.view',            'مشاهده‌ی جاب‌های پردازش و وضعیت آن‌ها'),
    ('users.view',           'مشاهده‌ی لیست و جزئیات کاربران'),
    ('users.create',         'ایجاد کاربر جدید'),
    ('users.edit',           'ویرایش اطلاعات کاربر'),
    ('users.delete',         'حذف کاربر'),
    ('users.reset_password', 'بازنشانی رمز عبور سایر کاربران'),
    ('audit.view',           'مشاهده‌ی گزارش رخدادها (audit log)'),
    ('roles.manage',         'مدیریت نقش‌ها، دسترسی‌ها، و تخصیص نقش به کاربران')
ON CONFLICT (code) DO NOTHING;

-- ── Seed built-in roles ────────────────────────────────────────────────────────
INSERT INTO roles (name, description, is_builtin) VALUES
    ('admin',  'دسترسی کامل به تمام بخش‌های سامانه', TRUE),
    ('editor', 'مشاهده و ویرایش متادیتا',            TRUE),
    ('viewer', 'فقط مشاهده',                          TRUE)
ON CONFLICT (name) DO NOTHING;

-- ── Seed role → permission mapping (matches current hard-coded behaviour) ──────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'  -- admin: everything
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'editor' AND p.code IN ('files.view', 'jobs.view', 'files.metadata.edit')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'viewer' AND p.code IN ('files.view', 'jobs.view')
ON CONFLICT DO NOTHING;

-- ── Backfill: give every existing user their current users.role as a real row ──
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u JOIN roles r ON r.name = u.role
ON CONFLICT DO NOTHING;
