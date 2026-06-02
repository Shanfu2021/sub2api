import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DirectUsersView from '@/views/agent/DirectUsersView.vue'
import DirectAgentsView from '@/views/agent/DirectAgentsView.vue'
import DirectEnterprisesView from '@/views/agent/DirectEnterprisesView.vue'
import MyGroupsView from '@/views/agent/MyGroupsView.vue'
import type { AgentDirectChildrenResponse, AgentManagedUser, User, UserRole } from '@/types'

const {
  getCurrentUser,
  getSummary,
  listDirectUsers,
  listDirectAgents,
  listDirectEnterprises,
  updateAllocation,
  createDirectUser,
  updateInviteDefaults,
  upgradeChild,
  deleteDirectChild,
  listGroups,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getCurrentUser: vi.fn(),
  getSummary: vi.fn(),
  listDirectUsers: vi.fn(),
  listDirectAgents: vi.fn(),
  listDirectEnterprises: vi.fn(),
  updateAllocation: vi.fn(),
  createDirectUser: vi.fn(),
  updateInviteDefaults: vi.fn(),
  upgradeChild: vi.fn(),
  deleteDirectChild: vi.fn(),
  listGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/agentManagement', () => ({
  agentManagementAPI: {
    getSummary,
    listDirectUsers,
    listDirectAgents,
    listDirectEnterprises,
    updateAllocation,
    createDirectUser,
    updateInviteDefaults,
    upgradeChild,
    deleteDirectChild,
    listGroups,
    setChildGroupDelegation: vi.fn(),
    removeChildGroupDelegation: vi.fn(),
  },
}))

vi.mock('@/api', () => ({
  authAPI: {
    getCurrentUser,
    login: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
  },
  isTotp2FARequired: () => false,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
}
const DataTableStub = {
  props: ['columns', 'data'],
  template: `
    <div data-test="agent-table">
      <div data-test="columns">{{ columns.map((column) => column.key).join(',') }}</div>
      <div v-for="row in data" :key="row.id" data-test="row">
        <slot name="cell-email" :row="row" :value="row.email" />
        <slot name="cell-role" :row="row" :value="row.role" />
        <slot name="cell-balance" :row="row" :value="row.balance" />
        <slot name="cell-allocation" :row="row" />
        <slot name="cell-name" :row="row" :value="row.name" />
        <slot name="cell-source" :row="row" :value="row.source" />
        <slot name="cell-effective_rate" :row="row" :value="row.effective_rate" />
        <slot name="cell-can_delegate" :row="row" :value="row.can_delegate" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `,
}

function makeUser(role: UserRole): User {
  return {
    id: 1,
    username: `${role}-manager`,
    email: `${role}@example.com`,
    role,
    balance: 0,
    concurrency: 20,
    rpm_limit: 200,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  }
}

function makeChild(overrides: Partial<AgentManagedUser> = {}): AgentManagedUser {
  return {
    id: 12,
    email: 'child@example.com',
    username: 'child',
    role: 'user',
    parent_user_id: 1,
    concurrency: 10,
    rpm_limit: 100,
    pool_concurrency: 0,
    pool_rpm: 0,
    invite_default_concurrency: 1,
    invite_default_rpm: 1,
    balance: 12.5,
    allocated_concurrency: 3,
    allocated_rpm: 30,
    status: 'active',
    created_at: '2026-06-01T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
    ...overrides,
  }
}

function makeChildrenResponse(items: AgentManagedUser[]): AgentDirectChildrenResponse {
  return {
    items,
    pagination: { total: items.length, page: 1, page_size: 20, pages: 1 },
  }
}

function mountAgentView(component: unknown, role: UserRole = 'admin') {
  localStorage.setItem('auth_token', `${role}-token`)
  localStorage.setItem('auth_user', JSON.stringify(makeUser(role)))
  getCurrentUser.mockResolvedValue({ data: makeUser(role) })

  return mount(component as any, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: true,
        BaseDialog: { template: '<div v-if="show"><slot /><slot name="footer" /></div>', props: ['show'] },
        Select: true,
        Icon: true,
        Teleport: true,
      },
    },
  })
}

describe('agent management pages', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()

    getSummary.mockResolvedValue({
      allocation: {
        total_concurrency: 20,
        allocated_concurrency: 3,
        remaining_concurrency: 17,
        total_rpm: 200,
        allocated_rpm: 30,
        remaining_rpm: 170,
        unlimited_capacity: false,
        unlimited_concurrency: false,
        unlimited_rpm: false,
      },
      invite_defaults: {
        invite_default_concurrency: 2,
        invite_default_rpm: 20,
      },
    })
    listDirectUsers.mockResolvedValue(makeChildrenResponse([makeChild()]))
    listDirectAgents.mockResolvedValue(makeChildrenResponse([makeChild({ role: 'agent_level2' })]))
    listDirectEnterprises.mockResolvedValue(makeChildrenResponse([makeChild({ role: 'enterprise' })]))
    updateAllocation.mockResolvedValue({
      total_concurrency: 20,
      allocated_concurrency: 5,
      remaining_concurrency: 15,
      total_rpm: 200,
      allocated_rpm: 50,
      remaining_rpm: 150,
      unlimited_capacity: false,
      unlimited_concurrency: false,
      unlimited_rpm: false,
    })
    createDirectUser.mockResolvedValue(makeChild({ id: 99, email: 'direct@example.com', username: 'direct' }))
    updateInviteDefaults.mockResolvedValue({
      user_id: 1,
      pool_concurrency: 20,
      pool_rpm: 200,
      invite_default_concurrency: 4,
      invite_default_rpm: 40,
    })
    upgradeChild.mockResolvedValue(makeChild({ role: 'agent_level1' }))
    deleteDirectChild.mockResolvedValue({ id: 12 })
    listGroups.mockResolvedValue([
      {
        group: {
          id: 7,
          name: 'Exclusive Retail',
          description: null,
          platform: 'openai',
          rate_multiplier: 1.7,
          is_exclusive: true,
          status: 'active',
          subscription_type: 'standard',
          daily_limit_usd: null,
          weekly_limit_usd: null,
          monthly_limit_usd: null,
          allow_image_generation: false,
          image_rate_independent: false,
          image_rate_multiplier: 1,
          image_price_1k: null,
          image_price_2k: null,
          image_price_4k: null,
          claude_code_only: false,
          fallback_group_id: null,
          fallback_group_id_on_invalid_request: null,
          require_oauth_only: false,
          require_privacy_set: false,
          created_at: '2026-06-01T00:00:00Z',
          updated_at: '2026-06-01T00:00:00Z',
        },
        effective_rate: 2.4,
        can_delegate: true,
        source: 'delegated',
      },
    ])
  })

  it('does not render balance, recharge, disable, or official delete actions on direct users', async () => {
    const wrapper = mountAgentView(DirectUsersView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Recharge')
    expect(wrapper.text()).not.toContain('Disable')
    expect(wrapper.text()).not.toContain('Official Delete')
  })

  it('renders direct child balance as read-only context', async () => {
    const wrapper = mountAgentView(DirectUsersView)
    await flushPromises()

    expect(wrapper.text()).toContain('12.50')
    expect(wrapper.find('[data-test="allocation-balance-12"]').exists()).toBe(false)
  })

  it('renders concurrency and RPM allocation controls for direct users', async () => {
    const wrapper = mountAgentView(DirectUsersView)
    await flushPromises()

    expect(wrapper.get('[data-test="allocation-concurrency-12"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="allocation-rpm-12"]').exists()).toBe(true)
  })

  it('lets agents save invitation registration defaults on the direct users page', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    expect((wrapper.get('[data-test="invite-default-concurrency"]').element as HTMLInputElement).value).toBe('2')
    expect((wrapper.get('[data-test="invite-default-rpm"]').element as HTMLInputElement).value).toBe('20')

    await wrapper.get('[data-test="invite-default-concurrency"]').setValue('4')
    await wrapper.get('[data-test="invite-default-rpm"]').setValue('40')
    await wrapper.get('[data-test="invite-default-submit"]').trigger('submit')
    await flushPromises()

    expect(updateInviteDefaults).toHaveBeenCalledWith({
      invite_default_concurrency: 4,
      invite_default_rpm: 40,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.direct.inviteDefaultsSaved')
  })

  it('sends search query when filtering direct children', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="direct-child-search"]').setValue('alice')
    await wrapper.get('[data-test="direct-child-search-submit"]').trigger('click')
    await flushPromises()

    expect(listDirectUsers).toHaveBeenLastCalledWith({ search: 'alice' })
  })

  it('uses effective concurrency and RPM fields for direct ordinary users', async () => {
    listDirectUsers.mockResolvedValue(makeChildrenResponse([
      makeChild({ concurrency: 10, rpm_limit: 120, allocated_concurrency: 0, allocated_rpm: 0 }),
    ]))
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    expect((wrapper.get('[data-test="allocation-concurrency-12"]').element as HTMLInputElement).value).toBe('10')
    expect((wrapper.get('[data-test="allocation-rpm-12"]').element as HTMLInputElement).value).toBe('120')
  })

  it('uses pool fields for direct agents', async () => {
    listDirectAgents.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'agent_level2', concurrency: 5, rpm_limit: 50, pool_concurrency: 30, pool_rpm: 300 }),
    ]))
    const wrapper = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()

    expect((wrapper.get('[data-test="allocation-concurrency-12"]').element as HTMLInputElement).value).toBe('30')
    expect((wrapper.get('[data-test="allocation-rpm-12"]').element as HTMLInputElement).value).toBe('300')
  })

  it('uses effective concurrency and RPM fields for direct enterprises', async () => {
    listDirectEnterprises.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'enterprise', concurrency: 18, rpm_limit: 190, allocated_concurrency: 0, allocated_rpm: 0 }),
    ]))
    const wrapper = mountAgentView(DirectEnterprisesView, 'admin')
    await flushPromises()

    expect((wrapper.get('[data-test="allocation-concurrency-12"]').element as HTMLInputElement).value).toBe('18')
    expect((wrapper.get('[data-test="allocation-rpm-12"]').element as HTMLInputElement).value).toBe('190')
  })

  it('opens a modal and creates direct users from the direct users page', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="create-direct-user"]').trigger('click')
    expect(wrapper.find('[data-test="create-direct-user-modal"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('admin.users.columns.balance')

    await wrapper.get('[data-test="create-direct-user-email"]').setValue('direct@example.com')
    await wrapper.get('[data-test="create-direct-user-password"]').setValue('secret123')
    await wrapper.get('[data-test="create-direct-user-username"]').setValue('direct')
    await wrapper.get('[data-test="create-direct-user-concurrency"]').setValue('5')
    await wrapper.get('[data-test="create-direct-user-rpm"]').setValue('60')
    await wrapper.get('[data-test="create-direct-user-submit"]').trigger('submit')
    await flushPromises()

    expect(createDirectUser).toHaveBeenCalledWith({
      email: 'direct@example.com',
      password: 'secret123',
      username: 'direct',
      allocated_concurrency: 5,
      allocated_rpm: 60,
    })
    expect(listDirectUsers).toHaveBeenCalledTimes(2)
  })

  it('blocks over-allocation in the create user modal for agents', async () => {
    getSummary.mockResolvedValue({
      allocation: {
        total_concurrency: 20,
        allocated_concurrency: 18,
        remaining_concurrency: 2,
        total_rpm: 200,
        allocated_rpm: 190,
        remaining_rpm: 10,
        unlimited_capacity: false,
        unlimited_concurrency: false,
        unlimited_rpm: false,
      },
    })
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="create-direct-user"]').trigger('click')
    await wrapper.get('[data-test="create-direct-user-email"]').setValue('direct@example.com')
    await wrapper.get('[data-test="create-direct-user-password"]').setValue('secret123')
    await wrapper.get('[data-test="create-direct-user-concurrency"]').setValue('3')
    await wrapper.get('[data-test="create-direct-user-rpm"]').setValue('11')
    await wrapper.get('[data-test="create-direct-user-submit"]').trigger('submit')
    await flushPromises()

    expect(createDirectUser).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('agentManagement.direct.insufficientAllocation')
  })

  it('does not show direct user creation on agent or enterprise pages', async () => {
    const agents = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()
    expect(agents.find('[data-test="create-direct-user"]').exists()).toBe(false)

    const enterprises = mountAgentView(DirectEnterprisesView, 'agent_level1')
    await flushPromises()
    expect(enterprises.find('[data-test="create-direct-user"]').exists()).toBe(false)
  })

  it('shows unlimited allocation status for admins instead of remaining quota badges', async () => {
    getSummary.mockResolvedValue({
      allocation: {
        total_concurrency: 5,
        allocated_concurrency: 500,
        remaining_concurrency: 0,
        total_rpm: 50,
        allocated_rpm: 5000,
        remaining_rpm: 0,
        unlimited_capacity: true,
      },
    })

    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    expect(wrapper.text()).toContain('agentManagement.direct.adminUnlimitedCapacity')
    expect(wrapper.text()).not.toContain('agentManagement.direct.remainingConcurrency')
    expect(wrapper.text()).not.toContain('agentManagement.direct.remainingRpm')
  })

  it('keeps remaining quota badges visible for agents', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    expect(wrapper.text()).toContain('agentManagement.direct.remainingConcurrency')
    expect(wrapper.text()).toContain('agentManagement.direct.remainingRpm')
    expect(wrapper.text()).not.toContain('agentManagement.direct.adminUnlimitedCapacity')
  })

  it('shows unlimited labels for agents with unlimited pool dimensions', async () => {
    getSummary.mockResolvedValue({
      allocation: {
        total_concurrency: 0,
        allocated_concurrency: 0,
        remaining_concurrency: 0,
        total_rpm: 0,
        allocated_rpm: 0,
        remaining_rpm: 0,
        unlimited_capacity: true,
        unlimited_concurrency: true,
        unlimited_rpm: true,
      },
      invite_defaults: {
        invite_default_concurrency: 1,
        invite_default_rpm: 1,
      },
    })
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    expect(wrapper.text()).toContain('agentManagement.direct.remainingConcurrency: common.unlimited')
    expect(wrapper.text()).toContain('agentManagement.direct.remainingRpm: common.unlimited')
    expect(wrapper.text()).not.toContain('agentManagement.direct.adminUnlimitedCapacity')
  })

  it.each([
    { role: 'admin' as const, allowed: ['agent_level1', 'enterprise'], denied: ['agent_level2'] },
    { role: 'agent_level1' as const, allowed: ['agent_level2', 'enterprise'], denied: ['agent_level1'] },
    { role: 'agent_level2' as const, allowed: ['enterprise'], denied: ['agent_level1', 'agent_level2'] },
  ])('renders upgrade choices for $role permissions', async ({ role, allowed, denied }) => {
    const wrapper = mountAgentView(DirectUsersView, role)
    await flushPromises()

    for (const targetRole of allowed) {
      expect(wrapper.find(`[data-test="upgrade-${targetRole}-12"]`).exists()).toBe(true)
    }
    for (const targetRole of denied) {
      expect(wrapper.find(`[data-test="upgrade-${targetRole}-12"]`).exists()).toBe(false)
    }
  })

  it('does not offer agent upgrade actions from the direct agents page', async () => {
    const wrapper = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()

    expect(wrapper.find('[data-test^="upgrade-"]').exists()).toBe(false)
  })

  it('sends the row draft as pool quota when upgrading a direct user to an agent', async () => {
    listDirectUsers.mockResolvedValue(makeChildrenResponse([
      makeChild({ concurrency: 10, rpm_limit: 120, allocated_concurrency: 0, allocated_rpm: 0 }),
    ]))
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="allocation-concurrency-12"]').setValue('80')
    await wrapper.get('[data-test="allocation-rpm-12"]').setValue('900')
    await wrapper.get('[data-test="upgrade-agent_level1-12"]').trigger('click')
    await flushPromises()

    expect(upgradeChild).toHaveBeenCalledWith(12, {
      target_role: 'agent_level1',
      pool_concurrency: 80,
      pool_rpm: 900,
    })
  })

  it('does not send pool fields when upgrading a direct user to enterprise', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="upgrade-enterprise-12"]').trigger('click')
    await flushPromises()

    expect(upgradeChild).toHaveBeenCalledWith(12, {
      target_role: 'enterprise',
    })
  })

  it('renders effective group rates without upstream cost fields', async () => {
    const wrapper = mountAgentView(MyGroupsView, 'agent_level1')
    await flushPromises()

    expect(wrapper.text()).toContain('Exclusive Retail')
    expect(wrapper.text()).toContain('2.4')
    expect(wrapper.text()).not.toContain('1.7')
    expect(wrapper.text()).not.toContain('upstream')
    expect(wrapper.text()).not.toContain('cost')
    expect(wrapper.text()).not.toContain('admin cost')
  })
})
