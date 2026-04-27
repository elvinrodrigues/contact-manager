-- 003_admin_role.sql
-- Adds role column and sets up admin user

ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'user';

-- Make the primary user an admin
UPDATE users SET role = 'admin' WHERE email = 'elvinrodrigues3456@gmail.com';
