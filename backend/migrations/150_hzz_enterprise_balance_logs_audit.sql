ALTER TABLE enterprise_employee_balance_logs
    DROP CONSTRAINT IF EXISTS enterprise_employee_balance_logs_employee_user_id_fkey,
    DROP CONSTRAINT IF EXISTS enterprise_employee_balance_logs_enterprise_user_id_fkey,
    DROP CONSTRAINT IF EXISTS enterprise_employee_balance_logs_operator_user_id_fkey;
