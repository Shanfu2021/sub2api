ALTER TABLE enterprise_tenants
    ADD COLUMN IF NOT EXISTS balance_overdraft_limit DECIMAL(18,6) NOT NULL DEFAULT 0;

ALTER TABLE enterprise_tenants
    ADD COLUMN IF NOT EXISTS balance_quota_spent DECIMAL(18,6) NOT NULL DEFAULT 0;

COMMENT ON COLUMN enterprise_tenants.balance_overdraft_limit IS 'Enterprise credit limit for platform-side spending before top-up.';
COMMENT ON COLUMN enterprise_tenants.balance_quota_spent IS 'Platform-side cost consumed by enterprise members. Member allocations remain in balance_quota_used.';
