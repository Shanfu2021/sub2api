package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type enterpriseRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewEnterpriseRepository(client *dbent.Client, sqlDB *sql.DB) service.EnterpriseTenantRepository {
	return &enterpriseRepository{
		client: client,
		sql:    sqlDB,
	}
}

func (r *enterpriseRepository) exec(ctx context.Context) sqlQueryExecutor {
	return txAwareSQLExecutor(ctx, r.sql, r.client)
}

func (r *enterpriseRepository) ListTenants(ctx context.Context, params pagination.PaginationParams, filters service.EnterpriseTenantListFilters) ([]service.EnterpriseTenant, int64, error) {
	exec := r.exec(ctx)
	where, args := buildEnterpriseTenantWhere(filters)
	countQuery := "SELECT COUNT(*) FROM enterprise_tenants t" + where
	var total int64
	if err := scanSingleRow(ctx, exec, countQuery, args, &total); err != nil {
		return nil, 0, err
	}

	query := `
SELECT t.id,
       t.name,
       t.code,
       t.status,
       COALESCE(t.notes, ''),
       COALESCE(t.portal_host, ''),
       COALESCE(t.pricing_floor_factor::double precision, 1.0),
       COALESCE(t.member_default_pricing_factor::double precision, 0),
       COALESCE(t.pricing_scope, 'balance'),
       COALESCE(t.concurrency, 0),
       COALESCE(t.balance_quota_total::double precision, 0),
       COALESCE(t.balance_quota_used::double precision, 0),
       COALESCE(t.balance_quota_spent::double precision, 0),
       COALESCE(t.balance_overdraft_limit::double precision, 0),
       t.created_by,
       t.updated_by,
       t.created_at,
       t.updated_at,
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id AND em.member_role = 'manager'), 0),
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id), 0)
FROM enterprise_tenants t` + where + `
ORDER BY ` + enterpriseTenantOrderBy(params) + `
OFFSET $` + fmt.Sprint(len(args)+1) + ` LIMIT $` + fmt.Sprint(len(args)+2)
	args = append(args, params.Offset(), params.Limit())
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}

	items := make([]service.EnterpriseTenant, 0)
	for rows.Next() {
		item, err := scanEnterpriseTenant(rows)
		if err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	if err := r.hydrateTenantAllowedGroups(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *enterpriseRepository) GetTenantByID(ctx context.Context, tenantID int64) (*service.EnterpriseTenant, error) {
	return r.getTenant(ctx, tenantID, false)
}

func (r *enterpriseRepository) LockTenantByID(ctx context.Context, tenantID int64) (*service.EnterpriseTenant, error) {
	return r.getTenant(ctx, tenantID, true)
}

func (r *enterpriseRepository) GetTenantByCode(ctx context.Context, code string) (*service.EnterpriseTenant, error) {
	exec := r.exec(ctx)
	query := `
SELECT t.id,
       t.name,
       t.code,
       t.status,
       COALESCE(t.notes, ''),
       COALESCE(t.portal_host, ''),
       COALESCE(t.pricing_floor_factor::double precision, 1.0),
       COALESCE(t.member_default_pricing_factor::double precision, 0),
       COALESCE(t.pricing_scope, 'balance'),
       COALESCE(t.concurrency, 0),
       COALESCE(t.balance_quota_total::double precision, 0),
       COALESCE(t.balance_quota_used::double precision, 0),
       COALESCE(t.balance_quota_spent::double precision, 0),
       COALESCE(t.balance_overdraft_limit::double precision, 0),
       t.created_by,
       t.updated_by,
       t.created_at,
       t.updated_at,
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id AND em.member_role = 'manager'), 0),
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id), 0)
FROM enterprise_tenants t
WHERE UPPER(t.code) = UPPER($1)
LIMIT 1`
	rows, err := exec.QueryContext(ctx, query, code)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrEnterpriseTenantNotFound
	}
	item, err := scanEnterpriseTenant(rows)
	if err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	groups, err := r.GetTenantAllowedGroups(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.AllowedGroupIDs = groups[item.ID]
	rates, err := r.GetTenantGroupRates(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.GroupRates = rates[item.ID]
	memberRates, err := r.GetTenantMemberGroupRates(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.MemberGroupRates = memberRates[item.ID]
	return &item, nil
}

func (r *enterpriseRepository) NextTenantCode(ctx context.Context) (string, error) {
	exec := r.exec(ctx)
	const prefix = "ENT"
	if _, err := exec.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('enterprise_tenants_next_code'))`); err != nil {
		return "", err
	}
	var maxSuffix sql.NullInt64
	if err := scanSingleRow(ctx, exec, `
SELECT MAX(NULLIF(regexp_replace(code, '^ENT0*', ''), '')::bigint)
FROM enterprise_tenants
WHERE code ~ '^ENT[0-9]+$'
`, nil, &maxSuffix); err != nil {
		return "", err
	}
	next := int64(1)
	if maxSuffix.Valid {
		next = maxSuffix.Int64 + 1
	}
	return prefix + leftPadInt(next, 4), nil
}

func (r *enterpriseRepository) getTenant(ctx context.Context, tenantID int64, forUpdate bool) (*service.EnterpriseTenant, error) {
	exec := r.exec(ctx)
	query := `
SELECT t.id,
       t.name,
       t.code,
       t.status,
       COALESCE(t.notes, ''),
       COALESCE(t.portal_host, ''),
       COALESCE(t.pricing_floor_factor::double precision, 1.0),
       COALESCE(t.member_default_pricing_factor::double precision, 0),
       COALESCE(t.pricing_scope, 'balance'),
       COALESCE(t.concurrency, 0),
       COALESCE(t.balance_quota_total::double precision, 0),
       COALESCE(t.balance_quota_used::double precision, 0),
       COALESCE(t.balance_quota_spent::double precision, 0),
       COALESCE(t.balance_overdraft_limit::double precision, 0),
       t.created_by,
       t.updated_by,
       t.created_at,
       t.updated_at,
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id AND em.member_role = 'manager'), 0),
       COALESCE((SELECT COUNT(*) FROM enterprise_memberships em WHERE em.tenant_id = t.id), 0)
FROM enterprise_tenants t
WHERE t.id = $1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := exec.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrEnterpriseTenantNotFound
	}
	item, err := scanEnterpriseTenant(rows)
	if err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	groups, err := r.GetTenantAllowedGroups(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.AllowedGroupIDs = groups[item.ID]
	rates, err := r.GetTenantGroupRates(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.GroupRates = rates[item.ID]
	memberRates, err := r.GetTenantMemberGroupRates(ctx, []int64{item.ID})
	if err != nil {
		return nil, err
	}
	item.MemberGroupRates = memberRates[item.ID]
	return &item, nil
}

func (r *enterpriseRepository) CreateTenant(ctx context.Context, tenant *service.EnterpriseTenant) error {
	exec := r.exec(ctx)
	query := `
INSERT INTO enterprise_tenants (
    name, code, status, notes, portal_host, pricing_floor_factor, pricing_scope,
    member_default_pricing_factor, concurrency, balance_quota_total, balance_quota_used,
    balance_quota_spent, balance_overdraft_limit, created_by, updated_by, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NOW(), NOW())
RETURNING id, created_at, updated_at`
	var createdAt, updatedAt time.Time
	if err := scanSingleRow(ctx, exec, query, []any{
		tenant.Name,
		tenant.Code,
		tenant.Status,
		tenant.Notes,
		tenant.PortalHost,
		service.NormalizePricingDiscountFactorForRepo(tenant.PricingFloorFactor),
		service.NormalizeEnterprisePricingScopeForRepo(tenant.PricingScope),
		tenant.MemberDefaultPricingFactor,
		tenant.Concurrency,
		tenant.BalanceQuotaTotal,
		tenant.BalanceQuotaUsed,
		tenant.BalanceQuotaSpent,
		tenant.BalanceOverdraftLimit,
		tenant.CreatedBy,
		tenant.UpdatedBy,
	}, &tenant.ID, &createdAt, &updatedAt); err != nil {
		return err
	}
	tenant.CreatedAt = createdAt
	tenant.UpdatedAt = updatedAt
	return nil
}

func (r *enterpriseRepository) UpdateTenant(ctx context.Context, tenant *service.EnterpriseTenant) error {
	exec := r.exec(ctx)
	res, err := exec.ExecContext(ctx, `
UPDATE enterprise_tenants
SET name = $2,
    code = $3,
    status = $4,
    notes = $5,
    portal_host = $6,
    pricing_floor_factor = $7,
    pricing_scope = $8,
    member_default_pricing_factor = $9,
    concurrency = $10,
    balance_quota_total = $11,
    balance_quota_used = $12,
    balance_quota_spent = $13,
    balance_overdraft_limit = $14,
    updated_by = $15,
    updated_at = NOW()
WHERE id = $1
	`, tenant.ID, tenant.Name, tenant.Code, tenant.Status, tenant.Notes, tenant.PortalHost, service.NormalizePricingDiscountFactorForRepo(tenant.PricingFloorFactor), service.NormalizeEnterprisePricingScopeForRepo(tenant.PricingScope), tenant.MemberDefaultPricingFactor, tenant.Concurrency, tenant.BalanceQuotaTotal, tenant.BalanceQuotaUsed, tenant.BalanceQuotaSpent, tenant.BalanceOverdraftLimit, tenant.UpdatedBy)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrEnterpriseTenantNotFound
	}
	return nil
}

func (r *enterpriseRepository) SetTenantAllowedGroups(ctx context.Context, tenantID int64, groupIDs []int64, groupRates map[int64]*float64, memberGroupRates map[int64]*float64) error {
	exec := r.exec(ctx)
	if _, err := exec.ExecContext(ctx, `DELETE FROM enterprise_tenant_groups WHERE tenant_id = $1`, tenantID); err != nil {
		return err
	}
	unique := uniquePositiveInt64s(groupIDs)
	if len(unique) == 0 {
		return nil
	}
	for _, groupID := range unique {
		var rate any
		if groupRates != nil {
			if v, ok := groupRates[groupID]; ok && v != nil {
				rate = service.NormalizePricingDiscountFactorForRepo(*v)
			}
		}
		var memberRate any
		if memberGroupRates != nil {
			if v, ok := memberGroupRates[groupID]; ok && v != nil {
				memberRate = service.NormalizePricingDiscountFactorForRepo(*v)
			}
		}
		if _, err := exec.ExecContext(ctx, `
INSERT INTO enterprise_tenant_groups (tenant_id, group_id, pricing_floor_multiplier, member_default_multiplier, created_at)
VALUES ($1, $2, $3, $4, NOW())
`, tenantID, groupID, rate, memberRate); err != nil {
			return err
		}
	}
	return nil
}

func (r *enterpriseRepository) GetTenantAllowedGroups(ctx context.Context, tenantIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(tenantIDs))
	tenantIDs = uniquePositiveInt64s(tenantIDs)
	if len(tenantIDs) == 0 {
		return out, nil
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT tenant_id, group_id
FROM enterprise_tenant_groups
WHERE tenant_id = ANY($1)
ORDER BY tenant_id, group_id
`, pq.Array(tenantIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tenantID, groupID int64
		if err := rows.Scan(&tenantID, &groupID); err != nil {
			return nil, err
		}
		out[tenantID] = append(out[tenantID], groupID)
	}
	return out, rows.Err()
}

func (r *enterpriseRepository) ListTenantGroupSummaries(ctx context.Context, tenantID int64) ([]service.EnterpriseGroupSummary, error) {
	if tenantID <= 0 {
		return []service.EnterpriseGroupSummary{}, nil
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT g.id,
       COALESCE(g.name, ''),
       COALESCE(g.platform, ''),
       COALESCE(g.subscription_type, ''),
       COALESCE(g.rate_multiplier::double precision, 1.0),
       COALESCE(g.is_exclusive, false),
       COALESCE(g.status, '')
FROM enterprise_tenant_groups etg
JOIN groups g ON g.id = etg.group_id AND g.deleted_at IS NULL
WHERE etg.tenant_id = $1
ORDER BY g.sort_order ASC, g.id ASC
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.EnterpriseGroupSummary, 0)
	for rows.Next() {
		var item service.EnterpriseGroupSummary
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Platform,
			&item.SubscriptionType,
			&item.RateMultiplier,
			&item.IsExclusive,
			&item.Status,
		); err != nil {
			return nil, err
		}
		item.RateMultiplier = service.NormalizePricingDiscountFactorForRepo(item.RateMultiplier)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *enterpriseRepository) GetTenantGroupRates(ctx context.Context, tenantIDs []int64) (map[int64]map[int64]float64, error) {
	out := make(map[int64]map[int64]float64, len(tenantIDs))
	tenantIDs = uniquePositiveInt64s(tenantIDs)
	if len(tenantIDs) == 0 {
		return out, nil
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT tenant_id, group_id, pricing_floor_multiplier
FROM enterprise_tenant_groups
WHERE tenant_id = ANY($1) AND pricing_floor_multiplier IS NOT NULL
ORDER BY tenant_id, group_id
`, pq.Array(tenantIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tenantID, groupID int64
		var rate float64
		if err := rows.Scan(&tenantID, &groupID, &rate); err != nil {
			return nil, err
		}
		if out[tenantID] == nil {
			out[tenantID] = make(map[int64]float64)
		}
		out[tenantID][groupID] = service.NormalizePricingDiscountFactorForRepo(rate)
	}
	return out, rows.Err()
}

func (r *enterpriseRepository) GetTenantMemberGroupRates(ctx context.Context, tenantIDs []int64) (map[int64]map[int64]float64, error) {
	out := make(map[int64]map[int64]float64, len(tenantIDs))
	tenantIDs = uniquePositiveInt64s(tenantIDs)
	if len(tenantIDs) == 0 {
		return out, nil
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT tenant_id, group_id, member_default_multiplier
FROM enterprise_tenant_groups
WHERE tenant_id = ANY($1) AND member_default_multiplier IS NOT NULL
ORDER BY tenant_id, group_id
`, pq.Array(tenantIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var tenantID, groupID int64
		var rate float64
		if err := rows.Scan(&tenantID, &groupID, &rate); err != nil {
			return nil, err
		}
		if out[tenantID] == nil {
			out[tenantID] = make(map[int64]float64)
		}
		out[tenantID][groupID] = service.NormalizePricingDiscountFactorForRepo(rate)
	}
	return out, rows.Err()
}

func (r *enterpriseRepository) GetMembershipByUserID(ctx context.Context, userID int64) (*service.EnterpriseMembership, error) {
	return r.getMembership(ctx, "em.user_id = $1", []any{userID}, false)
}

func (r *enterpriseRepository) GetMembershipByTenantAndUserID(ctx context.Context, tenantID, userID int64) (*service.EnterpriseMembership, error) {
	return r.getMembership(ctx, "em.tenant_id = $1 AND em.user_id = $2", []any{tenantID, userID}, false)
}

func (r *enterpriseRepository) GetMembershipByTenantAndUserIDForUpdate(ctx context.Context, tenantID, userID int64) (*service.EnterpriseMembership, error) {
	return r.getMembership(ctx, "em.tenant_id = $1 AND em.user_id = $2", []any{tenantID, userID}, true)
}

func (r *enterpriseRepository) ListMembershipUserIDs(ctx context.Context, tenantID int64) ([]int64, error) {
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT user_id
FROM enterprise_memberships
WHERE tenant_id = $1
ORDER BY user_id
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	userIDs := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

func (r *enterpriseRepository) getMembership(ctx context.Context, where string, args []any, forUpdate bool) (*service.EnterpriseMembership, error) {
	exec := r.exec(ctx)
	query := membershipBaseSelect() + `
WHERE ` + where + `
LIMIT 1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, service.ErrEnterpriseMembershipNotFound
	}
	item, err := scanEnterpriseMembership(rows)
	if err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	items := []service.EnterpriseMembership{item}
	if err := r.hydrateMembershipAllowedGroups(ctx, items); err != nil {
		return nil, err
	}
	if err := r.hydrateMembershipGroupRates(ctx, items); err != nil {
		return nil, err
	}
	item = items[0]
	return &item, nil
}

func (r *enterpriseRepository) ListMemberships(ctx context.Context, tenantID int64, params pagination.PaginationParams, filters service.EnterpriseMemberListFilters) ([]service.EnterpriseMembership, int64, error) {
	exec := r.exec(ctx)
	where, args := buildEnterpriseMembershipWhere(tenantID, filters)
	countQuery := `SELECT COUNT(*) FROM enterprise_memberships em JOIN users u ON u.id = em.user_id AND u.deleted_at IS NULL` + where
	var total int64
	if err := scanSingleRow(ctx, exec, countQuery, args, &total); err != nil {
		return nil, 0, err
	}

	query := membershipBaseSelect() + where + `
ORDER BY ` + enterpriseMembershipOrderBy(params) + `
OFFSET $` + fmt.Sprint(len(args)+1) + ` LIMIT $` + fmt.Sprint(len(args)+2)
	args = append(args, params.Offset(), params.Limit())
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	items := make([]service.EnterpriseMembership, 0)
	for rows.Next() {
		item, err := scanEnterpriseMembership(rows)
		if err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, 0, err
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	if err := r.hydrateMembershipAllowedGroups(ctx, items); err != nil {
		return nil, 0, err
	}
	if err := r.hydrateMembershipGroupRates(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *enterpriseRepository) CreateMembership(ctx context.Context, membership *service.EnterpriseMembership) error {
	exec := r.exec(ctx)
	query := `
INSERT INTO enterprise_memberships (
    tenant_id, user_id, member_role, member_note, joined_via, joined_source,
    pricing_factor, pricing_scope, created_by, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
RETURNING id, created_at, updated_at`
	if err := scanSingleRow(ctx, exec, query, []any{
		membership.TenantID,
		membership.UserID,
		membership.MemberRole,
		membership.MemberNote,
		membership.JoinedVia,
		membership.JoinedSource,
		service.NormalizeEnterpriseMemberPricingFactorForRepo(membership.PricingFactor),
		service.NormalizeEnterprisePricingScopeForRepo(membership.PricingScope),
		membership.CreatedBy,
	}, &membership.ID, &membership.CreatedAt, &membership.UpdatedAt); err != nil {
		return err
	}
	return nil
}

func (r *enterpriseRepository) UpdateMembership(ctx context.Context, membership *service.EnterpriseMembership) error {
	res, err := r.exec(ctx).ExecContext(ctx, `
UPDATE enterprise_memberships
SET member_role = $2,
    member_note = $3,
    joined_via = $4,
    joined_source = $5,
    pricing_factor = $6,
    pricing_scope = $7,
    updated_at = NOW()
WHERE id = $1
	`, membership.ID, membership.MemberRole, membership.MemberNote, membership.JoinedVia, membership.JoinedSource, service.NormalizeEnterpriseMemberPricingFactorForRepo(membership.PricingFactor), service.NormalizeEnterprisePricingScopeForRepo(membership.PricingScope))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrEnterpriseMembershipNotFound
	}
	return nil
}

func (r *enterpriseRepository) DeleteMembership(ctx context.Context, tenantID, userID int64) error {
	res, err := r.exec(ctx).ExecContext(ctx, `
DELETE FROM enterprise_memberships
WHERE tenant_id = $1 AND user_id = $2
`, tenantID, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrEnterpriseMembershipNotFound
	}
	return nil
}

func (r *enterpriseRepository) ListInviteCodes(ctx context.Context, tenantID int64, params pagination.PaginationParams, filters service.EnterpriseInviteCodeListFilters) ([]service.EnterpriseInviteCode, int64, error) {
	exec := r.exec(ctx)
	where, args := buildEnterpriseInviteWhere(tenantID, filters)
	countQuery := `SELECT COUNT(*) FROM enterprise_invite_codes e` + where
	var total int64
	if err := scanSingleRow(ctx, exec, countQuery, args, &total); err != nil {
		return nil, 0, err
	}
	query := `
SELECT id, tenant_id, code, status, max_uses, used_count, expires_at, COALESCE(notes, ''), created_by, created_at, updated_at
FROM enterprise_invite_codes e` + where + `
ORDER BY ` + enterpriseInviteOrderBy(params) + `
OFFSET $` + fmt.Sprint(len(args)+1) + ` LIMIT $` + fmt.Sprint(len(args)+2)
	args = append(args, params.Offset(), params.Limit())
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.EnterpriseInviteCode, 0)
	for rows.Next() {
		item, err := scanEnterpriseInviteCode(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *enterpriseRepository) GetInviteCodeByID(ctx context.Context, inviteID int64) (*service.EnterpriseInviteCode, error) {
	return r.getInviteCode(ctx, "id = $1", []any{inviteID}, false)
}

func (r *enterpriseRepository) GetInviteCodeByCode(ctx context.Context, code string) (*service.EnterpriseInviteCode, error) {
	return r.getInviteCode(ctx, "UPPER(code) = UPPER($1)", []any{code}, false)
}

func (r *enterpriseRepository) GetInviteCodeByCodeForUpdate(ctx context.Context, code string) (*service.EnterpriseInviteCode, error) {
	return r.getInviteCode(ctx, "UPPER(code) = UPPER($1)", []any{code}, true)
}

func (r *enterpriseRepository) getInviteCode(ctx context.Context, where string, args []any, forUpdate bool) (*service.EnterpriseInviteCode, error) {
	query := `
SELECT id, tenant_id, code, status, max_uses, used_count, expires_at, COALESCE(notes, ''), created_by, created_at, updated_at
FROM enterprise_invite_codes
WHERE ` + where + `
LIMIT 1`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := r.exec(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrEnterpriseInviteCodeNotFound
	}
	item, err := scanEnterpriseInviteCode(rows)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *enterpriseRepository) CreateInviteCode(ctx context.Context, invite *service.EnterpriseInviteCode) error {
	query := `
INSERT INTO enterprise_invite_codes (tenant_id, code, status, max_uses, used_count, expires_at, notes, created_by, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
RETURNING id, created_at, updated_at`
	if err := scanSingleRow(ctx, r.exec(ctx), query, []any{
		invite.TenantID,
		invite.Code,
		invite.Status,
		invite.MaxUses,
		invite.UsedCount,
		invite.ExpiresAt,
		invite.Notes,
		invite.CreatedBy,
	}, &invite.ID, &invite.CreatedAt, &invite.UpdatedAt); err != nil {
		return err
	}
	return nil
}

func (r *enterpriseRepository) UpdateInviteCode(ctx context.Context, invite *service.EnterpriseInviteCode) error {
	res, err := r.exec(ctx).ExecContext(ctx, `
UPDATE enterprise_invite_codes
SET status = $2,
    max_uses = $3,
    expires_at = $4,
    notes = $5,
    updated_at = NOW()
WHERE id = $1
`, invite.ID, invite.Status, invite.MaxUses, invite.ExpiresAt, invite.Notes)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrEnterpriseInviteCodeNotFound
	}
	return nil
}

func (r *enterpriseRepository) IncrementInviteCodeUsage(ctx context.Context, inviteID int64) error {
	res, err := r.exec(ctx).ExecContext(ctx, `
UPDATE enterprise_invite_codes
SET used_count = used_count + 1,
    updated_at = NOW()
WHERE id = $1
`, inviteID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrEnterpriseInviteCodeNotFound
	}
	return nil
}

func (r *enterpriseRepository) CreateLedgerEntry(ctx context.Context, entry *service.EnterpriseWalletLedgerEntry) error {
	query := `
INSERT INTO enterprise_wallet_ledger (
    tenant_id, operator_user_id, target_user_id, direction, amount,
    balance_before, balance_after, notes, created_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
RETURNING id, created_at`
	if err := scanSingleRow(ctx, r.exec(ctx), query, []any{
		entry.TenantID,
		entry.OperatorUserID,
		entry.TargetUserID,
		entry.Direction,
		entry.Amount,
		entry.BalanceBefore,
		entry.BalanceAfter,
		entry.Notes,
	}, &entry.ID, &entry.CreatedAt); err != nil {
		return err
	}
	return nil
}

func (r *enterpriseRepository) ListLedger(ctx context.Context, tenantID int64, params pagination.PaginationParams) ([]service.EnterpriseWalletLedgerEntry, int64, error) {
	exec := r.exec(ctx)
	var total int64
	if err := scanSingleRow(ctx, exec, `SELECT COUNT(*) FROM enterprise_wallet_ledger WHERE tenant_id = $1`, []any{tenantID}, &total); err != nil {
		return nil, 0, err
	}
	query := `
SELECT l.id,
       l.tenant_id,
       l.operator_user_id,
       l.target_user_id,
       l.direction,
       COALESCE(l.amount::double precision, 0),
       COALESCE(l.balance_before::double precision, 0),
       COALESCE(l.balance_after::double precision, 0),
       COALESCE(l.notes, ''),
       l.created_at,
       COALESCE(op.email, ''),
       COALESCE(tu.email, ''),
       COALESCE(tu.username, ''),
       COALESCE(t.name, ''),
       COALESCE(t.code, '')
FROM enterprise_wallet_ledger l
JOIN enterprise_tenants t ON t.id = l.tenant_id
LEFT JOIN users op ON op.id = l.operator_user_id
LEFT JOIN users tu ON tu.id = l.target_user_id
WHERE l.tenant_id = $1
ORDER BY ` + enterpriseLedgerOrderBy(params) + `
OFFSET $2 LIMIT $3`
	rows, err := exec.QueryContext(ctx, query, tenantID, params.Offset(), params.Limit())
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]service.EnterpriseWalletLedgerEntry, 0)
	for rows.Next() {
		item, err := scanEnterpriseLedger(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *enterpriseRepository) GetEnterpriseContextByUserID(ctx context.Context, userID int64) (*service.EnterpriseContext, error) {
	query := `
SELECT em.tenant_id,
       COALESCE(t.name, ''),
       COALESCE(t.code, ''),
       COALESCE(t.status, 'active'),
       COALESCE(t.portal_host, ''),
       COALESCE(em.member_role, ''),
       COALESCE(em.member_note, ''),
       COALESCE(em.joined_via, ''),
       COALESCE(em.joined_source, ''),
       COALESCE(em.pricing_factor::double precision, 0),
       COALESCE(em.pricing_scope, 'balance'),
       COALESCE(t.pricing_floor_factor::double precision, 1.0),
       COALESCE(t.member_default_pricing_factor::double precision, 0),
       COALESCE(t.concurrency, 0),
       COALESCE(t.balance_quota_total::double precision, 0),
       COALESCE(t.balance_quota_used::double precision, 0),
       COALESCE(t.balance_quota_spent::double precision, 0),
       COALESCE(t.balance_overdraft_limit::double precision, 0)
FROM enterprise_memberships em
JOIN enterprise_tenants t ON t.id = em.tenant_id
WHERE em.user_id = $1
LIMIT 1`
	rows, err := r.exec(ctx).QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
		return nil, nil
	}
	var out service.EnterpriseContext
	if err := rows.Scan(
		&out.TenantID,
		&out.TenantName,
		&out.TenantCode,
		&out.TenantStatus,
		&out.PortalHost,
		&out.MemberRole,
		&out.MemberNote,
		&out.JoinedVia,
		&out.JoinedSource,
		&out.PricingFactor,
		&out.PricingScope,
		&out.PricingFloorFactor,
		&out.MemberDefaultPricingFactor,
		&out.Concurrency,
		&out.BalanceQuotaTotal,
		&out.BalanceQuotaUsed,
		&out.BalanceQuotaSpent,
		&out.BalanceOverdraftLimit,
	); err != nil {
		_ = rows.Close()
		return nil, err
	}
	out.PricingFactor = service.NormalizeEnterpriseMemberPricingFactorForRepo(out.PricingFactor)
	out.PricingScope = service.NormalizeEnterprisePricingScopeForRepo(out.PricingScope)
	out.PricingFloorFactor = service.NormalizePricingDiscountFactorForRepo(out.PricingFloorFactor)
	out.MemberDefaultPricingFactor = service.NormalizeEnterpriseMemberDefaultPricingFactor(out.MemberDefaultPricingFactor)
	out.AllowedGroupIDs = nil
	if err := rows.Close(); err != nil {
		return nil, err
	}
	groups, err := r.GetTenantAllowedGroups(ctx, []int64{out.TenantID})
	if err != nil {
		return nil, err
	}
	out.AllowedGroupIDs = groups[out.TenantID]
	rates, err := r.GetTenantGroupRates(ctx, []int64{out.TenantID})
	if err != nil {
		return nil, err
	}
	out.GroupRates = rates[out.TenantID]
	memberRates, err := r.GetTenantMemberGroupRates(ctx, []int64{out.TenantID})
	if err != nil {
		return nil, err
	}
	out.MemberGroupRates = memberRates[out.TenantID]
	out.SelfRechargeBlocked = true
	out.SelfRedeemBlocked = true
	return &out, nil
}

func (r *enterpriseRepository) hydrateTenantAllowedGroups(ctx context.Context, tenants []service.EnterpriseTenant) error {
	if len(tenants) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(tenants))
	for i := range tenants {
		ids = append(ids, tenants[i].ID)
	}
	groups, err := r.GetTenantAllowedGroups(ctx, ids)
	if err != nil {
		return err
	}
	rates, err := r.GetTenantGroupRates(ctx, ids)
	if err != nil {
		return err
	}
	memberRates, err := r.GetTenantMemberGroupRates(ctx, ids)
	if err != nil {
		return err
	}
	for i := range tenants {
		tenants[i].AllowedGroupIDs = groups[tenants[i].ID]
		tenants[i].GroupRates = rates[tenants[i].ID]
		tenants[i].MemberGroupRates = memberRates[tenants[i].ID]
	}
	return nil
}

func (r *enterpriseRepository) hydrateMembershipAllowedGroups(ctx context.Context, memberships []service.EnterpriseMembership) error {
	if len(memberships) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(memberships))
	for i := range memberships {
		userIDs = append(userIDs, memberships[i].UserID)
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT user_id, group_id
FROM user_allowed_groups
WHERE user_id = ANY($1)
ORDER BY user_id, group_id
`, pq.Array(uniquePositiveInt64s(userIDs)))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	groupMap := make(map[int64][]int64)
	for rows.Next() {
		var userID, groupID int64
		if err := rows.Scan(&userID, &groupID); err != nil {
			return err
		}
		groupMap[userID] = append(groupMap[userID], groupID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range memberships {
		memberships[i].AllowedGroups = groupMap[memberships[i].UserID]
	}
	return nil
}

func (r *enterpriseRepository) hydrateMembershipGroupRates(ctx context.Context, memberships []service.EnterpriseMembership) error {
	if len(memberships) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(memberships))
	for i := range memberships {
		userIDs = append(userIDs, memberships[i].UserID)
	}
	rows, err := r.exec(ctx).QueryContext(ctx, `
SELECT user_id, group_id, rate_multiplier
FROM user_group_rate_multipliers
WHERE user_id = ANY($1) AND rate_multiplier IS NOT NULL
ORDER BY user_id, group_id
`, pq.Array(uniquePositiveInt64s(userIDs)))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	rateMap := make(map[int64]map[int64]float64)
	for rows.Next() {
		var userID, groupID int64
		var rate float64
		if err := rows.Scan(&userID, &groupID, &rate); err != nil {
			return err
		}
		if rateMap[userID] == nil {
			rateMap[userID] = make(map[int64]float64)
		}
		rateMap[userID][groupID] = rate
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range memberships {
		memberships[i].GroupRates = rateMap[memberships[i].UserID]
	}
	return nil
}

func buildEnterpriseTenantWhere(filters service.EnterpriseTenantListFilters) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if status := strings.TrimSpace(filters.Status); status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("t.status = $%d", len(args)))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(t.name ILIKE $%d OR t.code ILIKE $%d OR t.notes ILIKE $%d)", len(args), len(args), len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildEnterpriseMembershipWhere(tenantID int64, filters service.EnterpriseMemberListFilters) (string, []any) {
	clauses := []string{"em.tenant_id = $1"}
	args := []any{tenantID}
	if status := strings.TrimSpace(filters.Status); status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("u.status = $%d", len(args)))
	}
	if role := strings.TrimSpace(filters.Role); role != "" {
		args = append(args, role)
		clauses = append(clauses, fmt.Sprintf("em.member_role = $%d", len(args)))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(u.email ILIKE $%d OR u.username ILIKE $%d OR em.member_note ILIKE $%d)", len(args), len(args), len(args)))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func buildEnterpriseInviteWhere(tenantID int64, filters service.EnterpriseInviteCodeListFilters) (string, []any) {
	clauses := []string{"e.tenant_id = $1"}
	args := []any{tenantID}
	if status := strings.TrimSpace(filters.Status); status != "" {
		args = append(args, status)
		clauses = append(clauses, fmt.Sprintf("e.status = $%d", len(args)))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		args = append(args, "%"+search+"%")
		clauses = append(clauses, fmt.Sprintf("(e.code ILIKE $%d OR e.notes ILIKE $%d)", len(args), len(args)))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func enterpriseTenantOrderBy(params pagination.PaginationParams) string {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := "t.created_at"
	switch sortBy {
	case "name":
		field = "t.name"
	case "code":
		field = "t.code"
	case "status":
		field = "t.status"
	case "updated_at":
		field = "t.updated_at"
	case "balance_quota_total":
		field = "t.balance_quota_total"
	case "balance_quota_used":
		field = "t.balance_quota_used"
	case "balance_quota_spent":
		field = "t.balance_quota_spent"
	case "balance_overdraft_limit":
		field = "t.balance_overdraft_limit"
	}
	return field + " " + sortOrder + ", t.id DESC"
}

func enterpriseMembershipOrderBy(params pagination.PaginationParams) string {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := "em.created_at"
	switch sortBy {
	case "email":
		field = "u.email"
	case "username":
		field = "u.username"
	case "balance":
		field = "u.balance"
	case "status":
		field = "u.status"
	case "member_role":
		field = "em.member_role"
	case "pricing_factor":
		field = "em.pricing_factor"
	case "updated_at":
		field = "em.updated_at"
	}
	return field + " " + sortOrder + ", em.id DESC"
}

func enterpriseInviteOrderBy(params pagination.PaginationParams) string {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := "created_at"
	switch sortBy {
	case "code":
		field = "code"
	case "status":
		field = "status"
	case "used_count":
		field = "used_count"
	case "expires_at":
		field = "expires_at"
	case "updated_at":
		field = "updated_at"
	}
	return field + " " + sortOrder + ", id DESC"
}

func enterpriseLedgerOrderBy(params pagination.PaginationParams) string {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)
	field := "l.created_at"
	switch sortBy {
	case "direction":
		field = "l.direction"
	case "amount":
		field = "l.amount"
	}
	return field + " " + sortOrder + ", l.id DESC"
}

func membershipBaseSelect() string {
	return `
SELECT em.id,
       em.tenant_id,
       em.user_id,
       COALESCE(em.member_role, 'member'),
       COALESCE(em.member_note, ''),
       COALESCE(em.joined_via, ''),
       COALESCE(em.joined_source, ''),
       COALESCE(em.pricing_factor::double precision, 0),
       COALESCE(em.pricing_scope, 'balance'),
       em.created_by,
       em.created_at,
       em.updated_at,
       COALESCE(u.email, ''),
       COALESCE(u.username, ''),
       COALESCE(u.status, ''),
       COALESCE(u.balance::double precision, 0),
       COALESCE(u.concurrency, 0)
FROM enterprise_memberships em
JOIN users u ON u.id = em.user_id AND u.deleted_at IS NULL
`
}

func scanEnterpriseTenant(rows *sql.Rows) (service.EnterpriseTenant, error) {
	var item service.EnterpriseTenant
	var createdBy, updatedBy sql.NullInt64
	if err := rows.Scan(
		&item.ID,
		&item.Name,
		&item.Code,
		&item.Status,
		&item.Notes,
		&item.PortalHost,
		&item.PricingFloorFactor,
		&item.MemberDefaultPricingFactor,
		&item.PricingScope,
		&item.Concurrency,
		&item.BalanceQuotaTotal,
		&item.BalanceQuotaUsed,
		&item.BalanceQuotaSpent,
		&item.BalanceOverdraftLimit,
		&createdBy,
		&updatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ManagerCount,
		&item.MemberCount,
	); err != nil {
		return service.EnterpriseTenant{}, err
	}
	item.PricingFloorFactor = service.NormalizePricingDiscountFactorForRepo(item.PricingFloorFactor)
	item.MemberDefaultPricingFactor = service.NormalizeEnterpriseMemberDefaultPricingFactor(item.MemberDefaultPricingFactor)
	item.PricingScope = service.NormalizeEnterprisePricingScopeForRepo(item.PricingScope)
	if createdBy.Valid {
		v := createdBy.Int64
		item.CreatedBy = &v
	}
	if updatedBy.Valid {
		v := updatedBy.Int64
		item.UpdatedBy = &v
	}
	return item, nil
}

func scanEnterpriseMembership(rows *sql.Rows) (service.EnterpriseMembership, error) {
	var item service.EnterpriseMembership
	var createdBy sql.NullInt64
	if err := rows.Scan(
		&item.ID,
		&item.TenantID,
		&item.UserID,
		&item.MemberRole,
		&item.MemberNote,
		&item.JoinedVia,
		&item.JoinedSource,
		&item.PricingFactor,
		&item.PricingScope,
		&createdBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.UserEmail,
		&item.UserUsername,
		&item.UserStatus,
		&item.UserBalance,
		&item.UserConcurrency,
	); err != nil {
		return service.EnterpriseMembership{}, err
	}
	item.PricingFactor = service.NormalizeEnterpriseMemberPricingFactorForRepo(item.PricingFactor)
	item.PricingScope = service.NormalizeEnterprisePricingScopeForRepo(item.PricingScope)
	if createdBy.Valid {
		v := createdBy.Int64
		item.CreatedBy = &v
	}
	return item, nil
}

func scanEnterpriseInviteCode(rows *sql.Rows) (service.EnterpriseInviteCode, error) {
	var item service.EnterpriseInviteCode
	var createdBy sql.NullInt64
	if err := rows.Scan(
		&item.ID,
		&item.TenantID,
		&item.Code,
		&item.Status,
		&item.MaxUses,
		&item.UsedCount,
		&item.ExpiresAt,
		&item.Notes,
		&createdBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return service.EnterpriseInviteCode{}, err
	}
	if createdBy.Valid {
		v := createdBy.Int64
		item.CreatedBy = &v
	}
	return item, nil
}

func scanEnterpriseLedger(rows *sql.Rows) (service.EnterpriseWalletLedgerEntry, error) {
	var item service.EnterpriseWalletLedgerEntry
	var operatorID, targetID sql.NullInt64
	if err := rows.Scan(
		&item.ID,
		&item.TenantID,
		&operatorID,
		&targetID,
		&item.Direction,
		&item.Amount,
		&item.BalanceBefore,
		&item.BalanceAfter,
		&item.Notes,
		&item.CreatedAt,
		&item.OperatorEmail,
		&item.TargetUserEmail,
		&item.TargetUserName,
		&item.TenantName,
		&item.TenantCode,
	); err != nil {
		return service.EnterpriseWalletLedgerEntry{}, err
	}
	if operatorID.Valid {
		v := operatorID.Int64
		item.OperatorUserID = &v
	}
	if targetID.Valid {
		v := targetID.Int64
		item.TargetUserID = &v
	}
	return item, nil
}

func uniquePositiveInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func leftPadInt(v int64, width int) string {
	out := strconv.FormatInt(v, 10)
	for len(out) < width {
		out = "0" + out
	}
	return out
}
