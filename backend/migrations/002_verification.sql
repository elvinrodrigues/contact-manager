-- 002_verification.sql
-- Adds columns for email verification and password reset flows with Resend

ALTER TABLE users
ADD COLUMN IF NOT EXISTS is_verified BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS verification_token_hash TEXT,
ADD COLUMN IF NOT EXISTS verification_token_expiry TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS reset_token_hash TEXT,
ADD COLUMN IF NOT EXISTS reset_token_expiry TIMESTAMPTZ;

-- Performance: index token hashes for fast lookup during verify/reset flows
CREATE INDEX IF NOT EXISTS idx_users_verification_token_hash ON users (verification_token_hash);
CREATE INDEX IF NOT EXISTS idx_users_reset_token_hash ON users (reset_token_hash);

-- Mark ALL pre-existing users as verified so they don't get locked out
UPDATE users SET is_verified = true WHERE is_verified = false;

-- Create default user (verified, password: rodrigues)
INSERT INTO users (name, email, password_hash, is_verified)
VALUES (
  'Elvin Rodrigues',
  'elvinrodrigues3456@gmail.com',
  '$2a$10$E1NIsTqg/mDBtMw1H5FmZ.v7xyQpRaBbeJqTI8m085XPmLkJvEbdW',
  true
)
ON CONFLICT (email) DO NOTHING;
