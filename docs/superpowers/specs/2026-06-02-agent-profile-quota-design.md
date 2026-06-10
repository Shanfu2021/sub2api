# Agent Profile Quota Design

## Goal

Add a dedicated agent extension model so the official `users` quota fields keep their original meaning while agents get a separate quota pool for downstream allocation.

## Confirmed Model

- The official `users` table remains the source of truth for an account's effective runtime limits.
- `users.concurrency` is the account's current usable concurrency.
- `users.rpm_limit` is the account's current usable RPM.
- Ordinary users do not have an agent quota pool.
- An agent quota pool is created only when a direct ordinary user is upgraded to an agent.
- Agents allocate from their own quota pool to direct children.
- A child user's effective runtime quota is written to `users.concurrency` and `users.rpm_limit`.
- A child agent's allocatable pool is written to the new agent extension table.
- An agent's own effective runtime quota is recalculated as its pool minus the quota it has allocated to direct children.
- Admins are not quota-limited by this allocation system.

## Data Model

Create a dedicated table, tentatively named `agent_profiles`.

Fields:

- `user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE`
- `pool_concurrency INTEGER NOT NULL DEFAULT 0`
- `pool_rpm INTEGER NOT NULL DEFAULT 0`
- `created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`
- `updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`
- `deleted_at TIMESTAMP NULL`

`users.parent_user_id` remains the ownership tree:

- Admin direct users point to the root admin user.
- Level 1 agents point to the root admin user.
- Level 2 agents point to their level 1 agent.
- Users and enterprises point to their direct owner.
- If a level 2 agent is detached from a level 1 agent, it becomes a level 1 agent under the root admin.

The existing `users.allocated_concurrency` and `users.allocated_rpm` fields should not be used as the ordinary-user display source. They can remain for compatibility until a later cleanup, but new behavior must not depend on them for the quota-pool model.

## Allocation Rules

### Admin

Admins can create direct ordinary users from the native admin user page or the agent-management page.

When an admin creates or edits a direct ordinary user:

- Write effective quota directly to `users.concurrency` and `users.rpm_limit`.
- Do not create an `agent_profiles` row.

When an admin upgrades a direct ordinary user to a level 1 agent:

- Create or update that user's `agent_profiles` row with the admin-provided pool.
- Recalculate the upgraded agent's effective runtime quota as `pool - downstream allocations`.
- Admin does not consume any admin quota.

### Level 1 Agent

Level 1 agents can create direct ordinary users, direct enterprises, and level 2 agents within their own pool.

When allocating to ordinary users or enterprises:

- Check the allocation does not exceed the level 1 agent's remaining pool.
- Write the child's effective quota to `users.concurrency` and `users.rpm_limit`.
- Do not create an `agent_profiles` row for ordinary users or enterprises.
- Recalculate the level 1 agent's own `users.concurrency` and `users.rpm_limit`.

When upgrading a direct ordinary user to level 2 agent:

- Check the new pool does not exceed the level 1 agent's remaining pool.
- Create or update the level 2 agent's `agent_profiles` row.
- Recalculate both the level 1 agent's effective quota and the new level 2 agent's effective quota.

### Level 2 Agent

Level 2 agents can allocate to direct ordinary users and direct enterprises within their own pool.

Level 2 agents cannot create level 3 agents.

When allocating:

- Check the allocation does not exceed the level 2 agent's remaining pool.
- Write the child's effective quota to `users.concurrency` and `users.rpm_limit`.
- Recalculate the level 2 agent's own `users.concurrency` and `users.rpm_limit`.

## Remaining Pool Calculation

For an agent:

```text
allocated_concurrency = sum of direct ordinary/enterprise child users.concurrency
                      + sum of direct child agents agent_profiles.pool_concurrency

remaining_concurrency = agent_profiles.pool_concurrency - allocated_concurrency

effective_agent_concurrency = remaining_concurrency
```

RPM uses the same formula with `rpm_limit` and `pool_rpm`.

The recalculated effective agent quota is persisted back to `users.concurrency` and `users.rpm_limit`.

## Page Behavior

Agent-management direct user, direct enterprise, and direct agent pages remain additive UI. The original admin user management page is not removed or replaced.

Direct ordinary user page:

- Show and edit `users.concurrency` and `users.rpm_limit`.
- Do not show a quota-pool concept.

Direct enterprise page:

- Same quota behavior as ordinary users unless later enterprise-specific rules are added.

Direct agent page:

- Show and edit the child agent's `agent_profiles.pool_concurrency` and `agent_profiles.pool_rpm`.
- Also show the child agent's remaining/effective quota if useful, derived from its downstream allocations.

Manager summary:

- Admin shows unlimited allocation status.
- Agent shows pool, allocated downstream quota, and remaining quota.

## Invitation Ownership

The existing invitation-parent rule remains:

- If invited by an admin, the new user belongs to the root admin.
- If invited by an agent, the new user belongs to that agent.
- If invited by an ordinary user, the new user belongs to the nearest upstream agent.
- If no upstream agent exists, the new user belongs to the root admin.

New invite-created ordinary users do not get an `agent_profiles` row.

## Deletion and Detach Rules

Detaching ordinary users or enterprises from a manager moves them to the root admin. Their effective `users.concurrency` and `users.rpm_limit` remain unchanged.

Detaching a level 2 agent from a level 1 agent moves it under the root admin and promotes it to level 1. Its `agent_profiles` row remains unchanged.

Deleting a level 1 agent from the admin agent-management page deletes that level 1 agent account. Its direct users and enterprises move to the root admin. Its direct level 2 agents move to the root admin and become level 1 agents. Their `agent_profiles` rows remain unchanged.

## Error Handling

Reject negative pool or quota values.

Reject agent allocations that exceed the manager's remaining pool.

Admins bypass quota-limit checks.

Reject unsupported upgrades:

- Admin can upgrade direct ordinary user to level 1 agent or enterprise.
- Level 1 agent can upgrade direct ordinary user to level 2 agent or enterprise.
- Level 2 agent can upgrade direct ordinary user to enterprise only.

## Testing Strategy

Backend tests:

- Native admin-created ordinary users display their effective `users.concurrency` and `users.rpm_limit`.
- Ordinary users do not create agent profile rows.
- Upgrading a direct ordinary user to an agent creates an agent profile row.
- Agent child allocation checks use the manager's profile pool, not `users.allocated_*`.
- Agent effective quota is recalculated after child allocation changes.
- Direct child listing returns ordinary user effective quota and child agent pool quota using the correct source.
- Detach and delete preserve or move agent profiles according to the rules.

Frontend tests:

- Direct ordinary users render effective concurrency/RPM from the official fields.
- Direct agents render editable pool fields.
- Admin pages show unlimited allocation status.
- Agent pages show pool remaining status and block obvious over-allocation before submitting.

## Deployment

Implementation stays on branch `hzz`.

Each small stage should be committed and pushed to `origin/hzz`.

Heavy builds and image packaging should run on GitHub Actions. Local verification should stay focused on targeted tests because the machine is hardware-constrained.

After a successful GitHub image build, deploy the `hzz` image locally for visual testing.
