ALTER TABLE enterprise_tenant_groups
    ADD COLUMN IF NOT EXISTS pricing_floor_multiplier DECIMAL(10,4) NULL;

COMMENT ON COLUMN enterprise_tenant_groups.pricing_floor_multiplier IS '企业在该分组的默认计费倍率；NULL 表示沿用企业默认倍率。';
