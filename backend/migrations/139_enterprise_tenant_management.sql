CREATE TABLE IF NOT EXISTS enterprise_tenants (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    code VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    notes TEXT NOT NULL DEFAULT '',
    portal_host VARCHAR(255) NOT NULL DEFAULT '',
    pricing_floor_factor DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    pricing_scope VARCHAR(32) NOT NULL DEFAULT 'balance',
    balance_quota_total DECIMAL(18,6) NOT NULL DEFAULT 0,
    balance_quota_used DECIMAL(18,6) NOT NULL DEFAULT 0,
    created_by BIGINT,
    updated_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enterprise_tenants_status
    ON enterprise_tenants(status);

CREATE TABLE IF NOT EXISTS enterprise_memberships (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES enterprise_tenants(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_role VARCHAR(32) NOT NULL DEFAULT 'member',
    member_note TEXT NOT NULL DEFAULT '',
    joined_via VARCHAR(32) NOT NULL DEFAULT 'manual_bind',
    joined_source VARCHAR(255) NOT NULL DEFAULT '',
    pricing_factor DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    pricing_scope VARCHAR(32) NOT NULL DEFAULT 'balance',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enterprise_memberships_user_unique UNIQUE (user_id),
    CONSTRAINT enterprise_memberships_tenant_user_unique UNIQUE (tenant_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_enterprise_memberships_tenant_id
    ON enterprise_memberships(tenant_id);

CREATE INDEX IF NOT EXISTS idx_enterprise_memberships_member_role
    ON enterprise_memberships(member_role);

CREATE TABLE IF NOT EXISTS enterprise_invite_codes (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES enterprise_tenants(id) ON DELETE CASCADE,
    code VARCHAR(64) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    max_uses INTEGER NOT NULL DEFAULT 0,
    used_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    created_by BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enterprise_invite_codes_tenant_id
    ON enterprise_invite_codes(tenant_id);

CREATE INDEX IF NOT EXISTS idx_enterprise_invite_codes_status
    ON enterprise_invite_codes(status);

CREATE TABLE IF NOT EXISTS enterprise_wallet_ledger (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES enterprise_tenants(id) ON DELETE CASCADE,
    operator_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    direction VARCHAR(32) NOT NULL,
    amount DECIMAL(18,6) NOT NULL DEFAULT 0,
    balance_before DECIMAL(18,6) NOT NULL DEFAULT 0,
    balance_after DECIMAL(18,6) NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    related_user_balance_log_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enterprise_wallet_ledger_tenant_id
    ON enterprise_wallet_ledger(tenant_id);

CREATE INDEX IF NOT EXISTS idx_enterprise_wallet_ledger_target_user_id
    ON enterprise_wallet_ledger(target_user_id);

CREATE INDEX IF NOT EXISTS idx_enterprise_wallet_ledger_created_at
    ON enterprise_wallet_ledger(created_at DESC);

CREATE TABLE IF NOT EXISTS enterprise_tenant_groups (
    id BIGSERIAL PRIMARY KEY,
    tenant_id BIGINT NOT NULL REFERENCES enterprise_tenants(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enterprise_tenant_groups_unique UNIQUE (tenant_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_enterprise_tenant_groups_tenant_id
    ON enterprise_tenant_groups(tenant_id);

CREATE INDEX IF NOT EXISTS idx_enterprise_tenant_groups_group_id
    ON enterprise_tenant_groups(group_id);

COMMENT ON TABLE enterprise_tenants IS '企业租户定义';
COMMENT ON TABLE enterprise_memberships IS '企业成员归属与角色关系';
COMMENT ON TABLE enterprise_invite_codes IS '企业邀请码';
COMMENT ON TABLE enterprise_wallet_ledger IS '企业额度池台账';
COMMENT ON TABLE enterprise_tenant_groups IS '企业允许使用的分组集合';
