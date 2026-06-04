-- Promote legacy level-2 agents to first-level agents under the root admin.
-- Their own children, balances, quotas, and group rate/delegation records are preserved.

WITH root_admin AS (
  SELECT id
  FROM users
  WHERE role = 'admin' AND deleted_at IS NULL
  ORDER BY id
  LIMIT 1
)
UPDATE users
SET role = 'agent_level1',
    parent_user_id = (SELECT id FROM root_admin),
    updated_at = NOW()
WHERE role = 'agent_level2'
  AND deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM root_admin);
