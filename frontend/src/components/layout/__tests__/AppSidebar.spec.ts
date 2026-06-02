import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { createRouter, createWebHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AppSidebar from '../AppSidebar.vue'
import { useAdminSettingsStore, useAppStore, useAuthStore } from '@/stores'
import type { User, UserRole } from '@/types'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

const navMessages = {
  agentManagement: 'Agent Management',
  agentDirectUsers: 'Direct Users',
  agentDirectAgents: 'Direct Agents',
  agentDirectEnterprises: 'Direct Enterprises',
  agentGroups: 'Groups & Rates',
  dashboard: 'Dashboard',
  apiKeys: 'API Keys',
  usage: 'Usage',
  availableChannels: 'Available Channels',
  channelStatus: 'Channel Status',
  mySubscriptions: 'My Subscriptions',
  buySubscription: 'Buy Subscription',
  myOrders: 'My Orders',
  redeem: 'Redeem',
  affiliate: 'Affiliate',
  profile: 'Profile',
  myAccount: 'My Account',
  ops: 'Operations',
  users: 'Users',
  groups: 'Groups',
  channelManagement: 'Channel Management',
  channelPricing: 'Channel Pricing',
  channelMonitor: 'Channel Monitor',
  subscriptions: 'Subscriptions',
  accounts: 'Accounts',
  announcements: 'Announcements',
  proxies: 'Proxies',
  riskControl: 'Risk Control',
  redeemCodes: 'Redeem Codes',
  promoCodes: 'Promo Codes',
  affiliateManagement: 'Affiliate Management',
  affiliateInviteRecords: 'Invite Records',
  affiliateRebateRecords: 'Rebate Records',
  affiliateTransferRecords: 'Transfer Records',
  orderManagement: 'Orders',
  paymentDashboard: 'Payment Dashboard',
  paymentPlans: 'Payment Plans',
  settings: 'Settings',
  lightMode: 'Light Mode',
  darkMode: 'Dark Mode',
  expand: 'Expand',
  collapse: 'Collapse'
} as const

const messages = {
  en: {
    nav: Object.fromEntries(
      Object.entries(navMessages).map(([key, value]) => [key, () => value])
    )
  }
}

function makeUser(role: UserRole, id = 1): User {
  return {
    id,
    username: `${role}-${id}`,
    email: `${role}-${id}@example.test`,
    role,
    balance: 0,
    concurrency: 0,
    rpm_limit: 0,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: false,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z'
  }
}

async function mountSidebar(options: {
  role: UserRole
  runMode?: 'standard' | 'simple'
  backendModeEnabled?: boolean
}) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/keys', component: { template: '<div />' } },
      { path: '/usage', component: { template: '<div />' } },
      { path: '/available-channels', component: { template: '<div />' } },
      { path: '/monitor', component: { template: '<div />' } },
      { path: '/subscriptions', component: { template: '<div />' } },
      { path: '/purchase', component: { template: '<div />' } },
      { path: '/orders', component: { template: '<div />' } },
      { path: '/redeem', component: { template: '<div />' } },
      { path: '/affiliate', component: { template: '<div />' } },
      { path: '/profile', component: { template: '<div />' } },
      { path: '/agent/direct-users', component: { template: '<div />' } },
      { path: '/agent/direct-agents', component: { template: '<div />' } },
      { path: '/agent/direct-enterprises', component: { template: '<div />' } },
      { path: '/agent/groups', component: { template: '<div />' } },
      { path: '/admin/dashboard', component: { template: '<div />' } },
      { path: '/admin/ops', component: { template: '<div />' } },
      { path: '/admin/users', component: { template: '<div />' } },
      { path: '/admin/groups', component: { template: '<div />' } },
      { path: '/admin/channels/pricing', component: { template: '<div />' } },
      { path: '/admin/channels/monitor', component: { template: '<div />' } },
      { path: '/admin/subscriptions', component: { template: '<div />' } },
      { path: '/admin/accounts', component: { template: '<div />' } },
      { path: '/admin/announcements', component: { template: '<div />' } },
      { path: '/admin/proxies', component: { template: '<div />' } },
      { path: '/admin/risk-control', component: { template: '<div />' } },
      { path: '/admin/redeem', component: { template: '<div />' } },
      { path: '/admin/promo-codes', component: { template: '<div />' } },
      { path: '/admin/affiliates/invites', component: { template: '<div />' } },
      { path: '/admin/affiliates/rebates', component: { template: '<div />' } },
      { path: '/admin/affiliates/transfers', component: { template: '<div />' } },
      { path: '/admin/orders/dashboard', component: { template: '<div />' } },
      { path: '/admin/orders', component: { template: '<div />' } },
      { path: '/admin/orders/plans', component: { template: '<div />' } },
      { path: '/admin/usage', component: { template: '<div />' } },
      { path: '/admin/settings', component: { template: '<div />' } }
    ]
  })
  router.push('/')
  await router.isReady()

  const authStore = useAuthStore()
  authStore.token = 'test-token'
  authStore.user = makeUser(options.role)
  if (options.runMode === 'simple') {
    Object.defineProperty(authStore, 'isSimpleMode', { configurable: true, value: true })
  }

  const appStore = useAppStore()
  appStore.cachedPublicSettings = {
    custom_menu_items: [],
    backend_mode_enabled: options.backendModeEnabled ?? false
  } as any

  const adminSettingsStore = useAdminSettingsStore()
  vi.spyOn(adminSettingsStore, 'fetch').mockResolvedValue(undefined)

  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    messages
  })

  return mount(AppSidebar, {
    global: {
      plugins: [pinia, router, i18n],
      stubs: {
        VersionBadge: true
      }
    }
  })
}

function agentLinkCount(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('a').filter((link) => link.attributes('href')?.startsWith('/agent/')).length
}

describe('AppSidebar agent management visibility', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.restoreAllMocks()
    vi.stubGlobal('matchMedia', vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn()
    }))
  })

  it('shows agent management links to admins in simple mode', async () => {
    const wrapper = await mountSidebar({ role: 'admin', runMode: 'simple' })

    expect(wrapper.text()).toContain('Agent Management')
    expect(agentLinkCount(wrapper)).toBe(4)
  })

  it('shows agent management links to agents in simple mode', async () => {
    const wrapper = await mountSidebar({ role: 'agent_level1', runMode: 'simple' })

    expect(wrapper.text()).toContain('Agent Management')
    expect(agentLinkCount(wrapper)).toBe(4)
  })

  it('shows agent management links to agents when backend mode is enabled', async () => {
    const wrapper = await mountSidebar({ role: 'agent_level1', backendModeEnabled: true })

    expect(wrapper.text()).toContain('Agent Management')
    expect(agentLinkCount(wrapper)).toBe(4)
    expect(wrapper.text()).not.toContain('Dashboard')
  })

  it('hides agent management links from regular users', async () => {
    const wrapper = await mountSidebar({ role: 'user' })

    expect(wrapper.text()).not.toContain('Agent Management')
    expect(agentLinkCount(wrapper)).toBe(0)
  })
})

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})
