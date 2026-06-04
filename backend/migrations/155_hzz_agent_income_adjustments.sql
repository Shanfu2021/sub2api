CREATE TABLE IF NOT EXISTS agent_income_adjustments (
  id BIGSERIAL PRIMARY KEY,
  agent_user_id BIGINT NOT NULL,
  admin_user_id BIGINT NULL,
  amount DECIMAL(20,10) NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_income_adjustments_agent_user_id
  ON agent_income_adjustments(agent_user_id);

CREATE INDEX IF NOT EXISTS idx_agent_income_adjustments_created_at
  ON agent_income_adjustments(created_at);

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'agent_income_adjustments_agent_user_id_fk'
  ) THEN
    ALTER TABLE agent_income_adjustments
      ADD CONSTRAINT agent_income_adjustments_agent_user_id_fk
      FOREIGN KEY (agent_user_id) REFERENCES users(id)
      ON DELETE CASCADE;
  END IF;

  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'agent_income_adjustments_admin_user_id_fk'
  ) THEN
    ALTER TABLE agent_income_adjustments
      ADD CONSTRAINT agent_income_adjustments_admin_user_id_fk
      FOREIGN KEY (admin_user_id) REFERENCES users(id)
      ON DELETE SET NULL;
  END IF;
END $$;
