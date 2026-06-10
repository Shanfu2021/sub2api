-- Add per-group OpenAI scheduling strategy.
-- Keep this migration backwards-compatible with already-running services:
-- old code ignores the column, new code should treat NULL/unknown as weighted.
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS scheduling_strategy varchar(32);

ALTER TABLE groups
  ALTER COLUMN scheduling_strategy SET DEFAULT 'weighted';

UPDATE groups
SET scheduling_strategy = 'weighted'
WHERE scheduling_strategy IS NULL;

COMMENT ON COLUMN groups.scheduling_strategy IS '账号调度策略：weighted=加权负载均衡，strict_priority=严格优先级';
