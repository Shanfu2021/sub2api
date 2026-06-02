import { apiClient } from './client'
import type {
  EnterpriseAllocationSummary,
  EnterpriseEmployee,
  EnterpriseEmployeeAllocationUpdate,
  EnterpriseEmployeeCreateRequest,
  EnterpriseEmployeeGroupOption,
  EnterpriseEmployeeGroupRequest,
  EnterpriseEmployeeGroupResponse,
  EnterpriseEmployeesResponse,
  EnterpriseGroupRate,
  EnterpriseManagementSummary,
} from '@/types'

const BASE_PATH = '/enterprise-management'

export interface EnterpriseEmployeesQuery {
  search?: string
}

export async function getSummary(): Promise<EnterpriseManagementSummary> {
  const { data } = await apiClient.get<EnterpriseManagementSummary>(`${BASE_PATH}/summary`)
  return data
}

export const summary = getSummary

export async function listEmployees(query: EnterpriseEmployeesQuery = {}): Promise<EnterpriseEmployeesResponse> {
  const { data } = await apiClient.get<EnterpriseEmployeesResponse>(`${BASE_PATH}/employees`, { params: query })
  return data
}

export async function createEmployee(payload: EnterpriseEmployeeCreateRequest): Promise<EnterpriseEmployee> {
  const { data } = await apiClient.post<EnterpriseEmployee>(`${BASE_PATH}/employees`, payload)
  return data
}

export async function updateEmployeeAllocation(
  employeeId: number,
  payload: EnterpriseEmployeeAllocationUpdate
): Promise<EnterpriseAllocationSummary> {
  const { data } = await apiClient.put<EnterpriseAllocationSummary>(
    `${BASE_PATH}/employees/${employeeId}/allocation`,
    payload
  )
  return data
}

export async function deleteEmployee(employeeId: number): Promise<{ id: number }> {
  const { data } = await apiClient.delete<{ id: number }>(`${BASE_PATH}/employees/${employeeId}`)
  return data
}

export async function listGroups(): Promise<EnterpriseGroupRate[]> {
  const { data } = await apiClient.get<EnterpriseGroupRate[]>(`${BASE_PATH}/groups`)
  return data
}

export async function listEmployeeGroupOptions(employeeId: number): Promise<EnterpriseEmployeeGroupOption[]> {
  const { data } = await apiClient.get<EnterpriseEmployeeGroupOption[]>(
    `${BASE_PATH}/employees/${employeeId}/groups`
  )
  return data
}

export async function setEmployeeGroup(
  employeeId: number,
  groupId: number,
  payload: EnterpriseEmployeeGroupRequest
): Promise<EnterpriseEmployeeGroupResponse> {
  const { data } = await apiClient.put<EnterpriseEmployeeGroupResponse>(
    `${BASE_PATH}/employees/${employeeId}/groups/${groupId}`,
    payload
  )
  return data
}

export async function removeEmployeeGroup(
  employeeId: number,
  groupId: number
): Promise<EnterpriseEmployeeGroupResponse> {
  const { data } = await apiClient.delete<EnterpriseEmployeeGroupResponse>(
    `${BASE_PATH}/employees/${employeeId}/groups/${groupId}`
  )
  return data
}

export const enterpriseManagementAPI = {
  summary,
  getSummary,
  listEmployees,
  createEmployee,
  updateEmployeeAllocation,
  deleteEmployee,
  listGroups,
  listEmployeeGroupOptions,
  setEmployeeGroup,
  removeEmployeeGroup,
}

export default enterpriseManagementAPI
