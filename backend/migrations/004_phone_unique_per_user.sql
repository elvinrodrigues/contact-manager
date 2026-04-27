-- 004_phone_unique_per_user.sql
-- Fixes phone uniqueness to be scoped per user instead of globally.
-- Safe to run multiple times (fully idempotent).

-- 1. Drop ANY global UNIQUE constraint on phone alone (regardless of name).
--    Queries pg_constraint to find it dynamically instead of guessing names.
DO $$
DECLARE
    cname TEXT;
BEGIN
    FOR cname IN
        SELECT con.conname
        FROM pg_constraint con
        JOIN pg_class rel ON con.conrelid = rel.oid
        JOIN pg_attribute att ON att.attrelid = rel.oid AND att.attnum = ANY(con.conkey)
        WHERE rel.relname = 'contacts'
          AND con.contype = 'u'
          AND att.attname = 'phone'
          AND array_length(con.conkey, 1) = 1  -- single-column unique on phone only
    LOOP
        EXECUTE format('ALTER TABLE contacts DROP CONSTRAINT %I', cname);
        RAISE NOTICE 'Dropped global UNIQUE constraint: %', cname;
    END LOOP;
END $$;

-- Also drop any standalone unique index on phone alone (not a table constraint)
DROP INDEX IF EXISTS contacts_phone_key;
DROP INDEX IF EXISTS contacts_phone_unique;
DROP INDEX IF EXISTS unique_phone;
DROP INDEX IF EXISTS idx_contacts_phone;

-- 2. Add the correct per-user unique index (partial — only active contacts).
--    Two users can have the same phone number.
--    The same user cannot have two active contacts with the same phone.
CREATE UNIQUE INDEX IF NOT EXISTS contacts_user_phone_active_unique
ON contacts (user_id, phone)
WHERE deleted_at IS NULL;
