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

import { enterpriseManagementAPI } from '@/api/enterpriseManagement'

describe('enterprise management api', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
    post.mockReset()
    del.mockReset()
  })

  it('imports employees through the enterprise import route', async () => {
    const payload = {
      employees: [{
        email: 'employee@example.com',
        username: 'employee',
        password: 'secret123',
        concurrency: 2,
        rpm: 30,
      }],
    }
    const response = { created_count: 1, skipped_count: 0, skipped: [], created: [], allocation: {} }
    post.mockResolvedValue({ data: response })

    await expect(enterpriseManagementAPI.importEmployees(payload)).resolves.toEqual(response)

    expect(post).toHaveBeenCalledWith('/enterprise-management/employees/import', payload)
  })

  it('initializes all employee balances through the enterprise balance route', async () => {
    const response = {
      employee_count: 2,
      target_balance: 10,
      current_balance: 1,
      required_balance: 19,
      enterprise_balance_before: 100,
      enterprise_balance_after: 81,
    }
    put.mockResolvedValue({ data: response })

    await expect(enterpriseManagementAPI.initializeEmployeeBalances({ balance: 10 })).resolves.toEqual(response)

    expect(put).toHaveBeenCalledWith('/enterprise-management/employees/balances/initialize', { balance: 10 })
  })
})
