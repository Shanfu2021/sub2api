-- Add automatic schedulable-control fields to scheduled account tests.
ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS auto_schedulable_control BOOLEAN DEFAULT false;

ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS first_token_timeout_ms BIGINT DEFAULT 0;

ALTER TABLE scheduled_test_results
    ADD COLUMN IF NOT EXISTS first_token_ms BIGINT;

ALTER TABLE scheduled_test_results
    ADD COLUMN IF NOT EXISTS decision TEXT DEFAULT 'no_action';

ALTER TABLE scheduled_test_results
    ADD COLUMN IF NOT EXISTS decision_reason TEXT DEFAULT '';

ALTER TABLE scheduled_test_plans
    ALTER COLUMN auto_schedulable_control SET DEFAULT false;

ALTER TABLE scheduled_test_plans
    ALTER COLUMN first_token_timeout_ms SET DEFAULT 0;

ALTER TABLE scheduled_test_results
    ALTER COLUMN decision SET DEFAULT 'no_action';

ALTER TABLE scheduled_test_results
    ALTER COLUMN decision_reason SET DEFAULT '';
