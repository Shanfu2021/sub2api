import { apiClient } from './client'
import type {
  EnterpriseContext,
  EnterpriseGroupSummary,
  EnterpriseInviteCode,
  EnterpriseLedgerEntry,
  EnterpriseMembership,
  EnterpriseTenant,
  PaginatedResponse,
  User,
} from '@/types'

export interface EnterpriseMeResponse {
  enterprise: EnterpriseContext | null
  tenant?: EnterpriseTenant | null
}

export async function getMe(): Promise<EnterpriseMeResponse> {
  const { data } = await apiClient.get<EnterpriseMeResponse>('/enterprise/me')
  return data
}

export async function bindInviteCode(code: string): Promise<EnterpriseMembership> {
  const { data } = await apiClient.post<EnterpriseMembership>('/enterprise/bind-invite', { code })
  return data
}

export async function listGroups(): Promise<EnterpriseGroupSummary[]> {
  const { data } = await apiClient.get<EnterpriseGroupSummary[]>('/enterprise/groups')
  return data
}

export async function listMembers(
  page = 1,
  pageSize = 20,
  filters?: { status?: string; role?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseMembership>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseMembership>>('/enterprise/members', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function createMember(payload: {
  email: string
  password: string
  username?: string
  notes?: string
  concurrency?: number
  rpm_limit?: number
  allowed_groups?: number[]
  member_note?: string
  pricing_factor?: number
  pricing_scope?: string
  group_rates?: Record<number, number | null>
  initial_balance?: number
}): Promise<{ membership: EnterpriseMembership; user: User }> {
  const { data } = await apiClient.post<{ membership: EnterpriseMembership; user: User }>('/enterprise/members', payload)
  return data
}

export async function updateMember(
  userId: number,
  payload: { member_role?: string; member_note?: string; pricing_factor?: number; pricing_scope?: string; concurrency?: number; group_rates?: Record<number, number | null>; status?: string; allowed_groups?: number[] }
): Promise<EnterpriseMembership> {
  const { data } = await apiClient.put<EnterpriseMembership>(`/enterprise/members/${userId}`, payload)
  return data
}

export async function updatePricingDefaults(payload: {
  member_default_pricing_factor?: number
  member_default_concurrency?: number
  member_group_rates?: Record<number, number | null>
}): Promise<EnterpriseTenant> {
  const { data } = await apiClient.put<EnterpriseTenant>('/enterprise/pricing-defaults', payload)
  return data
}

export async function adjustMemberBalance(
  userId: number,
  payload: { amount: number; operation?: string; notes?: string }
): Promise<{ membership: EnterpriseMembership; user: User }> {
  const { data } = await apiClient.post<{ membership: EnterpriseMembership; user: User }>(`/enterprise/members/${userId}/balance`, payload)
  return data
}

export async function listInviteCodes(
  page = 1,
  pageSize = 20,
  filters?: { status?: string; search?: string; sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseInviteCode>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseInviteCode>>('/enterprise/invite-codes', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export async function createInviteCode(payload: { code?: string; max_uses?: number; expires_at?: string | null; notes?: string }): Promise<EnterpriseInviteCode> {
  const { data } = await apiClient.post<EnterpriseInviteCode>('/enterprise/invite-codes', payload)
  return data
}

export async function updateInviteCode(inviteId: number, payload: { status?: string; max_uses?: number; expires_at?: string | null; notes?: string }): Promise<EnterpriseInviteCode> {
  const { data } = await apiClient.put<EnterpriseInviteCode>(`/enterprise/invite-codes/${inviteId}`, payload)
  return data
}

export async function listLedger(
  page = 1,
  pageSize = 20,
  filters?: { sort_by?: string; sort_order?: 'asc' | 'desc' }
): Promise<PaginatedResponse<EnterpriseLedgerEntry>> {
  const { data } = await apiClient.get<PaginatedResponse<EnterpriseLedgerEntry>>('/enterprise/ledger', {
    params: { page, page_size: pageSize, ...filters },
  })
  return data
}

export default {
  getMe,
  bindInviteCode,
  listGroups,
  listMembers,
  createMember,
  updateMember,
  updatePricingDefaults,
  adjustMemberBalance,
  listInviteCodes,
  createInviteCode,
  updateInviteCode,
  listLedger,
}
