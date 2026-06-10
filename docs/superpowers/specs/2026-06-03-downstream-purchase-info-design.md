# Downstream Purchase Info Design

## Goal

Add a small upstream-propagated purchase information feature for redeem-code sourcing.

Managers can publish "where to buy redeem codes / service contact" cards to their direct downstream accounts. The feature is informational only: redeem codes are still created by admins and redeemed through the existing redeem flow. Agents who buy admin-generated redeem codes can list their own purchase links or contact details for their downstream users.

## Role Rules

- Admin:
  - Can edit the admin-owned purchase info cards.
  - When viewing the page, sees the admin-owned cards because admins have no upstream.
  - The admin-owned cards are what admin-direct users, admin-direct level-1 agents, and admin-direct enterprises see.
- Agent:
  - Can view purchase info cards published by their upstream.
  - Can edit their own cards that will be shown to their direct users, direct child agents, and direct enterprises.
  - A level-2 agent can publish cards to its own direct users and enterprises if such direct children exist.
- Enterprise:
  - Can view purchase info cards published by its upstream.
  - Cannot edit or propagate cards.
- Normal user:
  - Can view purchase info cards published by its upstream.
  - Cannot edit or propagate cards.
- Employee:
  - Cannot view the page.
  - Cannot call the purchase info APIs.
  - Reason: employees cannot redeem/recharge, so purchase information is irrelevant.

## Data Model

Create a dedicated `purchase_info_cards` table instead of reusing global settings, because this data is owner-scoped and inherited by downstream users.

Fields:

- `id`
- `owner_user_id`: nullable for admin-owned/root cards, or set to the publishing agent user ID.
- `title`: required, short service/card title.
- `description`: optional text shown on the card.
- `purchase_url`: optional URL for buying or contacting.
- `contact`: optional text such as Telegram, WeChat, email, or support handle.
- `sort_order`: integer, lower first.
- `enabled`: boolean.
- `created_at`, `updated_at`, `deleted_at`.

Root admin cards use `owner_user_id IS NULL`, so all admin accounts share the same admin/root purchase information. This matches the existing "all admins are common root" ownership model.

## Inheritance

The viewer resolves a publisher:

- Admin viewer: publisher is root admin (`owner_user_id IS NULL`).
- Agent, enterprise, normal user: publisher is their `parent_user_id`.
- If the parent is nil or not found, fall back to root admin cards.
- Employee: blocked before resolution.

No historical copy is made to children. Changes by an admin or agent are immediately reflected for their downstream viewers.

## API

Add authenticated user routes:

- `GET /api/v1/purchase-info`
  - Returns the cards visible to the current user.
  - Blocks employees.
- `GET /api/v1/purchase-info/manage`
  - Returns cards owned by the current publisher.
  - Allowed for admins and agents only.
- `POST /api/v1/purchase-info/manage`
  - Creates a card.
  - Allowed for admins and agents only.
- `PUT /api/v1/purchase-info/manage/:id`
  - Updates an owned card.
  - Allowed for admins and agents only.
- `DELETE /api/v1/purchase-info/manage/:id`
  - Soft-deletes an owned card.
  - Allowed for admins and agents only.

Ownership checks:

- Admin can only manage root cards.
- Agent can only manage cards whose `owner_user_id` equals the agent ID.
- Enterprise, normal user, and employee cannot manage cards.

Validation:

- `title` required, max 100.
- `description` max 2000.
- `purchase_url` max 1024 and must be empty or `http://` / `https://`.
- `contact` max 500.
- `sort_order` default 0.

## Frontend

Add a new sidebar/page entry for all roles except employees.

Suggested route:

- `/purchase-info`

Page behavior:

- Viewer section: all allowed roles see visible purchase cards.
- Management section: only admins and agents see card create/edit/delete controls.
- Enterprises and normal users see a read-only card list.
- Employees do not see the menu item and direct navigation returns a forbidden/404-like state via backend denial.

The UI should be simple and operational: a searchable/listed card area is unnecessary initially because card counts are expected to be small. Use the existing table/page layout style and modals/forms already used by agent/admin management.

## Non-Goals

- Do not let agents generate redeem codes.
- Do not transfer or resell redeem-code inventory in this feature.
- Do not alter redeem-code redemption, affiliate rebate, or payment logic.
- Do not show this page to employees.

## Tests

Backend:

- Root admin cards are visible to admin-direct users.
- Agent-owned cards are visible to direct users/enterprises/agents under that agent.
- Agents see their upstream cards but manage only their own cards.
- Employees are forbidden for both view and manage APIs.
- URL validation rejects non-HTTP(S) schemes.

Frontend:

- Sidebar hides purchase info for employees.
- Sidebar shows purchase info for admin, agent, enterprise, and normal user.
- Management controls render only for admin and agent.

