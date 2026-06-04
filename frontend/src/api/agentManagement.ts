import { apiClient } from './client'
import type {
  AgentAllocationSummary,
  AgentAdminTreeResponse,
  AgentChildGroupDelegationOption,
  AgentDirectChildKind,
  AgentDirectChildrenGroupDelegationBatchResponse,
  AgentDirectChildrenResponse,
  AgentGroupDelegationBatchRequest,
  AgentGroupDelegationBatchResponse,
  AgentDirectUserCreateRequest,
  AgentGroupDelegationRequest,
  AgentGroupDelegationResponse,
  AgentGroupRate,
  AgentIncomeSetRequest,
  AgentInviteGroupDefaultBatchRequest,
  AgentInviteGroupDefaultBatchResponse,
  AgentInviteGroupDefaultRequest,
  AgentInviteGroupDefaultResponse,
  AgentInviteDefaultsUpdate,
  AgentManagedUser,
  AgentManagementSummary,
  AgentProfile,
  AgentUpgradeRequest,
  AgentAllocationUpdate,
  AgentStructureResponse,
  AgentUsageLog,
  PaginatedResponse
} from '@/types'
import type { AdminUsageQueryParams, AdminUsageStatsResponse } from '@/api/admin/usage'

const BASE_PATH = '/agent-management'

export interface AgentDirectChildrenQuery {
  search?: string
  page?: number
  page_size?: number
}

const DIRECT_CHILD_KIND_PATH: Record<AgentDirectChildKind, string> = {
  users: 'direct-users',
  agents: 'direct-agents',
  enterprises: 'direct-enterprises',
}

export async function getSummary(): Promise<AgentManagementSummary> {
  const { data } = await apiClient.get<AgentManagementSummary>(`${BASE_PATH}/summary`)
  return data
}

export async function listDirectUsers(query: AgentDirectChildrenQuery = {}): Promise<AgentDirectChildrenResponse> {
  const { data } = await apiClient.get<AgentDirectChildrenResponse>(`${BASE_PATH}/direct-users`, { params: query })
  return data
}

export async function listDirectAgents(query: AgentDirectChildrenQuery = {}): Promise<AgentDirectChildrenResponse> {
  const { data } = await apiClient.get<AgentDirectChildrenResponse>(`${BASE_PATH}/direct-agents`, { params: query })
  return data
}

export async function listDirectEnterprises(query: AgentDirectChildrenQuery = {}): Promise<AgentDirectChildrenResponse> {
  const { data } = await apiClient.get<AgentDirectChildrenResponse>(`${BASE_PATH}/direct-enterprises`, { params: query })
  return data
}

export async function getAdminAgentTree(): Promise<AgentAdminTreeResponse> {
  const { data } = await apiClient.get<AgentAdminTreeResponse>(`${BASE_PATH}/admin-agent-tree`)
  return data
}

export async function getStructure(ownerId?: number): Promise<AgentStructureResponse> {
  const params = ownerId ? { owner_id: ownerId } : {}
  const { data } = await apiClient.get<AgentStructureResponse>(`${BASE_PATH}/structure`, { params })
  return data
}

export async function listUsage(
  params: AdminUsageQueryParams,
  options?: { signal?: AbortSignal }
): Promise<PaginatedResponse<AgentUsageLog>> {
  const { data } = await apiClient.get<PaginatedResponse<AgentUsageLog>>(`${BASE_PATH}/usage`, {
    params,
    signal: options?.signal
  })
  return data
}

export async function getUsageStats(params: AdminUsageQueryParams): Promise<AdminUsageStatsResponse> {
  const { data } = await apiClient.get<AdminUsageStatsResponse>(`${BASE_PATH}/usage/stats`, { params })
  return data
}

export async function listUsageUsers(): Promise<AgentManagedUser[]> {
  const { data } = await apiClient.get<AgentManagedUser[]>(`${BASE_PATH}/usage/users`)
  return data
}

export async function createDirectUser(payload: AgentDirectUserCreateRequest): Promise<AgentManagedUser> {
  const { data } = await apiClient.post<AgentManagedUser>(`${BASE_PATH}/direct-users`, payload)
  return data
}

export async function updateAllocation(childId: number, payload: AgentAllocationUpdate): Promise<AgentAllocationSummary> {
  const { data } = await apiClient.put<AgentAllocationSummary>(`${BASE_PATH}/children/${childId}/allocation`, payload)
  return data
}

export async function updateInviteDefaults(payload: AgentInviteDefaultsUpdate): Promise<AgentProfile> {
  const { data } = await apiClient.put<AgentProfile>(`${BASE_PATH}/invite-defaults`, payload)
  return data
}

export async function upgradeChild(childId: number, payload: AgentUpgradeRequest): Promise<AgentManagedUser> {
  const { data } = await apiClient.post<AgentManagedUser>(`${BASE_PATH}/children/${childId}/upgrade`, payload)
  return data
}

export async function deleteDirectChild(childId: number): Promise<{ id: number }> {
  const { data } = await apiClient.delete<{ id: number }>(`${BASE_PATH}/children/${childId}`)
  return data
}

export async function listGroups(): Promise<AgentGroupRate[]> {
  const { data } = await apiClient.get<AgentGroupRate[]>(`${BASE_PATH}/groups`)
  return data
}

export async function listChildGroupDelegationOptions(childId: number): Promise<AgentChildGroupDelegationOption[]> {
  const { data } = await apiClient.get<AgentChildGroupDelegationOption[]>(`${BASE_PATH}/children/${childId}/groups`)
  return data
}

export async function listInviteGroupDefaultOptions(): Promise<AgentChildGroupDelegationOption[]> {
  const { data } = await apiClient.get<AgentChildGroupDelegationOption[]>(`${BASE_PATH}/invite-default-groups`)
  return data
}

export async function setChildGroupDelegation(
  childId: number,
  groupId: number,
  payload: AgentGroupDelegationRequest
): Promise<AgentGroupDelegationResponse> {
  const { data } = await apiClient.put<AgentGroupDelegationResponse>(
    `${BASE_PATH}/children/${childId}/groups/${groupId}`,
    payload
  )
  return data
}

export async function setChildGroupDelegationsBatch(
  childId: number,
  payload: AgentGroupDelegationBatchRequest
): Promise<AgentGroupDelegationBatchResponse> {
  const { data } = await apiClient.put<AgentGroupDelegationBatchResponse>(
    `${BASE_PATH}/children/${childId}/groups/batch`,
    payload
  )
  return data
}

export async function setDirectChildrenGroupDelegationsBatch(
  kind: AgentDirectChildKind,
  payload: AgentGroupDelegationBatchRequest
): Promise<AgentDirectChildrenGroupDelegationBatchResponse> {
  const { data } = await apiClient.put<AgentDirectChildrenGroupDelegationBatchResponse>(
    `${BASE_PATH}/${DIRECT_CHILD_KIND_PATH[kind]}/groups/batch`,
    payload
  )
  return data
}

export async function setAgentIncome(childId: number, payload: AgentIncomeSetRequest): Promise<AgentManagedUser> {
  const { data } = await apiClient.put<AgentManagedUser>(`${BASE_PATH}/children/${childId}/agent-income`, payload)
  return data
}

export async function removeChildGroupDelegation(childId: number, groupId: number): Promise<AgentGroupDelegationResponse> {
  const { data } = await apiClient.delete<AgentGroupDelegationResponse>(
    `${BASE_PATH}/children/${childId}/groups/${groupId}`
  )
  return data
}

export async function setInviteGroupDefault(
  groupId: number,
  payload: AgentInviteGroupDefaultRequest
): Promise<AgentInviteGroupDefaultResponse> {
  const { data } = await apiClient.put<AgentInviteGroupDefaultResponse>(
    `${BASE_PATH}/invite-default-groups/${groupId}`,
    payload
  )
  return data
}

export async function setInviteGroupDefaultsBatch(
  payload: AgentInviteGroupDefaultBatchRequest
): Promise<AgentInviteGroupDefaultBatchResponse> {
  const { data } = await apiClient.put<AgentInviteGroupDefaultBatchResponse>(
    `${BASE_PATH}/invite-default-groups/batch`,
    payload
  )
  return data
}

export async function removeInviteGroupDefault(groupId: number): Promise<AgentInviteGroupDefaultResponse> {
  const { data } = await apiClient.delete<AgentInviteGroupDefaultResponse>(
    `${BASE_PATH}/invite-default-groups/${groupId}`
  )
  return data
}

export const agentManagementAPI = {
  getSummary,
  listDirectUsers,
  listDirectAgents,
  listDirectEnterprises,
  getAdminAgentTree,
  getStructure,
  listUsage,
  getUsageStats,
  listUsageUsers,
  createDirectUser,
  updateAllocation,
  updateInviteDefaults,
  upgradeChild,
  deleteDirectChild,
  listGroups,
  listChildGroupDelegationOptions,
  listInviteGroupDefaultOptions,
  setChildGroupDelegation,
  setChildGroupDelegationsBatch,
  setDirectChildrenGroupDelegationsBatch,
  setAgentIncome,
  removeChildGroupDelegation,
  setInviteGroupDefault,
  setInviteGroupDefaultsBatch,
  removeInviteGroupDefault,
}

export default agentManagementAPI
