ALTER TABLE enterprise_tenants
    ADD COLUMN IF NOT EXISTS member_default_concurrency INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN enterprise_tenants.member_default_concurrency IS 'Default concurrency assigned to enterprise members created by invite or manager. 0 means use global signup/default form value.';
