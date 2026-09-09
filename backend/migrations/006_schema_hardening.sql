-- 006_schema_hardening.sql
-- Constraints, indexes and lifecycle rules the original schema left undeclared.

-- ─── 1. Tighten user columns ─────────────────────────────────────────────
-- Both were added nullable. The Go code scans them into bool and string, so a
-- NULL would surface as a scan failure (a 500 on login) rather than a default.
UPDATE users SET is_verified = false WHERE is_verified IS NULL;
UPDATE users SET role = 'user' WHERE role IS NULL OR role NOT IN ('user', 'admin');

ALTER TABLE users ALTER COLUMN is_verified SET NOT NULL;
ALTER TABLE users ALTER COLUMN is_verified SET DEFAULT false;
ALTER TABLE users ALTER COLUMN role SET NOT NULL;
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'user';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_role_valid') THEN
        ALTER TABLE users ADD CONSTRAINT users_role_valid CHECK (role IN ('user', 'admin'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_name_not_blank') THEN
        ALTER TABLE users ADD CONSTRAINT users_name_not_blank CHECK (length(trim(name)) > 0);
    END IF;
END $$;

-- ─── 2. Contact column constraints ───────────────────────────────────────
-- The application rejects a blank name, but '' is not NULL, so NOT NULL alone
-- never enforced it.
UPDATE contacts SET name = '(unnamed)' WHERE length(trim(name)) = 0;
-- An empty-string email is not a value, it is a missing one.
UPDATE contacts SET email = NULL WHERE email IS NOT NULL AND length(trim(email)) = 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'contacts_name_not_blank') THEN
        ALTER TABLE contacts ADD CONSTRAINT contacts_name_not_blank CHECK (length(trim(name)) > 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'contacts_email_not_blank') THEN
        ALTER TABLE contacts ADD CONSTRAINT contacts_email_not_blank
            CHECK (email IS NULL OR length(trim(email)) > 0);
    END IF;
END $$;

-- ─── 3. Foreign key lifecycle ────────────────────────────────────────────
-- Both keys previously defaulted to NO ACTION, and the application emulated a
-- cascade in Go with an unchecked DELETE. Declare the intent in the schema.
DO $$
DECLARE
    cname TEXT;
BEGIN
    -- contacts.user_id -> users.id : deleting a user removes their contacts.
    SELECT con.conname INTO cname
    FROM pg_constraint con
    JOIN pg_class rel ON con.conrelid = rel.oid
    WHERE rel.relname = 'contacts' AND con.contype = 'f'
      AND con.conkey = ARRAY[(SELECT attnum FROM pg_attribute
                              WHERE attrelid = rel.oid AND attname = 'user_id')];
    IF cname IS NOT NULL THEN
        EXECUTE format('ALTER TABLE contacts DROP CONSTRAINT %I', cname);
    END IF;
    ALTER TABLE contacts
        ADD CONSTRAINT contacts_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

    -- contacts.category_id -> categories.id : a category in use cannot vanish.
    cname := NULL;
    SELECT con.conname INTO cname
    FROM pg_constraint con
    JOIN pg_class rel ON con.conrelid = rel.oid
    WHERE rel.relname = 'contacts' AND con.contype = 'f'
      AND con.conkey = ARRAY[(SELECT attnum FROM pg_attribute
                              WHERE attrelid = rel.oid AND attname = 'category_id')];
    IF cname IS NOT NULL THEN
        EXECUTE format('ALTER TABLE contacts DROP CONSTRAINT %I', cname);
    END IF;
    ALTER TABLE contacts
        ADD CONSTRAINT contacts_category_id_fkey
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;
END $$;

-- ─── 4. Indexes the queries actually need ────────────────────────────────
-- user_id leads every index because it is the equality predicate on every query;
-- an index that does not lead with it cannot serve them.

-- Serves the list endpoint: filter by owner (and optionally category), then
-- ORDER BY name, id. Without this each page sorts the user's whole active set.
CREATE INDEX IF NOT EXISTS idx_contacts_active_owner_name
    ON contacts (user_id, name, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_active_owner_category
    ON contacts (user_id, category_id, name, id)
    WHERE deleted_at IS NULL;

-- Serves the trash listing (ORDER BY deleted_at DESC) and the cleanup worker.
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_owner
    ON contacts (user_id, deleted_at DESC, id)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_contacts_purge_at
    ON contacts (purge_at)
    WHERE deleted_at IS NOT NULL AND purge_at IS NOT NULL;

-- Serves the dashboard's "added this week" counter.
CREATE INDEX IF NOT EXISTS idx_contacts_owner_created
    ON contacts (user_id, created_at DESC, id)
    WHERE deleted_at IS NULL;

-- ILIKE '%term%' cannot use a B-tree at all. Trigram indexes are what make a
-- leading-wildcard match indexable.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_contacts_name_trgm  ON contacts USING gin (name  gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_contacts_phone_trgm ON contacts USING gin (phone gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_contacts_email_trgm ON contacts USING gin (email gin_trgm_ops);

-- ─── 5. Backfill purge_at for contacts deleted before it was populated ───
UPDATE contacts
SET purge_at = deleted_at + interval '30 days'
WHERE deleted_at IS NOT NULL AND purge_at IS NULL;

-- ─── 6. updated_at maintained by the database ────────────────────────────
-- Previously only one UPDATE statement bumped it, so any other write path left
-- it stale.
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS contacts_set_updated_at ON contacts;
CREATE TRIGGER contacts_set_updated_at
    BEFORE UPDATE ON contacts
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
