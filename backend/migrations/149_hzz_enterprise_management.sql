CREATE TABLE IF NOT EXISTS enterprise_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pool_concurrency INTEGER NOT NULL DEFAULT 0,
    pool_rpm INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_enterprise_profiles_user_id
    ON enterprise_profiles(user_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS enterprise_employee_balance_logs (
    id BIGSERIAL PRIMARY KEY,
    enterprise_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operator_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta DECIMAL(18,6) NOT NULL,
    enterprise_balance_before DECIMAL(18,6) NOT NULL,
    enterprise_balance_after DECIMAL(18,6) NOT NULL,
    employee_balance_before DECIMAL(18,6) NOT NULL,
    employee_balance_after DECIMAL(18,6) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_balance_logs_enterprise
    ON enterprise_employee_balance_logs(enterprise_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_balance_logs_employee
    ON enterprise_employee_balance_logs(employee_user_id, created_at DESC);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS employee_disabled_by_enterprise BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_enterprise_employees
    ON users(parent_user_id, role, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_employee_enterprise_disable
    ON users(parent_user_id, employee_disabled_by_enterprise)
    WHERE deleted_at IS NULL AND role = 'employee';
