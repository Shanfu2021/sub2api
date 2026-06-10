CREATE TABLE IF NOT EXISTS agent_invite_group_defaults (
    id BIGSERIAL PRIMARY KEY,
    agent_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS agent_invite_group_defaults_agent_group_active_key
    ON agent_invite_group_defaults(agent_user_id, group_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_invite_group_defaults_agent_user_id
    ON agent_invite_group_defaults(agent_user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_invite_group_defaults_group_id
    ON agent_invite_group_defaults(group_id)
    WHERE deleted_at IS NULL;
