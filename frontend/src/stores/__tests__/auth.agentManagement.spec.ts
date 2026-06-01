import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import type { User } from '@/types'

vi.mock('@/api', () => ({
  authAPI: {
    getCurrentUser: vi.fn(),
    login: vi.fn(),
    login2FA: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
    register: vi.fn(),
  },
  isTotp2FARequired: () => false,
}))

function userWithRole(role: User['role']): User {
  return {
    id: 1,
    username: `${role}-user`,
    email: `${role}@example.com`,
    role,
    balance: 0,
    concurrency: 0,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  }
}

async function loginAs(role: User['role']) {
  const { authAPI } = await import('@/api')
  vi.mocked(authAPI.login).mockResolvedValueOnce({
    access_token: `${role}-token`,
    refresh_token: `${role}-refresh`,
    expires_in: 3600,
    token_type: 'Bearer',
    user: userWithRole(role),
  })

  const authStore = useAuthStore()
  await authStore.login({ email: `${role}@example.com`, password: 'password' })
  return authStore
}

describe('auth agent management permissions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it.each([
    { role: 'admin', expected: true },
    { role: 'agent_level1', expected: true },
    { role: 'agent_level2', expected: true },
    { role: 'enterprise', expected: false },
    { role: 'user', expected: false },
  ] as const)('sets canUseAgentManagement=$expected for $role role', async ({ role, expected }) => {
    const authStore = await loginAs(role)

    expect(authStore.canUseAgentManagement).toBe(expected)
  })
})
