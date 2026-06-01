package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type agentManagementRepository struct {
	client *dbent.Client
}

func NewAgentManagementRepository(client *dbent.Client) service.AgentManagementRepository {
	return &agentManagementRepository{client: client}
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

	if _, err := txClient.User.Delete().
		Where(dbuser.IDEQ(agent.ID)).
		Exec(txCtx); err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	if _, err := txClient.User.Query().
		Where(dbuser.IDEQ(agent.ID)).
		Only(mixins.SkipSoftDelete(txCtx)); err != nil {
		return translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
