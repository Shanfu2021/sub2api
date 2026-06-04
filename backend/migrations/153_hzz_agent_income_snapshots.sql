ALTER TABLE usage_logs
  ADD COLUMN IF NOT EXISTS agent_owner_user_id BIGINT NULL,
  ADD COLUMN IF NOT EXISTS agent_user_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS agent_cost_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS agent_income DECIMAL(20,10) NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_usage_logs_agent_owner_user_id_created_at
  ON usage_logs(agent_owner_user_id, created_at)
  WHERE agent_owner_user_id IS NOT NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'usage_logs_agent_owner_user_id_fk'
  ) THEN
    ALTER TABLE usage_logs
      ADD CONSTRAINT usage_logs_agent_owner_user_id_fk
      FOREIGN KEY (agent_owner_user_id) REFERENCES users(id)
      ON DELETE SET NULL;
  END IF;
END $$;
