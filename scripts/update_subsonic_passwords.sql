-- Script to update subsonic_password for existing users
-- Run this after applying migration 004_add_subsonic_password.sql

-- First, check if the column exists
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name='users' AND column_name='subsonic_password'
    ) THEN
        RAISE EXCEPTION 'Column subsonic_password does not exist. Please run migration 004_add_subsonic_password.sql first.';
    END IF;
END $$;

-- Update the admin user with the default password
UPDATE users 
SET subsonic_password = 'admin123' 
WHERE username = 'admin' AND (subsonic_password IS NULL OR subsonic_password = 'changeme');

-- For user 'antonio', you need to set the actual password
-- Replace 'your_password_here' with the actual password for antonio
-- UPDATE users 
-- SET subsonic_password = 'your_password_here' 
-- WHERE username = 'antonio';

-- Display users that still need subsonic_password configured
SELECT id, username, email, 
       CASE 
           WHEN subsonic_password IS NULL THEN 'NOT SET'
           WHEN subsonic_password = 'changeme' THEN 'DEFAULT (needs update)'
           ELSE 'CONFIGURED'
       END as subsonic_status
FROM users
ORDER BY id;