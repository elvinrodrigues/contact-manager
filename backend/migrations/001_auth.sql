-- 001_auth.sql
-- Adds authentication support and makes contacts user-scoped.
-- Safe to run multiple times (fully idempotent).

-- 1. Create users table
CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- 2. Add user_id column to contacts (nullable first so we can backfill)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS user_id INTEGER REFERENCES users(id);

-- 3. Create default admin user for existing data
--    Password hash = bcrypt("admin123") — change immediately after first login.
INSERT INTO users (name, email, password_hash)
VALUES ('admin', 'admin@local', '$2a$10$PbRiV6NbH9KNiHL9zwJ/luHvj7p1xlp6YP6XQw4ymvZ49A3Q9dgMu')
ON CONFLICT (email) DO NOTHING;

-- 4. Assign all orphaned contacts to the admin user
UPDATE contacts SET user_id = (SELECT id FROM users WHERE email = 'admin@local') WHERE user_id IS NULL;

-- 5. Make user_id NOT NULL (idempotent — only if currently nullable)
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

-- 6. Index for query performance
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts (user_id);
