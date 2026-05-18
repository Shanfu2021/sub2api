-- Add per-group OpenAI scheduling strategy.
-- weighted: existing weighted Top-K load-aware scheduling.
-- strict_priority: only falls back to lower priority when every higher-priority
-- account is unavailable, incompatible, full, or cannot accept waiting.
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS scheduling_strategy varchar(32) NOT NULL DEFAULT 'weighted';

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'groups_scheduling_strategy_check'
  ) THEN
    ALTER TABLE groups
      ADD CONSTRAINT groups_scheduling_strategy_check
      CHECK (scheduling_strategy IN ('weighted', 'strict_priority'));
  END IF;
END $$;

COMMENT ON COLUMN groups.scheduling_strategy IS 'OpenAI账号调度策略：weighted=加权负载均衡，strict_priority=严格优先级';
