ALTER TABLE promo_codes
    ADD COLUMN IF NOT EXISTS discount_scope VARCHAR(20) NOT NULL DEFAULT 'all';

ALTER TABLE user_promo_discounts
    ADD COLUMN IF NOT EXISTS discount_scope VARCHAR(20) NOT NULL DEFAULT 'all';

UPDATE promo_codes
SET discount_scope = 'all'
WHERE discount_scope IS NULL OR discount_scope = '';

UPDATE user_promo_discounts
SET discount_scope = 'all'
WHERE discount_scope IS NULL OR discount_scope = '';

COMMENT ON COLUMN promo_codes.discount_scope IS '优惠码折扣生效范围: all, balance, subscription';
COMMENT ON COLUMN user_promo_discounts.discount_scope IS '用户折扣生效范围快照: all, balance, subscription';
