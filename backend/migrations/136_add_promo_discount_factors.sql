ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS discount_factor DECIMAL(10,4);

ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS discount_label TEXT;

CREATE TABLE IF NOT EXISTS user_promo_discounts (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    promo_code_id BIGINT REFERENCES promo_codes(id) ON DELETE SET NULL,
    discount_factor DECIMAL(10,4) NOT NULL DEFAULT 1.0,
    discount_label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_promo_discounts_promo_code_id
    ON user_promo_discounts(promo_code_id);

COMMENT ON COLUMN promo_codes.discount_factor IS '优惠码绑定的长期折扣因子，1.0 表示无折扣';
COMMENT ON COLUMN promo_codes.discount_label IS '优惠码折扣标签，用于前端展示';
COMMENT ON TABLE user_promo_discounts IS '用户长期优惠码折扣绑定';
