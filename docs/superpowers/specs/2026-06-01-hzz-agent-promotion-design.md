# HZZ Agent Promotion Design

## Goal

Build an additive agent promotion and resource-distribution system on top of the official Sub2API baseline. The official admin user management, official group management, official redeem-code flow, and official payment flow remain upstream-compatible. All new behavior lives behind separate agent-management routes, services, and frontend pages.

This phase includes affiliate/referral codes as valid invitation credentials for invitation-only registration. It does not implement enterprise employee management, agent balance chains, agent income settlement, or direct agent-to-user recharge.

## Execution Constraints

- Keep `main` equal to the official upstream baseline.
- Develop on `hzz`.
- Commit each completed stage and push to `origin/hzz`.
- Run build and image work through GitHub Actions, not local Docker builds.
- The currently running local official baseline should not be replaced unless explicitly requested.

## Roles And Hierarchy

Supported roles for this phase:

- `admin`
- `agent_level1`
- `agent_level2`
- `enterprise`
- `user`

Supported hierarchy:

```text
admin -> level 1 agent -> level 2 agent -> user
admin -> level 1 agent -> user
admin -> level 1 agent -> enterprise
admin -> level 1 agent -> level 2 agent -> enterprise
admin -> user
admin -> enterprise
```

There is no level 3 agent. A level 2 agent can never upgrade another account into an agent.

## Additive Management Surface

Add a new management area shared by admins and agents. It is separate from the official admin user management page.

Pages:

- Direct Users
- Direct Agents
- Direct Enterprises
- My Groups And Rates

The official admin user management page keeps the upstream global behavior. Admins can still use that page however the official application allows. The new management area intentionally applies direct-child rules even for admins.

## Direct Visibility

In the new management area:

- Admins see only accounts whose `parent_user_id` is the root admin account.
- Level 1 agents see only accounts whose `parent_user_id` is their own user ID.
- Level 2 agents see only accounts whose `parent_user_id` is their own user ID.

Direct Users shows ordinary users only.

Direct Agents shows direct agents only. A level 1 agent can have level 2 direct agents. A level 2 agent cannot have agent children, so this page may be hidden or show an empty state.

Direct Enterprises shows enterprise accounts only. Enterprise employee management is outside this phase.

## Agent Capabilities

Agents are promotion and distribution accounts. They do not control downstream balances.

Allowed in the additive management area:

- View direct users, direct agents, and direct enterprises.
- Allocate concurrency to direct children.
- Allocate RPM to direct children.
- Upgrade direct ordinary users according to hierarchy rules.
- Remove direct-child relationships according to the deletion rules below.
- View their own available public groups and delegated exclusive groups.
- Delegate exclusive groups they received to direct children.
- Set downstream rates for delegated exclusive groups.

Not allowed in the additive management area:

- Recharge a child account directly.
- Deduct child balance.
- Set child balance.
- Generate redeem codes.
- Disable or enable child accounts.
- Physically delete child accounts, except the admin's special level 1 agent deletion rule.
- View or manage non-direct descendants.
- See upstream exclusive-group cost.

Agent balance does not affect downstream usage. If an agent has no balance, direct and indirect children can still use API access as long as their own official account state, balance, keys, group access, and rate limits allow it.

## Upgrade Rules

Only direct ordinary users can be upgraded in the additive management area.

Admin:

- Can upgrade a direct ordinary user to `agent_level1`.
- Can upgrade a direct ordinary user to `enterprise`.
- Cannot upgrade a direct ordinary user directly to `agent_level2`.

Level 1 agent:

- Can upgrade a direct ordinary user to `agent_level2`.
- Can upgrade a direct ordinary user to `enterprise`.

Level 2 agent:

- Can upgrade a direct ordinary user to `enterprise`.
- Cannot upgrade any account into an agent.

Upgrading preserves the account's balance, API keys, allocated concurrency, allocated RPM, group access, and other user-owned state. The upgrade changes the role and keeps the same parent.

## Deletion And Detach Rules

Deletion behavior in the additive management area is intentionally different from the official admin user management page.

Admin deleting a direct level 1 agent:

- The level 1 agent account itself is deleted using the application's normal user deletion behavior, preferably soft delete.
- Direct ordinary users of that deleted agent become direct children of the root admin.
- Direct enterprises of that deleted agent become direct children of the root admin.
- Direct level 2 agents of that deleted agent become direct children of the root admin and are promoted to `agent_level1`.
- Descendants below moved agents stay attached to those moved agents.
- Moved accounts keep balances, API keys, allocated resources, and delegated groups.

Admin deleting a direct ordinary user or direct enterprise:

- The account is detached to root admin. Because it is already under root admin, the operation should be idempotent and non-destructive in the additive management area.

Agent deleting a direct ordinary user:

- The user is not deleted.
- Set `parent_user_id` to the root admin user ID.
- Role remains `user`.

Agent deleting a direct enterprise:

- The enterprise is not deleted.
- Set `parent_user_id` to the root admin user ID.
- Role remains `enterprise`.

Level 1 agent deleting a direct level 2 agent:

- The level 2 agent is not deleted.
- Set `parent_user_id` to the root admin user ID.
- Promote role from `agent_level2` to `agent_level1`.
- Its own direct children remain attached.

Level 2 agents cannot have direct agent children.

## Concurrency And RPM Allocation

Each managed account has assigned capacity:

- `allocated_concurrency`
- `allocated_rpm`

The existing user concurrency and RPM enforcement should use these assigned values for agent-managed accounts, or these fields should be kept synchronized with the existing enforcement fields if the official application already has them.

For agents, remaining capacity is calculated from direct-child allocations:

```text
remaining_concurrency = allocated_concurrency - sum(direct_children.allocated_concurrency)
remaining_rpm = allocated_rpm - sum(direct_children.allocated_rpm)
```

When a manager changes a direct child's allocation:

```text
new_child_allocation <= manager_remaining + old_child_allocation
```

Allocation changes must be effective immediately for API usage. A manager's own usable concurrency/RPM is the remaining capacity after direct-child allocations.

Admin root capacity is special. The root admin is not a quota pool and does not distribute from the admin account's own concurrency or RPM fields. Admins can assign any non-negative concurrency/RPM values to direct children in the additive agent-management area. Summary responses should make this explicit, for example with an `unlimited_capacity` flag, and the frontend should not show admin users a remaining-capacity pool.

## Groups And Rates

Public groups:

- Are visible to all eligible users.
- Do not require propagation.
- Show public effective rates.

Exclusive groups:

- Must be delegated from an upstream manager.
- A manager can delegate only exclusive groups that are effective for that manager.
- A manager can set the downstream rate for direct children.
- A child sees only the rate assigned to that child.
- A child never sees upstream cost, admin cost, or intermediate margin.

My Groups And Rates page shows:

- Public groups available to the current account.
- Exclusive groups delegated to the current account.
- The current account's effective rate for each group.
- Whether each exclusive group can be delegated further.

API usage for a delegated exclusive group deducts from the using account according to the user's effective rate. Agent balance is not checked for downstream use in this phase.

## Invitation Registration Ownership

The deployment will use invitation-only registration.

Valid invitation credentials:

- Official invitation redeem codes.
- Affiliate/referral codes.

Ownership resolution:

- If the inviter is admin, the new user becomes a direct child of the root admin.
- If the inviter is `agent_level1` or `agent_level2`, the new user becomes a direct child of that agent.
- If the inviter is an ordinary user, walk up the parent chain and assign the new user to the nearest upstream agent.
- If the ordinary user's parent chain has no agent, assign the new user to the root admin.
- Enterprise-specific invitation behavior is outside this phase unless the inviter is already represented as an enterprise account; detailed employee ownership will be designed later.

Using an affiliate/referral code as an invitation credential does not require implementing agent income settlement in this phase. Existing affiliate binding can continue to run if enabled, but it is separate from agent hierarchy ownership.

## Backend Shape

Add focused services and routes instead of extending official admin handlers broadly:

- `AgentManagementService`
- `AgentManagementHandler`
- Additive routes such as `/api/v1/agent-management/...`

Representative endpoints:

- `GET /summary`
- `GET /direct-users`
- `GET /direct-agents`
- `GET /direct-enterprises`
- `PUT /children/:id/allocation`
- `POST /children/:id/upgrade`
- `DELETE /children/:id`
- `GET /groups`
- `PUT /children/:id/groups/:group_id`
- `DELETE /children/:id/groups/:group_id`

Authorization should accept admin, level 1 agent, and level 2 agent, then apply direct-child and role-specific rules inside the service.

## Frontend Shape

Add a new sidebar section for users who are admin, level 1 agent, or level 2 agent.

Pages:

- `frontend/src/views/agent/DirectUsersView.vue`
- `frontend/src/views/agent/DirectAgentsView.vue`
- `frontend/src/views/agent/DirectEnterprisesView.vue`
- `frontend/src/views/agent/MyGroupsView.vue`

The direct-management tables can reuse admin user-table patterns visually, but actions are reduced to the allowed additive operations. Balance, disable, and official delete actions must not appear here.

## Testing Strategy

Backend tests:

- Role constants and parent assignment.
- Direct-only list visibility.
- Admin can upgrade direct users to level 1 agent, not level 2 agent.
- Level 1 agent can upgrade direct users to level 2 agent or enterprise.
- Level 2 agent can upgrade direct users to enterprise only.
- Agent cannot allocate more concurrency/RPM than remaining capacity plus the child's old allocation.
- Manager own remaining capacity changes when child allocations change.
- Agent deletion/detach semantics.
- Admin deleting direct level 1 agent moves children and promotes direct level 2 agents.
- Affiliate/referral code can satisfy invitation-only registration.
- Invited user ownership resolves to the nearest upstream agent or root admin.
- Exclusive group delegation hides upstream cost and exposes only effective rates.

Frontend tests:

- New management routes are visible only to admin and agent roles.
- Direct pages call additive management endpoints, not official admin user endpoints.
- Balance and disable actions are absent from additive management pages.
- Upgrade actions match current role permissions.
- My Groups And Rates displays public groups and delegated exclusive groups without upstream cost fields.

## Out Of Scope For This Phase

- Enterprise employee management.
- Employee restricted UI.
- Agent income settlement.
- Agent balance-chain enforcement.
- Direct manual recharge by agents.
- Official admin user management behavior changes.
- Full multi-level billing-chain deductions.
