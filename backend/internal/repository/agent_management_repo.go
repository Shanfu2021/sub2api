package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbagentgroupdelegation "github.com/Wei-Shaw/sub2api/ent/agentgroupdelegation"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type agentManagementRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewAgentManagementRepository(client *dbent.Client, sqlDB *sql.DB) service.AgentManagementRepository {
	return &agentManagementRepository{client: client, sql: sqlDB}
}

func (r *agentManagementRepository) GetRootAdmin(ctx context.Context) (*service.User, error) {
	admin, err := clientFromContext(ctx, r.client).User.Query().
		Where(
			dbuser.RoleEQ(service.RoleAdmin),
			dbuser.StatusEQ(service.StatusActive),
		).
		Order(dbent.Asc(dbuser.FieldID)).
		First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return userEntityToService(admin), nil
}

func (r *agentManagementRepository) CreateUser(ctx context.Context, user *service.User) error {
	return newUserRepositoryWithSQL(clientFromContext(ctx, r.client), r.sql).Create(ctx, user)
}

func (r *agentManagementRepository) ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return r.ListDirectChildrenWithSearch(ctx, parentID, roles, params, "")
}

func (r *agentManagementRepository) ListDirectChildrenWithSearch(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]service.User, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	q := client.User.Query().
		Where(dbuser.ParentUserIDEQ(parentID))
	if len(roles) > 0 {
		q = q.Where(dbuser.RoleIn(roles...))
	}
	search = strings.TrimSpace(search)
	if search != "" {
		q = q.Where(dbuser.Or(
			dbuser.EmailContainsFold(search),
			dbuser.UsernameContainsFold(search),
		))
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	children, err := q.
		Order(dbent.Desc(dbuser.FieldID)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := make([]service.User, 0, len(children))
	for i := range children {
		out = append(out, *userEntityToService(children[i]))
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *agentManagementRepository) SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error) {
	q := clientFromContext(ctx, r.client).User.Query().
		Where(dbuser.ParentUserIDEQ(parentID))
	if excludeChildID != nil {
		q = q.Where(dbuser.IDNEQ(*excludeChildID))
	}

	children, err := q.All(ctx)
	if err != nil {
		return 0, 0, err
	}
	for _, child := range children {
		concurrency += child.AllocatedConcurrency
		rpm += child.AllocatedRpm
	}
	return concurrency, rpm, nil
}

func (r *agentManagementRepository) GetAgentProfile(ctx context.Context, userID int64) (*service.AgentProfile, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `
SELECT user_id, pool_concurrency, pool_rpm, invite_default_concurrency, invite_default_rpm
FROM agent_profiles
WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return nil, rows.Err()
	}
	var profile service.AgentProfile
	if err := rows.Scan(&profile.UserID, &profile.PoolConcurrency, &profile.PoolRPM, &profile.InviteDefaultConcurrency, &profile.InviteDefaultRPM); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *agentManagementRepository) UpsertAgentProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO agent_profiles (user_id, pool_concurrency, pool_rpm, created_at, updated_at)
VALUES ($1, CASE WHEN $2 < 0 THEN 0 ELSE $2 END, CASE WHEN $3 < 0 THEN 0 ELSE $3 END, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    pool_concurrency = EXCLUDED.pool_concurrency,
    pool_rpm = EXCLUDED.pool_rpm,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL`,
		userID,
		poolConcurrency,
		poolRPM,
	)
	return err
}

func (r *agentManagementRepository) GetEnterpriseProfile(ctx context.Context, userID int64) (*service.EnterpriseProfile, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	var profile service.EnterpriseProfile
	err := scanSingleRow(ctx, exec, `
SELECT user_id, pool_concurrency, pool_rpm
FROM enterprise_profiles
WHERE user_id = $1 AND deleted_at IS NULL`,
		[]any{userID},
		&profile.UserID,
		&profile.PoolConcurrency,
		&profile.PoolRPM,
	)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return &profile, nil
}

func (r *agentManagementRepository) UpsertEnterpriseProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO enterprise_profiles (user_id, pool_concurrency, pool_rpm, created_at, updated_at)
VALUES ($1, CASE WHEN $2 < 0 THEN 0 ELSE $2 END, CASE WHEN $3 < 0 THEN 0 ELSE $3 END, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    pool_concurrency = EXCLUDED.pool_concurrency,
    pool_rpm = EXCLUDED.pool_rpm,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL`,
		userID,
		poolConcurrency,
		poolRPM,
	)
	return err
}

func (r *agentManagementRepository) GetEnterpriseEmployeeQuotaUsage(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64) (service.QuotaUsageSummary, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return service.QuotaUsageSummary{}, errors.New("sql executor is not configured")
	}
	var exclude any
	if excludeEmployeeID != nil {
		exclude = *excludeEmployeeID
	}
	var usage service.QuotaUsageSummary
	err := scanSingleRow(ctx, exec, `
SELECT
  COALESCE(SUM(CASE WHEN concurrency < 0 THEN 0 ELSE concurrency END), 0) AS concurrency,
  COALESCE(SUM(CASE WHEN rpm_limit <= 0 THEN 0 ELSE rpm_limit END), 0) AS rpm,
  false AS unlimited_concurrency,
  COALESCE(BOOL_OR(rpm_limit = 0), false) AS unlimited_rpm
FROM users
WHERE parent_user_id = $1
  AND role = $2
  AND deleted_at IS NULL
  AND ($3::bigint IS NULL OR id <> $3::bigint)`,
		[]any{enterpriseID, service.RoleEmployee, exclude},
		&usage.Concurrency,
		&usage.RPM,
		&usage.UnlimitedConcurrency,
		&usage.UnlimitedRPM,
	)
	if err != nil {
		return service.QuotaUsageSummary{}, err
	}
	return usage, nil
}

func (r *agentManagementRepository) UpdateAgentInviteDefaults(ctx context.Context, userID int64, inviteConcurrency int, inviteRPM int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO agent_profiles (user_id, pool_concurrency, pool_rpm, invite_default_concurrency, invite_default_rpm, created_at, updated_at)
VALUES ($1, 0, 0, CASE WHEN $2 < 0 THEN 1 ELSE $2 END, CASE WHEN $3 < 0 THEN 1 ELSE $3 END, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    invite_default_concurrency = EXCLUDED.invite_default_concurrency,
    invite_default_rpm = EXCLUDED.invite_default_rpm,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL`,
		userID,
		inviteConcurrency,
		inviteRPM,
	)
	return err
}

func (r *agentManagementRepository) GetDirectChildQuotaUsage(ctx context.Context, parentID int64, excludeChildID *int64) (service.QuotaUsageSummary, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return service.QuotaUsageSummary{}, errors.New("sql executor is not configured")
	}
	var exclude any
	if excludeChildID != nil {
		exclude = *excludeChildID
	}
	rows, err := exec.QueryContext(ctx, `
WITH direct_children AS (
  SELECT
    CASE
      WHEN u.role IN ('agent_level1', 'agent_level2') THEN COALESCE(ap.pool_concurrency, 0)
      WHEN u.role = 'enterprise' THEN COALESCE(ep.pool_concurrency, 0)
      ELSE u.concurrency
    END AS concurrency,
    CASE
      WHEN u.role IN ('agent_level1', 'agent_level2') THEN COALESCE(ap.pool_rpm, 0)
      WHEN u.role = 'enterprise' THEN COALESCE(ep.pool_rpm, 0)
      ELSE u.rpm_limit
    END AS rpm
  FROM users u
  LEFT JOIN agent_profiles ap ON ap.user_id = u.id AND ap.deleted_at IS NULL
  LEFT JOIN enterprise_profiles ep ON ep.user_id = u.id AND ep.deleted_at IS NULL
  WHERE u.parent_user_id = $1
    AND u.deleted_at IS NULL
    AND ($2::bigint IS NULL OR u.id <> $2::bigint)
)
SELECT
  COALESCE(SUM(CASE WHEN concurrency < 0 THEN 0 ELSE concurrency END), 0) AS concurrency,
  COALESCE(SUM(CASE WHEN rpm <= 0 THEN 0 ELSE rpm END), 0) AS rpm,
  false AS unlimited_concurrency,
  COALESCE(BOOL_OR(rpm = 0), false) AS unlimited_rpm
FROM direct_children`,
		parentID,
		exclude,
	)
	if err != nil {
		return service.QuotaUsageSummary{}, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return service.QuotaUsageSummary{}, rows.Err()
	}
	var usage service.QuotaUsageSummary
	if err := rows.Scan(&usage.Concurrency, &usage.RPM, &usage.UnlimitedConcurrency, &usage.UnlimitedRPM); err != nil {
		return service.QuotaUsageSummary{}, err
	}
	return usage, rows.Err()
}

func (r *agentManagementRepository) SetEffectiveQuota(ctx context.Context, userID int64, concurrency int, rpm int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	res, err := exec.ExecContext(ctx, `
UPDATE users
SET concurrency = $2,
    rpm_limit = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL`,
		userID,
		concurrency,
		rpm,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return service.ErrUserNotFound
	}
	return nil
}

func (r *agentManagementRepository) SetParent(ctx context.Context, userID int64, parentID *int64) error {
	update := clientFromContext(ctx, r.client).User.UpdateOneID(userID)
	if parentID == nil {
		update = update.ClearParentUserID()
	} else {
		update = update.SetParentUserID(*parentID)
	}
	_, err := update.Save(ctx)
	return translatePersistenceError(err, service.ErrUserNotFound, nil)
}

func (r *agentManagementRepository) SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error {
	update := clientFromContext(ctx, r.client).User.UpdateOneID(userID).
		SetRole(role)
	if parentID == nil {
		update = update.ClearParentUserID()
	} else {
		update = update.SetParentUserID(*parentID)
	}
	_, err := update.Save(ctx)
	return translatePersistenceError(err, service.ErrUserNotFound, nil)
}

func (r *agentManagementRepository) SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error {
	if concurrency < 0 {
		concurrency = 0
	}
	if rpm < 0 {
		rpm = 0
	}
	_, err := clientFromContext(ctx, r.client).User.UpdateOneID(userID).
		SetAllocatedConcurrency(concurrency).
		SetAllocatedRpm(rpm).
		SetConcurrency(concurrency).
		SetRpmLimit(rpm).
		Save(ctx)
	return translatePersistenceError(err, service.ErrUserNotFound, nil)
}

func (r *agentManagementRepository) DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	txClient := tx.Client()
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}

	agent, err := txClient.User.Query().
		Where(
			dbuser.IDEQ(agentID),
			dbuser.RoleEQ(service.RoleAgentLevel1),
		).
		Only(txCtx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	if _, err := txClient.User.Update().
		Where(
			dbuser.ParentUserIDEQ(agent.ID),
			dbuser.RoleIn(service.RoleUser, service.RoleEnterprise),
		).
		SetParentUserID(rootAdminID).
		Save(txCtx); err != nil {
		return err
	}

	if err := r.rehomeManagedGroupDelegations(txCtx, exec, agent.ID, rootAdminID); err != nil {
		return err
	}

	if _, err := txClient.User.Update().
		Where(
			dbuser.ParentUserIDEQ(agent.ID),
			dbuser.RoleEQ(service.RoleAgentLevel2),
		).
		SetRole(service.RoleAgentLevel1).
		SetParentUserID(rootAdminID).
		Save(txCtx); err != nil {
		return err
	}

	if err := r.deleteAgentAccountRows(txCtx, exec, agent.ID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(txCtx, `UPDATE usage_cleanup_tasks SET created_by = $2 WHERE created_by = $1`, agent.ID, rootAdminID); err != nil {
		return err
	}
	deleted, err := txClient.User.Delete().
		Where(
			dbuser.IDEQ(agent.ID),
			dbuser.RoleEQ(service.RoleAgentLevel1),
		).
		Exec(mixins.SkipSoftDelete(txCtx))
	if err != nil {
		return err
	}
	if deleted == 0 {
		return service.ErrUserNotFound
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *agentManagementRepository) deleteAgentAccountRows(ctx context.Context, exec sqlQueryExecutor, userID int64) error {
	statements := []string{
		`UPDATE user_affiliates SET inviter_id = NULL, updated_at = NOW() WHERE inviter_id = $1`,
		`UPDATE redeem_codes SET used_by = NULL WHERE used_by = $1`,
		`UPDATE pending_auth_sessions SET target_user_id = NULL WHERE target_user_id = $1`,
		`UPDATE user_subscriptions SET assigned_by = NULL WHERE assigned_by = $1`,
		`DELETE FROM identity_adoption_decisions WHERE identity_id IN (SELECT id FROM auth_identities WHERE user_id = $1)`,
		`DELETE FROM auth_identity_channels WHERE identity_id IN (SELECT id FROM auth_identities WHERE user_id = $1)`,
		`DELETE FROM auth_identities WHERE user_id = $1`,
		`DELETE FROM agent_profiles WHERE user_id = $1`,
		`DELETE FROM agent_invite_group_defaults WHERE agent_user_id = $1`,
		`DELETE FROM enterprise_profiles WHERE user_id = $1`,
		`DELETE FROM enterprise_employee_balance_logs WHERE enterprise_user_id = $1 OR employee_user_id = $1 OR operator_user_id = $1`,
		`DELETE FROM user_allowed_groups WHERE user_id = $1`,
		`DELETE FROM user_group_rate_multipliers WHERE user_id = $1`,
		`DELETE FROM user_attribute_values WHERE user_id = $1`,
		`DELETE FROM user_platform_quotas WHERE user_id = $1`,
		`DELETE FROM announcement_reads WHERE user_id = $1`,
		`DELETE FROM promo_code_usages WHERE user_id = $1`,
		`DELETE FROM user_affiliate_ledger WHERE user_id = $1`,
		`UPDATE user_affiliate_ledger SET source_user_id = NULL WHERE source_user_id = $1`,
		`UPDATE user_affiliate_ledger SET source_order_id = NULL WHERE source_order_id IN (SELECT id FROM payment_orders WHERE user_id = $1)`,
		`DELETE FROM user_affiliates WHERE user_id = $1`,
		`DELETE FROM billing_usage_entries WHERE user_id = $1 OR api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`,
		`DELETE FROM usage_billing_dedup_archive WHERE api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`,
		`DELETE FROM usage_billing_dedup WHERE api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`,
		`DELETE FROM usage_logs WHERE user_id = $1 OR api_key_id IN (SELECT id FROM api_keys WHERE user_id = $1)`,
		`DELETE FROM usage_dashboard_hourly_users WHERE user_id = $1`,
		`DELETE FROM usage_dashboard_daily_users WHERE user_id = $1`,
		`DELETE FROM api_keys WHERE user_id = $1`,
		`DELETE FROM user_subscriptions WHERE user_id = $1`,
		`DELETE FROM payment_orders WHERE user_id = $1`,
	}
	for _, statement := range statements {
		if _, err := exec.ExecContext(ctx, statement, userID); err != nil {
			return err
		}
	}
	return nil
}

func (r *agentManagementRepository) DeleteAgentForAdminUserDeletion(ctx context.Context, user *service.User) ([]int64, error) {
	if user == nil || (user.Role != service.RoleAgentLevel1 && user.Role != service.RoleAgentLevel2) {
		return nil, nil
	}
	rootAdmin, err := r.GetRootAdmin(ctx)
	if err != nil {
		return nil, err
	}

	client := clientFromContext(ctx, r.client)
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	affected := map[int64]struct{}{user.ID: {}}
	if user.ParentUserID != nil {
		affected[*user.ParentUserID] = struct{}{}
	}

	childRows, err := client.User.Query().
		Where(dbuser.ParentUserIDEQ(user.ID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, child := range childRows {
		affected[child.ID] = struct{}{}
	}

	if _, err := client.User.Update().
		Where(
			dbuser.ParentUserIDEQ(user.ID),
			dbuser.RoleIn(service.RoleUser, service.RoleEnterprise),
		).
		SetParentUserID(rootAdmin.ID).
		Save(ctx); err != nil {
		return nil, err
	}
	if err := r.rehomeManagedGroupDelegations(ctx, exec, user.ID, rootAdmin.ID); err != nil {
		return nil, err
	}
	if _, err := client.User.Update().
		Where(
			dbuser.ParentUserIDEQ(user.ID),
			dbuser.RoleEQ(service.RoleAgentLevel2),
		).
		SetRole(service.RoleAgentLevel1).
		SetParentUserID(rootAdmin.ID).
		Save(ctx); err != nil {
		return nil, err
	}
	if _, err := client.User.Update().
		Where(
			dbuser.ParentUserIDEQ(user.ID),
			dbuser.RoleEQ(service.RoleAgentLevel1),
		).
		SetParentUserID(rootAdmin.ID).
		Save(ctx); err != nil {
		return nil, err
	}

	if err := r.deleteAgentAccountRows(ctx, exec, user.ID); err != nil {
		return nil, err
	}
	if _, err := exec.ExecContext(ctx, `UPDATE usage_cleanup_tasks SET created_by = $2 WHERE created_by = $1`, user.ID, rootAdmin.ID); err != nil {
		return nil, err
	}
	deleted, err := client.User.Delete().
		Where(
			dbuser.IDEQ(user.ID),
			dbuser.RoleIn(service.RoleAgentLevel1, service.RoleAgentLevel2),
		).
		Exec(mixins.SkipSoftDelete(ctx))
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	if deleted == 0 {
		return nil, service.ErrUserNotFound
	}

	if user.ParentUserID != nil {
		if parent, err := client.User.Get(ctx, *user.ParentUserID); err == nil && parent.Role != service.RoleAdmin && isAgentRole(parent.Role) {
			totalConcurrency := parent.Concurrency
			totalRPM := parent.RpmLimit
			if profile, err := r.GetAgentProfile(ctx, parent.ID); err == nil && profile != nil {
				totalConcurrency = profile.PoolConcurrency
				totalRPM = profile.PoolRPM
			}
			usage, err := r.GetDirectChildQuotaUsage(ctx, parent.ID, nil)
			if err != nil {
				return nil, err
			}
			if err := r.SetEffectiveQuota(ctx, parent.ID, repoConcurrencyRemainingForStorage(totalConcurrency, usage.Concurrency), repoRPMRemainingForStorage(totalRPM, usage.RPM, usage.UnlimitedRPM)); err != nil {
				return nil, err
			}
		} else if err != nil && !dbent.IsNotFound(err) {
			return nil, err
		}
	}

	return int64Keys(affected), nil
}

func (r *agentManagementRepository) rehomeManagedGroupDelegations(ctx context.Context, exec sqlQueryExecutor, oldManagerID int64, newManagerID int64) error {
	_, err := exec.ExecContext(ctx, `
WITH source AS (
  SELECT child_user_id, group_id, rate_multiplier, can_delegate
  FROM agent_group_delegations
  WHERE manager_user_id = $1
    AND deleted_at IS NULL
),
updated AS (
  UPDATE agent_group_delegations target
  SET rate_multiplier = source.rate_multiplier,
      can_delegate = source.can_delegate,
      updated_at = CURRENT_TIMESTAMP
  FROM source
  WHERE target.manager_user_id = $2
    AND target.child_user_id = source.child_user_id
    AND target.group_id = source.group_id
    AND target.deleted_at IS NULL
  RETURNING target.child_user_id, target.group_id
),
inserted AS (
  INSERT INTO agent_group_delegations (manager_user_id, child_user_id, group_id, rate_multiplier, can_delegate, created_at, updated_at)
  SELECT $2, source.child_user_id, source.group_id, source.rate_multiplier, source.can_delegate, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
  FROM source
  WHERE NOT EXISTS (
    SELECT 1
    FROM updated
    WHERE updated.child_user_id = source.child_user_id
      AND updated.group_id = source.group_id
  )
  RETURNING child_user_id, group_id
)
DELETE FROM agent_group_delegations
WHERE manager_user_id = $1
  AND deleted_at IS NULL`,
		oldManagerID,
		newManagerID,
	)
	return err
}

func (r *agentManagementRepository) RehomeChildGroupDelegations(ctx context.Context, oldManagerID int64, newManagerID int64, childID int64) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
WITH source AS (
  SELECT group_id, rate_multiplier, can_delegate
  FROM agent_group_delegations
  WHERE manager_user_id = $1
    AND child_user_id = $3
    AND deleted_at IS NULL
),
updated AS (
  UPDATE agent_group_delegations target
  SET rate_multiplier = source.rate_multiplier,
      can_delegate = source.can_delegate,
      updated_at = CURRENT_TIMESTAMP
  FROM source
  WHERE target.manager_user_id = $2
    AND target.child_user_id = $3
    AND target.group_id = source.group_id
    AND target.deleted_at IS NULL
  RETURNING target.group_id
),
inserted AS (
  INSERT INTO agent_group_delegations (manager_user_id, child_user_id, group_id, rate_multiplier, can_delegate, created_at, updated_at)
  SELECT $2, $3, source.group_id, source.rate_multiplier, source.can_delegate, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
  FROM source
  WHERE NOT EXISTS (
    SELECT 1
    FROM updated
    WHERE updated.group_id = source.group_id
  )
  RETURNING group_id
)
DELETE FROM agent_group_delegations
WHERE manager_user_id = $1
  AND child_user_id = $3
  AND deleted_at IS NULL`,
		oldManagerID,
		newManagerID,
		childID,
	)
	return err
}

func (r *agentManagementRepository) RecalculateAgentQuota(ctx context.Context, agentID int64) error {
	client := clientFromContext(ctx, r.client)
	agent, err := client.User.Get(ctx, agentID)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil
		}
		return err
	}
	if agent.Role == service.RoleAdmin || !isAgentRole(agent.Role) {
		return nil
	}
	totalConcurrency := agent.Concurrency
	totalRPM := agent.RpmLimit
	if profile, err := r.GetAgentProfile(ctx, agent.ID); err == nil && profile != nil {
		totalConcurrency = profile.PoolConcurrency
		totalRPM = profile.PoolRPM
	} else if err != nil {
		return err
	}
	usage, err := r.GetDirectChildQuotaUsage(ctx, agent.ID, nil)
	if err != nil {
		return err
	}
	return r.SetEffectiveQuota(ctx, agent.ID, repoConcurrencyRemainingForStorage(totalConcurrency, usage.Concurrency), repoRPMRemainingForStorage(totalRPM, usage.RPM, usage.UnlimitedRPM))
}

func (r *agentManagementRepository) ListGroupDelegationsForChild(ctx context.Context, childID int64) ([]service.AgentGroupDelegation, error) {
	rows, err := clientFromContext(ctx, r.client).AgentGroupDelegation.Query().
		Where(dbagentgroupdelegation.ChildUserIDEQ(childID)).
		WithGroup().
		Order(dbent.Asc(dbagentgroupdelegation.FieldGroupID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.AgentGroupDelegation, 0, len(rows))
	for i := range rows {
		out = append(out, *agentGroupDelegationEntityToService(rows[i]))
	}
	return out, nil
}

func (r *agentManagementRepository) GetGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) (*service.AgentGroupDelegation, error) {
	row, err := clientFromContext(ctx, r.client).AgentGroupDelegation.Query().
		Where(
			dbagentgroupdelegation.ManagerUserIDEQ(managerID),
			dbagentgroupdelegation.ChildUserIDEQ(childID),
			dbagentgroupdelegation.GroupIDEQ(groupID),
			dbagentgroupdelegation.DeletedAtIsNil(),
		).
		WithGroup().
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return agentGroupDelegationEntityToService(row), nil
}

func (r *agentManagementRepository) UpsertGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error {
	client := clientFromContext(ctx, r.client)
	updated, err := client.AgentGroupDelegation.Update().
		Where(
			dbagentgroupdelegation.ManagerUserIDEQ(managerID),
			dbagentgroupdelegation.ChildUserIDEQ(childID),
			dbagentgroupdelegation.GroupIDEQ(groupID),
			dbagentgroupdelegation.DeletedAtIsNil(),
		).
		SetRateMultiplier(rateMultiplier).
		SetCanDelegate(canDelegate).
		Save(ctx)
	if err != nil {
		return err
	}
	if updated > 0 {
		return nil
	}
	return client.AgentGroupDelegation.Create().
		SetManagerUserID(managerID).
		SetChildUserID(childID).
		SetGroupID(groupID).
		SetRateMultiplier(rateMultiplier).
		SetCanDelegate(canDelegate).
		Exec(ctx)
}

func (r *agentManagementRepository) DeleteGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) error {
	_, err := clientFromContext(ctx, r.client).AgentGroupDelegation.Delete().
		Where(
			dbagentgroupdelegation.ManagerUserIDEQ(managerID),
			dbagentgroupdelegation.ChildUserIDEQ(childID),
			dbagentgroupdelegation.GroupIDEQ(groupID),
			dbagentgroupdelegation.DeletedAtIsNil(),
		).
		Exec(ctx)
	return err
}

func (r *agentManagementRepository) ListInviteGroupDefaults(ctx context.Context, agentID int64) ([]service.AgentInviteGroupDefault, error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `
SELECT id, agent_user_id, group_id, rate_multiplier
FROM agent_invite_group_defaults
WHERE agent_user_id = $1 AND deleted_at IS NULL
ORDER BY group_id`,
		agentID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.AgentInviteGroupDefault, 0)
	for rows.Next() {
		var item service.AgentInviteGroupDefault
		if err := rows.Scan(&item.ID, &item.AgentUserID, &item.GroupID, &item.RateMultiplier); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *agentManagementRepository) UpsertInviteGroupDefault(ctx context.Context, agentID int64, groupID int64, rateMultiplier float64) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	updated, err := exec.ExecContext(ctx, `
UPDATE agent_invite_group_defaults
SET rate_multiplier = $3,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL
WHERE agent_user_id = $1
  AND group_id = $2
  AND deleted_at IS NULL`,
		agentID,
		groupID,
		rateMultiplier,
	)
	if err != nil {
		return err
	}
	if affected, _ := updated.RowsAffected(); affected > 0 {
		return nil
	}
	_, err = exec.ExecContext(ctx, `
INSERT INTO agent_invite_group_defaults (agent_user_id, group_id, rate_multiplier, created_at, updated_at)
VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		agentID,
		groupID,
		rateMultiplier,
	)
	return err
}

func (r *agentManagementRepository) DeleteInviteGroupDefault(ctx context.Context, agentID int64, groupID int64) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	_, err := exec.ExecContext(ctx, `
DELETE FROM agent_invite_group_defaults
WHERE agent_user_id = $1 AND group_id = $2 AND deleted_at IS NULL`,
		agentID,
		groupID,
	)
	return err
}

func agentGroupDelegationEntityToService(m *dbent.AgentGroupDelegation) *service.AgentGroupDelegation {
	if m == nil {
		return nil
	}
	out := &service.AgentGroupDelegation{
		ID:             m.ID,
		ManagerUserID:  m.ManagerUserID,
		ChildUserID:    m.ChildUserID,
		GroupID:        m.GroupID,
		RateMultiplier: m.RateMultiplier,
		CanDelegate:    m.CanDelegate,
	}
	if m.Edges.Group != nil {
		out.Group = groupEntityToService(m.Edges.Group)
	}
	return out
}

func isAgentRole(role string) bool {
	return role == service.RoleAgentLevel1 || role == service.RoleAgentLevel2
}

func repoQuotaRemaining(total int, allocated int, allocatedUnlimited bool) int {
	if total == 0 {
		return 0
	}
	if allocatedUnlimited {
		return 0
	}
	remaining := total - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

func repoConcurrencyRemaining(total int, allocated int) int {
	remaining := total - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

func repoConcurrencyRemainingForStorage(total int, allocated int) int {
	remaining := repoConcurrencyRemaining(total, allocated)
	if remaining == 0 {
		return -1
	}
	return remaining
}

func repoRPMRemainingForStorage(total int, allocated int, allocatedUnlimited bool) int {
	remaining := repoQuotaRemaining(total, allocated, allocatedUnlimited)
	if total > 0 && remaining == 0 {
		return -1
	}
	return remaining
}

func int64Keys(in map[int64]struct{}) []int64 {
	out := make([]int64, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	return out
}
