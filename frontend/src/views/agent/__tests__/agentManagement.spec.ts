import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DirectUsersView from '@/views/agent/DirectUsersView.vue'
import DirectAgentsView from '@/views/agent/DirectAgentsView.vue'
import DirectEnterprisesView from '@/views/agent/DirectEnterprisesView.vue'
import AdminOverviewView from '@/views/agent/AdminOverviewView.vue'
import AgentUsageView from '@/views/agent/UsageView.vue'
import MyGroupsView from '@/views/agent/MyGroupsView.vue'
import type { AgentAdminTreeResponse, AgentChildGroupDelegationOption, AgentDirectChildrenResponse, AgentGroupRate, AgentManagedUser, AgentStructureResponse, AdminUsageLog, Group, PaginatedResponse, User, UserRole } from '@/types'

const {
  getCurrentUser,
  getSummary,
  listDirectUsers,
  listDirectAgents,
  listDirectEnterprises,
  getAdminAgentTree,
  getStructure,
  listUsage,
  getUsageStats,
  listUsageUsers,
  updateAllocation,
  createDirectUser,
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
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  getCurrentUser: vi.fn(),
  getSummary: vi.fn(),
  listDirectUsers: vi.fn(),
  listDirectAgents: vi.fn(),
  listDirectEnterprises: vi.fn(),
  getAdminAgentTree: vi.fn(),
  getStructure: vi.fn(),
  listUsage: vi.fn(),
  getUsageStats: vi.fn(),
  listUsageUsers: vi.fn(),
  updateAllocation: vi.fn(),
  createDirectUser: vi.fn(),
  updateInviteDefaults: vi.fn(),
  upgradeChild: vi.fn(),
  deleteDirectChild: vi.fn(),
  listGroups: vi.fn(),
  listChildGroupDelegationOptions: vi.fn(),
  listInviteGroupDefaultOptions: vi.fn(),
  setChildGroupDelegation: vi.fn(),
  setChildGroupDelegationsBatch: vi.fn(),
  setDirectChildrenGroupDelegationsBatch: vi.fn(),
  setAgentIncome: vi.fn(),
  removeChildGroupDelegation: vi.fn(),
  setInviteGroupDefault: vi.fn(),
  setInviteGroupDefaultsBatch: vi.fn(),
  removeInviteGroupDefault: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/agentManagement', () => ({
  agentManagementAPI: {
    getSummary,
    listDirectUsers,
    listDirectAgents,
    listDirectEnterprises,
    getAdminAgentTree,
    getStructure,
    listUsage,
    getUsageStats,
    listUsageUsers,
    updateAllocation,
    createDirectUser,
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
        <slot name="cell-agent_income" :row="row" :value="row.agent_income" />
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
const ConfirmDialogStub = {
  props: ['show', 'title', 'message', 'confirmText', 'danger'],
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-test="confirm-dialog">
      <button data-test="confirm-dialog-confirm" @click="$emit('confirm')">confirm</button>
      <button data-test="confirm-dialog-cancel" @click="$emit('cancel')">cancel</button>
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

function makeGroup(overrides: Partial<Group> = {}): Group {
  return {
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
    ...overrides,
  }
}

function makeAgentGroupRate(overrides: Partial<AgentGroupRate> = {}): AgentGroupRate {
  return {
    group: makeGroup(),
    effective_rate: 2.4,
    can_delegate: true,
    source: 'delegated',
    ...overrides,
  }
}

function makeChildGroupOption(overrides: Partial<AgentChildGroupDelegationOption> = {}): AgentChildGroupDelegationOption {
  return {
    group: makeGroup(),
    effective_rate: 2.4,
    can_delegate: true,
    source: 'delegated',
    assigned: true,
    child_rate_multiplier: 2.8,
    child_can_delegate: false,
    ...overrides,
  }
}

function makeChildrenResponse(items: AgentManagedUser[]): AgentDirectChildrenResponse {
  return {
    items,
    pagination: { total: items.length, page: 1, page_size: 20, pages: 1 },
  }
}

function makeAdminTreeResponse(): AgentAdminTreeResponse {
  return {
    items: [
      {
        agent: makeChild({
          id: 21,
          role: 'agent_level1',
          email: 'agent@example.com',
          username: 'agent',
          pool_concurrency: 100,
          pool_rpm: 1000,
          agent_income: 8.75,
        }),
        users: [
          makeChild({ id: 22, email: 'agent-user@example.com', username: 'agent-user', concurrency: 5, rpm_limit: 50, balance: 3.25 }),
        ],
        enterprises: [
          {
            enterprise: makeChild({
              id: 23,
              role: 'enterprise',
              email: 'enterprise@example.com',
              username: 'enterprise',
              pool_concurrency: 20,
              pool_rpm: 200,
            }),
            employees: [
              makeChild({
                id: 24,
                role: 'employee',
                parent_user_id: 23,
                email: 'employee@example.com',
                username: 'employee',
                concurrency: 2,
                rpm_limit: 20,
                balance: 1.5,
              }),
            ],
          },
        ],
      },
    ],
  }
}

function makeStructureResponse(): AgentStructureResponse {
  return {
    owner_options: [
      makeChild({ id: 1, role: 'admin', email: 'admin@example.com', username: 'admin' }),
      makeChild({
        id: 21,
        role: 'agent_level1',
        email: 'agent@example.com',
        username: 'agent',
        pool_concurrency: 100,
        pool_rpm: 1000,
        agent_income: 8.75,
      }),
    ],
    selected_owner: makeChild({
      id: 21,
      role: 'agent_level1',
      email: 'agent@example.com',
      username: 'agent',
      pool_concurrency: 100,
      pool_rpm: 1000,
      agent_income: 8.75,
    }),
    users: [
      makeChild({ id: 22, email: 'agent-user@example.com', username: 'agent-user', concurrency: 5, rpm_limit: 50, balance: 3.25 }),
    ],
    enterprises: [
      {
        enterprise: makeChild({
          id: 23,
          role: 'enterprise',
          email: 'enterprise@example.com',
          username: 'enterprise',
          pool_concurrency: 20,
          pool_rpm: 200,
        }),
        employees: [
          makeChild({
            id: 24,
            role: 'employee',
            parent_user_id: 23,
            email: 'employee@example.com',
            username: 'employee',
            concurrency: 2,
            rpm_limit: 20,
            balance: 1.5,
          }),
        ],
      },
    ],
  }
}

function makeLargeStructureResponse(): AgentStructureResponse {
  const base = makeStructureResponse()
  return {
    ...base,
    users: Array.from({ length: 7 }, (_, index) => makeChild({
      id: 220 + index,
      email: `agent-user-${index + 1}@example.com`,
      username: `agent-user-${index + 1}`,
      concurrency: 5,
      rpm_limit: 50,
      balance: 3.25,
    })),
    enterprises: Array.from({ length: 7 }, (_, index) => ({
      enterprise: makeChild({
        id: 230 + index,
        role: 'enterprise',
        email: `enterprise-${index + 1}@example.com`,
        username: `enterprise-${index + 1}`,
        pool_concurrency: 20,
        pool_rpm: 200,
      }),
      employees: [
        makeChild({
          id: 240 + index,
          role: 'employee',
          parent_user_id: 230 + index,
          email: `employee-${index + 1}@example.com`,
          username: `employee-${index + 1}`,
          concurrency: 2,
          rpm_limit: 20,
          balance: 1.5,
        }),
      ],
    })),
  }
}

function makeUsageLog(overrides: Partial<AdminUsageLog> = {}): AdminUsageLog {
  return {
    id: 9001,
    user_id: 22,
    api_key_id: 10,
    account_id: 0,
    request_id: 'req-agent-usage',
    model: 'gpt-test',
    group_id: null,
    subscription_id: null,
    input_tokens: 10,
    output_tokens: 20,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    cache_creation_5m_tokens: 0,
    cache_creation_1h_tokens: 0,
    input_cost: 0.1,
    output_cost: 0.2,
    cache_creation_cost: 0,
    cache_read_cost: 0,
    total_cost: 0.3,
    actual_cost: 0.6,
    rate_multiplier: 2,
    billing_type: 0,
    request_type: 'sync',
    stream: false,
    duration_ms: 100,
    first_token_ms: null,
    image_count: 0,
    image_size: null,
    image_input_size: null,
    image_output_size: null,
    image_size_source: null,
    image_size_breakdown: null,
    user_agent: null,
    cache_ttl_overridden: false,
    created_at: '2026-06-04T10:00:00Z',
    user: makeUser('user'),
    ...overrides,
  }
}

function makeUsageResponse(items: AdminUsageLog[] = [makeUsageLog()]): PaginatedResponse<AdminUsageLog> {
  return { items, total: items.length, page: 1, page_size: 20, pages: 1 }
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
        UsageStatsCards: true,
        UsageTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: ConfirmDialogStub,
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
    listDirectAgents.mockResolvedValue(makeChildrenResponse([makeChild({ role: 'agent_level1', agent_income: 8.75 })]))
    listDirectEnterprises.mockResolvedValue(makeChildrenResponse([makeChild({ role: 'enterprise' })]))
    getAdminAgentTree.mockResolvedValue(makeAdminTreeResponse())
    getStructure.mockResolvedValue(makeStructureResponse())
    listUsage.mockResolvedValue(makeUsageResponse())
    getUsageStats.mockResolvedValue({
      total_requests: 1,
      total_input_tokens: 10,
      total_output_tokens: 20,
      total_cache_tokens: 0,
      total_tokens: 30,
      total_cost: 0.3,
      total_actual_cost: 0.6,
      total_account_cost: 0,
      average_duration_ms: 100,
    })
    listUsageUsers.mockResolvedValue([
      makeChild({ id: 22, email: 'agent-user@example.com', role: 'user' }),
      makeChild({ id: 24, email: 'employee@example.com', role: 'employee' }),
    ])
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
    listGroups.mockResolvedValue([makeAgentGroupRate()])
    listChildGroupDelegationOptions.mockResolvedValue([makeChildGroupOption()])
    listInviteGroupDefaultOptions.mockResolvedValue([makeChildGroupOption()])
    setChildGroupDelegation.mockResolvedValue({ child_id: 12, group_id: 7 })
    setChildGroupDelegationsBatch.mockResolvedValue({ child_id: 12, group_ids: [7], all: false })
    setDirectChildrenGroupDelegationsBatch.mockResolvedValue({ kind: 'enterprises', group_ids: [7], all: false, updated_children: 2 })
    setAgentIncome.mockResolvedValue(makeChild({ role: 'agent_level1', agent_income: 0 }))
    removeChildGroupDelegation.mockResolvedValue({ child_id: 12, group_id: 7 })
    setInviteGroupDefault.mockResolvedValue({ group_id: 7 })
    setInviteGroupDefaultsBatch.mockResolvedValue({ group_ids: [7], all: false })
    removeInviteGroupDefault.mockResolvedValue({ group_id: 7 })
  })

  it('does not render balance, recharge, disable, or official delete actions on direct users', async () => {
    const wrapper = mountAgentView(DirectUsersView)
    await flushPromises()

    expect(wrapper.text()).not.toContain('Recharge')
    expect(wrapper.text()).not.toContain('Disable')
    expect(wrapper.text()).not.toContain('Official Delete')
  })

  it('renders the subordinate structure as a read-only overview', async () => {
    const wrapper = mountAgentView(AdminOverviewView, 'admin')
    await flushPromises()

    expect(getStructure).toHaveBeenCalledWith(undefined)
    expect(wrapper.text()).toContain('agent@example.com')
    expect(wrapper.text()).toContain('agent-user@example.com')
    expect(wrapper.text()).toContain('enterprise@example.com')
    expect(wrapper.text()).not.toContain('employee@example.com')
    await wrapper.get('[data-test="overview-enterprise-toggle-23"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('employee@example.com')
    expect(wrapper.text()).toContain('$8.75')
    expect(wrapper.text()).toContain('agentManagement.overview.directUsers')
    expect(wrapper.text()).toContain('agentManagement.overview.employeeCount')
    expect(wrapper.text()).not.toContain('agentManagement.direct.saveAllocation')
    expect(wrapper.text()).not.toContain('agentManagement.direct.deleteAgent')
    expect(wrapper.find('[data-test="save-allocation-21"]').exists()).toBe(false)
  })

  it('paginates subordinate structure sections and expands enterprise employees on demand', async () => {
    getStructure.mockResolvedValue(makeLargeStructureResponse())
    const wrapper = mountAgentView(AdminOverviewView, 'admin')
    await flushPromises()

    expect(wrapper.text()).toContain('agent-user-1@example.com')
    expect(wrapper.text()).not.toContain('agent-user-7@example.com')
    expect(wrapper.text()).toContain('enterprise-1@example.com')
    expect(wrapper.text()).not.toContain('enterprise-7@example.com')
    expect(wrapper.text()).not.toContain('employee-1@example.com')

    await wrapper.get('[data-test="overview-enterprise-toggle-230"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('employee-1@example.com')

    await wrapper.get('[data-test="overview-users-next"]').trigger('click')
    await wrapper.get('[data-test="overview-enterprises-next"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('agent-user-7@example.com')
    expect(wrapper.text()).toContain('enterprise-7@example.com')
  })

  it('lets admins switch subordinate structure owner', async () => {
    const wrapper = mountAgentView(AdminOverviewView, 'admin')
    await flushPromises()

    const select = wrapper.get('[data-test="structure-owner-select"]')
    await select.setValue('21')
    await flushPromises()

    expect(getStructure).toHaveBeenLastCalledWith(21)
  })

  it('lets agents view their own subordinate structure without admin-only owner switch', async () => {
    const wrapper = mountAgentView(AdminOverviewView, 'agent_level1')
    await flushPromises()

    expect(getStructure).toHaveBeenCalledWith(undefined)
    expect(wrapper.text()).toContain('agent-user@example.com')
    expect(wrapper.find('[data-test="structure-owner-select"]').exists()).toBe(false)
  })

  it('renders agent usage as read-only scoped records', async () => {
    const wrapper = mountAgentView(AgentUsageView, 'agent_level1')
    await flushPromises()

    expect(listUsageUsers).toHaveBeenCalled()
    expect(listUsage).toHaveBeenCalled()
    expect(getUsageStats).toHaveBeenCalled()
    expect(wrapper.text()).toContain('agentManagement.usage.title')
    expect(wrapper.text()).toContain('agent-user@example.com')
    expect(wrapper.text()).not.toContain('admin.usage.cleanup.button')
    expect(wrapper.text()).not.toContain('usage.exportExcel')
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

  it('lets admins save global registration defaults from the direct users page', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    expect((wrapper.get('[data-test="invite-default-concurrency"]').element as HTMLInputElement).value).toBe('2')
    expect((wrapper.get('[data-test="invite-default-rpm"]').element as HTMLInputElement).value).toBe('20')

    await wrapper.get('[data-test="invite-default-concurrency"]').setValue('6')
    await wrapper.get('[data-test="invite-default-rpm"]').setValue('60')
    await wrapper.get('[data-test="invite-default-submit"]').trigger('submit')
    await flushPromises()

    expect(updateInviteDefaults).toHaveBeenCalledWith({
      invite_default_concurrency: 6,
      invite_default_rpm: 60,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.direct.inviteDefaultsSaved')
  })

  it('sends search query when filtering direct children', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="direct-child-search"]').setValue('alice')
    await wrapper.get('[data-test="direct-child-search-submit"]').trigger('click')
    await flushPromises()

    expect(listDirectUsers).toHaveBeenLastCalledWith({ search: 'alice', page: 1, page_size: 20 })
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

  it('uses pool fields and shows income for direct agents', async () => {
    listDirectAgents.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'agent_level1', concurrency: 5, rpm_limit: 50, pool_concurrency: 30, pool_rpm: 300, agent_income: 8.75 }),
    ]))
    const wrapper = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()

    expect(wrapper.get('[data-test="columns"]').text()).toContain('agent_income')
    expect(wrapper.text()).toContain('$8.75')
    expect((wrapper.get('[data-test="allocation-concurrency-12"]').element as HTMLInputElement).value).toBe('30')
    expect((wrapper.get('[data-test="allocation-rpm-12"]').element as HTMLInputElement).value).toBe('300')
  })

  it('lets admins set a direct agent income target for settlement', async () => {
    listDirectAgents.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'agent_level1', agent_income: 8.75 }),
    ]))
    const wrapper = mountAgentView(DirectAgentsView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="set-agent-income-12"]').trigger('click')
    expect(wrapper.find('[data-test="agent-income-modal"]').exists()).toBe(true)
    expect((wrapper.get('[data-test="agent-income-input"]').element as HTMLInputElement).value).toBe('8.75')

    await wrapper.get('[data-test="agent-income-input"]').setValue('0')
    await wrapper.get('[data-test="agent-income-reason"]').setValue('settled')
    await wrapper.get('[data-test="agent-income-submit"]').trigger('click')
    await flushPromises()

    expect(setAgentIncome).toHaveBeenCalledWith(12, { agent_income: 0, reason: 'settled' })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.direct.agentIncomeSaved')
    expect(listDirectAgents).toHaveBeenCalledTimes(2)
  })

  it('does not show agent income settlement controls to agents', async () => {
    listDirectAgents.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'agent_level1', agent_income: 8.75 }),
    ]))
    const wrapper = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()

    expect(wrapper.find('[data-test="set-agent-income-12"]').exists()).toBe(false)
  })

  it('uses pool fields for direct enterprises instead of live remaining quota', async () => {
    listDirectEnterprises.mockResolvedValue(makeChildrenResponse([
      makeChild({ role: 'enterprise', concurrency: 5, rpm_limit: 50, pool_concurrency: 30, pool_rpm: 300 }),
    ]))
    const wrapper = mountAgentView(DirectEnterprisesView, 'admin')
    await flushPromises()

    expect((wrapper.get('[data-test="allocation-concurrency-12"]').element as HTMLInputElement).value).toBe('30')
    expect((wrapper.get('[data-test="allocation-rpm-12"]').element as HTMLInputElement).value).toBe('300')
  })

  it('opens a modal and creates direct users from the direct users page', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="create-direct-user"]').trigger('click')
    expect(wrapper.find('[data-test="create-direct-user-modal"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('admin.users.columns.balance')
    expect((wrapper.get('[data-test="create-direct-user-concurrency"]').element as HTMLInputElement).value).toBe('2')
    expect((wrapper.get('[data-test="create-direct-user-rpm"]').element as HTMLInputElement).value).toBe('20')

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
    await wrapper.get('[data-test="create-direct-user-concurrency"]').setValue('2')
    await wrapper.get('[data-test="create-direct-user-rpm"]').setValue('11')
    await wrapper.get('[data-test="create-direct-user-submit"]').trigger('submit')
    await flushPromises()

    expect(createDirectUser).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('agentManagement.direct.insufficientAllocation')
  })

  it('blocks direct allocation updates that would leave the agent with no own capacity', async () => {
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
    listDirectUsers.mockResolvedValue(makeChildrenResponse([
      makeChild({ concurrency: 3, rpm_limit: 30, allocated_concurrency: 0, allocated_rpm: 0 }),
    ]))
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="allocation-concurrency-12"]').setValue('5')
    await wrapper.get('[data-test="allocation-rpm-12"]').setValue('40')
    await wrapper.get('[data-test="save-allocation-12"]').trigger('click')
    await flushPromises()

    expect(updateAllocation).not.toHaveBeenCalled()
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

  it('shows only RPM unlimited labels for agents with unlimited RPM pool', async () => {
    getSummary.mockResolvedValue({
      allocation: {
        total_concurrency: 10,
        allocated_concurrency: 0,
        remaining_concurrency: 10,
        total_rpm: 0,
        allocated_rpm: 0,
        remaining_rpm: 0,
        unlimited_capacity: false,
        unlimited_concurrency: false,
        unlimited_rpm: true,
      },
      invite_defaults: {
        invite_default_concurrency: 1,
        invite_default_rpm: 1,
      },
    })
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    expect(wrapper.text()).toContain('agentManagement.direct.remainingConcurrency: 10')
    expect(wrapper.text()).toContain('agentManagement.direct.remainingRpm: common.unlimited')
    expect(wrapper.text()).not.toContain('agentManagement.direct.adminUnlimitedCapacity')
  })

  it.each([
    { role: 'admin' as const, allowed: ['agent_level1', 'enterprise'], denied: [] },
    { role: 'agent_level1' as const, allowed: ['enterprise'], denied: ['agent_level1'] },
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

    expect(upgradeChild).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-dialog-confirm"]').trigger('click')
    await flushPromises()

    expect(upgradeChild).toHaveBeenCalledWith(12, {
      target_role: 'agent_level1',
      pool_concurrency: 80,
      pool_rpm: 900,
    })
  })

  it('sends the row draft as pool quota when upgrading a direct user to enterprise', async () => {
    listDirectUsers.mockResolvedValue(makeChildrenResponse([
      makeChild({ concurrency: 10, rpm_limit: 120, allocated_concurrency: 0, allocated_rpm: 0 }),
    ]))
    const wrapper = mountAgentView(DirectUsersView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="allocation-concurrency-12"]').setValue('40')
    await wrapper.get('[data-test="allocation-rpm-12"]').setValue('500')
    await wrapper.get('[data-test="upgrade-enterprise-12"]').trigger('click')
    await flushPromises()

    expect(upgradeChild).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-dialog-confirm"]').trigger('click')
    await flushPromises()

    expect(upgradeChild).toHaveBeenCalledWith(12, {
      target_role: 'enterprise',
      pool_concurrency: 40,
      pool_rpm: 500,
    })
  })

  it('loads child group assignment state and updates a checked group', async () => {
    const wrapper = mountAgentView(DirectUsersView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="manage-groups-12"]').trigger('click')
    await flushPromises()

    expect(listChildGroupDelegationOptions).toHaveBeenCalledWith(12)
    expect(listGroups).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Exclusive Retail')
    expect(wrapper.text()).not.toContain('1.7')
    expect((wrapper.get('[data-test="group-assigned-7"]').element as HTMLInputElement).checked).toBe(true)
    expect((wrapper.get('[data-test="group-rate-7"]').element as HTMLInputElement).value).toBe('2.8')

    await wrapper.get('[data-test="group-rate-7"]').setValue('3.1')
    await wrapper.get('[data-test="group-can-delegate-7"]').setValue(true)
    await wrapper.get('[data-test="save-group-7"]').trigger('click')
    await flushPromises()

    expect(setChildGroupDelegation).toHaveBeenCalledWith(12, 7, {
      rate_multiplier: 3.1,
      can_delegate: true,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.delegationSaved')
  })

  it('reclaims a group when it is unchecked and saved', async () => {
    const wrapper = mountAgentView(DirectAgentsView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="manage-groups-12"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="group-assigned-7"]').setValue(false)
    await wrapper.get('[data-test="save-group-7"]').trigger('click')
    await flushPromises()

    expect(removeChildGroupDelegation).toHaveBeenCalledWith(12, 7)
    expect(setChildGroupDelegation).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.delegationRemoved')
  })

  it('only offers child group delegation for groups the manager can delegate', async () => {
    listChildGroupDelegationOptions.mockResolvedValue([
      makeChildGroupOption({
        group: makeGroup({ id: 4, name: 'Public Shared', rate_multiplier: 1, is_exclusive: false }),
        effective_rate: 1,
        can_delegate: false,
        source: 'public',
        assigned: false,
      }),
      makeChildGroupOption(),
    ])
    const wrapper = mountAgentView(DirectEnterprisesView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="manage-groups-12"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Exclusive Retail')
    expect(wrapper.text()).not.toContain('Public Shared')
    expect(wrapper.find('[data-test="save-group-4"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="save-group-7"]').exists()).toBe(true)
  })

  it('batch updates selected child group delegations', async () => {
    listChildGroupDelegationOptions.mockResolvedValue([
      makeChildGroupOption(),
      makeChildGroupOption({
        group: makeGroup({ id: 8, name: 'Enterprise Boost' }),
        effective_rate: 1.6,
        assigned: false,
        child_rate_multiplier: 0,
      }),
    ])
    const wrapper = mountAgentView(DirectEnterprisesView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="manage-groups-12"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="group-batch-select-7"]').setValue(true)
    await wrapper.get('[data-test="group-batch-select-8"]').setValue(true)
    await wrapper.get('[data-test="group-batch-rate"]').setValue('3.2')
    await wrapper.get('[data-test="group-batch-can-delegate"]').setValue(true)
    await wrapper.get('[data-test="apply-group-batch"]').trigger('click')
    await flushPromises()

    expect(setChildGroupDelegationsBatch).toHaveBeenCalledWith(12, {
      group_ids: [7, 8],
      all: false,
      rate_multiplier: 3.2,
      can_delegate: true,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.batchDelegationSaved')
  })

  it('batch deploys selected groups to all direct agents on the current management page', async () => {
    listGroups.mockResolvedValue([
      makeAgentGroupRate(),
      makeAgentGroupRate({
        group: makeGroup({ id: 8, name: 'Enterprise Boost' }),
        effective_rate: 1.6,
        can_delegate: true,
      }),
    ])
    const wrapper = mountAgentView(DirectAgentsView, 'admin')
    await flushPromises()

    await wrapper.get('[data-test="open-direct-group-batch"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="direct-group-batch-select-7"]').setValue(true)
    await wrapper.get('[data-test="direct-group-batch-select-8"]').setValue(true)
    await wrapper.get('[data-test="direct-group-batch-rate"]').setValue('2.8')
    await wrapper.get('[data-test="direct-group-batch-can-delegate"]').setValue(true)
    await wrapper.get('[data-test="apply-direct-group-batch"]').trigger('click')
    await flushPromises()

    expect(setDirectChildrenGroupDelegationsBatch).toHaveBeenCalledWith('agents', {
      group_ids: [7, 8],
      all: false,
      rate_multiplier: 2.8,
      can_delegate: true,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.directBatchSaved')
  })

  it('batch deploys all groups to direct enterprises on the current management page', async () => {
    const wrapper = mountAgentView(DirectEnterprisesView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="open-direct-group-batch"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="direct-group-batch-all"]').setValue(true)
    await wrapper.get('[data-test="direct-group-batch-rate"]').setValue('2.2')
    await wrapper.get('[data-test="apply-direct-group-batch"]').trigger('click')
    await flushPromises()

    expect(setDirectChildrenGroupDelegationsBatch).toHaveBeenCalledWith('enterprises', {
      group_ids: [],
      all: true,
      rate_multiplier: 2.2,
      can_delegate: false,
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

  it('shows invite default group propagation config for agents and saves a checked group rate', async () => {
    listInviteGroupDefaultOptions.mockResolvedValue([
      makeChildGroupOption({
        assigned: false,
        child_rate_multiplier: 0,
        effective_rate: 2.4,
      }),
    ])
    const wrapper = mountAgentView(MyGroupsView, 'agent_level1')
    await flushPromises()

    expect(listInviteGroupDefaultOptions).toHaveBeenCalled()
    expect(wrapper.get('[data-test="invite-default-groups-section"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Exclusive Retail')
    expect((wrapper.get('[data-test="invite-default-assigned-7"]').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.get('[data-test="invite-default-rate-7"]').element as HTMLInputElement).value).toBe('2.4')

    await wrapper.get('[data-test="invite-default-assigned-7"]').setValue(true)
    await wrapper.get('[data-test="invite-default-rate-7"]').setValue('3.1')
    await wrapper.get('[data-test="save-invite-default-7"]').trigger('click')
    await flushPromises()

    expect(setInviteGroupDefault).toHaveBeenCalledWith(7, {
      rate_multiplier: 3.1,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.inviteDefaultGroupSaved')
  })

  it('removes an invite default group when it is unchecked and saved', async () => {
    const wrapper = mountAgentView(MyGroupsView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="invite-default-assigned-7"]').setValue(false)
    await wrapper.get('[data-test="save-invite-default-7"]').trigger('click')
    await flushPromises()

    expect(removeInviteGroupDefault).toHaveBeenCalledWith(7)
    expect(setInviteGroupDefault).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.inviteDefaultGroupRemoved')
  })

  it('batch updates all invite default group rates', async () => {
    listInviteGroupDefaultOptions.mockResolvedValue([
      makeChildGroupOption(),
      makeChildGroupOption({
        group: makeGroup({ id: 8, name: 'Enterprise Boost' }),
        effective_rate: 1.6,
        assigned: false,
        child_rate_multiplier: 0,
      }),
    ])
    const wrapper = mountAgentView(MyGroupsView, 'agent_level1')
    await flushPromises()

    await wrapper.get('[data-test="invite-default-batch-all"]').setValue(true)
    await wrapper.get('[data-test="invite-default-batch-rate"]').setValue('3.2')
    await wrapper.get('[data-test="apply-invite-default-batch"]').trigger('click')
    await flushPromises()

    expect(setInviteGroupDefaultsBatch).toHaveBeenCalledWith({
      group_ids: [],
      all: true,
      rate_multiplier: 3.2,
    })
    expect(showSuccess).toHaveBeenCalledWith('agentManagement.groups.inviteDefaultBatchSaved')
  })

  it('shows invite default group propagation config for admins', async () => {
    const wrapper = mountAgentView(MyGroupsView, 'admin')
    await flushPromises()

    expect(listGroups).toHaveBeenCalled()
    expect(listInviteGroupDefaultOptions).toHaveBeenCalled()
    expect(wrapper.get('[data-test="invite-default-groups-section"]').exists()).toBe(true)
  })
})
