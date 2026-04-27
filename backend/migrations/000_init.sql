-- 000_init.sql
-- Creates the base tables: categories and contacts.
-- Safe to run multiple times (fully idempotent).

-- 1. Categories lookup table
CREATE TABLE IF NOT EXISTS categories (
    id   SERIAL PRIMARY KEY,
    name TEXT   NOT NULL UNIQUE
);

-- Seed default categories (skip if already present)
INSERT INTO categories (name) VALUES
    ('General'),
    ('Family'),
    ('Friends'),
    ('Work'),
    ('College')
ON CONFLICT (name) DO NOTHING;

-- 2. Contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id          SERIAL       PRIMARY KEY,
    name        TEXT         NOT NULL,
    phone       TEXT         NOT NULL,
    email       TEXT,
    category_id INTEGER      NOT NULL DEFAULT 1 REFERENCES categories(id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    purge_at    TIMESTAMPTZ
);
