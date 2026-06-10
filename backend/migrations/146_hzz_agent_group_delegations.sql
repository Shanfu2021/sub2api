CREATE TABLE IF NOT EXISTS agent_group_delegations (
    id BIGSERIAL PRIMARY KEY,
    manager_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    child_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    can_delegate BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS agent_group_delegations_manager_child_group_active_key
    ON agent_group_delegations(manager_user_id, child_user_id, group_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_group_delegations_child_user_id
    ON agent_group_delegations(child_user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_group_delegations_manager_user_id
    ON agent_group_delegations(manager_user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_agent_group_delegations_group_id
    ON agent_group_delegations(group_id)
    WHERE deleted_at IS NULL;
