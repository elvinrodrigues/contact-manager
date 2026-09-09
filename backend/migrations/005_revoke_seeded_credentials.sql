-- 005_revoke_seeded_credentials.sql
-- Revokes the accounts that earlier revisions of this migration set created
-- with passwords published in this repository.
--
-- 001_auth.sql seeded admin@local and 002_verification.sql seeded a personal
-- address that 003_admin_role.sql then promoted to admin. Those INSERTs are
-- gone, but a database that already ran them still holds the rows, so they are
-- neutralised here.

-- ─── 1. Token versioning (revokes JWTs on password reset) ────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INTEGER NOT NULL DEFAULT 0;

-- ─── 2. Revoke the previously seeded accounts ────────────────────────────
--
-- The rows are not deleted, because they may own contacts. Instead the password
-- hash is replaced with '!', which no bcrypt comparison can ever match, and the
-- admin role is revoked. Regain access with the password-reset flow, and grant
-- the admin role through the ADMIN_EMAIL environment variable.
DO $$
DECLARE
    revoked INTEGER;
BEGIN
    UPDATE users
    SET password_hash = '!',
        role          = 'user',
        token_version = COALESCE(token_version, 0) + 1
    WHERE password_hash IN (
        -- bcrypt('admin123') seeded by the old 001_auth.sql
        '$2a$10$PbRiV6NbH9KNiHL9zwJ/luHvj7p1xlp6YP6XQw4ymvZ49A3Q9dgMu',
        -- bcrypt('rodrigues') seeded by the old 002_verification.sql
        '$2a$10$E1NIsTqg/mDBtMw1H5FmZ.v7xyQpRaBbeJqTI8m085XPmLkJvEbdW'
    );

    GET DIAGNOSTICS revoked = ROW_COUNT;
    IF revoked > 0 THEN
        RAISE WARNING 'Revoked % account(s) that used a password published in this repository. Use the password-reset flow to regain access.', revoked;
    END IF;
END $$;
