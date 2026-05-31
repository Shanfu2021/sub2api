ALTER TABLE enterprise_tenants
    ADD COLUMN IF NOT EXISTS member_default_pricing_factor DECIMAL(10,4) NOT NULL DEFAULT 0;

ALTER TABLE enterprise_tenants
    ADD COLUMN IF NOT EXISTS concurrency INTEGER NOT NULL DEFAULT 0;

ALTER TABLE enterprise_tenant_groups
    ADD COLUMN IF NOT EXISTS member_default_multiplier DECIMAL(10,4) NULL;

ALTER TABLE enterprise_memberships
    ALTER COLUMN pricing_factor SET DEFAULT 0;

UPDATE enterprise_memberships
SET pricing_factor = 0
WHERE pricing_factor = 1.0;

COMMENT ON COLUMN enterprise_tenants.member_default_pricing_factor IS 'Default member-facing pricing multiplier for enterprise members. 0 means unset and falls back to enterprise floor.';
COMMENT ON COLUMN enterprise_tenants.concurrency IS 'Enterprise-wide concurrent request limit. 0 means unlimited.';
COMMENT ON COLUMN enterprise_tenant_groups.member_default_multiplier IS 'Default member-facing pricing multiplier for this enterprise group. NULL falls back to tenant member default or enterprise floor.';
COMMENT ON COLUMN enterprise_memberships.pricing_factor IS 'Member-facing default pricing multiplier. 0 means inherit enterprise/group default.';
