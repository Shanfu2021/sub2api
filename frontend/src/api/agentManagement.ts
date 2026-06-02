import { apiClient } from './client'
import type {
  AgentAllocationSummary,
  AgentDirectChildrenResponse,
  AgentDirectUserCreateRequest,
  AgentGroupDelegationRequest,
  AgentGroupDelegationResponse,
  AgentGroupRate,
  AgentManagedUser,
  AgentManagementSummary,
  AgentUpgradeRequest,
  AgentAllocationUpdate
} from '@/types'

const BASE_PATH = '/agent-management'

export interface AgentDirectChildrenQuery {
  search?: string
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

export async function createDirectUser(payload: AgentDirectUserCreateRequest): Promise<AgentManagedUser> {
  const { data } = await apiClient.post<AgentManagedUser>(`${BASE_PATH}/direct-users`, payload)
  return data
}

export async function updateAllocation(childId: number, payload: AgentAllocationUpdate): Promise<AgentAllocationSummary> {
  const { data } = await apiClient.put<AgentAllocationSummary>(`${BASE_PATH}/children/${childId}/allocation`, payload)
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

export async function removeChildGroupDelegation(childId: number, groupId: number): Promise<AgentGroupDelegationResponse> {
  const { data } = await apiClient.delete<AgentGroupDelegationResponse>(
    `${BASE_PATH}/children/${childId}/groups/${groupId}`
  )
  return data
}

export const agentManagementAPI = {
  getSummary,
  listDirectUsers,
  listDirectAgents,
  listDirectEnterprises,
  createDirectUser,
  updateAllocation,
  upgradeChild,
  deleteDirectChild,
  listGroups,
  setChildGroupDelegation,
  removeChildGroupDelegation,
}

export default agentManagementAPI
