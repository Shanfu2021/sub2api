package repository

import (
	"context"
	"database/sql"
	"errors"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbagentgroupdelegation "github.com/Wei-Shaw/sub2api/ent/agentgroupdelegation"
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
	client := clientFromContext(ctx, r.client)
	q := client.User.Query().
		Where(dbuser.ParentUserIDEQ(parentID))
	if len(roles) > 0 {
		q = q.Where(dbuser.RoleIn(roles...))
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
SELECT user_id, pool_concurrency, pool_rpm
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
	if err := rows.Scan(&profile.UserID, &profile.PoolConcurrency, &profile.PoolRPM); err != nil {
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

func (r *agentManagementRepository) SumDirectChildQuotaUsage(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error) {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return 0, 0, errors.New("sql executor is not configured")
	}
	var exclude any
	if excludeChildID != nil {
		exclude = *excludeChildID
	}
	rows, err := exec.QueryContext(ctx, `
SELECT
  COALESCE(SUM(
    CASE
      WHEN u.role IN ('agent_level1', 'agent_level2') THEN COALESCE(ap.pool_concurrency, 0)
      ELSE u.concurrency
    END
  ), 0) AS concurrency,
  COALESCE(SUM(
    CASE
      WHEN u.role IN ('agent_level1', 'agent_level2') THEN COALESCE(ap.pool_rpm, 0)
      ELSE u.rpm_limit
    END
  ), 0) AS rpm
FROM users u
LEFT JOIN agent_profiles ap ON ap.user_id = u.id AND ap.deleted_at IS NULL
WHERE u.parent_user_id = $1
  AND u.deleted_at IS NULL
  AND ($2::bigint IS NULL OR u.id <> $2::bigint)`,
		parentID,
		exclude,
	)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		return 0, 0, rows.Err()
	}
	if err := rows.Scan(&concurrency, &rpm); err != nil {
		return 0, 0, err
	}
	return concurrency, rpm, rows.Err()
}

func (r *agentManagementRepository) SetEffectiveQuota(ctx context.Context, userID int64, concurrency int, rpm int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	res, err := exec.ExecContext(ctx, `
UPDATE users
SET concurrency = CASE WHEN $2 < 0 THEN 0 ELSE $2 END,
    rpm_limit = CASE WHEN $3 < 0 THEN 0 ELSE $3 END,
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

func (r *agentManagementRepository) DetachLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	txClient := tx.Client()

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

	if _, err := txClient.User.UpdateOneID(agent.ID).
		SetRole(service.RoleUser).
		SetParentUserID(rootAdminID).
		Save(txCtx); err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
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
