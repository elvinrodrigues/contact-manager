-- 002_verification.sql
-- Adds columns for the email verification and password reset flows.

-- Whether the verification columns already existed decides how existing users
-- are treated below, so record it before the ALTER changes the answer.
DO $$
DECLARE
    fresh_install BOOLEAN;
BEGIN
    fresh_install := NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'is_verified'
    );

    ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_verified               BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS verification_token_hash   TEXT,
    ADD COLUMN IF NOT EXISTS verification_token_expiry TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reset_token_hash          TEXT,
    ADD COLUMN IF NOT EXISTS reset_token_expiry        TIMESTAMPTZ;

    -- Accounts that predate verification have no way to receive a link, so they
    -- are grandfathered in — but only on the run that actually introduces the
    -- column. Previously this was an unconditional
    --     UPDATE users SET is_verified = true WHERE is_verified = false;
    -- which, because migrations re-ran on every boot, verified every pending
    -- signup each time the process restarted.
    IF fresh_install THEN
        UPDATE users SET is_verified = true;
        RAISE NOTICE 'Grandfathered % pre-verification account(s)',
            (SELECT count(*) FROM users);
    END IF;
END $$;

-- Token lookups happen once per verify/reset request and always on the hash.
-- Partial indexes keep them small: almost every row has NULL in both columns.
CREATE INDEX IF NOT EXISTS idx_users_verification_token_hash
    ON users (verification_token_hash)
    WHERE verification_token_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_reset_token_hash
    ON users (reset_token_hash)
    WHERE reset_token_hash IS NOT NULL;
