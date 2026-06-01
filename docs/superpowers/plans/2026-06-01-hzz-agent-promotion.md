# HZZ Agent Promotion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the additive two-level agent promotion system, direct-child management, concurrency/RPM distribution, delegated exclusive group rates, and affiliate-code invitation ownership.

**Architecture:** Keep official admin pages and official recharge/redeem/payment behavior untouched. Add focused agent-management repository/service/handler/routes and frontend pages under a separate management section shared by admins and agents. Persist hierarchy and allocation state on users, and persist exclusive-group delegation in a new table.

**Tech Stack:** Go, Gin, Ent, PostgreSQL migrations, Wire, Vue 3, TypeScript, Pinia, Vitest, GitHub Actions.

---

## File Structure

Create:

- `backend/internal/service/agent_management.go` - domain service for direct-child listing, allocation, upgrades, deletion/detach, group visibility, and invitation ownership helpers.
- `backend/internal/service/agent_management_test.go` - unit tests for hierarchy, allocation, upgrade, delete/detach, and ownership resolution.
- `backend/internal/repository/agent_management_repo.go` - SQL/Ent-backed repository helpers for parent updates, child listing, allocation sums, root admin lookup, and group delegation persistence.
- `backend/internal/repository/agent_management_repo_integration_test.go` - integration tests for repository persistence and delete/move semantics.
- `backend/internal/handler/agent_management_handler.go` - HTTP handler for additive agent-management endpoints.
- `backend/internal/handler/agent_management_handler_test.go` - handler authorization and payload tests.
- `backend/internal/server/routes/agent_management.go` - authenticated non-admin route registration for `/api/v1/agent-management`.
- `backend/migrations/145_hzz_agent_promotion.sql` - additive DB migration.
- `frontend/src/api/agentManagement.ts` - frontend API client and types.
- `frontend/src/views/agent/DirectUsersView.vue` - direct ordinary user management page.
- `frontend/src/views/agent/DirectAgentsView.vue` - direct agent management page.
- `frontend/src/views/agent/DirectEnterprisesView.vue` - direct enterprise management page.
- `frontend/src/views/agent/MyGroupsView.vue` - group/rate visibility and delegation page.
- `frontend/src/views/agent/__tests__/agentManagement.spec.ts` - page/action visibility tests.
- `frontend/src/api/__tests__/agentManagement.spec.ts` - API client path tests.

Modify:

- `backend/internal/domain/constants.go` - add role constants.
- `backend/internal/service/domain_constants.go` - re-export new role constants to service package.
- `backend/ent/schema/user.go` - add parent/allocation fields.
- `backend/internal/service/user.go` - expose parent/allocation fields on `service.User`.
- `backend/internal/repository/user_repo.go` - map new user fields.
- `backend/internal/service/auth_service.go` - accept affiliate code as invitation credential and assign `parent_user_id` after ownership resolution.
- `backend/internal/handler/handler.go` - add `AgentManagement *AgentManagementHandler`.
- `backend/internal/handler/wire.go` - provide `NewAgentManagementHandler` and include it in `Handlers`.
- `backend/internal/service/wire.go` - provide `NewAgentManagementService`.
- `backend/internal/repository/wire.go` - provide `NewAgentManagementRepository`.
- `backend/internal/server/routes/user.go` - register agent-management routes under JWT auth and normal backend user guard.
- `backend/cmd/server/wire_gen.go` - regenerate with `go generate ./cmd/server`.
- `frontend/src/types/index.ts` - add role literals and shared agent-management response types.
- `frontend/src/stores/auth.ts` - add `isAgent` and `canUseAgentManagement` computed values.
- `frontend/src/components/layout/AppSidebar.vue` - add additive management section for admin and agent roles.
- `frontend/src/router/index.ts` - add agent-management routes and guards.
- `frontend/src/i18n/locales/en.ts` and `frontend/src/i18n/locales/zh.ts` - add labels.
- `frontend/src/views/auth/RegisterView.vue` - allow affiliate-code invitation validation to submit without blocking.

Generated:

- Ent generated files under `backend/ent/**` after `go generate ./ent`.
- Wire generated file `backend/cmd/server/wire_gen.go` after `go generate ./cmd/server`.

## Remote Execution Rules

For each implementation task:

- Run the focused tests listed in the task locally when they are normal unit tests and do not require Docker image builds.
- Do not run local Docker build.
- Commit the task on `hzz`.
- Push to `origin/hzz`.
- Confirm GitHub Actions status before moving to the next larger phase if the task changes build, generated code, frontend routing, or hot-path backend behavior.

---

### Task 1: Schema And Constants

**Files:**
- Modify: `backend/internal/domain/constants.go`
- Modify: `backend/internal/service/domain_constants.go`
- Modify: `backend/ent/schema/user.go`
- Create: `backend/migrations/145_hzz_agent_promotion.sql`
- Modify: `backend/internal/service/user.go`
- Modify: `backend/internal/repository/user_repo.go`
- Generated: `backend/ent/**`

- [ ] **Step 1: Add failing schema-facing tests**

Add assertions to `backend/internal/repository/user_repo_integration_test.go`:

```go
func (s *UserRepoSuite) TestAgentPromotionFieldsRoundTrip() {
	parent := s.mustCreateUser(&service.User{Email: "parent-agent@test.com", Role: service.RoleAgentLevel1, Concurrency: 50, RPMLimit: 500})
	child := s.mustCreateUser(&service.User{
		Email:                "child-user@test.com",
		Role:                 service.RoleUser,
		ParentUserID:         &parent.ID,
		AllocatedConcurrency: 7,
		AllocatedRPM:         70,
		Concurrency:          7,
		RPMLimit:             70,
	})

	got, err := s.repo.GetByID(s.ctx, child.ID)
	s.Require().NoError(err)
	s.Require().NotNil(got.ParentUserID)
	s.Require().Equal(parent.ID, *got.ParentUserID)
	s.Require().Equal(7, got.AllocatedConcurrency)
	s.Require().Equal(70, got.AllocatedRPM)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend
go test ./internal/repository -run TestUserRepoSuite/TestAgentPromotionFieldsRoundTrip -count=1
```

Expected: FAIL because `service.User.ParentUserID`, `AllocatedConcurrency`, `AllocatedRPM`, and role constants are undefined.

- [ ] **Step 3: Add constants and fields**

Implement in `backend/internal/domain/constants.go` and re-export from `backend/internal/service/domain_constants.go`:

```go
const (
	RoleAdmin       = "admin"
	RoleUser        = "user"
	RoleAgentLevel1 = "agent_level1"
	RoleAgentLevel2 = "agent_level2"
	RoleEnterprise  = "enterprise"
)
```

Add user schema fields:

```go
field.Int64("parent_user_id").Optional().Nillable(),
field.Int("allocated_concurrency").Default(0),
field.Int("allocated_rpm").Default(0),
```

Add indexes:

```go
index.Fields("parent_user_id"),
index.Fields("role", "parent_user_id"),
```

Add service fields:

```go
ParentUserID *int64
AllocatedConcurrency int
AllocatedRPM int
```

Map fields in `Create`, `Update`, and `userEntityToService`.

- [ ] **Step 4: Add migration**

Create `backend/migrations/145_hzz_agent_promotion.sql`:

```sql
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS parent_user_id BIGINT NULL,
  ADD COLUMN IF NOT EXISTS allocated_concurrency INTEGER NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS allocated_rpm INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_parent_user_id
  ON users(parent_user_id)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_role_parent_user_id
  ON users(role, parent_user_id)
  WHERE deleted_at IS NULL;

ALTER TABLE users
  ADD CONSTRAINT users_parent_user_id_fk
  FOREIGN KEY (parent_user_id) REFERENCES users(id)
  ON DELETE SET NULL;
```

- [ ] **Step 5: Regenerate Ent**

Run:

```bash
cd backend
go generate ./ent
```

Expected: generated Ent files include the new user fields and predicates.

- [ ] **Step 6: Run focused test**

Run:

```bash
cd backend
go test ./internal/repository -run TestUserRepoSuite/TestAgentPromotionFieldsRoundTrip -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit and push**

```bash
git add backend/internal/domain/constants.go backend/internal/service/domain_constants.go backend/ent/schema/user.go backend/migrations/145_hzz_agent_promotion.sql backend/internal/service/user.go backend/internal/repository/user_repo.go backend/ent
git commit -m "feat: add agent hierarchy user fields"
git push origin hzz
```

---

### Task 2: Agent Management Repository

**Files:**
- Create: `backend/internal/repository/agent_management_repo.go`
- Create: `backend/internal/repository/agent_management_repo_integration_test.go`
- Modify: `backend/internal/repository/wire.go`
- Create/Modify service port definitions in `backend/internal/service/agent_management.go`

- [ ] **Step 1: Write repository integration tests**

Create these tests with fixtures for root admin, a level 1 agent, a level 2 agent, one ordinary user, and one enterprise:

- `TestAgentManagementRepoSuite/TestListDirectChildrenByRole`: asserts direct users, direct agents, and direct enterprises are filtered by `parent_user_id` and role, and unrelated users are excluded.
- `TestAgentManagementRepoSuite/TestSumDirectChildAllocations`: asserts allocation sums include direct children only and honor `excludeChildID`.
- `TestAgentManagementRepoSuite/TestDetachChildToRootAdmin`: asserts `SetParent` moves a child to the root admin without changing role, balance, API-key ownership, or allocation fields.
- `TestAgentManagementRepoSuite/TestDeleteLevel1AgentMovesChildrenAndPromotesLevel2`: asserts deleting a direct level 1 agent soft-deletes that account, moves its direct users and enterprises to root admin, promotes its direct level 2 agents to level 1, and keeps moved agents' children attached.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/repository -run TestAgentManagementRepoSuite -count=1
```

Expected: FAIL because repository does not exist.

- [ ] **Step 3: Implement repository**

Implement methods:

```go
type AgentManagementRepository interface {
	GetRootAdmin(ctx context.Context) (*User, error)
	ListDirectChildren(ctx context.Context, parentID int64, roles []string, params pagination.PaginationParams) ([]User, *pagination.PaginationResult, error)
	SumDirectChildAllocations(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	SetParent(ctx context.Context, userID int64, parentID *int64) error
	SetRoleAndParent(ctx context.Context, userID int64, role string, parentID *int64) error
	SetAllocation(ctx context.Context, userID int64, concurrency int, rpm int) error
	DeleteLevel1AgentAndMoveChildren(ctx context.Context, agentID int64, rootAdminID int64) error
}
```

Use Ent transactions for multi-row operations.

- [ ] **Step 4: Run repository tests**

Run:

```bash
cd backend
go test ./internal/repository -run TestAgentManagementRepoSuite -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add backend/internal/repository/agent_management_repo.go backend/internal/repository/agent_management_repo_integration_test.go backend/internal/repository/wire.go backend/internal/service/agent_management.go
git commit -m "feat: add agent management repository"
git push origin hzz
```

---

### Task 3: Agent Management Service Rules

**Files:**
- Modify: `backend/internal/service/agent_management.go`
- Create: `backend/internal/service/agent_management_test.go`
- Modify: `backend/internal/service/wire.go`

- [ ] **Step 1: Write service tests**

Create these tests:

- `TestAgentManagementUpgradeRules`: asserts admin can upgrade a direct user to `agent_level1` and `enterprise`, admin cannot upgrade directly to `agent_level2`, level 1 can upgrade a direct user to `agent_level2` and `enterprise`, and level 2 can upgrade a direct user only to `enterprise`.
- `TestAgentManagementAllocationCannotExceedRemaining`: uses a manager with 100 concurrency and 1000 RPM, existing direct children using 70/700, and a child currently using 10/100; updating that child to 40/400 succeeds, while 41/401 fails.
- `TestAgentManagementDeleteRules`: asserts agent deletion of a user detaches to root admin, level 1 deletion of a level 2 agent detaches to root admin and promotes it to level 1, and admin deletion of a direct level 1 agent calls the repository delete-and-move operation.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/service -run 'TestAgentManagement(UpgradeRules|AllocationCannotExceedRemaining|DeleteRules)' -count=1
```

Expected: FAIL because methods are not implemented.

- [ ] **Step 3: Implement service**

Implement:

```go
func NewAgentManagementService(repo AgentManagementRepository, userRepo UserRepository, groupRepo GroupRepository, authCacheInvalidator APIKeyAuthCacheInvalidator) *AgentManagementService
func (s *AgentManagementService) ListDirectUsers(ctx context.Context, actorID int64) (*DirectChildrenResult, error)
func (s *AgentManagementService) ListDirectAgents(ctx context.Context, actorID int64) (*DirectChildrenResult, error)
func (s *AgentManagementService) ListDirectEnterprises(ctx context.Context, actorID int64) (*DirectChildrenResult, error)
func (s *AgentManagementService) UpdateAllocation(ctx context.Context, actorID int64, childID int64, req AllocationUpdate) (*AllocationSummary, error)
func (s *AgentManagementService) UpgradeDirectUser(ctx context.Context, actorID int64, childID int64, targetRole string) (*User, error)
func (s *AgentManagementService) DeleteDirectChild(ctx context.Context, actorID int64, childID int64) error
```

Rules:

- Actor must be admin, level 1 agent, or level 2 agent.
- Target must be a direct child unless actor is root admin deleting its direct level 1 agent.
- Admin can upgrade direct user only to `agent_level1` or `enterprise`.
- Level 1 can upgrade direct user to `agent_level2` or `enterprise`.
- Level 2 can upgrade direct user only to `enterprise`.
- Balance fields are never changed.
- Status fields are never changed.
- Invalidate API key auth cache when allocation, role, or parent changes.

- [ ] **Step 4: Run service tests**

Run:

```bash
cd backend
go test ./internal/service -run 'TestAgentManagement(UpgradeRules|AllocationCannotExceedRemaining|DeleteRules)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add backend/internal/service/agent_management.go backend/internal/service/agent_management_test.go backend/internal/service/wire.go
git commit -m "feat: enforce agent management rules"
git push origin hzz
```

---

### Task 4: Agent Management HTTP Routes

**Files:**
- Create: `backend/internal/handler/agent_management_handler.go`
- Create: `backend/internal/handler/agent_management_handler_test.go`
- Create: `backend/internal/server/routes/agent_management.go`
- Modify: `backend/internal/server/routes/user.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/handler/wire.go`
- Generated: `backend/cmd/server/wire_gen.go`

- [ ] **Step 1: Write handler tests**

Create these tests:

- `TestAgentManagementHandlerRejectsBalancePayload`: sends `balance` in an allocation request and expects HTTP 400 with no service call.
- `TestAgentManagementHandlerRoutesUseAuthenticatedUser`: sets authenticated user ID in Gin context and asserts the service receives that actor ID rather than any request-body actor.
- `TestAgentManagementHandlerUpgradePayload`: sends `{"target_role":"agent_level1"}` and asserts the handler passes `agent_level1` to the service.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/handler -run TestAgentManagementHandler -count=1
```

Expected: FAIL because handler does not exist.

- [ ] **Step 3: Implement handler and routes**

Routes:

```go
GET    /api/v1/agent-management/summary
GET    /api/v1/agent-management/direct-users
GET    /api/v1/agent-management/direct-agents
GET    /api/v1/agent-management/direct-enterprises
PUT    /api/v1/agent-management/children/:id/allocation
POST   /api/v1/agent-management/children/:id/upgrade
DELETE /api/v1/agent-management/children/:id
GET    /api/v1/agent-management/groups
PUT    /api/v1/agent-management/children/:id/groups/:group_id
DELETE /api/v1/agent-management/children/:id/groups/:group_id
```

Use existing JWT middleware and `BackendModeUserGuard`.

- [ ] **Step 4: Regenerate Wire**

Run:

```bash
cd backend
go generate ./cmd/server
```

Expected: `backend/cmd/server/wire_gen.go` includes the new service and handler.

- [ ] **Step 5: Run tests**

Run:

```bash
cd backend
go test ./internal/handler ./internal/server/routes -run 'TestAgentManagement|Test.*Routes' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit and push**

```bash
git add backend/internal/handler/agent_management_handler.go backend/internal/handler/agent_management_handler_test.go backend/internal/server/routes/agent_management.go backend/internal/server/routes/user.go backend/internal/handler/handler.go backend/internal/handler/wire.go backend/cmd/server/wire_gen.go
git commit -m "feat: add agent management API routes"
git push origin hzz
```

---

### Task 5: Invitation Ownership With Affiliate Codes

**Files:**
- Modify: `backend/internal/service/auth_service.go`
- Modify: `backend/internal/service/auth_service_register_test.go`
- Modify: `backend/internal/handler/auth_handler.go`
- Modify: `frontend/src/views/auth/RegisterView.vue`

- [ ] **Step 1: Write auth service tests**

Add these tests:

- `TestRegisterInvitationOnlyAcceptsAffiliateCodeAsInvitation`: with invitation-only mode enabled, empty `invitation_code`, and a valid `aff_code`, registration succeeds.
- `TestRegisterAssignsParentToAgentInviter`: an agent affiliate code creates the new user with `parent_user_id` equal to the agent ID.
- `TestRegisterAssignsParentToNearestAgentForOrdinaryInviter`: an ordinary user under a level 2 agent invites a new user, and the new user parent is that level 2 agent.
- `TestRegisterAssignsParentToRootAdminWhenNoAgentInChain`: an ordinary user directly under admin invites a new user, and the new user parent is root admin.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/service -run 'TestRegister(InvitationOnlyAcceptsAffiliateCodeAsInvitation|AssignsParent)' -count=1
```

Expected: FAIL because affiliate code does not satisfy invitation-only mode and parent assignment is not implemented.

- [ ] **Step 3: Implement ownership resolution**

Add a service helper:

```go
func (s *AgentManagementService) ResolveInvitationParent(ctx context.Context, inviterID int64) (*int64, error)
```

Rules:

- Admin inviter -> root admin ID.
- Agent inviter -> inviter ID.
- Ordinary user inviter -> nearest upstream agent, else root admin.

In `AuthService.RegisterWithVerification` and OAuth completion paths, when invitation-only mode is enabled:

- Accept valid official invitation code.
- Or accept valid affiliate code as invitation credential.
- Preserve existing affiliate binding behavior.
- Set `ParentUserID` before creating the user.

- [ ] **Step 4: Run auth tests**

Run:

```bash
cd backend
go test ./internal/service -run 'TestRegister(InvitationOnlyAcceptsAffiliateCodeAsInvitation|AssignsParent)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add backend/internal/service/auth_service.go backend/internal/service/auth_service_register_test.go backend/internal/handler/auth_handler.go frontend/src/views/auth/RegisterView.vue
git commit -m "feat: resolve agent ownership during invitation signup"
git push origin hzz
```

---

### Task 6: Exclusive Group Delegation And Effective Rates

**Files:**
- Create: `backend/ent/schema/agent_group_delegation.go`
- Create: `backend/migrations/146_hzz_agent_group_delegations.sql`
- Modify: `backend/internal/service/agent_management.go`
- Modify: `backend/internal/repository/agent_management_repo.go`
- Modify: `backend/internal/service/api_key_service.go`
- Modify: `backend/internal/repository/api_key_repo.go`
- Tests: `backend/internal/service/agent_management_test.go`, `backend/internal/repository/agent_management_repo_integration_test.go`
- Generated: `backend/ent/**`

- [ ] **Step 1: Write tests**

Add these tests:

- `TestAgentGroupsShowsPublicAndDelegatedExclusiveGroups`: public groups and the actor's delegated exclusive groups appear on the group/rate list.
- `TestDelegateExclusiveGroupRequiresManagerAccess`: a manager cannot delegate an exclusive group it has not received.
- `TestDelegatedExclusiveGroupHidesUpstreamRate`: child group responses contain `effective_rate` and do not contain upstream/admin cost fields.
- `TestEffectiveGroupRateUsesDirectDelegation`: a child with a delegated exclusive group receives the direct delegation rate as the effective rate.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/service -run 'TestAgent.*Group|TestEffectiveGroupRate' -count=1
```

Expected: FAIL because delegation model is not implemented.

- [ ] **Step 3: Add delegation schema**

Create Ent schema `AgentGroupDelegation` with fields `manager_user_id`, `child_user_id`, `group_id`, `rate_multiplier`, `can_delegate`, timestamps, and soft delete. Create migration `146_hzz_agent_group_delegations.sql` with table `agent_group_delegations`, foreign keys to `users` and `groups`, and a partial unique index on `(manager_user_id, child_user_id, group_id)` where `deleted_at IS NULL`.

- [ ] **Step 4: Implement delegation service**

Implement:

```go
func (s *AgentManagementService) ListMyGroups(ctx context.Context, actorID int64) ([]AgentGroupRate, error)
func (s *AgentManagementService) SetChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64, rate float64, canDelegate bool) error
func (s *AgentManagementService) RemoveChildGroupDelegation(ctx context.Context, actorID int64, childID int64, groupID int64) error
```

Responses must not include upstream original cost fields.

- [ ] **Step 5: Regenerate Ent and run tests**

Run:

```bash
cd backend
go generate ./ent
go test ./internal/service ./internal/repository -run 'TestAgent.*Group|TestEffectiveGroupRate|TestAgentManagementRepoSuite' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit and push**

```bash
git add backend/ent/schema/agent_group_delegation.go backend/migrations backend/internal/service/agent_management.go backend/internal/repository/agent_management_repo.go backend/internal/service/api_key_service.go backend/internal/repository/api_key_repo.go backend/internal/service/agent_management_test.go backend/internal/repository/agent_management_repo_integration_test.go backend/ent
git commit -m "feat: add agent group delegation"
git push origin hzz
```

---

### Task 7: Frontend Agent Management API And Routes

**Files:**
- Create: `frontend/src/api/agentManagement.ts`
- Create: `frontend/src/api/__tests__/agentManagement.spec.ts`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/stores/auth.ts`
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/components/layout/AppSidebar.vue`
- Modify: `frontend/src/i18n/locales/en.ts`
- Modify: `frontend/src/i18n/locales/zh.ts`

- [ ] **Step 1: Write frontend tests**

Create API tests that mock `apiClient` and assert `listDirectUsers`, `updateAllocation(12, { allocated_concurrency: 5, allocated_rpm: 60 })`, and `upgradeChild(12, 'agent_level1')` call `/agent-management/direct-users`, `/agent-management/children/12/allocation`, and `/agent-management/children/12/upgrade` respectively. Add router/sidebar tests asserting `canUseAgentManagement` is true for `admin`, `agent_level1`, and `agent_level2`, and false for `user`.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd frontend
pnpm vitest run src/api/__tests__/agentManagement.spec.ts src/__tests__/integration/navigation.spec.ts
```

Expected: FAIL because API/routes are absent.

- [ ] **Step 3: Implement API, roles, routes, sidebar**

Routes:

```ts
/agent/direct-users
/agent/direct-agents
/agent/direct-enterprises
/agent/groups
```

Add route meta such as:

```ts
requiresAgentManagement: true
```

Guard: allow admin, `agent_level1`, `agent_level2`.

- [ ] **Step 4: Run focused frontend tests**

Run:

```bash
cd frontend
pnpm vitest run src/api/__tests__/agentManagement.spec.ts src/__tests__/integration/navigation.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add frontend/src/api/agentManagement.ts frontend/src/api/__tests__/agentManagement.spec.ts frontend/src/router/index.ts frontend/src/stores/auth.ts frontend/src/types/index.ts frontend/src/components/layout/AppSidebar.vue frontend/src/i18n/locales/en.ts frontend/src/i18n/locales/zh.ts
git commit -m "feat: add agent management frontend routes"
git push origin hzz
```

---

### Task 8: Frontend Direct Management Pages

**Files:**
- Create: `frontend/src/views/agent/DirectUsersView.vue`
- Create: `frontend/src/views/agent/DirectAgentsView.vue`
- Create: `frontend/src/views/agent/DirectEnterprisesView.vue`
- Create: `frontend/src/views/agent/MyGroupsView.vue`
- Create: `frontend/src/views/agent/__tests__/agentManagement.spec.ts`

- [ ] **Step 1: Write page tests**

Create page tests with these assertions:

- Direct user page does not render balance, recharge, disable, or official delete actions.
- Direct user page renders allocation controls for concurrency and RPM.
- Upgrade action choices match admin, level 1 agent, and level 2 agent permissions.
- My Groups page renders effective group rates and does not render upstream cost fields.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts
```

Expected: FAIL because pages are absent.

- [ ] **Step 3: Implement pages**

Use existing components:

- `DataTable`
- `Pagination`
- `ConfirmDialog`
- common `Input`/`Select` controls

Keep actions limited to:

- allocation edit
- role upgrade where allowed
- detach/delete relationship
- group delegation in My Groups page

Do not include:

- balance update
- status toggle
- redeem code generation
- official admin delete

- [ ] **Step 4: Run page tests**

Run:

```bash
cd frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add frontend/src/views/agent
git commit -m "feat: add agent management pages"
git push origin hzz
```

---

### Task 9: API Usage Enforcement For Allocated Capacity And Effective Rates

**Files:**
- Modify: `backend/internal/repository/api_key_repo.go`
- Modify: `backend/internal/service/billing_cache_service.go`
- Modify: `backend/internal/service/api_key_service.go`
- Modify: `backend/internal/service/gateway_service.go`
- Modify: `backend/internal/service/openai_gateway_service.go`
- Tests: existing hot-path tests plus new focused service tests.

- [ ] **Step 1: Write tests**

Add these tests:

- `TestAuthSnapshotUsesAllocatedConcurrencyAndRPM`: API-key auth snapshots use allocated concurrency/RPM for managed accounts.
- `TestManagerOwnCapacityUsesRemainingAllocation`: a manager with allocated capacity and direct-child allocations receives remaining concurrency/RPM for self-use.
- `TestExclusiveGroupUsesDelegatedEffectiveRate`: exclusive group rate resolution uses the direct delegation rate for a child.
- `TestAgentBalanceDoesNotBlockChildUsage`: a child request remains eligible when its upstream agent has zero balance, as long as the child itself is eligible.

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
cd backend
go test ./internal/service ./internal/repository -run 'TestAuthSnapshotUsesAllocated|TestManagerOwnCapacity|TestExclusiveGroupUsesDelegated|TestAgentBalanceDoesNotBlock' -count=1
```

Expected: FAIL until hot-path snapshot logic is updated.

- [ ] **Step 3: Implement enforcement integration**

Rules:

- Users and agents use their effective allocated concurrency/RPM.
- A manager's own capacity is allocated capacity minus direct-child allocations.
- Agent balances are not part of downstream eligibility.
- Delegated exclusive group effective rates are returned to the user and used by billing/rate resolver.

- [ ] **Step 4: Run focused hot-path tests**

Run:

```bash
cd backend
go test ./internal/service ./internal/repository -run 'TestAuthSnapshotUsesAllocated|TestManagerOwnCapacity|TestExclusiveGroupUsesDelegated|TestAgentBalanceDoesNotBlock|TestGateway|TestBillingCache' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit and push**

```bash
git add backend/internal/repository/api_key_repo.go backend/internal/service/billing_cache_service.go backend/internal/service/api_key_service.go backend/internal/service/gateway_service.go backend/internal/service/openai_gateway_service.go backend/internal/service/*test.go backend/internal/repository/*test.go
git commit -m "feat: apply agent allocations to API usage"
git push origin hzz
```

---

### Task 10: Remote Verification And CI Gate

**Files:**
- No planned code files. If GitHub Actions does not run for `hzz`, modify `.github/workflows/backend-ci.yml` and `.github/workflows/release.yml` in a separate commit named `ci: enable hzz branch workflows`.

- [ ] **Step 1: Run local focused verification**

Run:

```bash
cd backend
go test ./internal/service ./internal/repository ./internal/handler ./internal/server/routes -count=1
```

Expected: PASS. If it fails, stop and fix the failing tests before pushing the final branch state.

Run:

```bash
cd frontend
pnpm vitest run src/api/__tests__/agentManagement.spec.ts src/views/agent/__tests__/agentManagement.spec.ts src/__tests__/integration/navigation.spec.ts
```

Expected: PASS.

- [ ] **Step 2: Push final branch state**

```bash
git status --short
git push origin hzz
```

Expected: clean working tree and pushed branch.

- [ ] **Step 3: Confirm remote Actions**

Use GitHub Actions UI or `gh` if authenticated:

```bash
gh run list --repo H-2Szz/sub2api --branch hzz --limit 5
```

Expected: latest relevant workflow for `hzz` succeeds before starting the next major feature area.

---

## Self-Review Notes

- Spec coverage: hierarchy, additive admin/agent surface, direct visibility, capabilities, upgrade rules, delete/detach rules, concurrency/RPM allocation, group/rate delegation, invitation ownership, backend/frontend shape, and tests are each covered by at least one task.
- Scope: enterprise employee management, employee UI restrictions, agent income settlement, direct agent recharge, and full multi-level billing-chain deduction remain out of scope.
- Type consistency: roles use `agent_level1`, `agent_level2`, `enterprise`; allocation fields use `allocated_concurrency` and `allocated_rpm`; routes use `/agent-management`.
