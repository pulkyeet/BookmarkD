DROP INDEX IF EXISTS idx_users_google_id;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users DROP COLUMN google_id;
