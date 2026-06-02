CREATE TABLE IF NOT EXISTS agent_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pool_concurrency INTEGER NOT NULL DEFAULT 0,
    pool_rpm INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_user_id
    ON agent_profiles(user_id)
    WHERE deleted_at IS NULL;
