# Enterprise Management And Invite Default Groups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add enterprise employee management and affiliate-invite default exclusive-group assignment while preserving the official admin user-management baseline and the existing agent-management extension.

**Architecture:** This is an additive extension. Enterprise employee operations get their own service, repository, handler, routes, and frontend views; shared ownership, quota, and group-delegation behavior reuses the existing `users`, `agent_profiles`, and `agent_group_delegations` patterns where that keeps billing and auth caches stable. Employee balance remains in `users.balance` for official API billing compatibility, but only enterprise allocation flows can fund it, with an enterprise allocation ledger for audit.

**Tech Stack:** Go 1.26.3, Gin, Ent plus raw SQL repository methods for hzz extension tables, PostgreSQL migrations, Vue 3, Pinia, Vue Router, Vitest, GitHub Actions/remote Docker build, local deployment only from remote-built artifacts/images when possible.

---

## Execution Rules

- Work on branch `hzz` only.
- Keep `main` as the official clean baseline.
- Do not run full local builds unless the user explicitly approves. This machine is resource constrained.
- Local verification should be targeted:
  - backend: focused `go test` package/test names only;
  - frontend: focused Vitest tests only, or type-level file checks when cheap;
  - formatting: `gofmt` for touched Go files.
- After each task group:
  - run the listed targeted verification;
  - commit;
  - push `hzz`;
  - trigger or wait for GitHub remote build/CI where available.
- After backend and frontend are both ready, deploy locally for user UI testing from the remote-built image/artifact.

## Source Design

- Design spec: `docs/superpowers/specs/2026-06-02-enterprise-management-and-invite-default-groups-design.md`
- Current latest design commit: `fdf2be4d docs: refine enterprise employee allocation design`

## File Structure

### Backend Migrations

- Create `backend/migrations/149_hzz_enterprise_management.sql`
  - Adds `enterprise_profiles`.
  - Adds `enterprise_employee_balance_logs`.
  - Adds `employee_disabled_by_enterprise` to `users`.
  - Adds indexes for enterprise employee listing and status restoration.
- Modify `backend/migrations/migrations.go`
  - Register migration `149_hzz_enterprise_management.sql` if migration registration is explicit.

### Backend Domain And Service Types

- Modify `backend/internal/domain/constants.go`
  - Add `RoleEmployee = "employee"`.
- Modify `backend/internal/service/domain_constants.go`
  - Re-export `RoleEmployee`.
- Modify `backend/internal/service/user.go`
  - Add `EnterpriseProfile *EnterpriseProfile` for enterprise list responses if needed.
- Create `backend/internal/service/enterprise_management.go`
  - Defines enterprise DTOs, errors, repository interfaces, and enterprise-management service methods.
- Modify `backend/internal/service/agent_management.go`
  - Treat `enterprise` as a profile-backed quota manager for allocation updates.
  - Create `enterprise_profiles` when upgrading a direct user to enterprise.
  - Apply correct delete/detach behavior for direct enterprises.
  - Include employees in recursive group removal where enterprise groups cascade to employees.
- Modify `backend/internal/service/admin_service.go`
  - Hard-delete enterprises with employees when admin deletes an enterprise.
  - Hard-delete employees with balance/quota return when admin deletes an employee.
  - Cascade enterprise disable/enable to employee status with durable marker.
- Modify `backend/internal/service/auth_service.go`
  - Resolve enterprise invitations to the enterprise upstream owner.
  - Apply invite default groups only for affiliate invitation registration.
  - Ensure employee role is never created by public registration.
- Modify `backend/internal/service/auth_oauth_email_flow.go`
  - Apply the same affiliate invite default group behavior to OAuth registration completion flows.
- Modify user-facing services and handlers:
  - `backend/internal/service/redeem_service.go`
  - `backend/internal/service/payment_order.go`
  - `backend/internal/service/affiliate_service.go`
  - `backend/internal/handler/user_handler.go`
  - `backend/internal/handler/redeem_handler.go`
  - `backend/internal/handler/payment_handler.go`
  - `backend/internal/handler/subscription_handler.go`
  - These block employee wallet, redeem, payment, affiliate, and my-subscription capabilities.

### Backend Repositories And Wiring

- Create `backend/internal/repository/enterprise_management_repo.go`
  - Owns raw SQL for `enterprise_profiles`, employee allocation transactions, enterprise/employee deletion cleanup, status cascade, and employee group propagation.
- Modify `backend/internal/repository/agent_management_repo.go`
  - Include enterprise profiles in direct-child quota usage.
  - Add invite group default CRUD if not already fully present in current branch.
  - Reuse recursive group cleanup for enterprise employee descendants.
- Modify `backend/internal/repository/wire.go`
  - Add `NewEnterpriseManagementRepository`.
- Modify `backend/internal/service/wire.go`
  - Add `NewEnterpriseManagementService`.
  - Inject enterprise cleanup service/repository into admin and agent-management services as needed.
- Modify `backend/internal/handler/wire.go`
  - Add `NewEnterpriseManagementHandler`.
  - Add field to `handler.Handlers`.

### Backend Handlers And Routes

- Create `backend/internal/handler/enterprise_management_handler.go`
  - Handles enterprise summary, employees, allocation, groups.
- Create `backend/internal/server/routes/enterprise_management.go`
  - Registers authenticated routes under `/enterprise-management`.
- Modify `backend/internal/server/routes/user.go`
  - Register enterprise-management routes beside agent-management routes.
- Modify `backend/internal/server/routes/agent_management.go`
  - Keep agent routes intact; add only invite default group endpoint changes if needed.

### Frontend API, State, And Routes

- Modify `frontend/src/types/index.ts`
  - Add `employee` role.
  - Add enterprise management DTOs.
  - Add invite default group fields on `AgentGroupRate`.
- Modify `frontend/src/stores/auth.ts`
  - Add `isEnterprise`, `isEmployee`, `canUseEnterpriseManagement`.
  - Keep `canUseAgentManagement = admin || agent`.
- Modify `frontend/src/router/index.ts`
  - Add `/enterprise/employees` and `/enterprise/groups`.
  - Add route meta `requiresEnterpriseManagement`.
  - Block employee access to redeem, affiliate, purchase/orders, subscriptions.
- Modify `frontend/src/router/meta.d.ts`
  - Add `requiresEnterpriseManagement`.
- Modify `frontend/src/components/layout/AppSidebar.vue`
  - Add enterprise sidebar section for enterprise accounts.
  - Hide wallet/redeem/affiliate/subscription entries for employees.
  - Keep admin and agent agent-management sidebar visible.
- Create `frontend/src/api/enterpriseManagement.ts`
  - Calls new enterprise management endpoints.
- Modify `frontend/src/api/agentManagement.ts`
  - Add invite group default API if backend endpoint shape changes.

### Frontend Views

- Create `frontend/src/views/enterprise/EmployeeManagementView.vue`
  - Employee list, search, create employee, edit allocation, group assignment, delete employee.
- Create `frontend/src/views/enterprise/MyGroupsView.vue`
  - Enterprise group/rate visibility only; no repricing controls.
- Create `frontend/src/components/enterprise/EmployeeCreateModal.vue`
  - Modal patterned after `frontend/src/components/agent/AgentDirectUserCreateModal.vue`.
- Modify `frontend/src/views/agent/MyGroupsView.vue`
  - Add admin/agent invite default group toggle and default rate field for eligible exclusive groups.
- Modify `frontend/src/views/agent/AgentDirectChildrenView.vue`
  - Ensure direct enterprise allocation treats enterprise pool correctly.
  - Keep balance read-only in agent/admin direct children views.

### Tests

- Create or modify backend tests:
  - `backend/internal/service/enterprise_management_test.go`
  - `backend/internal/repository/enterprise_management_repo_integration_test.go`
  - `backend/internal/handler/enterprise_management_handler_test.go`
  - `backend/internal/server/routes/enterprise_management_test.go`
  - `backend/internal/service/admin_service_delete_test.go`
  - `backend/internal/service/admin_service_enterprise_status_test.go`
  - `backend/internal/service/auth_service_register_test.go`
  - `backend/internal/service/auth_oauth_email_flow_test.go`
  - `backend/internal/service/agent_management_test.go`
  - `backend/internal/repository/agent_management_repo_integration_test.go`
- Create or modify frontend tests:
  - `frontend/src/views/enterprise/__tests__/enterpriseManagement.spec.ts`
  - `frontend/src/views/agent/__tests__/agentManagement.spec.ts`
  - `frontend/src/router/__tests__/guards.spec.ts`
  - `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`

---

## Task 1: Enterprise Schema And Role Foundation

**Files:**
- Create: `backend/migrations/149_hzz_enterprise_management.sql`
- Modify: `backend/migrations/migrations.go` if explicit registration is required
- Modify: `backend/internal/domain/constants.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/internal/service/user.go`
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: Add the migration**

Create `backend/migrations/149_hzz_enterprise_management.sql` with:

```sql
CREATE TABLE IF NOT EXISTS enterprise_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pool_concurrency INTEGER NOT NULL DEFAULT 0,
    pool_rpm INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_enterprise_profiles_user_id
    ON enterprise_profiles(user_id)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS enterprise_employee_balance_logs (
    id BIGSERIAL PRIMARY KEY,
    enterprise_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operator_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta DECIMAL(18,6) NOT NULL,
    enterprise_balance_before DECIMAL(18,6) NOT NULL,
    enterprise_balance_after DECIMAL(18,6) NOT NULL,
    employee_balance_before DECIMAL(18,6) NOT NULL,
    employee_balance_after DECIMAL(18,6) NOT NULL,
    reason VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_balance_logs_enterprise
    ON enterprise_employee_balance_logs(enterprise_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_enterprise_employee_balance_logs_employee
    ON enterprise_employee_balance_logs(employee_user_id, created_at DESC);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS employee_disabled_by_enterprise BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_users_enterprise_employees
    ON users(parent_user_id, role, id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_employee_enterprise_disable
    ON users(parent_user_id, employee_disabled_by_enterprise)
    WHERE deleted_at IS NULL AND role = 'employee';
```

- [ ] **Step 2: Register migration if needed**

Inspect `backend/migrations/migrations.go`. If migrations are listed explicitly, add:

```go
// 149_hzz_enterprise_management.sql
```

Keep the registration style exactly consistent with the existing file.

- [ ] **Step 3: Add backend role constant**

In `backend/internal/domain/constants.go`, add:

```go
RoleEmployee = "employee"
```

In `backend/internal/service/domain_constants.go`, re-export:

```go
RoleEmployee = domain.RoleEmployee
```

- [ ] **Step 4: Add optional enterprise profile response field**

In `backend/internal/service/user.go`, add this field near `AgentProfile`:

```go
// EnterpriseProfile is populated only by enterprise-management or direct-enterprise list responses.
EnterpriseProfile *EnterpriseProfile
```

- [ ] **Step 5: Add frontend role type**

In `frontend/src/types/index.ts`, change:

```ts
export type UserRole = 'admin' | 'agent_level1' | 'agent_level2' | 'enterprise' | 'user'
```

to:

```ts
export type UserRole = 'admin' | 'agent_level1' | 'agent_level2' | 'enterprise' | 'employee' | 'user'
```

- [ ] **Step 6: Verify foundation edits**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/domain/constants.go backend/internal/service/domain_constants.go backend/internal/service/user.go
git diff --check
```

Expected:

```text
no output from git diff --check
```

- [ ] **Step 7: Commit and push**

Run:

```bash
git add backend/migrations/149_hzz_enterprise_management.sql backend/migrations/migrations.go backend/internal/domain/constants.go backend/internal/service/domain_constants.go backend/internal/service/user.go frontend/src/types/index.ts
git commit -m "feat: add enterprise employee schema foundation"
git push origin hzz
```

---

## Task 2: Enterprise Repository Transactions

**Files:**
- Create: `backend/internal/repository/enterprise_management_repo.go`
- Create: `backend/internal/repository/enterprise_management_repo_integration_test.go`
- Modify: `backend/internal/repository/wire.go`

- [ ] **Step 1: Define repository responsibilities in tests first**

Create integration tests covering these repository behaviors:

```go
func (s *EnterpriseManagementRepoSuite) TestCreateEmployeeMovesBalanceAndWritesLedger() {
    // enterprise starts with balance 100, pool concurrency 10, pool rpm 100.
    // create employee with balance 25, concurrency 3, rpm 30.
    // assert enterprise balance 75, employee balance 25, employee role "employee",
    // employee parent is enterprise, and one ledger row delta=25 reason="create_employee".
}

func (s *EnterpriseManagementRepoSuite) TestUpdateEmployeeAllocationMovesOnlyBalanceDelta() {
    // employee starts with balance 25.
    // set target balance to 10.
    // assert enterprise receives 15 back and ledger delta=-15.
    // then set target balance to 40.
    // assert enterprise pays 30 and ledger delta=30.
}

func (s *EnterpriseManagementRepoSuite) TestDeleteEmployeeReturnsBalanceAndHardDeletes() {
    // employee balance is returned to enterprise.
    // employee user row is gone with hard-delete context.
    // allowed groups and group delegations involving employee are removed by FK/cascade or explicit cleanup.
}

func (s *EnterpriseManagementRepoSuite) TestEnterpriseDisableEnableRestoresOnlyMarkedEmployees() {
    // active employee becomes disabled and employee_disabled_by_enterprise=true.
    // already disabled employee remains disabled and marker=false.
    // enable enterprise restores only marker=true employee.
}
```

- [ ] **Step 2: Create repository interface implementation**

Create `backend/internal/repository/enterprise_management_repo.go` with a concrete repository that exposes these operations to service:

```go
type enterpriseManagementRepository struct {
    client *dbent.Client
    sql    sqlExecutor
}

func NewEnterpriseManagementRepository(client *dbent.Client, sqlDB *sql.DB) service.EnterpriseManagementRepository {
    return &enterpriseManagementRepository{client: client, sql: sqlDB}
}
```

The repository must use transactions for:

```go
CreateEmployee(ctx, enterpriseID, operatorID int64, user *service.User) error
UpdateEmployeeAllocation(ctx, enterpriseID, employeeID, operatorID int64, target service.EmployeeAllocationUpdate) (*service.EmployeeAllocationResult, error)
DeleteEmployeeAndReturnAllocation(ctx, enterpriseID, employeeID, operatorID int64) ([]int64, error)
HardDeleteEnterpriseWithEmployees(ctx context.Context, enterpriseID int64) ([]int64, error)
CascadeEnterpriseStatus(ctx context.Context, enterpriseID int64, targetStatus string) ([]int64, error)
```

For balance moves, lock both enterprise and employee rows:

```sql
SELECT id, balance FROM users WHERE id = $1 AND role = 'enterprise' AND deleted_at IS NULL FOR UPDATE;
SELECT id, balance FROM users WHERE id = $1 AND role = 'employee' AND parent_user_id = $2 AND deleted_at IS NULL FOR UPDATE;
```

Then compute:

```go
delta := targetBalance - employeeBalance
if delta > 0 && enterpriseBalance < delta {
    return service.ErrEnterpriseManagementBalanceExceeded
}
```

- [ ] **Step 3: Add profile methods**

Implement:

```go
GetEnterpriseProfile(ctx context.Context, userID int64) (*service.EnterpriseProfile, error)
UpsertEnterpriseProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
RecalculateEnterpriseQuota(ctx context.Context, enterpriseID int64) error
GetEnterpriseEmployeeQuotaUsage(ctx context.Context, enterpriseID int64, excludeEmployeeID *int64) (service.QuotaUsageSummary, error)
```

`RecalculateEnterpriseQuota` must set enterprise `users.concurrency` and `users.rpm_limit` to remaining self-use quota:

```go
remainingConcurrency := repoQuotaRemaining(poolConcurrency, usage.Concurrency, usage.UnlimitedConcurrency)
remainingRPM := repoQuotaRemaining(poolRPM, usage.RPM, usage.UnlimitedRPM)
```

- [ ] **Step 4: Wire repository provider**

In `backend/internal/repository/wire.go`, add:

```go
NewEnterpriseManagementRepository,
```

beside `NewAgentManagementRepository`.

- [ ] **Step 5: Run targeted repository verification**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/repository/enterprise_management_repo.go backend/internal/repository/enterprise_management_repo_integration_test.go backend/internal/repository/wire.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/repository -run 'TestEnterpriseManagementRepoSuite' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/repository
```

- [ ] **Step 6: Commit and push**

Run:

```bash
git add backend/internal/repository/enterprise_management_repo.go backend/internal/repository/enterprise_management_repo_integration_test.go backend/internal/repository/wire.go
git commit -m "feat: add enterprise management repository"
git push origin hzz
```

---

## Task 3: Enterprise Service Rules

**Files:**
- Create: `backend/internal/service/enterprise_management.go`
- Create: `backend/internal/service/enterprise_management_test.go`
- Modify: `backend/internal/service/wire.go`

- [ ] **Step 1: Write service tests for employee creation and allocation**

Add tests:

```go
func TestEnterpriseManagementCreateEmployeeRequiresEnterpriseActor(t *testing.T) {
    // actor role user -> ErrEnterpriseManagementForbidden
}

func TestEnterpriseManagementCreateEmployeeRejectsBalanceOverEnterpriseBalance(t *testing.T) {
    // enterprise balance 5, requested employee balance 10 -> ErrEnterpriseManagementBalanceExceeded
}

func TestEnterpriseManagementUpdateEmployeeAllocationTreatsZeroQuotaAsUnlimited(t *testing.T) {
    // enterprise profile pool_concurrency=0 and pool_rpm=0.
    // assigning employee concurrency=0 and rpm=0 is allowed.
}

func TestEnterpriseManagementUpdateEmployeeAllocationRejectsFinitePoolOverAllocation(t *testing.T) {
    // enterprise profile pool 10/100, existing employee usage 8/80.
    // increasing another employee to 3/30 exceeds remaining -> ErrEnterpriseManagementAllocationExceeded
}
```

- [ ] **Step 2: Define service DTOs and errors**

In `backend/internal/service/enterprise_management.go`, define:

```go
var (
    ErrEnterpriseManagementForbidden          = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_FORBIDDEN", "enterprise management operation is not allowed")
    ErrEnterpriseManagementNotEmployee        = infraerrors.Forbidden("ENTERPRISE_MANAGEMENT_NOT_EMPLOYEE", "target user is not an enterprise employee")
    ErrEnterpriseManagementInvalidAllocation  = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_ALLOCATION", "allocation must be non-negative")
    ErrEnterpriseManagementAllocationExceeded = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_ALLOCATION_EXCEEDED", "allocation exceeds enterprise remaining capacity")
    ErrEnterpriseManagementBalanceExceeded    = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_BALANCE_EXCEEDED", "employee balance exceeds enterprise available balance")
    ErrEnterpriseManagementInvalidGroup       = infraerrors.BadRequest("ENTERPRISE_MANAGEMENT_INVALID_GROUP", "only active exclusive groups visible to enterprise can be assigned")
)

type EnterpriseProfile struct {
    UserID          int64 `json:"user_id"`
    PoolConcurrency int   `json:"pool_concurrency"`
    PoolRPM         int   `json:"pool_rpm"`
}

type EmployeeCreateInput struct {
    Email       string  `json:"email"`
    Password    string  `json:"password"`
    Username    string  `json:"username"`
    Balance     float64 `json:"balance"`
    Concurrency int     `json:"concurrency"`
    RPM         int     `json:"rpm"`
}

type EmployeeAllocationUpdate struct {
    Balance     float64 `json:"balance"`
    Concurrency int     `json:"concurrency"`
    RPM         int     `json:"rpm"`
}
```

- [ ] **Step 3: Implement service methods**

Add:

```go
func (s *EnterpriseManagementService) ListEmployeesWithQuery(ctx context.Context, actorID int64, query DirectChildrenQuery) (*DirectChildrenResult, error)
func (s *EnterpriseManagementService) CreateEmployee(ctx context.Context, actorID int64, input EmployeeCreateInput) (*User, error)
func (s *EnterpriseManagementService) UpdateEmployeeAllocation(ctx context.Context, actorID int64, employeeID int64, input EmployeeAllocationUpdate) (*AllocationSummary, error)
func (s *EnterpriseManagementService) DeleteEmployee(ctx context.Context, actorID int64, employeeID int64) error
func (s *EnterpriseManagementService) GetSummary(ctx context.Context, actorID int64) (*AgentManagementSummary, error)
```

Rules:

```go
actor.Role == RoleEnterprise
employee.Role == RoleEmployee
employee.ParentUserID == actor.ID
input.Balance >= 0
input.Concurrency >= 0
input.RPM >= 0
```

- [ ] **Step 4: Add group propagation service methods**

Add:

```go
func (s *EnterpriseManagementService) ListMyGroups(ctx context.Context, actorID int64) ([]AgentGroupRate, error)
func (s *EnterpriseManagementService) ListEmployeeGroupOptions(ctx context.Context, actorID int64, employeeID int64) ([]ChildGroupDelegationOption, error)
func (s *EnterpriseManagementService) SetEmployeeGroup(ctx context.Context, actorID int64, employeeID int64, groupID int64, assigned bool) error
```

Rules:

```go
// Enterprise cannot reprice employee groups.
employeeRate := enterpriseEffectiveRate
childCanDelegate := false
```

- [ ] **Step 5: Wire service provider**

In `backend/internal/service/wire.go`, add:

```go
NewEnterpriseManagementService,
```

near `NewAffiliateService` and `ProvidePaymentService`.

- [ ] **Step 6: Run targeted service verification**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/service/enterprise_management.go backend/internal/service/enterprise_management_test.go backend/internal/service/wire.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/service -run 'TestEnterpriseManagement' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/service
```

- [ ] **Step 7: Commit and push**

Run:

```bash
git add backend/internal/service/enterprise_management.go backend/internal/service/enterprise_management_test.go backend/internal/service/wire.go
git commit -m "feat: add enterprise management service"
git push origin hzz
```

---

## Task 4: Enterprise Handler And Routes

**Files:**
- Create: `backend/internal/handler/enterprise_management_handler.go`
- Create: `backend/internal/handler/enterprise_management_handler_test.go`
- Create: `backend/internal/server/routes/enterprise_management.go`
- Create: `backend/internal/server/routes/enterprise_management_test.go`
- Modify: `backend/internal/handler/wire.go`
- Modify: `backend/internal/server/routes/user.go`

- [ ] **Step 1: Write handler tests**

Add tests:

```go
func TestEnterpriseManagementHandlerCreatesEmployeeWithAuthenticatedEnterprise(t *testing.T) {
    // POST /enterprise-management/employees uses current subject user ID as actor.
    // Assert service receives actorID and EmployeeCreateInput.
}

func TestEnterpriseManagementHandlerRejectsNegativeBalance(t *testing.T) {
    // POST employee with balance -1 returns 400.
}

func TestEnterpriseManagementHandlerSetsEmployeeGroupWithoutRatePayload(t *testing.T) {
    // PUT /enterprise-management/employees/:id/groups/:group_id with {"assigned":true}
    // Assert no rate/can_delegate input is accepted from enterprise UI/API.
}
```

- [ ] **Step 2: Implement handler**

Create handler methods:

```go
GET    /enterprise-management/summary
GET    /enterprise-management/employees
POST   /enterprise-management/employees
PUT    /enterprise-management/employees/:id/allocation
DELETE /enterprise-management/employees/:id
GET    /enterprise-management/groups
GET    /enterprise-management/employees/:id/groups
PUT    /enterprise-management/employees/:id/groups/:group_id
DELETE /enterprise-management/employees/:id/groups/:group_id
```

Use `currentActorID(c)` exactly like `AgentManagementHandler`.

- [ ] **Step 3: Register routes**

Create `backend/internal/server/routes/enterprise_management.go`:

```go
func RegisterEnterpriseManagementRoutes(authenticated *gin.RouterGroup, h *handler.Handlers) {
    enterpriseManagement := authenticated.Group("/enterprise-management")
    {
        enterpriseManagement.GET("/summary", h.EnterpriseManagement.Summary)
        enterpriseManagement.GET("/employees", h.EnterpriseManagement.ListEmployees)
        enterpriseManagement.POST("/employees", h.EnterpriseManagement.CreateEmployee)
        enterpriseManagement.PUT("/employees/:id/allocation", h.EnterpriseManagement.UpdateEmployeeAllocation)
        enterpriseManagement.DELETE("/employees/:id", h.EnterpriseManagement.DeleteEmployee)
        enterpriseManagement.GET("/groups", h.EnterpriseManagement.ListMyGroups)
        enterpriseManagement.GET("/employees/:id/groups", h.EnterpriseManagement.ListEmployeeGroupOptions)
        enterpriseManagement.PUT("/employees/:id/groups/:group_id", h.EnterpriseManagement.SetEmployeeGroup)
        enterpriseManagement.DELETE("/employees/:id/groups/:group_id", h.EnterpriseManagement.RemoveEmployeeGroup)
    }
}
```

In `backend/internal/server/routes/user.go`, call:

```go
RegisterEnterpriseManagementRoutes(authenticated, h)
```

beside agent-management registration.

- [ ] **Step 4: Wire handler**

In `backend/internal/handler/wire.go`, add `EnterpriseManagementHandler` to:

```go
ProvideHandlers(...)
type Handlers struct
ProviderSet
```

Use the same pattern as `AgentManagementHandler`.

- [ ] **Step 5: Run targeted route/handler verification**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/handler/enterprise_management_handler.go backend/internal/handler/enterprise_management_handler_test.go backend/internal/server/routes/enterprise_management.go backend/internal/server/routes/enterprise_management_test.go backend/internal/handler/wire.go backend/internal/server/routes/user.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/handler -run 'TestEnterpriseManagementHandler' -count=1
cd backend && /tmp/go1.26.3/bin/go test ./internal/server/routes -run 'TestEnterpriseManagementRoutes' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/handler
ok   github.com/Wei-Shaw/sub2api/internal/server/routes
```

- [ ] **Step 6: Commit and push**

Run:

```bash
git add backend/internal/handler/enterprise_management_handler.go backend/internal/handler/enterprise_management_handler_test.go backend/internal/server/routes/enterprise_management.go backend/internal/server/routes/enterprise_management_test.go backend/internal/handler/wire.go backend/internal/server/routes/user.go
git commit -m "feat: add enterprise management api"
git push origin hzz
```

---

## Task 5: Agent/Admin Enterprise Integration

**Files:**
- Modify: `backend/internal/service/agent_management.go`
- Modify: `backend/internal/repository/agent_management_repo.go`
- Modify: `backend/internal/service/agent_management_test.go`
- Modify: `backend/internal/repository/agent_management_repo_integration_test.go`

- [ ] **Step 1: Add failing tests for enterprise profile pools**

Add tests:

```go
func TestAgentManagementUpgradeDirectUserToEnterpriseCreatesEnterpriseProfile(t *testing.T) {
    // admin/agent upgrades direct user to enterprise with pool 20/200.
    // assert role enterprise, enterprise profile pool 20/200,
    // enterprise effective concurrency/rpm equals pool minus employee usage.
}

func TestAgentManagementUpdateEnterpriseAllocationCannotGoBelowEmployeeAllocations(t *testing.T) {
    // enterprise has employee allocated 8/80.
    // parent attempts to set enterprise pool 5/50.
    // expect ErrAgentManagementPoolReclaimExceeded metadata.
}

func TestAgentManagementAgentDetachesEnterpriseToAdminAndRecalculatesAgentQuota(t *testing.T) {
    // agent direct enterprise is rehomed to root admin.
    // enterprise account remains role enterprise.
    // employees remain children of enterprise.
    // agent remaining quota increases.
}
```

- [ ] **Step 2: Include enterprise profiles in quota usage**

In `backend/internal/repository/agent_management_repo.go`, update direct child usage SQL:

```sql
CASE
  WHEN u.role IN ('agent_level1', 'agent_level2') THEN COALESCE(ap.pool_concurrency, 0)
  WHEN u.role = 'enterprise' THEN COALESCE(ep.pool_concurrency, u.concurrency)
  ELSE u.concurrency
END
```

and join:

```sql
LEFT JOIN enterprise_profiles ep ON ep.user_id = u.id AND ep.deleted_at IS NULL
```

Do the same for RPM.

- [ ] **Step 3: Treat enterprise as profile-backed allocation target**

In `backend/internal/service/agent_management.go`, when `UpdateAllocation` targets `RoleEnterprise`, call an enterprise pool update path:

```go
if child.Role == RoleEnterprise {
    return s.updateDirectEnterprisePool(ctx, actor, child, requestedConcurrency, requestedRPM)
}
```

The enterprise path mirrors `updateDirectAgentPool`, but uses enterprise profile methods and employee usage.

- [ ] **Step 4: Upgrade direct user to enterprise**

When `UpgradeDirectUser` target is `RoleEnterprise`:

```go
if targetRole == RoleEnterprise {
    // validate pool values
    // check actor capacity
    // upsert enterprise profile
    // set role enterprise
    // recalculate enterprise effective quota
    // recalculate parent agent quota when actor is not admin
}
```

- [ ] **Step 5: Fix direct enterprise delete rules**

Rules:

```go
actor admin + child enterprise -> hard delete enterprise and employees
actor agent + child enterprise -> SetParent(child, rootAdmin.ID), preserve enterprise and employees
```

Do not use native soft delete for enterprise detach.

- [ ] **Step 6: Include employees in recursive group cleanup**

In recursive group removal role lists, include:

```go
RoleEmployee
```

so upstream group removal from enterprise cascades to employees.

- [ ] **Step 7: Run targeted agent verification**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/service/agent_management.go backend/internal/service/agent_management_test.go backend/internal/repository/agent_management_repo.go backend/internal/repository/agent_management_repo_integration_test.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/service -run 'TestAgentManagement.*Enterprise|TestAgentManagement.*DeleteRules|TestAgentManagement.*Group' -count=1
cd backend && /tmp/go1.26.3/bin/go test ./internal/repository -run 'TestAgentManagementRepoSuite' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/service
ok   github.com/Wei-Shaw/sub2api/internal/repository
```

- [ ] **Step 8: Commit and push**

Run:

```bash
git add backend/internal/service/agent_management.go backend/internal/service/agent_management_test.go backend/internal/repository/agent_management_repo.go backend/internal/repository/agent_management_repo_integration_test.go
git commit -m "feat: integrate enterprise pools with agent management"
git push origin hzz
```

---

## Task 6: Admin Native Delete And Disable Integration

**Files:**
- Modify: `backend/internal/service/admin_service.go`
- Modify: `backend/internal/service/admin_service_delete_test.go`
- Create: `backend/internal/service/admin_service_enterprise_status_test.go`

- [ ] **Step 1: Add failing admin deletion tests**

Add tests:

```go
func TestAdminDeleteEnterpriseHardDeletesEmployees(t *testing.T) {
    // admin native DeleteUser(enterpriseID)
    // asserts enterprise and employees are hard deleted.
}

func TestAdminDeleteAgentOwnedEnterpriseRecalculatesAgentQuota(t *testing.T) {
    // enterprise parent is agent.
    // deleting enterprise releases profile pool from parent usage.
}

func TestAdminDeleteEmployeeReturnsBalanceToEnterprise(t *testing.T) {
    // employee balance 12, enterprise balance 20.
    // DeleteUser(employeeID) returns enterprise balance 32 then hard deletes employee.
}
```

- [ ] **Step 2: Add failing status cascade tests**

Add tests:

```go
func TestAdminDisableEnterpriseDisablesOnlyActiveEmployeesAndMarksThem(t *testing.T) {
    // active employee becomes disabled + marker true.
    // already disabled employee stays disabled + marker false.
}

func TestAdminEnableEnterpriseRestoresOnlyEnterpriseMarkedEmployees(t *testing.T) {
    // marker true employee becomes active and marker false.
    // manually disabled employee remains disabled.
}
```

- [ ] **Step 3: Extend DeleteUser**

In `admin_service.go`:

```go
if user.Role == RoleEnterprise {
    affectedUserIDs, err := s.enterpriseDeletionCleanupRepo.HardDeleteEnterpriseWithEmployees(ctx, user.ID)
    // recalc parent agent if parent is agent
    // invalidate enterprise and affected employees
    return nil
}

if user.Role == RoleEmployee {
    affectedUserIDs, err := s.enterpriseDeletionCleanupRepo.DeleteEmployeeAndReturnAllocation(ctx, *user.ParentUserID, user.ID, adminOperatorID)
    // invalidate employee and enterprise
    return nil
}
```

Use the same transaction style as `rehomeDeletedAgentFromAdminUsers`.

- [ ] **Step 4: Extend UpdateUser status behavior**

When admin changes an enterprise status:

```go
if user.Role == RoleEnterprise && oldStatus != user.Status {
    affectedUserIDs, err := s.enterpriseDeletionCleanupRepo.CascadeEnterpriseStatus(ctx, user.ID, user.Status)
    // invalidate affected employee auth caches
}
```

Do not restore employees that were manually disabled.

- [ ] **Step 5: Run targeted admin verification**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/service/admin_service.go backend/internal/service/admin_service_delete_test.go backend/internal/service/admin_service_enterprise_status_test.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/service -run 'TestAdmin.*Enterprise|TestAdmin.*Employee|TestAdmin.*DeleteUser' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/service
```

- [ ] **Step 6: Commit and push**

Run:

```bash
git add backend/internal/service/admin_service.go backend/internal/service/admin_service_delete_test.go backend/internal/service/admin_service_enterprise_status_test.go
git commit -m "feat: add enterprise admin cleanup rules"
git push origin hzz
```

---

## Task 7: Employee Capability Restrictions

**Files:**
- Modify: `backend/internal/service/redeem_service.go`
- Modify: `backend/internal/service/payment_order.go`
- Modify: `backend/internal/service/affiliate_service.go`
- Modify: `backend/internal/handler/user_handler.go`
- Modify: `backend/internal/handler/redeem_handler.go`
- Modify: `backend/internal/handler/payment_handler.go`
- Modify: `backend/internal/handler/subscription_handler.go`
- Add focused tests in the corresponding service/handler test files.

- [ ] **Step 1: Add employee restriction tests**

Add tests:

```go
func TestRedeemServiceRejectsEmployee(t *testing.T) {
    // user role employee attempts redeem -> forbidden.
}

func TestPaymentServiceRejectsEmployeeCreateOrder(t *testing.T) {
    // employee attempts balance/subscription order -> forbidden.
}

func TestAffiliateServiceRejectsEmployeeEnsureAndTransfer(t *testing.T) {
    // employee cannot create affiliate profile and cannot transfer rebate.
}

func TestSubscriptionHandlerRejectsEmployeeMySubscriptions(t *testing.T) {
    // employee GET /subscriptions returns forbidden or route guard response.
}
```

- [ ] **Step 2: Add service-level guards**

Prefer service-level checks so UI hiding is not the only protection:

```go
if user.Role == RoleEmployee {
    return ErrEnterpriseManagementForbidden
}
```

Apply to:

- redeem;
- payment order creation;
- affiliate detail/ensure/transfer;
- user subscription list/progress/summary where current user is employee.

- [ ] **Step 3: Keep API key and usage allowed**

Do not block:

```text
/keys
/usage
/groups/available
/groups/rates
/channels/available
/user/profile
```

- [ ] **Step 4: Run targeted restriction tests**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/service/redeem_service.go backend/internal/service/payment_order.go backend/internal/service/affiliate_service.go backend/internal/handler/user_handler.go backend/internal/handler/redeem_handler.go backend/internal/handler/payment_handler.go backend/internal/handler/subscription_handler.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/service -run 'Test.*Employee.*(Redeem|Payment|Affiliate|Subscription)' -count=1
cd backend && /tmp/go1.26.3/bin/go test ./internal/handler -run 'Test.*Employee.*(Redeem|Payment|Affiliate|Subscription)' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/service
ok   github.com/Wei-Shaw/sub2api/internal/handler
```

- [ ] **Step 5: Commit and push**

Run:

```bash
git add backend/internal/service backend/internal/handler
git commit -m "feat: restrict employee wallet and affiliate capabilities"
git push origin hzz
```

---

## Task 8: Invite Default Exclusive Groups Backend

**Files:**
- Modify: `backend/internal/service/agent_management.go`
- Modify: `backend/internal/repository/agent_management_repo.go`
- Modify: `backend/internal/handler/agent_management_handler.go`
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/service/auth_oauth_email_flow.go`
- Modify: tests in service/repository/handler auth files.

- [ ] **Step 1: Verify current invite default quota implementation**

Before editing, confirm whether current branch already includes:

```text
agent_profiles.invite_default_concurrency
agent_profiles.invite_default_rpm
AgentManagementService.UpdateInviteDefaults
AuthService.resolveInvitationRegistrationQuota
```

If present, keep it and only extend group defaults.

- [ ] **Step 2: Add group default DTO fields**

Extend `AgentGroupRate`:

```go
type AgentGroupRate struct {
    Group                Group   `json:"group"`
    EffectiveRate        float64 `json:"effective_rate"`
    CanDelegate          bool    `json:"can_delegate"`
    Source               string  `json:"source"`
    InviteDefaultEnabled bool    `json:"invite_default_enabled"`
    InviteDefaultRate    float64 `json:"invite_default_rate_multiplier"`
}
```

- [ ] **Step 3: Add repository methods for defaults**

Expose:

```go
ListInviteGroupDefaults(ctx context.Context, managerID int64) (map[int64]AgentInviteGroupDefault, error)
SyncInviteGroupDefaults(ctx context.Context, managerID int64, defaults []AgentInviteGroupDefaultInput) error
ApplyInviteGroupDefaultsToUser(ctx context.Context, managerID int64, childUserID int64) error
```

Validation:

```go
rateMultiplier > 0
group.IsExclusive == true
manager can delegate the group at apply time
```

- [ ] **Step 4: Add service tests**

Add tests:

```go
func TestAgentManagementListMyGroupsIncludesInviteDefaultsForAdminAndAgent(t *testing.T) {
    // default configured for an eligible exclusive group appears enabled with rate.
}

func TestAgentManagementSyncInviteGroupDefaultsRejectsNonDelegableGroup(t *testing.T) {
    // agent sees delegated group can_delegate=false, tries to default it -> forbidden.
}

func TestAgentManagementInviteDefaultsDoNotApplyToEnterprise(t *testing.T) {
    // enterprise ListMyGroups has no invite default controls.
}
```

- [ ] **Step 5: Add registration application tests**

Add tests:

```go
func TestRegisterWithAffiliateInviteAppliesOwnerDefaultGroups(t *testing.T) {
    // agent affiliate code resolves parent to agent.
    // new user gets delegation manager=agent, can_delegate=false, configured rate.
}

func TestRegisterWithRedeemInvitationDoesNotApplyInviteDefaultGroups(t *testing.T) {
    // admin-created invitation redeem code creates user under admin but no group defaults apply.
}

func TestEnterpriseAffiliateInviteUsesUpstreamOwnerDefaults(t *testing.T) {
    // enterprise inviter under agent resolves parent to agent.
    // new user gets agent defaults, rebate remains enterprise inviter.
}
```

- [ ] **Step 6: Apply defaults only after user creation**

In `AuthService`, enrich `registrationInvitationResolution` with:

```go
ApplyAffiliateInviteDefaults bool
DefaultGroupOwnerID          *int64
```

Set it only when invitation resolution used an affiliate code. Do not set it for redeem invitation codes.

After `userRepo.Create(ctx, user)` and before returning registration success:

```go
if invitationResolution != nil && invitationResolution.ApplyAffiliateInviteDefaults && invitationResolution.DefaultGroupOwnerID != nil {
    if err := s.agentManagementService.ApplyInviteGroupDefaultsToUser(ctx, *invitationResolution.DefaultGroupOwnerID, user.ID); err != nil {
        logger.LegacyPrintf("service.auth", "[Auth] failed to apply invite default groups: user_id=%d owner_id=%d err=%v", user.ID, *invitationResolution.DefaultGroupOwnerID, err)
    }
}
```

Use the same helper in OAuth registration completion paths.

- [ ] **Step 7: Run targeted invite default tests**

Run:

```bash
/tmp/go1.26.3/bin/gofmt -w backend/internal/service/agent_management.go backend/internal/repository/agent_management_repo.go backend/internal/handler/agent_management_handler.go backend/internal/service/auth_service.go backend/internal/service/auth_oauth_email_flow.go
cd backend && /tmp/go1.26.3/bin/go test ./internal/service -run 'TestAgentManagement.*Invite|TestRegister.*Invite.*Group|TestEnterpriseAffiliateInvite' -count=1
cd backend && /tmp/go1.26.3/bin/go test ./internal/handler -run 'TestAgentManagementHandler.*Invite' -count=1
```

Expected:

```text
ok   github.com/Wei-Shaw/sub2api/internal/service
ok   github.com/Wei-Shaw/sub2api/internal/handler
```

- [ ] **Step 8: Commit and push**

Run:

```bash
git add backend/internal/service/agent_management.go backend/internal/repository/agent_management_repo.go backend/internal/handler/agent_management_handler.go backend/internal/service/auth_service.go backend/internal/service/auth_oauth_email_flow.go backend/internal/service/*test.go backend/internal/handler/*test.go
git commit -m "feat: apply invite default exclusive groups"
git push origin hzz
```

---

## Task 9: Enterprise Frontend API, Routing, And Sidebar

**Files:**
- Create: `frontend/src/api/enterpriseManagement.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/router/meta.d.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/router/__tests__/guards.spec.ts`
- Modify: `frontend/src/components/layout/__tests__/AppSidebar.spec.ts`

- [ ] **Step 1: Add frontend types**

Add:

```ts
export interface EnterpriseManagementSummary {
  allocation: AgentAllocationSummary
}

export interface EmployeeCreateRequest {
  email: string
  password: string
  username?: string
  balance: number
  concurrency: number
  rpm: number
}

export interface EmployeeAllocationUpdate {
  balance: number
  concurrency: number
  rpm: number
}

export interface EnterpriseEmployeeGroupRequest {
  assigned: boolean
}
```

- [ ] **Step 2: Add enterprise API client**

Create `frontend/src/api/enterpriseManagement.ts`:

```ts
const BASE_PATH = '/enterprise-management'

export async function getSummary(): Promise<EnterpriseManagementSummary> {
  const { data } = await apiClient.get<EnterpriseManagementSummary>(`${BASE_PATH}/summary`)
  return data
}

export async function listEmployees(query: { search?: string } = {}): Promise<AgentDirectChildrenResponse> {
  const { data } = await apiClient.get<AgentDirectChildrenResponse>(`${BASE_PATH}/employees`, { params: query })
  return data
}
```

Add create, update allocation, delete, list groups, list employee group options, set/remove group methods using the backend route names from Task 4.

- [ ] **Step 3: Add auth computed state**

In `frontend/src/stores/auth.ts`:

```ts
const isEnterprise = computed(() => user.value?.role === 'enterprise')
const isEmployee = computed(() => user.value?.role === 'employee')
const canUseEnterpriseManagement = computed(() => isEnterprise.value)
```

Return these from the store.

- [ ] **Step 4: Add route guards**

In `frontend/src/router/meta.d.ts`:

```ts
requiresEnterpriseManagement?: boolean
disallowEmployee?: boolean
```

In router guard:

```ts
if (to.meta.requiresEnterpriseManagement === true && !authStore.canUseEnterpriseManagement) {
  next('/dashboard')
  return
}

if (to.meta.disallowEmployee === true && authStore.isEmployee) {
  next('/dashboard')
  return
}
```

Mark employee-disallowed routes:

```text
/redeem
/affiliate
/purchase
/orders
/subscriptions
```

- [ ] **Step 5: Add enterprise routes**

Add:

```ts
{
  path: '/enterprise',
  redirect: '/enterprise/employees'
},
{
  path: '/enterprise/employees',
  name: 'EnterpriseEmployees',
  component: () => import('@/views/enterprise/EmployeeManagementView.vue'),
  meta: { requiresAuth: true, requiresEnterpriseManagement: true, titleKey: 'nav.enterpriseEmployees' }
},
{
  path: '/enterprise/groups',
  name: 'EnterpriseGroups',
  component: () => import('@/views/enterprise/MyGroupsView.vue'),
  meta: { requiresAuth: true, requiresEnterpriseManagement: true, titleKey: 'nav.enterpriseGroups' }
}
```

- [ ] **Step 6: Update sidebar**

Add enterprise nav section visible only when:

```ts
authStore.canUseEnterpriseManagement
```

Entries:

```ts
{ path: '/enterprise/employees', label: t('nav.enterpriseEmployees'), icon: UsersIcon },
{ path: '/enterprise/groups', label: t('nav.enterpriseGroups'), icon: FolderIcon },
```

For employees, filter self nav items to remove:

```text
/subscriptions
/purchase
/orders
/redeem
/affiliate
```

- [ ] **Step 7: Add frontend guard/sidebar tests**

Tests:

```ts
it('allows enterprise users to enterprise management routes', () => {
  const redirect = simulateGuard('/enterprise/employees', { requiresEnterpriseManagement: true }, {
    isAuthenticated: true,
    isAdmin: false,
    canUseEnterpriseManagement: true
  })
  expect(redirect).toBeUndefined()
})

it('blocks employee users from wallet and affiliate routes', () => {
  const redirect = simulateGuard('/redeem', { disallowEmployee: true }, {
    isAuthenticated: true,
    isEmployee: true
  })
  expect(redirect).toBe('/dashboard')
})
```

- [ ] **Step 8: Run targeted frontend verification**

Run:

```bash
cd frontend
pnpm vitest run src/router/__tests__/guards.spec.ts src/components/layout/__tests__/AppSidebar.spec.ts
```

Expected:

```text
Test Files  2 passed
```

- [ ] **Step 9: Commit and push**

Run:

```bash
git add frontend/src/api/enterpriseManagement.ts frontend/src/types/index.ts frontend/src/stores/auth.ts frontend/src/router/index.ts frontend/src/router/meta.d.ts frontend/src/components/layout/AppSidebar.vue frontend/src/router/__tests__/guards.spec.ts frontend/src/components/layout/__tests__/AppSidebar.spec.ts
git commit -m "feat: add enterprise frontend routing"
git push origin hzz
```

---

## Task 10: Enterprise Employee Management Views

**Files:**
- Create: `frontend/src/views/enterprise/EmployeeManagementView.vue`
- Create: `frontend/src/views/enterprise/MyGroupsView.vue`
- Create: `frontend/src/components/enterprise/EmployeeCreateModal.vue`
- Create: `frontend/src/views/enterprise/__tests__/enterpriseManagement.spec.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Build create modal patterned after agent modal**

Fields:

```text
email
username
password
balance
concurrency
rpm
```

Rules:

```ts
balance >= 0
concurrency >= 0
rpm >= 0
```

The modal must show enterprise remaining quota and balance context from summary.

- [ ] **Step 2: Build employee table view**

`EmployeeManagementView.vue` includes:

```text
search box
create employee button
summary chips: enterprise balance, remaining concurrency, remaining rpm
columns: email, role, balance, allocation, actions
actions: save allocation, manage groups, delete employee
```

Allocation editor writes target values:

```ts
{
  balance: Number(balanceDraft),
  concurrency: Number(concurrencyDraft),
  rpm: Number(rpmDraft)
}
```

- [ ] **Step 3: Build group assignment dialog**

Enterprise group dialog:

```text
shows group name, source, effective rate
shows assigned toggle
does not show rate input
does not show can_delegate toggle
```

Submit:

```ts
enterpriseManagementAPI.setEmployeeGroup(employee.id, group.id, { assigned: true })
```

Remove:

```ts
enterpriseManagementAPI.removeEmployeeGroup(employee.id, group.id)
```

- [ ] **Step 4: Build enterprise my groups view**

`MyGroupsView.vue` displays:

```text
name
source
effective_rate
can_delegate
```

No invite default controls.
No reprice controls.

- [ ] **Step 5: Add i18n keys**

Add nav keys:

```ts
enterpriseManagement: '企业管理'
enterpriseEmployees: '员工管理'
enterpriseGroups: '企业分组'
```

Add English equivalents:

```ts
enterpriseManagement: 'Enterprise Management'
enterpriseEmployees: 'Employees'
enterpriseGroups: 'Enterprise Groups'
```

- [ ] **Step 6: Add view tests**

Tests:

```ts
it('creates an employee with target balance and quota', async () => {
  // fill modal and assert enterpriseManagementAPI.createEmployee payload.
})

it('saves employee allocation as target balance not delta', async () => {
  // edit balance to 30 and assert payload.balance === 30.
})

it('does not render rate input in enterprise employee group dialog', async () => {
  // open group dialog and assert no childRate field exists.
})
```

- [ ] **Step 7: Run targeted frontend view tests**

Run:

```bash
cd frontend
pnpm vitest run src/views/enterprise/__tests__/enterpriseManagement.spec.ts
```

Expected:

```text
Test Files  1 passed
```

- [ ] **Step 8: Commit and push**

Run:

```bash
git add frontend/src/views/enterprise frontend/src/components/enterprise frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: add enterprise employee management ui"
git push origin hzz
```

---

## Task 11: Invite Default Groups Frontend

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/agentManagement.ts`
- Modify: `frontend/src/views/agent/MyGroupsView.vue`
- Modify: `frontend/src/views/agent/__tests__/agentManagement.spec.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`
- Modify: `frontend/src/i18n/locales/en.ts`

- [ ] **Step 1: Extend frontend group type**

Add:

```ts
invite_default_enabled?: boolean
invite_default_rate_multiplier?: number
```

to `AgentGroupRate`.

- [ ] **Step 2: Add API method**

If backend uses batch sync:

```ts
export interface AgentInviteGroupDefaultUpdate {
  group_id: number
  enabled: boolean
  rate_multiplier: number
}

export async function updateInviteGroupDefaults(payload: AgentInviteGroupDefaultUpdate[]): Promise<void> {
  await apiClient.put(`${BASE_PATH}/invite-group-defaults`, { defaults: payload })
}
```

If backend folds this into existing `/invite-defaults`, keep endpoint naming consistent with Task 8.

- [ ] **Step 3: Add controls to agent groups view**

Only show controls for:

```ts
row.group.is_exclusive === true && row.can_delegate === true
```

Controls:

```text
checkbox/toggle: apply by default to invite registrations
number input: default rate multiplier
save button
```

Do not show controls to enterprise views.

- [ ] **Step 4: Add validation**

Before saving:

```ts
if (enabled && rate <= 0) {
  appStore.showError(t('agentManagement.groups.invalidRate'))
  return
}
```

- [ ] **Step 5: Add tests**

Tests:

```ts
it('shows invite default controls for delegable exclusive groups', async () => {
  // group exclusive + can_delegate true -> toggle visible.
})

it('hides invite default controls for public groups and non-delegable groups', async () => {
  // public group -> hidden.
  // exclusive can_delegate false -> hidden.
})

it('saves invite default group rate payload', async () => {
  // enable group, set rate 1.25, save -> API payload contains group_id and rate.
})
```

- [ ] **Step 6: Run targeted frontend agent tests**

Run:

```bash
cd frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts
```

Expected:

```text
Test Files  1 passed
```

- [ ] **Step 7: Commit and push**

Run:

```bash
git add frontend/src/types/index.ts frontend/src/api/agentManagement.ts frontend/src/views/agent/MyGroupsView.vue frontend/src/views/agent/__tests__/agentManagement.spec.ts frontend/src/i18n/locales/zh.ts frontend/src/i18n/locales/en.ts
git commit -m "feat: add invite default group controls"
git push origin hzz
```

---

## Task 12: Remote Build, Deployment, And Manual Test Pass

**Files:**
- Modify only if CI or deployment scripts need branch/image name adjustment.

- [ ] **Step 1: Run a local lightweight final sanity check**

Run:

```bash
git status --short --branch
git log --oneline -5
git diff --check
```

Expected:

```text
git diff --check has no output
branch is hzz
latest commits are the staged task commits
```

- [ ] **Step 2: Trigger remote GitHub build**

Use GitHub Actions or the repository's existing remote build workflow for branch `hzz`.

Expected:

```text
remote build passes on hzz
remote image/artifact is produced
```

- [ ] **Step 3: Deploy locally from remote-built artifact/image**

Pull the remote-built image/artifact for `hzz` and deploy to the local environment used for UI testing.

Expected:

```text
local app reachable at http://127.0.0.1:8080/
```

- [ ] **Step 4: Manual admin/agent/enterprise test script**

Manual checks:

```text
Admin:
- native user management still works.
- admin agent-management sidebar is visible.
- admin can upgrade direct user to enterprise.
- admin direct enterprise delete hard-deletes enterprise and employees.
- admin native disable enterprise disables employees.
- admin native enable enterprise restores only enterprise-disabled employees.

Agent:
- agent can create direct user.
- agent can upgrade direct user to enterprise.
- agent deleting direct enterprise rehomes it to admin.
- agent remaining quota recalculates.
- agent groups page can configure invite default exclusive groups.

Enterprise:
- enterprise sidebar shows employee management and enterprise groups.
- enterprise can create employee with balance/concurrency/rpm.
- balance allocation moves target balance without redeem code.
- employee can use API keys and usage pages.
- employee cannot see redeem, affiliate, payment, orders, or subscriptions.
- enterprise can assign exclusive group to employee without rate input.

Registration:
- affiliate invite from agent applies default groups and rate.
- affiliate invite from enterprise places user under enterprise upstream owner.
- redeem invitation does not apply affiliate default group template.
```

- [ ] **Step 5: Commit any deployment-script adjustment and push**

If no code changes are needed, do not create a commit.

If a script/config change is needed:

```bash
git add <changed-file>
git commit -m "chore: align hzz deployment build"
git push origin hzz
```

---

## Self-Review Checklist

- [ ] Enterprise remains a user role and employee is a final child role.
- [ ] Employees are not created by public registration, OAuth registration, affiliate invitation, or redeem invitation.
- [ ] Employee balance uses `users.balance` only as enterprise-assigned API-consumable balance.
- [ ] Enterprise employee balance changes are target-value edits with ledger records, not redeem codes.
- [ ] Enterprise can propagate groups to employees but cannot reprice and cannot set `can_delegate=true`.
- [ ] Admin native enterprise delete hard-deletes enterprise plus employees.
- [ ] Agent direct enterprise delete only rehomes enterprise to admin.
- [ ] Admin native employee delete returns employee balance/quota to enterprise, then hard-deletes employee.
- [ ] Enterprise disable/enable uses durable marker to restore only enterprise-disabled employees.
- [ ] Invite default exclusive groups apply only to affiliate invitation registrations.
- [ ] Admin redeem invitation codes do not apply affiliate invite default group templates.
- [ ] Enterprise affiliate invitee belongs to enterprise upstream owner, while rebate remains enterprise inviter.
- [ ] Existing admin native user management remains otherwise unchanged.
- [ ] Branch `hzz` is pushed after every completed phase.
