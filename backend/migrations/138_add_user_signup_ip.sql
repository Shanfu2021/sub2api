ALTER TABLE users
    ADD COLUMN IF NOT EXISTS signup_ip VARCHAR(64) NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS users_signup_ip_unique_active_nonempty
    ON users (signup_ip)
    WHERE deleted_at IS NULL AND signup_ip <> '';
