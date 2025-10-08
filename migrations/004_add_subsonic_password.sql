-- Add subsonic_password column to users table
-- This stores the plain text password for Subsonic API authentication
-- Subsonic clients use MD5(password + salt) for authentication
ALTER TABLE users ADD COLUMN IF NOT EXISTS subsonic_password VARCHAR(255);

-- Update existing users to have a default subsonic password
-- You should update this manually for each user with their actual password
UPDATE users SET subsonic_password = 'changeme' WHERE subsonic_password IS NULL;