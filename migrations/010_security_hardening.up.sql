-- The seeded admin (admin@fits.local) uses a bcrypt hash that is public in migration 005
-- and does not match the documented password. Neutralise it if it was never changed;
-- the application bootstraps a proper admin at startup (see ADMIN_EMAIL / ADMIN_PASSWORD).
UPDATE users
   SET is_active = FALSE, password_hash = '!disabled'
 WHERE password_hash = '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPj/o4BdLrG6.';

DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE password_hash = '!disabled');
