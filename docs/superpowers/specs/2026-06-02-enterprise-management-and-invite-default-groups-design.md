# Enterprise Management And Invite Default Groups Design

## Goal

Add enterprise management on top of the existing agent/admin hierarchy, and extend the existing agent "groups and rates" page so admins and agents can define default exclusive groups for users registered through their affiliate invitation codes.

This design is additive. The native admin user-management surface remains available and keeps its broad authority, while the new enterprise/agent surfaces enforce scoped management rules.

## Existing Context

- `users.role` already supports `enterprise`; this remains the enterprise account role.
- Add a new `employee` role for enterprise employees. Employees are users, but they are not normal registered users for ownership, recharge, invitation, rebate, or subscription-entry purposes.
- `users.parent_user_id` is the ownership tree:
  - admin root owns first-level agents, ordinary users, and enterprises.
  - agents own direct users, second-level agents, and enterprises.
  - enterprises own employees.
- Agent quota distribution already uses `agent_profiles` as the manager pool, with `users.concurrency` and `users.rpm_limit` representing the manager's current remaining self-use quota.
- Exclusive group propagation already uses `agent_group_delegations` plus `user_allowed_groups`, and billing resolves delegated rates before user-specific group rates.

## Enterprise Accounts

An enterprise is a `users` row with role `enterprise`. It can:

- use API keys like a normal user;
- redeem admin-created balance codes for itself;
- participate in affiliate invitation/rebate as a normal inviter;
- manage employees from a new "employee management" sidebar entry;
- view its own public and delegated exclusive groups;
- propagate its available exclusive groups to employees without repricing.

An enterprise cannot:

- manage agents;
- create redeem codes;
- create users outside its employee list;
- configure invite-default exclusive groups;
- reprice groups for employees.

Enterprise invitation/rebate behavior:

- If an enterprise invites a new user through affiliate invitation, the new user's direct owner is the enterprise's own upstream owner.
- If the enterprise is admin-owned, the invitee is admin direct.
- If the enterprise is agent-owned, the invitee is that agent's direct user.
- The affiliate rebate still belongs to the enterprise account as inviter.
- Invitees are not enterprise employees.

## Employees

An employee is a `users` row with role `employee` and `parent_user_id = enterprise_id`.

Employees are not created by public registration, OAuth registration, affiliate invitation, invitation redeem codes, or admin invite registration. They can only be created by their enterprise through the employee-management API.

Employee capabilities:

- can create and use API keys;
- can view usage;
- can view available groups;
- can edit profile and normal account basics;
- use their own assigned balance, concurrency, RPM, and groups.

Employee restrictions:

- cannot recharge;
- cannot redeem codes;
- cannot invite or participate in affiliate/rebate as inviter;
- cannot access "my subscriptions";
- cannot manage employees, enterprises, or agents.

## Enterprise Pools

Add `enterprise_profiles` to store the enterprise quota pool:

- `user_id`
- `pool_concurrency`
- `pool_rpm`
- `created_at`
- `updated_at`
- `deleted_at`

Balance does not need a separate pool table:

- enterprise `users.balance` is its current unallocated/self-use balance;
- employee `users.balance` is the employee's assigned balance.

Concurrency and RPM follow the same pattern as agents:

- enterprise profile stores total assignable pool;
- enterprise `users.concurrency` and `users.rpm_limit` store remaining self-use quota;
- employee `users.concurrency` and `users.rpm_limit` store assigned quota.

`0` for concurrency or RPM keeps the existing meaning: unlimited.

## Employee Allocation

Enterprise employee management edits employees directly, without redeem codes.

Creating an employee:

- enterprise provides email, username, password, initial balance, concurrency, and RPM;
- initial balance must not exceed enterprise current balance;
- initial concurrency/RPM must not exceed enterprise remaining pool unless the enterprise pool dimension is unlimited;
- assigned balance is moved from enterprise to employee in the same transaction;
- assigned concurrency/RPM is written to employee fields and enterprise remaining fields are recalculated.

Editing an employee:

- balance is updated by moving only the delta:
  - employee 10 -> 30: enterprise -20, employee +20;
  - employee 30 -> 10: employee -20, enterprise +20;
  - employee -> 0 is allowed and returns all employee balance to enterprise.
- concurrency/RPM are updated as direct employee fields;
- enterprise remaining quota is recalculated after each change;
- increases beyond enterprise remaining quota are rejected;
- reductions are allowed and return capacity to the enterprise.

Deleting an employee:

- employee remaining balance is returned to the enterprise;
- employee assigned concurrency/RPM are released by recalculating the enterprise remaining quota;
- employee is hard-deleted;
- employee API keys, allowed groups, group rates, subscriptions, and related auth/cache data are cleaned or invalidated.

## Enterprise Group Propagation

Enterprise group propagation reuses the existing delegated group model, but without repricing:

- enterprise can list its public groups and delegated exclusive groups.
- public groups do not need explicit propagation.
- exclusive groups can be assigned to employees only if the enterprise can see/use that group.
- when assigning an exclusive group to an employee:
  - create/update `agent_group_delegations` from enterprise to employee;
  - use the enterprise's effective group rate as the employee rate;
  - set `can_delegate=false`;
  - add `user_allowed_groups` for the employee.
- enterprise UI must not expose a rate input for employee group propagation.
- if the enterprise loses a delegated exclusive group from its upstream owner, the removal must cascade to employees.

Billing behavior:

- employees are billed from their own assigned balance.
- employee usage does not directly deduct enterprise balance.
- employees using enterprise-propagated exclusive groups are charged at the enterprise's effective rate.

## Enterprise Delete Rules

Native admin user management:

- deleting an `enterprise` hard-deletes the enterprise and all of its employees;
- if the enterprise was agent-owned, the agent's allocated enterprise quota is released/recalculated;
- no balance is returned to an upstream owner because the enterprise account itself is being deleted.

Admin direct enterprise management:

- deleting an admin-direct enterprise has the same behavior as native admin deletion:
  - hard-delete enterprise;
  - hard-delete employees;
  - clean associated enterprise/employee relationship data.

Agent direct enterprise management:

- deleting an agent-direct enterprise is not a real delete;
- it only moves the enterprise to admin direct ownership;
- employees remain under the enterprise;
- enterprise balance, employee balances, quota pools, RPM, groups, and employee settings remain unchanged;
- the agent's remaining quota is recalculated after the enterprise is detached.

Enterprise employee management:

- enterprise deleting its employee hard-deletes that employee after returning employee balance and quota to the enterprise.

Admin native user management deleting an employee:

- hard-delete employee;
- return employee remaining balance and quota to the employee's enterprise first;
- behavior is equivalent to the enterprise deleting its own employee, but performed by admin.

Agents:

- agents cannot see or delete employees directly.

## Enterprise Disable And Enable Rules

Only native admin user management can disable/enable enterprises globally.

Disabling an enterprise:

- sets the enterprise status to `disabled`;
- disables all currently active employees under that enterprise;
- records which employees were disabled because of the enterprise-level disable;
- invalidates API-key/auth caches for the enterprise and all affected employees.

Enabling an enterprise:

- sets the enterprise status to `active`;
- re-enables only employees that were disabled because of the enterprise-level disable;
- employees that were manually disabled before or after the enterprise disable remain disabled;
- clears the enterprise-disable marker only for employees restored by this operation.

Implementation needs a durable marker, such as an `enterprise_employee_states` table or an `employee_disabled_by_enterprise` field, so re-enable does not accidentally restore manually disabled employees.

## Agent/Admin Invite Default Groups

Admins and agents can configure default exclusive groups for users registered through their affiliate invitation codes.

This is not a separate page. It is added to the existing agent-management "my groups / groups and rates" page.

Per eligible group, show:

- whether this exclusive group should be applied by default to affiliate-code registrations;
- the default rate multiplier for newly registered users.

Eligibility:

- only exclusive groups are configurable;
- for admin, admin-owned exclusive groups are eligible;
- for agent, only groups visible to that agent with `can_delegate=true` are eligible;
- public groups are not configurable because they are already visible by default;
- enterprises do not see these controls.

Data model:

- add `agent_invite_group_defaults`;
- fields:
  - `manager_user_id`;
  - `group_id`;
  - `rate_multiplier`;
  - timestamps and soft-delete marker;
- active unique key on `manager_user_id + group_id`.

Registration behavior:

- applies only to affiliate invitation code registrations.
- admin-created invitation redeem codes do not use this default-group template.
- after resolving the new user's direct owner, load defaults for that owner.
- for each default:
  - verify the owner still has permission to delegate that group;
  - create `agent_group_delegations(manager_user_id=owner, child_user_id=new_user, group_id, rate_multiplier, can_delegate=false)`;
  - add `user_allowed_groups` for the new user;
  - invalidate the new user's auth/API-key cache if needed.

Per-user overrides:

- defaults are applied only once during registration;
- later edits from the direct-user management group dialog can reprice or remove the group for that one user;
- changing the default template does not rewrite existing users.

## UI Structure

Agent/admin existing agent-management sidebar remains:

- direct users;
- direct agents;
- direct enterprises;
- my groups / groups and rates.

Enterprise sidebar adds:

- employee management;
- my groups / group rates.

Employee sidebar hides:

- recharge/payment;
- redeem;
- affiliate;
- invitation;
- my subscriptions;
- all manager/admin/agent/enterprise sections.

Employee management UI should follow the existing direct-child table style:

- search box;
- employee rows with email, username, balance, concurrency, RPM, status;
- create employee modal;
- inline or modal edit for balance/concurrency/RPM;
- group assignment modal without rate input;
- delete employee action.

## Testing Strategy

Backend unit tests:

- enterprise creates employee and moves balance/quota;
- enterprise edits employee balance by delta;
- enterprise cannot over-allocate balance/concurrency/RPM;
- employee delete returns balance/quota and hard-deletes employee;
- admin native delete employee returns balance/quota;
- admin native delete enterprise hard-deletes enterprise and employees;
- agent delete enterprise rehomes enterprise to admin and preserves employees;
- enterprise disable/enable restores only enterprise-disabled employees;
- employee cannot redeem/recharge/invite/access subscription-only paths;
- affiliate invite by enterprise assigns the new user's parent to the enterprise's upstream owner.

Backend integration tests:

- hard-delete cascade cleans employee related rows;
- group delegation from enterprise to employee uses enterprise effective rate;
- upstream group removal cascades to employee group access;
- invite default group application writes delegation and allowed group rows.

Frontend tests:

- enterprise sidebar shows employee management and my groups;
- employee sidebar hides disallowed entries;
- employee management create/edit/delete payloads;
- enterprise group modal has no rate input;
- agent/admin my-groups page shows invite default controls only for eligible exclusive groups;
- enterprises do not see invite default controls.

Remote verification:

- run backend tests and frontend tests in GitHub CI where possible;
- build HZZ image remotely;
- deploy latest `ghcr.io/h-2szz/sub2api:hzz` locally only after remote image success.

## Open Design Decisions Resolved

- Employees use a distinct `employee` role.
- Employees are created only by enterprises.
- Employee balance is a directly assigned wallet, not an enterprise live charge-through balance.
- Enterprise-to-employee balance edits are direct balance moves, not redeem codes.
- Employee balance can be set to `0`.
- Enterprises can invite/rebate, but invitees are owned by the enterprise's upstream owner.
- Enterprises do not configure invite default groups.
- Invite default groups are configured in the existing admin/agent my-groups page, not in a separate page.
- Admin-created invitation redeem codes do not apply invite default group templates.

## Spec Self-Review

- No implementation task is hidden as a placeholder.
- The delete rules distinguish admin hard-delete from agent detach.
- The employee role restrictions are separate from normal user behavior.
- The billing model avoids double deduction by charging employees from assigned employee balance.
- Invite default group behavior is scoped to affiliate invitation codes only.
