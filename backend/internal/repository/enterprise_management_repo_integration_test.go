package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/suite"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const enterpriseManagementPostgresImage = "postgres:18.1-alpine3.23"

type EnterpriseManagementRepoSuite struct {
	suite.Suite

	ctx       context.Context
	db        *sql.DB
	client    *dbent.Client
	container *tcpostgres.PostgresContainer
	repo      service.EnterpriseManagementRepository
}

func TestEnterpriseManagementRepoSuite(t *testing.T) {
	suite.Run(t, new(EnterpriseManagementRepoSuite))
}

func (s *EnterpriseManagementRepoSuite) SetupSuite() {
	s.ctx = context.Background()
	if !enterpriseManagementDockerIsAvailable(s.ctx) {
		if os.Getenv("CI") != "" {
			s.T().Fatal("docker is not available for enterprise management integration suite")
		}
		s.T().Skip("docker is not available; skipping enterprise management integration suite")
	}

	container, err := tcpostgres.Run(
		s.ctx,
		enterpriseManagementPostgresImage,
		tcpostgres.WithDatabase("sub2api_enterprise_management_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	s.Require().NoError(err)
	s.container = container

	dsn, err := container.ConnectionString(s.ctx, "sslmode=disable", "TimeZone=UTC")
	s.Require().NoError(err)

	s.db, err = enterpriseManagementOpenSQLWithRetry(s.ctx, dsn, 30*time.Second)
	s.Require().NoError(err)
	s.Require().NoError(ApplyMigrations(s.ctx, s.db))

	drv := entsql.OpenDB(dialect.Postgres, s.db)
	s.client = dbent.NewClient(dbent.Driver(drv))
}

func (s *EnterpriseManagementRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.db != nil {
		_ = s.db.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(context.Background())
	}
}

func (s *EnterpriseManagementRepoSuite) SetupTest() {
	s.ctx = context.Background()
	s.repo = NewEnterpriseManagementRepository(s.client, s.db)
	s.resetData()
}

func (s *EnterpriseManagementRepoSuite) resetData() {
	_, err := s.db.ExecContext(s.ctx, `
TRUNCATE
	enterprise_employee_balance_logs,
	enterprise_profiles,
	agent_group_delegations,
	user_group_rate_multipliers,
	user_subscriptions,
	user_allowed_groups,
	api_keys,
	auth_identity_channels,
	auth_identities,
	users,
	groups
RESTART IDENTITY CASCADE`)
	s.Require().NoError(err)
}

func (s *EnterpriseManagementRepoSuite) TestCreateEmployeeMovesBalanceAndWritesLedger() {
	enterprise := s.mustCreateEnterprise("enterprise-create@test.local", 100, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 10, 100))

	employee := &service.User{
		Email:        "employee-create@test.local",
		Username:     "employee-create",
		PasswordHash: "test-password-hash",
		Balance:      25,
		Concurrency:  3,
		RPMLimit:     30,
		Status:       service.StatusActive,
	}

	s.Require().NoError(s.repo.CreateEmployee(s.ctx, enterprise.ID, enterprise.ID, employee))

	s.Require().Greater(employee.ID, int64(0))
	reloaded, err := s.client.User.Get(s.ctx, employee.ID)
	s.Require().NoError(err)
	s.Require().Equal(service.RoleEmployee, reloaded.Role)
	s.Require().NotNil(reloaded.ParentUserID)
	s.Require().Equal(enterprise.ID, *reloaded.ParentUserID)
	s.Require().InDelta(25, reloaded.Balance, 0.000001)
	s.Require().Equal(3, reloaded.Concurrency)
	s.Require().Equal(3, reloaded.AllocatedConcurrency)
	s.Require().Equal(30, reloaded.RpmLimit)
	s.Require().Equal(30, reloaded.AllocatedRpm)

	s.Require().InDelta(75, s.userBalance(enterprise.ID), 0.000001)
	log := s.latestBalanceLog(employee.ID)
	s.Require().Equal(enterprise.ID, log.enterpriseID)
	s.Require().Equal(enterprise.ID, log.operatorID)
	s.Require().Equal("create_employee", log.reason)
	s.Require().InDelta(25, log.delta, 0.000001)
	s.Require().InDelta(100, log.enterpriseBefore, 0.000001)
	s.Require().InDelta(75, log.enterpriseAfter, 0.000001)
	s.Require().InDelta(0, log.employeeBefore, 0.000001)
	s.Require().InDelta(25, log.employeeAfter, 0.000001)
}

func (s *EnterpriseManagementRepoSuite) TestCreateEmployeeRejectsBalanceAndQuotaExceeded() {
	enterprise := s.mustCreateEnterprise("enterprise-create-exceeded@test.local", 5, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 2, 20))

	tooExpensive := &service.User{
		Email:        "employee-too-expensive@test.local",
		PasswordHash: "test-password-hash",
		Balance:      10,
		Concurrency:  1,
		RPMLimit:     10,
		Status:       service.StatusActive,
	}
	err := s.repo.CreateEmployee(s.ctx, enterprise.ID, enterprise.ID, tooExpensive)
	s.Require().ErrorIs(err, service.ErrEnterpriseManagementBalanceExceeded)

	tooLarge := &service.User{
		Email:        "employee-too-large@test.local",
		PasswordHash: "test-password-hash",
		Balance:      1,
		Concurrency:  3,
		RPMLimit:     10,
		Status:       service.StatusActive,
	}
	err = s.repo.CreateEmployee(s.ctx, enterprise.ID, enterprise.ID, tooLarge)
	s.Require().ErrorIs(err, service.ErrEnterpriseManagementAllocationExceeded)

	s.Require().InDelta(5, s.userBalance(enterprise.ID), 0.000001)
	var employeeCount int
	s.Require().NoError(s.db.QueryRowContext(s.ctx, `
SELECT COUNT(*) FROM users WHERE parent_user_id = $1 AND role = $2`,
		enterprise.ID,
		service.RoleEmployee,
	).Scan(&employeeCount))
	s.Require().Equal(0, employeeCount)
}

func (s *EnterpriseManagementRepoSuite) TestUpdateEmployeeAllocationMovesOnlyBalanceDelta() {
	enterprise := s.mustCreateEnterprise("enterprise-update@test.local", 100, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 10, 100))
	employee := s.mustCreateEmployeeThroughRepo(enterprise.ID, "employee-update@test.local", 25, 3, 30)

	_, err := s.repo.UpdateEmployeeAllocation(s.ctx, enterprise.ID, employee.ID, enterprise.ID, service.EmployeeAllocationUpdate{
		Balance:     10,
		Concurrency: 3,
		RPM:         30,
	})
	s.Require().NoError(err)
	s.Require().InDelta(90, s.userBalance(enterprise.ID), 0.000001)
	s.Require().InDelta(10, s.userBalance(employee.ID), 0.000001)
	log := s.latestBalanceLog(employee.ID)
	s.Require().Equal("update_employee_allocation", log.reason)
	s.Require().InDelta(-15, log.delta, 0.000001)

	_, err = s.repo.UpdateEmployeeAllocation(s.ctx, enterprise.ID, employee.ID, enterprise.ID, service.EmployeeAllocationUpdate{
		Balance:     40,
		Concurrency: 4,
		RPM:         40,
	})
	s.Require().NoError(err)
	s.Require().InDelta(60, s.userBalance(enterprise.ID), 0.000001)
	s.Require().InDelta(40, s.userBalance(employee.ID), 0.000001)
	log = s.latestBalanceLog(employee.ID)
	s.Require().Equal("update_employee_allocation", log.reason)
	s.Require().InDelta(30, log.delta, 0.000001)

	reloaded, err := s.client.User.Get(s.ctx, employee.ID)
	s.Require().NoError(err)
	s.Require().Equal(4, reloaded.Concurrency)
	s.Require().Equal(4, reloaded.AllocatedConcurrency)
	s.Require().Equal(40, reloaded.RpmLimit)
	s.Require().Equal(40, reloaded.AllocatedRpm)
}

func (s *EnterpriseManagementRepoSuite) TestDeleteEmployeeReturnsBalanceAndHardDeletes() {
	enterprise := s.mustCreateEnterprise("enterprise-delete@test.local", 100, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 10, 100))
	employee := s.mustCreateEmployeeThroughRepo(enterprise.ID, "employee-delete@test.local", 25, 3, 30)
	groupID := s.mustCreateExclusiveGroup("exclusive-delete")

	_, err := s.db.ExecContext(s.ctx, `
INSERT INTO user_allowed_groups (user_id, group_id) VALUES ($1, $2)`,
		employee.ID,
		groupID,
	)
	s.Require().NoError(err)
	_, err = s.db.ExecContext(s.ctx, `
INSERT INTO agent_group_delegations (manager_user_id, child_user_id, group_id, rate_multiplier, can_delegate, created_at, updated_at)
VALUES ($1, $2, $3, 1.25, false, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		enterprise.ID,
		employee.ID,
		groupID,
	)
	s.Require().NoError(err)

	affected, err := s.repo.DeleteEmployeeAndReturnAllocation(s.ctx, enterprise.ID, employee.ID, enterprise.ID)
	s.Require().NoError(err)
	s.Require().Contains(affected, enterprise.ID)
	s.Require().Contains(affected, employee.ID)
	s.Require().InDelta(100, s.userBalance(enterprise.ID), 0.000001)

	exists, err := s.client.User.Query().
		Where(dbuser.IDEQ(employee.ID)).
		Exist(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	s.Require().False(exists)
	s.Require().Equal(0, s.countRows("user_allowed_groups", "user_id = $1", employee.ID))
	s.Require().Equal(0, s.countRows("agent_group_delegations", "child_user_id = $1", employee.ID))

	profile, err := s.repo.GetEnterpriseProfile(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().NotNil(profile)
	reloadedEnterprise, err := s.client.User.Get(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().Equal(profile.PoolConcurrency, reloadedEnterprise.Concurrency)
	s.Require().Equal(profile.PoolRPM, reloadedEnterprise.RpmLimit)
}

func (s *EnterpriseManagementRepoSuite) TestEnterpriseDisableEnableRestoresOnlyMarkedEmployees() {
	enterprise := s.mustCreateEnterprise("enterprise-status@test.local", 100, service.StatusActive)
	activeEmployee := s.mustCreateRepoEmployee(enterprise.ID, "employee-active@test.local", 5, 1, 10, service.StatusActive)
	disabledEmployee := s.mustCreateRepoEmployee(enterprise.ID, "employee-disabled@test.local", 5, 1, 10, service.StatusDisabled)

	affected, err := s.repo.CascadeEnterpriseStatus(s.ctx, enterprise.ID, service.StatusDisabled)
	s.Require().NoError(err)
	s.Require().Contains(affected, enterprise.ID)
	s.Require().Contains(affected, activeEmployee.ID)
	s.Require().NotContains(affected, disabledEmployee.ID)
	s.Require().Equal(service.StatusDisabled, s.userStatus(enterprise.ID))
	s.Require().Equal(service.StatusDisabled, s.userStatus(activeEmployee.ID))
	s.Require().Equal(service.StatusDisabled, s.userStatus(disabledEmployee.ID))
	s.Require().True(s.employeeDisabledMarker(activeEmployee.ID))
	s.Require().False(s.employeeDisabledMarker(disabledEmployee.ID))

	affected, err = s.repo.CascadeEnterpriseStatus(s.ctx, enterprise.ID, service.StatusActive)
	s.Require().NoError(err)
	s.Require().Contains(affected, enterprise.ID)
	s.Require().Contains(affected, activeEmployee.ID)
	s.Require().NotContains(affected, disabledEmployee.ID)
	s.Require().Equal(service.StatusActive, s.userStatus(enterprise.ID))
	s.Require().Equal(service.StatusActive, s.userStatus(activeEmployee.ID))
	s.Require().Equal(service.StatusDisabled, s.userStatus(disabledEmployee.ID))
	s.Require().False(s.employeeDisabledMarker(activeEmployee.ID))
	s.Require().False(s.employeeDisabledMarker(disabledEmployee.ID))
}

func (s *EnterpriseManagementRepoSuite) TestEnterpriseProfileUsageAndRecalculateQuota() {
	enterprise := s.mustCreateEnterprise("enterprise-quota@test.local", 0, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 10, 100))
	employeeA := s.mustCreateRepoEmployee(enterprise.ID, "employee-quota-a@test.local", 0, 3, 30, service.StatusActive)
	employeeB := s.mustCreateRepoEmployee(enterprise.ID, "employee-quota-b@test.local", 0, 4, 40, service.StatusActive)

	profile, err := s.repo.GetEnterpriseProfile(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().NotNil(profile)
	s.Require().Equal(10, profile.PoolConcurrency)
	s.Require().Equal(100, profile.PoolRPM)

	usage, err := s.repo.GetEnterpriseEmployeeQuotaUsage(s.ctx, enterprise.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(7, usage.Concurrency)
	s.Require().Equal(70, usage.RPM)
	s.Require().False(usage.UnlimitedConcurrency)
	s.Require().False(usage.UnlimitedRPM)

	usage, err = s.repo.GetEnterpriseEmployeeQuotaUsage(s.ctx, enterprise.ID, &employeeB.ID)
	s.Require().NoError(err)
	s.Require().Equal(3, usage.Concurrency)
	s.Require().Equal(30, usage.RPM)

	s.Require().NoError(s.repo.RecalculateEnterpriseQuota(s.ctx, enterprise.ID))
	reloaded, err := s.client.User.Get(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().Equal(3, reloaded.Concurrency)
	s.Require().Equal(30, reloaded.RpmLimit)

	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 0, 100))
	_, err = s.repo.UpdateEmployeeAllocation(s.ctx, enterprise.ID, employeeA.ID, enterprise.ID, service.EmployeeAllocationUpdate{
		Balance:     0,
		Concurrency: 0,
		RPM:         30,
	})
	s.Require().NoError(err)
	s.Require().NoError(s.repo.RecalculateEnterpriseQuota(s.ctx, enterprise.ID))
	reloaded, err = s.client.User.Get(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().Equal(0, reloaded.Concurrency)
	s.Require().Equal(30, reloaded.RpmLimit)
}

func (s *EnterpriseManagementRepoSuite) TestHardDeleteEnterpriseWithEmployees() {
	enterprise := s.mustCreateEnterprise("enterprise-hard-delete@test.local", 100, service.StatusActive)
	s.Require().NoError(s.repo.UpsertEnterpriseProfile(s.ctx, enterprise.ID, 10, 100))
	employeeA := s.mustCreateEmployeeThroughRepo(enterprise.ID, "employee-hard-delete-a@test.local", 25, 3, 30)
	employeeB := s.mustCreateEmployeeThroughRepo(enterprise.ID, "employee-hard-delete-b@test.local", 15, 2, 20)

	affected, err := s.repo.HardDeleteEnterpriseWithEmployees(s.ctx, enterprise.ID)
	s.Require().NoError(err)
	s.Require().Contains(affected, enterprise.ID)
	s.Require().Contains(affected, employeeA.ID)
	s.Require().Contains(affected, employeeB.ID)
	s.Require().False(s.userExistsIncludingDeleted(enterprise.ID))
	s.Require().False(s.userExistsIncludingDeleted(employeeA.ID))
	s.Require().False(s.userExistsIncludingDeleted(employeeB.ID))
}

func (s *EnterpriseManagementRepoSuite) mustCreateEnterprise(email string, balance float64, status string) *dbent.User {
	return s.mustCreateRepoUser(email, service.RoleEnterprise, nil, balance, 0, 0, status)
}

func (s *EnterpriseManagementRepoSuite) mustCreateRepoEmployee(enterpriseID int64, email string, balance float64, concurrency int, rpm int, status string) *dbent.User {
	return s.mustCreateRepoUser(email, service.RoleEmployee, &enterpriseID, balance, concurrency, rpm, status)
}

func (s *EnterpriseManagementRepoSuite) mustCreateRepoUser(email string, role string, parentID *int64, balance float64, concurrency int, rpm int, status string) *dbent.User {
	s.T().Helper()
	if status == "" {
		status = service.StatusActive
	}
	uniqueEmail := fmt.Sprintf("%s-%d", strings.TrimSuffix(email, "@test.local"), time.Now().UnixNano()) + "@test.local"
	create := s.client.User.Create().
		SetEmail(uniqueEmail).
		SetUsername(strings.TrimSuffix(email, "@test.local")).
		SetPasswordHash("test-password-hash").
		SetRole(role).
		SetStatus(status).
		SetBalance(balance).
		SetConcurrency(concurrency).
		SetRpmLimit(rpm).
		SetAllocatedConcurrency(concurrency).
		SetAllocatedRpm(rpm)
	if parentID != nil {
		create.SetParentUserID(*parentID)
	}
	created, err := create.Save(s.ctx)
	s.Require().NoError(err)
	return created
}

func (s *EnterpriseManagementRepoSuite) mustCreateEmployeeThroughRepo(enterpriseID int64, email string, balance float64, concurrency int, rpm int) *service.User {
	s.T().Helper()
	employee := &service.User{
		Email:        email,
		Username:     strings.TrimSuffix(email, "@test.local"),
		PasswordHash: "test-password-hash",
		Balance:      balance,
		Concurrency:  concurrency,
		RPMLimit:     rpm,
		Status:       service.StatusActive,
	}
	s.Require().NoError(s.repo.CreateEmployee(s.ctx, enterpriseID, enterpriseID, employee))
	return employee
}

func (s *EnterpriseManagementRepoSuite) mustCreateExclusiveGroup(name string) int64 {
	s.T().Helper()
	group, err := s.client.Group.Create().
		SetName(fmt.Sprintf("%s-%d", name, time.Now().UnixNano())).
		SetStatus(service.StatusActive).
		SetPlatform(service.PlatformAnthropic).
		SetRateMultiplier(1).
		SetIsExclusive(true).
		Save(s.ctx)
	s.Require().NoError(err)
	return group.ID
}

func (s *EnterpriseManagementRepoSuite) userBalance(userID int64) float64 {
	s.T().Helper()
	var balance float64
	s.Require().NoError(s.db.QueryRowContext(s.ctx, `
SELECT balance::double precision FROM users WHERE id = $1`,
		userID,
	).Scan(&balance))
	return balance
}

func (s *EnterpriseManagementRepoSuite) userStatus(userID int64) string {
	s.T().Helper()
	var status string
	s.Require().NoError(s.db.QueryRowContext(s.ctx, `
SELECT status FROM users WHERE id = $1`,
		userID,
	).Scan(&status))
	return status
}

func (s *EnterpriseManagementRepoSuite) employeeDisabledMarker(userID int64) bool {
	s.T().Helper()
	var marker bool
	s.Require().NoError(s.db.QueryRowContext(s.ctx, `
SELECT employee_disabled_by_enterprise FROM users WHERE id = $1`,
		userID,
	).Scan(&marker))
	return marker
}

func (s *EnterpriseManagementRepoSuite) userExistsIncludingDeleted(userID int64) bool {
	s.T().Helper()
	exists, err := s.client.User.Query().
		Where(dbuser.IDEQ(userID)).
		Exist(mixins.SkipSoftDelete(s.ctx))
	s.Require().NoError(err)
	return exists
}

func (s *EnterpriseManagementRepoSuite) countRows(table string, where string, args ...any) int {
	s.T().Helper()
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", table, where)
	s.Require().NoError(s.db.QueryRowContext(s.ctx, query, args...).Scan(&count))
	return count
}

type enterpriseEmployeeBalanceLog struct {
	enterpriseID     int64
	operatorID       int64
	delta            float64
	enterpriseBefore float64
	enterpriseAfter  float64
	employeeBefore   float64
	employeeAfter    float64
	reason           string
}

func (s *EnterpriseManagementRepoSuite) latestBalanceLog(employeeID int64) enterpriseEmployeeBalanceLog {
	s.T().Helper()
	var log enterpriseEmployeeBalanceLog
	err := s.db.QueryRowContext(s.ctx, `
SELECT
	enterprise_user_id,
	operator_user_id,
	delta::double precision,
	enterprise_balance_before::double precision,
	enterprise_balance_after::double precision,
	employee_balance_before::double precision,
	employee_balance_after::double precision,
	reason
FROM enterprise_employee_balance_logs
WHERE employee_user_id = $1
ORDER BY id DESC
LIMIT 1`,
		employeeID,
	).Scan(
		&log.enterpriseID,
		&log.operatorID,
		&log.delta,
		&log.enterpriseBefore,
		&log.enterpriseAfter,
		&log.employeeBefore,
		&log.employeeAfter,
		&log.reason,
	)
	s.Require().NoError(err)
	return log
}

func enterpriseManagementDockerIsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Env = os.Environ()
	return cmd.Run() == nil
}

func enterpriseManagementOpenSQLWithRetry(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return db, nil
		}
		lastErr = err
		_ = db.Close()
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("db not ready after %s: %w", timeout, lastErr)
}
