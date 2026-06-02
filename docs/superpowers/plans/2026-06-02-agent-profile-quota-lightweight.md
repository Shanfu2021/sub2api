# Agent Profile Quota Lightweight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Correct the agent quota model so official `users.concurrency` and `users.rpm_limit` remain effective runtime limits, while a new `agent_profiles` table stores only agent distributable quota pools.

**Architecture:** Keep the existing hzz agent management feature and replace only the quota-pool internals. Add `agent_profiles` with a SQL migration and raw SQL repository methods, avoiding local Ent generation on the weak machine. Service code will calculate agent remaining capacity from profile pools plus direct child effective quotas, then persist the manager's own effective quota back to `users`.

**Tech Stack:** Go service/repository tests, PostgreSQL SQL migrations, raw SQL repository helpers, Vue 3 + TypeScript + Vitest, GitHub Actions for full build and image packaging.

---

## Stage 1: Schema and Repository Boundary

**Files:**
- Create: `backend/migrations/147_hzz_agent_profiles.sql`
- Modify: `backend/internal/repository/migrations_schema_integration_test.go`
- Modify: `backend/internal/repository/agent_management_repo.go`
- Modify: `backend/internal/repository/agent_management_repo_integration_test.go`
- Modify: `backend/internal/service/agent_management.go`
- Modify: `backend/internal/service/agent_management_test.go`

- [ ] **Step 1: Add migration RED test**

In `backend/internal/repository/migrations_schema_integration_test.go`, add assertions for `agent_profiles` in `TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate`:

```go
	var agentProfilesRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.agent_profiles')").Scan(&agentProfilesRegclass))
	require.True(t, agentProfilesRegclass.Valid, "expected agent_profiles table to exist")
	requireColumn(t, tx, "agent_profiles", "user_id", "bigint", 0, false)
	requireColumn(t, tx, "agent_profiles", "pool_concurrency", "integer", 0, false)
	requireColumn(t, tx, "agent_profiles", "pool_rpm", "integer", 0, false)
	requireColumn(t, tx, "agent_profiles", "deleted_at", "timestamp with time zone", 0, true)
	requireIndex(t, tx, "agent_profiles", "agent_profiles_pkey")
	requireForeignKeyOnDelete(t, tx, "agent_profiles", "user_id", "users", "CASCADE")
```

Run:

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags integration ./internal/repository -run TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate -count=1
```

Expected: FAIL because `agent_profiles` does not exist.

- [ ] **Step 2: Add migration**

Create `backend/migrations/147_hzz_agent_profiles.sql`:

```sql
CREATE TABLE IF NOT EXISTS agent_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pool_concurrency INTEGER NOT NULL DEFAULT 0,
    pool_rpm INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_user_id
    ON agent_profiles(user_id)
    WHERE deleted_at IS NULL;
```

Run the migration test again. Expected: PASS.

- [ ] **Step 3: Add service model and repository interface methods**

In `backend/internal/service/agent_management.go`, add:

```go
type AgentProfile struct {
	UserID          int64 `json:"user_id"`
	PoolConcurrency int   `json:"pool_concurrency"`
	PoolRPM         int   `json:"pool_rpm"`
}
```

Extend `AgentManagementRepository`:

```go
	GetAgentProfile(ctx context.Context, userID int64) (*AgentProfile, error)
	UpsertAgentProfile(ctx context.Context, userID int64, poolConcurrency int, poolRPM int) error
	SumDirectChildQuotaUsage(ctx context.Context, parentID int64, excludeChildID *int64) (concurrency int, rpm int, err error)
	SetEffectiveQuota(ctx context.Context, userID int64, concurrency int, rpm int) error
```

- [ ] **Step 4: Write repository integration test**

Add to `backend/internal/repository/agent_management_repo_integration_test.go`:

```go
func (s *AgentManagementRepoSuite) TestAgentProfilePoolQuotaUsage() {
	root := s.mustCreateAgentUser("root-profile@test.com", service.RoleAdmin, nil, 0, 0)
	manager := s.mustCreateAgentUser("manager-profile@test.com", service.RoleAgentLevel1, &root.ID, 0, 0)
	ordinary := s.mustCreateAgentUser("ordinary-profile@test.com", service.RoleUser, &manager.ID, 10, 100)
	childAgent := s.mustCreateAgentUser("child-agent-profile@test.com", service.RoleAgentLevel2, &manager.ID, 0, 0)

	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, manager.ID, 100, 1000))
	s.Require().NoError(s.repo.UpsertAgentProfile(s.ctx, childAgent.ID, 30, 300))

	profile, err := s.repo.GetAgentProfile(s.ctx, manager.ID)
	s.Require().NoError(err)
	s.Require().NotNil(profile)
	s.Require().Equal(100, profile.PoolConcurrency)
	s.Require().Equal(1000, profile.PoolRPM)

	concurrency, rpm, err := s.repo.SumDirectChildQuotaUsage(s.ctx, manager.ID, nil)
	s.Require().NoError(err)
	s.Require().Equal(40, concurrency)
	s.Require().Equal(400, rpm)

	s.Require().NoError(s.repo.SetEffectiveQuota(s.ctx, ordinary.ID, 25, 250))
	updated, err := s.client.User.Get(s.ctx, ordinary.ID)
	s.Require().NoError(err)
	s.Require().Equal(25, updated.Concurrency)
	s.Require().Equal(250, updated.RpmLimit)
}
```

Run:

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags integration ./internal/repository -run 'TestAgentManagementRepoSuite/TestAgentProfilePoolQuotaUsage' -count=1
```

Expected: FAIL until raw SQL methods are implemented.

- [ ] **Step 5: Implement raw SQL repository methods**

In `backend/internal/repository/agent_management_repo.go`, implement the four new methods using `txAwareSQLExecutor(ctx, r.sql, r.client)`.

`GetAgentProfile` query:

```sql
SELECT user_id, pool_concurrency, pool_rpm
FROM agent_profiles
WHERE user_id = $1 AND deleted_at IS NULL
```

`UpsertAgentProfile` statement:

```sql
INSERT INTO agent_profiles (user_id, pool_concurrency, pool_rpm, created_at, updated_at)
VALUES ($1, CASE WHEN $2 < 0 THEN 0 ELSE $2 END, CASE WHEN $3 < 0 THEN 0 ELSE $3 END, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (user_id) DO UPDATE SET
    pool_concurrency = EXCLUDED.pool_concurrency,
    pool_rpm = EXCLUDED.pool_rpm,
    updated_at = CURRENT_TIMESTAMP,
    deleted_at = NULL
```

`SumDirectChildQuotaUsage` query:

```sql
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
  AND ($2::bigint IS NULL OR u.id <> $2::bigint)
```

`SetEffectiveQuota` statement:

```sql
UPDATE users
SET concurrency = CASE WHEN $2 < 0 THEN 0 ELSE $2 END,
    rpm_limit = CASE WHEN $3 < 0 THEN 0 ELSE $3 END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
```

Run repository tests. Expected: PASS.

- [ ] **Step 6: Update service test stub with new interface methods**

Add an `agentProfiles map[int64]AgentProfile` to `agentManagementRepoStub` and implement the same four interface methods in-memory.

Run:

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags unit ./internal/service -run TestAgentManagement -count=1
```

Expected: compile and PASS after stub is updated, even before service behavior is fully switched.

- [ ] **Step 7: Commit and push Stage 1**

```bash
cd /root/sub2api
git add backend/migrations/147_hzz_agent_profiles.sql backend/internal/repository/migrations_schema_integration_test.go backend/internal/repository/agent_management_repo.go backend/internal/repository/agent_management_repo_integration_test.go backend/internal/service/agent_management.go backend/internal/service/agent_management_test.go
git commit -m "feat: add agent profile quota persistence"
git push origin hzz
```

---

## Stage 2: Service Quota Semantics

**Files:**
- Modify: `backend/internal/service/agent_management.go`
- Modify: `backend/internal/service/agent_management_test.go`
- Modify: `backend/internal/handler/agent_management_handler.go`
- Modify: `backend/internal/handler/agent_management_handler_test.go`

- [ ] **Step 1: Rename request semantics**

Change `AllocationUpdate` to:

```go
type AllocationUpdate struct {
	Concurrency int `json:"concurrency"`
	RPM         int `json:"rpm"`
}
```

Keep accepting legacy JSON keys in handler during this stage by mapping `allocated_concurrency` to `Concurrency` and `allocated_rpm` to `RPM` if present.

- [ ] **Step 2: Add service RED tests**

Add tests proving:

- Direct ordinary users display and update `users.concurrency/rpm_limit`.
- Agent manager capacity comes from `agent_profiles`, not `users.allocated_*`.
- Updating a child user recalculates the manager's own `users.concurrency/rpm_limit`.
- Admin quota remains unlimited.

Run:

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags unit ./internal/service -run 'TestAgentManagement(DirectUsersUseEffectiveQuotaFields|AgentPoolControlsManagerCapacity|AdminAllocationIsUnconstrained)' -count=1
```

Expected: FAIL until service uses profile pools.

- [ ] **Step 3: Switch service calculations**

Replace `managerCapacity(actor)` with service method `managerCapacity(ctx, actor)`:

- Admin returns `(0, 0, nil)` and `UnlimitedCapacity = true`.
- Agents load `repo.GetAgentProfile(actor.ID)`.
- Missing agent profile means zero pool.

Replace calls to `SumDirectChildAllocations` with `SumDirectChildQuotaUsage`.

Replace `SetAllocation` for ordinary/enterprise children with `SetEffectiveQuota`.

After non-admin manager changes child quota, recalculate manager effective quota:

```text
manager.users.concurrency = max(profile.pool_concurrency - direct child usage, 0)
manager.users.rpm_limit = max(profile.pool_rpm - direct child usage, 0)
```

- [ ] **Step 4: Upgrade creates profile**

Change upgrade request to:

```go
type AgentUpgradeInput struct {
	TargetRole      string `json:"target_role"`
	PoolConcurrency int    `json:"pool_concurrency"`
	PoolRPM         int    `json:"pool_rpm"`
}
```

When upgrading to `agent_level1` or `agent_level2`, call `UpsertAgentProfile` and recalculate both child agent and non-admin parent agent.

When upgrading to `enterprise`, do not create a profile.

- [ ] **Step 5: Handler tests**

Update handler fake service and tests so:

- `PUT /children/:id/allocation` accepts `concurrency/rpm`.
- Legacy `allocated_concurrency/allocated_rpm` remains accepted during compatibility.
- `POST /children/:id/upgrade` sends `target_role/pool_concurrency/pool_rpm`.

Run:

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags unit ./internal/service ./internal/handler -run 'TestAgentManagement|TestAgentManagementHandler' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit and push Stage 2**

```bash
cd /root/sub2api
git add backend/internal/service/agent_management.go backend/internal/service/agent_management_test.go backend/internal/handler/agent_management_handler.go backend/internal/handler/agent_management_handler_test.go
git commit -m "feat: use agent profile quota pools"
git push origin hzz
```

---

## Stage 3: Frontend Field Sources

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/agentManagement.ts`
- Modify: `frontend/src/api/__tests__/agentManagement.spec.ts`
- Modify: `frontend/src/views/agent/AgentDirectChildrenView.vue`
- Modify: `frontend/src/views/agent/__tests__/agentManagement.spec.ts`

- [ ] **Step 1: Add frontend RED tests**

Update view tests to prove:

- Direct users use `concurrency/rpm_limit` even when `allocated_*` is zero.
- Direct enterprises use `concurrency/rpm_limit`.
- Direct agents use `pool_concurrency/pool_rpm`.
- Save allocation sends `concurrency/rpm`.
- Upgrade to agent sends pool fields.

Run:

```bash
cd /root/sub2api/frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts src/api/__tests__/agentManagement.spec.ts
```

Expected: FAIL until UI uses correct field sources.

- [ ] **Step 2: Update frontend types and API**

Add to `AgentManagedUser`:

```ts
pool_concurrency: number
pool_rpm: number
```

Change `AgentAllocationUpdate`:

```ts
export interface AgentAllocationUpdate {
  concurrency: number
  rpm: number
}
```

Change upgrade API to accept:

```ts
export interface AgentUpgradeRequest {
  target_role: AgentUpgradeTargetRole
  pool_concurrency?: number
  pool_rpm?: number
}
```

- [ ] **Step 3: Update view field source**

In `AgentDirectChildrenView.vue`:

- For `kind === 'agents'`, draft from `pool_concurrency/pool_rpm`.
- For `users` and `enterprises`, draft from `concurrency/rpm_limit`.
- Save sends `concurrency/rpm`.
- Upgrade agent button opens or uses a pool-value flow before calling API. Minimum implementation can reuse current row draft values as the pool payload for agent upgrade.

- [ ] **Step 4: Run frontend tests**

```bash
cd /root/sub2api/frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts src/api/__tests__/agentManagement.spec.ts
```

Expected: PASS.

- [ ] **Step 5: Commit and push Stage 3**

```bash
cd /root/sub2api
git add frontend/src/types/index.ts frontend/src/api/agentManagement.ts frontend/src/api/__tests__/agentManagement.spec.ts frontend/src/views/agent/AgentDirectChildrenView.vue frontend/src/views/agent/__tests__/agentManagement.spec.ts
git commit -m "feat: separate agent pool quota UI"
git push origin hzz
```

---

## Stage 4: Remote Build, Deploy, and Verify

**Files:**
- No source changes expected.

- [ ] **Step 1: Final targeted verification**

```bash
cd /root/sub2api/backend
/tmp/go1.26.3/bin/go test -tags unit ./internal/service ./internal/handler -run 'TestAgentManagement|TestAdminService_CreateUser' -count=1
/tmp/go1.26.3/bin/go test -tags integration ./internal/repository -run 'TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate|TestAgentManagementRepoSuite' -count=1
cd /root/sub2api/frontend
pnpm vitest run src/views/agent/__tests__/agentManagement.spec.ts src/api/__tests__/agentManagement.spec.ts
```

Expected: PASS.

- [ ] **Step 2: Wait for GitHub Actions on latest `hzz`**

Use GitHub CLI:

```bash
cd /root/sub2api
LATEST_SHA="$(git rev-parse HEAD)"
for i in $(seq 1 80); do
  gh run list --branch hzz --commit "$LATEST_SHA" --limit 10 --json name,status,conclusion,url \
    --jq '.[] | [.name,.status,(.conclusion // "-"),.url] | @tsv'
  if gh run list --branch hzz --commit "$LATEST_SHA" --limit 10 --json status,conclusion \
    --jq 'all(.[]; .status == "completed" and .conclusion == "success")' | grep -q true; then
    exit 0
  fi
  sleep 30
done
exit 1
```

Expected: CI, Security Scan, and HZZ Image are success.

- [ ] **Step 3: Deploy GitHub-built image locally**

```bash
docker inspect sub2api --format '{{range .Config.Env}}{{println .}}{{end}}' | grep -v '^PATH=' > /tmp/sub2api-env.list
docker inspect sub2api --format '{{range $k,$v := .NetworkSettings.Networks}}{{println $k}}{{end}}' | head -n1 > /tmp/sub2api-network.txt
docker pull ghcr.io/h-2szz/sub2api:hzz
docker stop sub2api
docker rm sub2api
docker run -d --name sub2api --restart unless-stopped --network "$(cat /tmp/sub2api-network.txt)" -p 0.0.0.0:8080:8080 -v /root/sub2api-deploy/data:/app/data --env-file /tmp/sub2api-env.list ghcr.io/h-2szz/sub2api:hzz
```

- [ ] **Step 4: Verify deployed behavior**

```bash
curl -fsS http://127.0.0.1:8080/health
docker exec sub2api-postgres sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select to_regclass('\''public.agent_profiles'\'') as agent_profiles;"'
docker exec sub2api-postgres sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select id,email,role,parent_user_id,concurrency,rpm_limit,allocated_concurrency,allocated_rpm from users where deleted_at is null order by id desc limit 8;"'
```

Manual checks:

- Native admin-created user with 10 concurrency displays 10 in direct users.
- Ordinary users have no `agent_profiles` row.
- Upgraded agents have `agent_profiles` pool rows.
- Agent remaining quota decreases after assigning quota to a child.
