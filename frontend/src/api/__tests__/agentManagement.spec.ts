import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put, post, del } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  del: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    put,
    post,
    delete: del,
  },
}))

import { agentManagementAPI } from '@/api/agentManagement'

describe('agent management api', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
    post.mockReset()
    del.mockReset()
  })

  it('loads direct users from the backend agent management route', async () => {
    const response = { items: [], pagination: { page: 1, page_size: 20, total: 0, total_pages: 0 } }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listDirectUsers()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/direct-users', { params: {} })
  })

  it('passes search query to direct user list route', async () => {
    const response = { items: [], pagination: { page: 1, page_size: 20, total: 0, total_pages: 0 } }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listDirectUsers({ search: 'alice' })).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/direct-users', { params: { search: 'alice' } })
  })

  it('loads the admin agent tree overview', async () => {
    const response = { items: [{ agent: { id: 12, email: 'agent@example.test' }, users: [], enterprises: [] }] }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.getAdminAgentTree()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/admin-agent-tree')
  })

  it('loads subordinate structure with optional owner id', async () => {
    const response = { owner_options: [], selected_owner: { id: 1 }, users: [], enterprises: [] }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.getStructure(12)).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/structure', { params: { owner_id: 12 } })
  })

  it('loads agent-scoped usage records and stats', async () => {
    const listResponse = { items: [], total: 0, page: 1, page_size: 20, pages: 1 }
    const statsResponse = { total_requests: 0, total_actual_cost: 0 }
    get.mockResolvedValueOnce({ data: listResponse })
    get.mockResolvedValueOnce({ data: statsResponse })

    await expect(agentManagementAPI.listUsage({ page: 1, page_size: 20, user_id: 99 })).resolves.toEqual(listResponse)
    await expect(agentManagementAPI.getUsageStats({ user_id: 99 })).resolves.toEqual(statsResponse)

    expect(get).toHaveBeenNthCalledWith(1, '/agent-management/usage', { params: { page: 1, page_size: 20, user_id: 99 } })
    expect(get).toHaveBeenNthCalledWith(2, '/agent-management/usage/stats', { params: { user_id: 99 } })
  })

  it('loads agent usage user options', async () => {
    const response = [{ id: 99, email: 'user@example.test', role: 'user' }]
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listUsageUsers()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/usage/users')
  })

  it('updates a direct child allocation without balance fields', async () => {
    const response = {
      total_concurrency: 100,
      allocated_concurrency: 5,
      remaining_concurrency: 95,
      total_rpm: 1000,
      allocated_rpm: 60,
      remaining_rpm: 940,
      unlimited_capacity: false,
    }
    const payload = { concurrency: 5, rpm: 60 }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.updateAllocation(12, payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/children/12/allocation', payload)
  })

  it('sets a direct agent income target through the backend settlement route', async () => {
    const response = { id: 12, role: 'agent_level1', agent_income: 0 }
    const payload = { agent_income: 0, reason: 'settled' }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.setAgentIncome(12, payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/children/12/agent-income', payload)
  })

  it('updates agent invitation registration defaults', async () => {
    const response = {
      user_id: 12,
      pool_concurrency: 100,
      pool_rpm: 1000,
      invite_default_concurrency: 3,
      invite_default_rpm: 30,
    }
    const payload = { invite_default_concurrency: 3, invite_default_rpm: 30 }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.updateInviteDefaults(payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/invite-defaults', payload)
  })

  it('creates a direct user through the backend agent management route', async () => {
    const response = { id: 99, email: 'direct@example.com', role: 'user' }
    const payload = {
      email: 'direct@example.com',
      password: 'secret123',
      username: 'direct',
      allocated_concurrency: 5,
      allocated_rpm: 60,
    }
    post.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.createDirectUser(payload)).resolves.toEqual(response)

    expect(post).toHaveBeenCalledWith('/agent-management/direct-users', payload)
  })

  it('upgrades a direct child through the backend upgrade route', async () => {
    const upgraded = { id: 12, role: 'agent_level1' }
    post.mockResolvedValue({ data: upgraded })

    await expect(agentManagementAPI.upgradeChild(12, {
      target_role: 'agent_level1',
      pool_concurrency: 100,
      pool_rpm: 1000,
    })).resolves.toEqual(upgraded)

    expect(post).toHaveBeenCalledWith('/agent-management/children/12/upgrade', {
      target_role: 'agent_level1',
      pool_concurrency: 100,
      pool_rpm: 1000,
    })
  })

  it('loads child group delegation options with assignment state', async () => {
    const response = [{ group: { id: 7, name: 'Exclusive Retail' }, assigned: true }]
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listChildGroupDelegationOptions(12)).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/children/12/groups')
  })

  it('loads invitation default group options', async () => {
    const response = [{ group: { id: 7, name: 'Exclusive Retail' }, assigned: true }]
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listInviteGroupDefaultOptions()).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/invite-default-groups')
  })

  it('updates and removes invitation default group assignments', async () => {
    const response = { group_id: 7 }
    put.mockResolvedValueOnce({ data: response })
    del.mockResolvedValueOnce({ data: response })

    await expect(agentManagementAPI.setInviteGroupDefault(7, { rate_multiplier: 2.4 })).resolves.toEqual(response)
    await expect(agentManagementAPI.removeInviteGroupDefault(7)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/invite-default-groups/7', { rate_multiplier: 2.4 })
    expect(del).toHaveBeenCalledWith('/agent-management/invite-default-groups/7')
  })

  it('updates child group delegations in batch', async () => {
    const response = { child_id: 12, group_ids: [7, 8], all: false }
    const payload = { group_ids: [7, 8], all: false, rate_multiplier: 3.2, can_delegate: true }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.setChildGroupDelegationsBatch(12, payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/children/12/groups/batch', payload)
  })

  it('updates direct children group delegations in batch by child kind', async () => {
    const response = { kind: 'enterprises', group_ids: [7, 8], all: false, child_ids: [12], all_children: false, updated_children: 3 }
    const payload = { group_ids: [7, 8], all: false, child_ids: [12], all_children: false, rate_multiplier: 3.2, can_delegate: true }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.setDirectChildrenGroupDelegationsBatch('enterprises', payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/direct-enterprises/groups/batch', payload)
  })

  it('loads direct children that already have a selected group', async () => {
    const response = { items: [], pagination: { page: 1, page_size: 20, total: 0, pages: 1 } }
    const query = { group_id: 7, search: 'alice', page: 1, page_size: 20 }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listDirectChildrenWithGroupDelegation('users', query)).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/direct-users/groups/assigned', { params: query })
  })

  it('loads direct children missing any selected group for deployment candidates', async () => {
    const response = { items: [], pagination: { page: 1, page_size: 20, total: 0, pages: 1 } }
    const query = { group_ids: [7, 8], search: 'alice', page: 1, page_size: 20 }
    get.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.listDirectChildrenWithoutGroupDelegation('users', query)).resolves.toEqual(response)

    expect(get).toHaveBeenCalledWith('/agent-management/direct-users/groups/unassigned', {
      params: { group_ids: '7,8', search: 'alice', page: 1, page_size: 20 },
    })
  })

  it('updates existing direct children group delegations without creating missing groups', async () => {
    const response = { kind: 'users', group_id: 7, requested_child_ids: [12], all: false, updated_children: 1, skipped_children: 0 }
    const payload = { group_id: 7, child_ids: [12], all: false, rate_multiplier: 2.4 }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.updateDirectChildrenExistingGroupDelegations('users', payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/direct-users/groups/existing', payload)
  })

  it('reclaims direct children group delegations by child kind', async () => {
    const response = { kind: 'enterprises', group_id: 7, requested_child_ids: [12], all: false, removed_children: 1, skipped_children: 0 }
    const payload = { group_id: 7, child_ids: [12], all: false }
    post.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.reclaimDirectChildrenGroupDelegations('enterprises', payload)).resolves.toEqual(response)

    expect(post).toHaveBeenCalledWith('/agent-management/direct-enterprises/groups/reclaim', payload)
  })

  it('updates invite default groups in batch', async () => {
    const response = { group_ids: [], all: true }
    const payload = { group_ids: [], all: true, rate_multiplier: 3.2 }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.setInviteGroupDefaultsBatch(payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/invite-default-groups/batch', payload)
  })
})
