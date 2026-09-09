-- 001_auth.sql
-- Adds authentication support and makes contacts user-scoped.

-- 1. Users table
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The UNIQUE constraint on email already creates an index; a second one on the
-- same column would only be dead weight to maintain.

-- 2. Add user_id to contacts (nullable first so existing rows can be backfilled)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id);

-- 3. Backfill any pre-auth contacts onto a placeholder owner.
--
--    This account exists only to satisfy the NOT NULL constraint in step 5 and
--    is deliberately NOT login-capable: '!' is not a valid bcrypt hash, so no
--    password can ever compare equal to it. Claim these contacts by reassigning
--    them to a real account:
--        UPDATE contacts SET user_id = (SELECT id FROM users WHERE email = 'you@example.com')
--        WHERE user_id = (SELECT id FROM users WHERE email = 'orphaned@invalid');
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM contacts WHERE user_id IS NULL) THEN
        INSERT INTO users (name, email, password_hash)
        VALUES ('Unclaimed contacts', 'orphaned@invalid', '!')
        ON CONFLICT (email) DO NOTHING;

        UPDATE contacts
        SET user_id = (SELECT id FROM users WHERE email = 'orphaned@invalid')
        WHERE user_id IS NULL;

        RAISE NOTICE 'Backfilled pre-auth contacts onto the placeholder owner orphaned@invalid';
    END IF;
END $$;

-- 4. Make user_id NOT NULL now that every row has an owner
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'contacts'
          AND column_name = 'user_id'
          AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE contacts ALTER COLUMN user_id SET NOT NULL;
    END IF;
END $$;

-- 5. Ownership lookup index
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts (user_id);
