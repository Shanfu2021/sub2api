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

    expect(get).toHaveBeenCalledWith('/agent-management/direct-users')
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
    const payload = { allocated_concurrency: 5, allocated_rpm: 60 }
    put.mockResolvedValue({ data: response })

    await expect(agentManagementAPI.updateAllocation(12, payload)).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/agent-management/children/12/allocation', payload)
  })

  it('upgrades a direct child through the backend upgrade route', async () => {
    const upgraded = { id: 12, role: 'agent_level1' }
    post.mockResolvedValue({ data: upgraded })

    await expect(agentManagementAPI.upgradeChild(12, 'agent_level1')).resolves.toEqual(upgraded)

    expect(post).toHaveBeenCalledWith('/agent-management/children/12/upgrade', {
      target_role: 'agent_level1',
    })
  })
})
