CREATE TABLE IF NOT EXISTS enterprise_employee_group_defaults (
  id BIGSERIAL PRIMARY KEY,
  enterprise_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS enterprise_employee_group_defaults_enterprise_group_active_key
  ON enterprise_employee_group_defaults(enterprise_user_id, group_id)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_group_defaults_enterprise_user_id
  ON enterprise_employee_group_defaults(enterprise_user_id)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_group_defaults_group_id
  ON enterprise_employee_group_defaults(group_id)
  WHERE deleted_at IS NULL;
