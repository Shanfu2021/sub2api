CREATE TABLE IF NOT EXISTS agent_child_notes (
  manager_user_id BIGINT NOT NULL,
  child_user_id BIGINT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (manager_user_id, child_user_id)
);

CREATE INDEX IF NOT EXISTS idx_agent_child_notes_child_user_id
  ON agent_child_notes(child_user_id);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'agent_child_notes_manager_user_id_fk'
  ) THEN
    ALTER TABLE agent_child_notes
      ADD CONSTRAINT agent_child_notes_manager_user_id_fk
      FOREIGN KEY (manager_user_id) REFERENCES users(id)
      ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'agent_child_notes_child_user_id_fk'
  ) THEN
    ALTER TABLE agent_child_notes
      ADD CONSTRAINT agent_child_notes_child_user_id_fk
      FOREIGN KEY (child_user_id) REFERENCES users(id)
      ON DELETE CASCADE;
  END IF;
END $$;
