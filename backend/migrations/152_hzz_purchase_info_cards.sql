CREATE TABLE IF NOT EXISTS purchase_info_cards (
    id BIGSERIAL PRIMARY KEY,
    owner_user_id BIGINT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    purchase_url TEXT NOT NULL DEFAULT '',
    contact TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_purchase_info_cards_owner_sort
    ON purchase_info_cards(owner_user_id, sort_order, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_purchase_info_cards_root_sort
    ON purchase_info_cards(sort_order, id)
    WHERE owner_user_id IS NULL AND deleted_at IS NULL;

