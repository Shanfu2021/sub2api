import { apiClient } from '../client'
import type {
  EnterpriseInviteCode,
  EnterpriseLedgerEntry,
  EnterpriseMembership,
  EnterpriseTenant,
  PaginatedResponse,
} from '@/types'

export async function listTenants(
  page = 1,
  pageSize = 20,
  filters?: { status?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseTenant>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseTenant>>('/admin/enterprise/tenants', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function getTenant(id: number): Promise<EnterpriseTenant> {
  const { data } = await apiClient.get<EnterpriseTenant>(`/admin/enterprise/tenants/${id}`)
  return data
}

export async function createTenant(payload: Partial<EnterpriseTenant> & { name: string; code?: string }): Promise<EnterpriseTenant> {
  const { data } = await apiClient.post<EnterpriseTenant>('/admin/enterprise/tenants', payload)
  return data
}

export async function updateTenant(id: number, payload: Partial<EnterpriseTenant>): Promise<EnterpriseTenant> {
  const { data } = await apiClient.put<EnterpriseTenant>(`/admin/enterprise/tenants/${id}`, payload)
  return data
}

export async function adjustQuota(id: number, payload: { amount: number; direction: string; notes?: string }): Promise<EnterpriseTenant> {
  const { data } = await apiClient.post<EnterpriseTenant>(`/admin/enterprise/tenants/${id}/quota`, payload)
  return data
}

export async function listMembers(
  tenantId: number,
  page = 1,
  pageSize = 20,
  filters?: { status?: string; role?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseMembership>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseMembership>>(`/admin/enterprise/tenants/${tenantId}/members`, {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function bindMember(
  tenantId: number,
  payload: { user_id: number; member_role?: string; member_note?: string; pricing_factor?: number; pricing_scope?: string; group_rates?: Record<number, number | null>; joined_via?: string; joined_source?: string }
): Promise<EnterpriseMembership> {
  const { data } = await apiClient.post<EnterpriseMembership>(`/admin/enterprise/tenants/${tenantId}/members`, payload)
  return data
}

export async function updateMember(
  tenantId: number,
  userId: number,
  payload: { member_role?: string; member_note?: string; pricing_factor?: number; pricing_scope?: string; group_rates?: Record<number, number | null>; status?: string; allowed_groups?: number[] }
): Promise<EnterpriseMembership> {
  const { data } = await apiClient.put<EnterpriseMembership>(`/admin/enterprise/tenants/${tenantId}/members/${userId}`, payload)
  return data
}

export async function deleteMember(tenantId: number, userId: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/enterprise/tenants/${tenantId}/members/${userId}`)
  return data
}

export async function listInviteCodes(
  tenantId: number,
  page = 1,
  pageSize = 20,
  filters?: { status?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseInviteCode>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseInviteCode>>(`/admin/enterprise/tenants/${tenantId}/invite-codes`, {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function createInviteCode(
  tenantId: number,
  payload: { code?: string; max_uses?: number; expires_at?: string | null; notes?: string }
): Promise<EnterpriseInviteCode> {
  const { data } = await apiClient.post<EnterpriseInviteCode>(`/admin/enterprise/tenants/${tenantId}/invite-codes`, payload)
  return data
}

export async function updateInviteCode(
  inviteId: number,
  payload: { status?: string; max_uses?: number; expires_at?: string | null; notes?: string }
): Promise<EnterpriseInviteCode> {
  const { data } = await apiClient.put<EnterpriseInviteCode>(`/admin/enterprise/invite-codes/${inviteId}`, payload)
  return data
}

export async function listLedger(
  tenantId: number,
  page = 1,
  pageSize = 20,
  filters?: { sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseLedgerEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseLedgerEntry>>(`/admin/enterprise/tenants/${tenantId}/ledger`, {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export default {
  listTenants,
  getTenant,
  createTenant,
  updateTenant,
  adjustQuota,
  listMembers,
  bindMember,
  updateMember,
  deleteMember,
  listInviteCodes,
  createInviteCode,
  updateInviteCode,
  listLedger,
}
