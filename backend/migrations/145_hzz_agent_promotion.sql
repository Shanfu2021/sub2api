ALTER TABLE users
  ADD COLUMN IF NOT EXISTS parent_user_id BIGINT NULL,
  ADD COLUMN IF NOT EXISTS allocated_concurrency INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS allocated_rpm INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_parent_user_id
  ON users(parent_user_id)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_role_parent_user_id
  ON users(role, parent_user_id)
  WHERE deleted_at IS NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'users_parent_user_id_fk'
  ) THEN
    ALTER TABLE users
      ADD CONSTRAINT users_parent_user_id_fk
      FOREIGN KEY (parent_user_id) REFERENCES users(id)
      ON DELETE SET NULL;
  END IF;
END $$;

