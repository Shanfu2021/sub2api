package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbagentgroupdelegation "github.com/Wei-Shaw/sub2api/ent/agentgroupdelegation"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	enterpriseBalanceLogReasonCreate     = "create_employee"
	enterpriseBalanceLogReasonUpdate     = "update_employee_allocation"
	enterpriseBalanceLogReasonDelete     = "delete_employee"
	enterpriseBalanceLogReasonInitialize = "initialize_employee_balances"
)

type enterpriseManagementRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewEnterpriseManagementRepository(client *dbent.Client, sqlDB *sql.DB) service.EnterpriseManagementRepository {
	return &enterpriseManagementRepository{client: client, sql: sqlDB}
}

func (r *enterpriseManagementRepository) ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]service.User, *pagination.PaginationResult, error) {
	return r.ListDirectChildrenWithSearch(ctx, parentID, roles, params, "")
}

func (r *enterpriseManagementRepository) ListDirectChildrenWithSearch(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams, search string) ([]service.User, *pagination.PaginationResult, error) {
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

func (r *enterpriseManagementRepository) GetEnterpriseProfile(ctx context.Context, userID int64) (*service.EnterpriseProfile, error) {
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

func (r *enterpriseManagementRepository) UpsertEnterpriseProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error {
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

func (r *enterpriseManagementRepository) GetEnterpriseEmployeeQuotaUsage(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64) (service.QuotaUsageSummary, error) {
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

func (r *enterpriseManagementRepository) RecalculateEnterpriseQuota(ctx context.Context, enterpriseID int64) error {
	return r.recalculateEnterpriseQuota(ctx, enterpriseID)
}

func (r *enterpriseManagementRepository) CreateEmployee(ctx context.Context, enterpriseID int64, operatorID int64, userIn *service.User) error {
	if userIn == nil {
		return nil
	}
	target := service.EmployeeAllocationUpdate{
		Balance:     userIn.Balance,
		Concurrency: userIn.Concurrency,
		RPM:         userIn.RPMLimit,
	}
	if err := validateEnterpriseAllocationTarget(target); err != nil {
		return err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}

	enterprise, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID)
	if err != nil {
		return err
	}
	if err := r.ensureEnterpriseAllocationAvailable(txCtx, enterpriseID, nil, target.Concurrency, target.RPM); err != nil {
		return err
	}
	if target.Balance > enterprise.Balance {
		return service.ErrEnterpriseManagementBalanceExceeded
	}

	employeeBalance := target.Balance
	userIn.Role = service.RoleEmployee
	userIn.ParentUserID = &enterpriseID
	userIn.Balance = 0
	userIn.Concurrency = target.Concurrency
	userIn.AllocatedConcurrency = target.Concurrency
	userIn.RPMLimit = target.RPM
	userIn.AllocatedRPM = target.RPM
	if userIn.Status == "" {
		userIn.Status = service.StatusActive
	}
	if err := newUserRepositoryWithSQL(clientFromContext(txCtx, r.client), r.sql).Create(txCtx, userIn); err != nil {
		return err
	}

	enterpriseAfter := enterprise.Balance - employeeBalance
	if employeeBalance != 0 {
		if err := r.setUserBalance(txCtx, exec, enterpriseID, enterpriseAfter); err != nil {
			return err
		}
		if err := r.setUserBalance(txCtx, exec, userIn.ID, employeeBalance); err != nil {
			return err
		}
	}
	if err := r.insertBalanceLog(txCtx, exec, balanceLogInput{
		EnterpriseID:     enterpriseID,
		EmployeeID:       userIn.ID,
		OperatorID:       operatorID,
		Delta:            employeeBalance,
		EnterpriseBefore: enterprise.Balance,
		EnterpriseAfter:  enterpriseAfter,
		EmployeeBefore:   0,
		EmployeeAfter:    employeeBalance,
		Reason:           enterpriseBalanceLogReasonCreate,
	}); err != nil {
		return err
	}
	userIn.Balance = employeeBalance

	if err := r.recalculateEnterpriseQuota(txCtx, enterpriseID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *enterpriseManagementRepository) CreateEmployees(ctx context.Context, enterpriseID int64, operatorID int64, users []*service.User) error {
	users = compactEmployeeUsers(users)
	if len(users) == 0 {
		return nil
	}
	requestedConcurrency, requestedRPM, requestedUnlimitedRPM, err := validateEnterpriseEmployeeBatch(users)
	if err != nil {
		return err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}

	enterprise, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID)
	if err != nil {
		return err
	}
	effectiveRequestedRPM := requestedRPM
	if requestedUnlimitedRPM {
		effectiveRequestedRPM = 0
	}
	if err := r.ensureEnterpriseAllocationAvailable(txCtx, enterpriseID, nil, requestedConcurrency, effectiveRequestedRPM); err != nil {
		return err
	}

	userRepo := newUserRepositoryWithSQL(clientFromContext(txCtx, r.client), r.sql)
	for _, userIn := range users {
		target := service.EmployeeAllocationUpdate{
			Balance:     0,
			Concurrency: userIn.Concurrency,
			RPM:         userIn.RPMLimit,
		}
		if err := validateEnterpriseAllocationTarget(target); err != nil {
			return err
		}
		userIn.Role = service.RoleEmployee
		userIn.ParentUserID = &enterpriseID
		userIn.Balance = 0
		userIn.AllocatedConcurrency = target.Concurrency
		userIn.AllocatedRPM = target.RPM
		if userIn.Status == "" {
			userIn.Status = service.StatusActive
		}
		if err := userRepo.Create(txCtx, userIn); err != nil {
			return err
		}
		if err := r.insertBalanceLog(txCtx, exec, balanceLogInput{
			EnterpriseID:     enterpriseID,
			EmployeeID:       userIn.ID,
			OperatorID:       operatorID,
			Delta:            0,
			EnterpriseBefore: enterprise.Balance,
			EnterpriseAfter:  enterprise.Balance,
			EmployeeBefore:   0,
			EmployeeAfter:    0,
			Reason:           enterpriseBalanceLogReasonCreate,
		}); err != nil {
			return err
		}
	}

	if err := r.recalculateEnterpriseQuota(txCtx, enterpriseID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *enterpriseManagementRepository) UpdateEmployeeAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64, target service.EmployeeAllocationUpdate) (*service.EmployeeAllocationResult, error) {
	if err := validateEnterpriseAllocationTarget(target); err != nil {
		return nil, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	enterprise, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID)
	if err != nil {
		return nil, err
	}
	employee, err := r.lockEmployeeForUpdate(txCtx, exec, enterpriseID, employeeID)
	if err != nil {
		return nil, err
	}
	if err := r.ensureEnterpriseAllocationAvailable(txCtx, enterpriseID, &employeeID, target.Concurrency, target.RPM); err != nil {
		return nil, err
	}

	delta := target.Balance - employee.Balance
	enterpriseAfter := enterprise.Balance - delta
	if delta > 0 && enterprise.Balance < delta {
		return nil, service.ErrEnterpriseManagementBalanceExceeded
	}

	_, err = exec.ExecContext(txCtx, `
UPDATE users
SET balance = $2,
    concurrency = $3,
    allocated_concurrency = $3,
    rpm_limit = $4,
    allocated_rpm = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND role = $5
  AND parent_user_id = $6
  AND deleted_at IS NULL`,
		employeeID,
		target.Balance,
		target.Concurrency,
		target.RPM,
		service.RoleEmployee,
		enterpriseID,
	)
	if err != nil {
		return nil, err
	}
	if delta != 0 {
		if err := r.setUserBalance(txCtx, exec, enterpriseID, enterpriseAfter); err != nil {
			return nil, err
		}
	}
	if err := r.insertBalanceLog(txCtx, exec, balanceLogInput{
		EnterpriseID:     enterpriseID,
		EmployeeID:       employeeID,
		OperatorID:       operatorID,
		Delta:            delta,
		EnterpriseBefore: enterprise.Balance,
		EnterpriseAfter:  enterpriseAfter,
		EmployeeBefore:   employee.Balance,
		EmployeeAfter:    target.Balance,
		Reason:           enterpriseBalanceLogReasonUpdate,
	}); err != nil {
		return nil, err
	}

	if err := r.recalculateEnterpriseQuota(txCtx, enterpriseID); err != nil {
		return nil, err
	}
	usage, err := r.GetEnterpriseEmployeeQuotaUsage(txCtx, enterpriseID, nil)
	if err != nil {
		return nil, err
	}
	updated, err := clientFromContext(txCtx, r.client).User.Get(txCtx, employeeID)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	profile, err := r.GetEnterpriseProfile(txCtx, enterpriseID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	result := &service.EmployeeAllocationResult{
		User:               userEntityToService(updated),
		BalanceDelta:       delta,
		EnterpriseBalance:  enterpriseAfter,
		EmployeeBalance:    target.Balance,
		AffectedUserIDs:    uniqueInt64s([]int64{enterpriseID, employeeID}),
		QuotaUsage:         usage,
		RemainingQuotaUser: nil,
	}
	result.Allocation = enterpriseAllocationSummary(profile, usage)
	return result, nil
}

func (r *enterpriseManagementRepository) InitializeEmployeeBalances(ctx context.Context, enterpriseID int64, operatorID int64, targetBalance float64) (*service.EmployeeBalanceInitializationResult, error) {
	if targetBalance < 0 {
		return nil, service.ErrEnterpriseManagementInvalidAllocation
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	enterprise, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID)
	if err != nil {
		return nil, err
	}
	employees, err := r.lockEmployeesForBalanceInitialization(txCtx, exec, enterpriseID)
	if err != nil {
		return nil, err
	}

	var currentBalance float64
	for _, employee := range employees {
		currentBalance += employee.Balance
	}
	targetTotal := targetBalance * float64(len(employees))
	requiredBalance := targetTotal - currentBalance
	if requiredBalance > enterprise.Balance {
		return nil, service.ErrEnterpriseEmployeeBalanceInitExceeded.WithMetadata(map[string]string{
			"current_balance":  fmt.Sprintf("%.6f", enterprise.Balance),
			"required_balance": fmt.Sprintf("%.6f", requiredBalance),
		})
	}

	enterpriseAfter := enterprise.Balance - requiredBalance
	if requiredBalance != 0 {
		if err := r.setUserBalance(txCtx, exec, enterpriseID, enterpriseAfter); err != nil {
			return nil, err
		}
	}
	affected := []int64{enterpriseID}
	runningEnterpriseBalance := enterprise.Balance
	for _, employee := range employees {
		delta := targetBalance - employee.Balance
		nextEnterpriseBalance := runningEnterpriseBalance - delta
		if _, err := exec.ExecContext(txCtx, `
UPDATE users
SET balance = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND role = $3
  AND parent_user_id = $4
  AND deleted_at IS NULL`,
			employee.ID,
			targetBalance,
			service.RoleEmployee,
			enterpriseID,
		); err != nil {
			return nil, err
		}
		if err := r.insertBalanceLog(txCtx, exec, balanceLogInput{
			EnterpriseID:     enterpriseID,
			EmployeeID:       employee.ID,
			OperatorID:       operatorID,
			Delta:            delta,
			EnterpriseBefore: runningEnterpriseBalance,
			EnterpriseAfter:  nextEnterpriseBalance,
			EmployeeBefore:   employee.Balance,
			EmployeeAfter:    targetBalance,
			Reason:           enterpriseBalanceLogReasonInitialize,
		}); err != nil {
			return nil, err
		}
		runningEnterpriseBalance = nextEnterpriseBalance
		affected = append(affected, employee.ID)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &service.EmployeeBalanceInitializationResult{
		EmployeeCount:           len(employees),
		TargetBalance:           targetBalance,
		CurrentBalance:          currentBalance,
		RequiredBalance:         requiredBalance,
		EnterpriseBalanceBefore: enterprise.Balance,
		EnterpriseBalanceAfter:  enterpriseAfter,
		AffectedUserIDs:         uniqueInt64s(affected),
	}, nil
}

func (r *enterpriseManagementRepository) DeleteEmployeeAndReturnAllocation(ctx context.Context, enterpriseID int64, employeeID int64, operatorID int64) ([]int64, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}

	enterprise, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID)
	if err != nil {
		return nil, err
	}
	employee, err := r.lockEmployeeForUpdate(txCtx, exec, enterpriseID, employeeID)
	if err != nil {
		return nil, err
	}
	enterpriseAfter := enterprise.Balance + employee.Balance
	if employee.Balance != 0 {
		if err := r.setUserBalance(txCtx, exec, enterpriseID, enterpriseAfter); err != nil {
			return nil, err
		}
	}
	if err := r.insertBalanceLog(txCtx, exec, balanceLogInput{
		EnterpriseID:     enterpriseID,
		EmployeeID:       employeeID,
		OperatorID:       operatorID,
		Delta:            -employee.Balance,
		EnterpriseBefore: enterprise.Balance,
		EnterpriseAfter:  enterpriseAfter,
		EmployeeBefore:   employee.Balance,
		EmployeeAfter:    0,
		Reason:           enterpriseBalanceLogReasonDelete,
	}); err != nil {
		return nil, err
	}

	if err := r.deleteEmployeeRows(txCtx, exec, []int64{employeeID}); err != nil {
		return nil, err
	}
	if err := r.recalculateEnterpriseQuota(txCtx, enterpriseID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return uniqueInt64s([]int64{enterpriseID, employeeID}), nil
}

func (r *enterpriseManagementRepository) HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	if _, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID); err != nil {
		return nil, err
	}

	employeeIDs, err := r.enterpriseEmployeeIDs(txCtx, exec, enterpriseID)
	if err != nil {
		return nil, err
	}
	affected := append([]int64{enterpriseID}, employeeIDs...)
	if len(employeeIDs) > 0 {
		if err := r.deleteEmployeeRows(txCtx, exec, employeeIDs); err != nil {
			return nil, err
		}
	}
	if err := r.deleteEnterpriseRows(txCtx, exec, enterpriseID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return uniqueInt64s(affected), nil
}

func (r *enterpriseManagementRepository) CascadeEnterpriseStatus(ctx context.Context, enterpriseID int64, targetStatus string) ([]int64, error) {
	if targetStatus != service.StatusActive && targetStatus != service.StatusDisabled {
		return nil, service.ErrEnterpriseManagementInvalidAllocation
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	exec := txAwareSQLExecutor(txCtx, r.sql, r.client)
	if exec == nil {
		return nil, errors.New("sql executor is not configured")
	}
	if _, err := r.lockEnterpriseForUpdate(txCtx, exec, enterpriseID); err != nil {
		return nil, err
	}

	affected := []int64{enterpriseID}
	if targetStatus == service.StatusDisabled {
		marked, err := r.markActiveEmployeesDisabled(txCtx, exec, enterpriseID)
		if err != nil {
			return nil, err
		}
		affected = append(affected, marked...)
	} else {
		restored, err := r.restoreEnterpriseDisabledEmployees(txCtx, exec, enterpriseID)
		if err != nil {
			return nil, err
		}
		affected = append(affected, restored...)
	}

	res, err := exec.ExecContext(txCtx, `
UPDATE users
SET status = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND role = $3
  AND deleted_at IS NULL`,
		enterpriseID,
		targetStatus,
		service.RoleEnterprise,
	)
	if err != nil {
		return nil, err
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, service.ErrUserNotFound
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return uniqueInt64s(affected), nil
}

func (r *enterpriseManagementRepository) ListGroupDelegationsForChild(ctx context.Context, childID int64) ([]service.AgentGroupDelegation, error) {
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

func (r *enterpriseManagementRepository) GetGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) (*service.AgentGroupDelegation, error) {
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

func (r *enterpriseManagementRepository) UpsertGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64, rateMultiplier float64, canDelegate bool) error {
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

func (r *enterpriseManagementRepository) DeleteGroupDelegation(ctx context.Context, managerID int64, childID int64, groupID int64) error {
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

type enterpriseUserLock struct {
	ID      int64
	Role    string
	Balance float64
}

type enterpriseEmployeeBalanceLock struct {
	ID      int64
	Balance float64
}

func (r *enterpriseManagementRepository) lockEnterpriseForUpdate(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) (enterpriseUserLock, error) {
	var out enterpriseUserLock
	err := scanSingleRow(ctx, exec, `
SELECT id, role, balance::double precision
FROM users
WHERE id = $1
  AND role = $2
  AND deleted_at IS NULL
FOR UPDATE`,
		[]any{enterpriseID, service.RoleEnterprise},
		&out.ID,
		&out.Role,
		&out.Balance,
	)
	if err != nil {
		return enterpriseUserLock{}, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}
	return out, nil
}

func (r *enterpriseManagementRepository) lockEmployeeForUpdate(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64, employeeID int64) (enterpriseUserLock, error) {
	var out enterpriseUserLock
	err := scanSingleRow(ctx, exec, `
SELECT id, role, balance::double precision
FROM users
WHERE id = $1
  AND role = $2
  AND parent_user_id = $3
  AND deleted_at IS NULL
FOR UPDATE`,
		[]any{employeeID, service.RoleEmployee, enterpriseID},
		&out.ID,
		&out.Role,
		&out.Balance,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return enterpriseUserLock{}, service.ErrEnterpriseManagementNotEmployee
		}
		return enterpriseUserLock{}, err
	}
	return out, nil
}

func (r *enterpriseManagementRepository) lockEmployeesForBalanceInitialization(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) ([]enterpriseEmployeeBalanceLock, error) {
	rows, err := exec.QueryContext(ctx, `
SELECT id, balance::double precision
FROM users
WHERE parent_user_id = $1
  AND role = $2
  AND deleted_at IS NULL
ORDER BY id
FOR UPDATE`,
		enterpriseID,
		service.RoleEmployee,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]enterpriseEmployeeBalanceLock, 0)
	for rows.Next() {
		var employee enterpriseEmployeeBalanceLock
		if err := rows.Scan(&employee.ID, &employee.Balance); err != nil {
			return nil, err
		}
		out = append(out, employee)
	}
	return out, rows.Err()
}

func (r *enterpriseManagementRepository) ensureEnterpriseAllocationAvailable(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64, targetConcurrency int, targetRPM int) error {
	profile, err := r.GetEnterpriseProfile(ctx, enterpriseID)
	if err != nil {
		return err
	}
	usage, err := r.GetEnterpriseEmployeeQuotaUsage(ctx, enterpriseID, excludeEmployeeID)
	if err != nil {
		return err
	}
	if targetConcurrency < 1 || usage.Concurrency+targetConcurrency >= profile.PoolConcurrency {
		return service.ErrEnterpriseManagementAllocationExceeded
	}
	if profile.PoolRPM != 0 {
		if targetRPM == 0 || usage.UnlimitedRPM || usage.RPM+targetRPM >= profile.PoolRPM {
			return service.ErrEnterpriseManagementAllocationExceeded
		}
	}
	return nil
}

func (r *enterpriseManagementRepository) recalculateEnterpriseQuota(ctx context.Context, enterpriseID int64) error {
	profile, err := r.GetEnterpriseProfile(ctx, enterpriseID)
	if err != nil {
		return err
	}
	usage, err := r.GetEnterpriseEmployeeQuotaUsage(ctx, enterpriseID, nil)
	if err != nil {
		return err
	}
	return r.setEnterpriseEffectiveQuota(ctx, enterpriseID,
		enterpriseConcurrencyRemainingForStorage(profile.PoolConcurrency, usage.Concurrency),
		enterpriseRPMRemainingForStorage(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM),
	)
}

func (r *enterpriseManagementRepository) setEnterpriseEffectiveQuota(ctx context.Context, enterpriseID int64, concurrency int, rpm int) error {
	exec := txAwareSQLExecutor(ctx, r.sql, r.client)
	if exec == nil {
		return errors.New("sql executor is not configured")
	}
	res, err := exec.ExecContext(ctx, `
UPDATE users
SET concurrency = $2,
    rpm_limit = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND role = $4
  AND deleted_at IS NULL`,
		enterpriseID,
		concurrency,
		rpm,
		service.RoleEnterprise,
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

func (r *enterpriseManagementRepository) setUserBalance(ctx context.Context, exec sqlQueryExecutor, userID int64, balance float64) error {
	res, err := exec.ExecContext(ctx, `
UPDATE users
SET balance = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND deleted_at IS NULL`,
		userID,
		balance,
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

type balanceLogInput struct {
	EnterpriseID     int64
	EmployeeID       int64
	OperatorID       int64
	Delta            float64
	EnterpriseBefore float64
	EnterpriseAfter  float64
	EmployeeBefore   float64
	EmployeeAfter    float64
	Reason           string
}

func (r *enterpriseManagementRepository) insertBalanceLog(ctx context.Context, exec sqlQueryExecutor, in balanceLogInput) error {
	_, err := exec.ExecContext(ctx, `
INSERT INTO enterprise_employee_balance_logs (
    enterprise_user_id,
    employee_user_id,
    operator_user_id,
    delta,
    enterprise_balance_before,
    enterprise_balance_after,
    employee_balance_before,
    employee_balance_after,
    reason,
    created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)`,
		in.EnterpriseID,
		in.EmployeeID,
		in.OperatorID,
		in.Delta,
		in.EnterpriseBefore,
		in.EnterpriseAfter,
		in.EmployeeBefore,
		in.EmployeeAfter,
		in.Reason,
	)
	return err
}

func (r *enterpriseManagementRepository) enterpriseEmployeeIDs(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) ([]int64, error) {
	rows, err := exec.QueryContext(ctx, `
SELECT id
FROM users
WHERE parent_user_id = $1
  AND role = $2
ORDER BY id`,
		enterpriseID,
		service.RoleEmployee,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *enterpriseManagementRepository) deleteEmployeeRows(ctx context.Context, exec sqlQueryExecutor, employeeIDs []int64) error {
	employeeIDs = uniqueInt64s(employeeIDs)
	if len(employeeIDs) == 0 {
		return nil
	}
	args, placeholders := int64InClause(employeeIDs)
	if _, err := exec.ExecContext(ctx, `
DELETE FROM agent_group_delegations
WHERE manager_user_id IN (`+placeholders+`)
   OR child_user_id IN (`+placeholders+`)`,
		args...,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM user_allowed_groups
WHERE user_id IN (`+placeholders+`)`,
		args...,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM user_group_rate_multipliers
WHERE user_id IN (`+placeholders+`)`,
		args...,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM api_keys
WHERE user_id IN (`+placeholders+`)`,
		args...,
	); err != nil {
		return err
	}
	_, err := clientFromContext(ctx, r.client).User.Delete().
		Where(
			dbuser.IDIn(employeeIDs...),
			dbuser.RoleEQ(service.RoleEmployee),
		).
		Exec(mixins.SkipSoftDelete(ctx))
	return err
}

func (r *enterpriseManagementRepository) deleteEnterpriseRows(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) error {
	if _, err := exec.ExecContext(ctx, `
DELETE FROM agent_group_delegations
WHERE manager_user_id = $1 OR child_user_id = $1`,
		enterpriseID,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM user_allowed_groups
WHERE user_id = $1`,
		enterpriseID,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM user_group_rate_multipliers
WHERE user_id = $1`,
		enterpriseID,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
DELETE FROM api_keys
WHERE user_id = $1`,
		enterpriseID,
	); err != nil {
		return err
	}
	_, err := clientFromContext(ctx, r.client).User.Delete().
		Where(
			dbuser.IDEQ(enterpriseID),
			dbuser.RoleEQ(service.RoleEnterprise),
		).
		Exec(mixins.SkipSoftDelete(ctx))
	return err
}

func (r *enterpriseManagementRepository) markActiveEmployeesDisabled(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) ([]int64, error) {
	rows, err := exec.QueryContext(ctx, `
UPDATE users
SET status = $3,
    employee_disabled_by_enterprise = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE parent_user_id = $1
  AND role = $2
  AND status = $4
  AND deleted_at IS NULL
RETURNING id`,
		enterpriseID,
		service.RoleEmployee,
		service.StatusDisabled,
		service.StatusActive,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return scanReturnedIDs(rows)
}

func (r *enterpriseManagementRepository) restoreEnterpriseDisabledEmployees(ctx context.Context, exec sqlQueryExecutor, enterpriseID int64) ([]int64, error) {
	rows, err := exec.QueryContext(ctx, `
UPDATE users
SET status = $3,
    employee_disabled_by_enterprise = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE parent_user_id = $1
  AND role = $2
  AND employee_disabled_by_enterprise = TRUE
  AND updated_at <= (
      SELECT updated_at
      FROM users
      WHERE id = $1
        AND role = $4
        AND deleted_at IS NULL
  )
  AND deleted_at IS NULL
RETURNING id`,
		enterpriseID,
		service.RoleEmployee,
		service.StatusActive,
		service.RoleEnterprise,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	restored, err := scanReturnedIDs(rows)
	if err != nil {
		return nil, err
	}
	_, err = exec.ExecContext(ctx, `
UPDATE users
SET employee_disabled_by_enterprise = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE parent_user_id = $1
  AND role = $2
  AND employee_disabled_by_enterprise = TRUE
  AND deleted_at IS NULL`,
		enterpriseID,
		service.RoleEmployee,
	)
	if err != nil {
		return nil, err
	}
	return restored, nil
}

func validateEnterpriseAllocationTarget(target service.EmployeeAllocationUpdate) error {
	if target.Balance < 0 || target.Concurrency < 1 || target.RPM < 0 {
		return service.ErrEnterpriseManagementInvalidAllocation
	}
	return nil
}

func compactEmployeeUsers(users []*service.User) []*service.User {
	if len(users) == 0 {
		return nil
	}
	out := make([]*service.User, 0, len(users))
	for _, user := range users {
		if user == nil {
			continue
		}
		out = append(out, user)
	}
	return out
}

func validateEnterpriseEmployeeBatch(users []*service.User) (int, int, bool, error) {
	var concurrency int
	var rpm int
	var unlimitedRPM bool
	for _, user := range users {
		target := service.EmployeeAllocationUpdate{
			Balance:     0,
			Concurrency: user.Concurrency,
			RPM:         user.RPMLimit,
		}
		if err := validateEnterpriseAllocationTarget(target); err != nil {
			return 0, 0, false, err
		}
		concurrency += user.Concurrency
		if user.RPMLimit == 0 {
			unlimitedRPM = true
			continue
		}
		if !unlimitedRPM {
			rpm += user.RPMLimit
		}
	}
	if unlimitedRPM {
		return concurrency, 0, true, nil
	}
	return concurrency, rpm, false, nil
}

func enterpriseAllocationSummary(profile *service.EnterpriseProfile, usage service.QuotaUsageSummary) service.AllocationSummary {
	if profile == nil {
		return service.AllocationSummary{}
	}
	return service.AllocationSummary{
		TotalConcurrency:     profile.PoolConcurrency,
		AllocatedConcurrency: usage.Concurrency,
		RemainingConcurrency: repoConcurrencyRemaining(profile.PoolConcurrency, usage.Concurrency),
		TotalRPM:             profile.PoolRPM,
		AllocatedRPM:         usage.RPM,
		RemainingRPM:         repoQuotaRemaining(profile.PoolRPM, usage.RPM, usage.UnlimitedRPM),
		UnlimitedCapacity:    false,
		UnlimitedConcurrency: false,
		UnlimitedRPM:         profile.PoolRPM == 0,
	}
}

func enterpriseConcurrencyRemainingForStorage(total int, allocated int) int {
	remaining := repoConcurrencyRemaining(total, allocated)
	if remaining == 0 {
		return -1
	}
	return remaining
}

func enterpriseRPMRemainingForStorage(total int, allocated int, allocatedUnlimited bool) int {
	remaining := repoQuotaRemaining(total, allocated, allocatedUnlimited)
	if total > 0 && remaining == 0 {
		return -1
	}
	return remaining
}

func scanReturnedIDs(rows *sql.Rows) ([]int64, error) {
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func uniqueInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func int64InClause(ids []int64) ([]any, string) {
	args := make([]any, 0, len(ids))
	placeholders := ""
	for i, id := range ids {
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", i+1)
		args = append(args, id)
	}
	return args, placeholders
}
