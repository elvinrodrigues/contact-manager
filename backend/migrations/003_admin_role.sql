-- 003_admin_role.sql
-- Adds the role column used for administrative authorization.
--
-- No account is promoted here. Administrators are granted at startup from the
-- ADMIN_EMAIL environment variable, so the admin set is a property of the
-- deployment rather than something baked into the repository.

ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'user';
