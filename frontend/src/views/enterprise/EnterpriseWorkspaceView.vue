<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">企业空间</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">企业成员默认使用企业分发的额度。企业管理员可在这里管理成员、邀请码和额度台账。</p>
          </div>
          <button class="btn btn-secondary" :disabled="loading" @click="loadAll">刷新</button>
        </div>
      </section>

      <section v-if="!me.enterprise" class="rounded-2xl border border-dashed border-gray-300 bg-white p-6 dark:border-dark-600 dark:bg-dark-800">
        <div class="max-w-xl space-y-3">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">绑定企业邀请码</h2>
          <p class="text-sm text-gray-500 dark:text-dark-300">如果你是已有账号，可以在这里补填企业邀请码，绑定后会自动归属对应企业。</p>
          <div class="flex flex-wrap gap-2">
            <input v-model="bindCode" class="input w-full sm:w-72" placeholder="请输入企业邀请码" />
            <button class="btn btn-primary" :disabled="submitting" @click="submitBindInvite">立即绑定</button>
          </div>
        </div>
      </section>

      <template v-else-if="managerEnterprise">
        <section class="grid gap-4 md:grid-cols-4">
          <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-300">企业名称</div>
            <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ me.enterprise.tenant_name }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ me.enterprise.tenant_code }}</div>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-300">我的角色</div>
            <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ me.enterprise.member_role }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ me.enterprise.member_note || '无成员备注' }}</div>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-300">企业分组价格</div>
            <div class="mt-2 flex flex-wrap gap-1">
              <span
                v-for="groupID in enterpriseGroupIDs"
                :key="groupID"
                class="rounded bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200"
              >
                {{ groupLabel(groupID) }} 成本 {{ tenantGroupFloor(groupID).toFixed(3) }}x / 默认 {{ tenantMemberDefaultRate(groupID).toFixed(3) }}x
              </span>
              <span v-if="!enterpriseGroupIDs.length" class="text-sm text-gray-500 dark:text-dark-300">
                成本 {{ enterpriseBaseFloorRate.toFixed(2) }}x / 默认 {{ tenantDefaultMemberRate.toFixed(2) }}x
              </span>
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">成本是平台向企业结算价；默认是新成员不单独设置时的售价</div>
          </div>
          <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
            <div class="text-xs text-gray-500 dark:text-dark-300">企业总账可用</div>
            <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ enterpriseAvailableBalance.toFixed(2) }}</div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">
              总额 {{ enterpriseTotalBalance.toFixed(2) }} / 已消耗 {{ enterpriseSpentBalance.toFixed(2) }} / 授信 {{ enterpriseOverdraftLimit.toFixed(2) }}
            </div>
            <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">
              企业总并发 {{ managerEnterprise?.concurrency || '不限' }}
            </div>
          </div>
        </section>

        <section class="grid gap-4 xl:grid-cols-[380px_minmax(0,1fr)]">
            <div class="space-y-4">
              <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex items-start justify-between gap-3">
                  <div>
                    <div class="text-sm font-medium text-gray-700 dark:text-dark-200">默认成员售价</div>
                    <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                      这是企业管理员对成员的默认售价。留空或 0 表示按平台给企业配置的企业成本价兜底；单个成员仍可单独覆盖。
                    </p>
                  </div>
                  <button class="btn btn-primary btn-sm" :disabled="submitting" @click="submitPricingDefaults">保存</button>
                </div>
                <div class="space-y-3">
                  <label class="block text-xs text-gray-500 dark:text-dark-300">
                    <span class="mb-1 block">通用默认倍率</span>
                    <input
                      v-model.number="pricingDefaultsForm.member_default_pricing_factor"
                      class="input h-9 text-xs"
                      type="number"
                      min="0"
                      step="0.001"
                      :placeholder="`未设置时 ${enterpriseBaseFloorRate.toFixed(3)}x`"
                    />
                  </label>
                  <label class="block text-xs text-gray-500 dark:text-dark-300">
                    <span class="mb-1 block">成员默认并发</span>
                    <input
                      v-model.number="pricingDefaultsForm.member_default_concurrency"
                      class="input h-9 text-xs"
                      type="number"
                      min="0"
                      step="1"
                      placeholder="0 表示沿用主站注册默认并发"
                    />
                  </label>
                  <div v-if="enterpriseGroupIDs.length" class="rounded-lg border border-gray-100 p-2 dark:border-dark-700">
                    <div class="mb-2 text-xs font-medium text-gray-600 dark:text-dark-200">分组默认售价</div>
                    <div class="grid gap-2 sm:grid-cols-2">
                      <label v-for="groupID in enterpriseGroupIDs" :key="groupID" class="text-xs text-gray-500 dark:text-dark-300">
                        <span class="mb-1 block truncate">{{ groupLabel(groupID) }}，企业成本 {{ tenantGroupFloor(groupID).toFixed(3) }}x</span>
                        <input
                          v-model.number="pricingDefaultsForm.member_group_rates[groupID]"
                          class="input h-9 text-xs"
                          type="number"
                          min="0"
                          step="0.001"
                          :placeholder="`默认 ${tenantMemberDefaultRate(groupID).toFixed(3)}x`"
                        />
                      </label>
                    </div>
                  </div>
                  <div class="text-xs text-gray-500 dark:text-dark-300">
                    当前生效：通用 {{ tenantDefaultMemberRate.toFixed(3) }}x
                  </div>
                </div>
              </div>

              <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">创建企业成员</div>
                <div class="space-y-2">
                  <input v-model="memberForm.email" class="input" placeholder="邮箱" />
                  <input v-model="memberForm.password" class="input" type="password" placeholder="密码" />
                  <input v-model="memberForm.username" class="input" placeholder="用户名（可选）" />
                  <input v-model="memberForm.notes" class="input" placeholder="用户备注（平台用户备注）" />
                  <input v-model="memberForm.member_note" class="input" placeholder="企业成员备注" />
                  <div class="grid grid-cols-2 gap-2">
                    <input v-model="memberForm.concurrency" class="input" type="number" min="0" :placeholder="`并发，0 使用企业默认 ${tenantDefaultMemberConcurrency || '主站默认'}`" />
                    <input v-model="memberForm.initial_balance" class="input" type="number" min="0" step="0.01" placeholder="初始额度" />
                  </div>
                  <div class="grid grid-cols-2 gap-2">
                    <input
                      v-model="memberForm.pricing_factor"
                      class="input"
                      type="number"
                      min="0"
                      step="0.01"
                      placeholder="留空/0 使用企业默认售价"
                    />
                    <div class="rounded-lg border border-gray-200 px-3 py-2 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-300">
                      成员自动继承企业可用分组
                    </div>
                  </div>
                  <div v-if="enterpriseGroupIDs.length" class="rounded-lg border border-gray-100 p-2 dark:border-dark-700">
                    <div class="mb-2 text-xs font-medium text-gray-600 dark:text-dark-200">成员分组倍率</div>
                    <div class="grid gap-2 sm:grid-cols-2">
                      <label v-for="groupID in enterpriseGroupIDs" :key="groupID" class="text-xs text-gray-500 dark:text-dark-300">
                        <span class="mb-1 block truncate">{{ groupLabel(groupID) }}，默认 {{ tenantMemberDefaultRate(groupID).toFixed(3) }}</span>
                        <input
                          v-model.number="memberForm.group_rates[groupID]"
                          class="input h-9 text-xs"
                          type="number"
                          min="0.01"
                          step="0.001"
                          :placeholder="`默认 ${tenantMemberDefaultRate(groupID).toFixed(3)}`"
                        />
                      </label>
                    </div>
                  </div>
                </div>
                <button class="btn btn-primary mt-3 w-full" :disabled="submitting" @click="submitCreateMember">创建成员</button>
              </div>

              <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">创建邀请码</div>
                <div class="space-y-2">
                  <input v-model="inviteForm.code" class="input" placeholder="留空自动生成" />
                  <input v-model="inviteForm.max_uses" class="input" type="number" min="0" placeholder="可使用次数，0 为无限" />
                  <input v-model="inviteForm.expires_at" class="input" type="datetime-local" />
                  <textarea v-model="inviteForm.notes" class="input min-h-[92px]" placeholder="备注"></textarea>
                </div>
                <button class="btn btn-primary mt-3 w-full" :disabled="submitting" @click="submitCreateInvite">创建邀请码</button>
              </div>
            </div>

            <div class="space-y-4">
              <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">企业成员</div>
                  <div class="flex flex-wrap items-center gap-2">
                    <input v-model="memberSearch" class="input w-full sm:w-60" placeholder="搜索邮箱 / 用户名 / 备注" @keyup.enter="applyMemberFilters" />
                    <button class="btn btn-secondary btn-sm" @click="applyMemberFilters">查询</button>
                  </div>
                </div>
                <div class="overflow-x-auto">
                  <table class="min-w-full text-sm">
                    <thead class="text-left text-gray-500 dark:text-dark-300">
                      <tr>
                        <th class="py-2">用户</th>
                        <th class="py-2">并发</th>
                        <th class="py-2">默认倍率</th>
                        <th class="py-2">分组倍率</th>
                        <th class="py-2">余额</th>
                        <th class="py-2">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-if="!members.length">
                        <td colspan="6" class="border-t border-gray-100 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-300">
                          没有找到成员
                        </td>
                      </tr>
                      <tr v-for="member in members" :key="member.id" class="border-t border-gray-100 dark:border-dark-700">
                        <td class="py-2">
                          <div class="font-medium text-gray-900 dark:text-white">{{ member.user_email }}</div>
                          <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">用户名：{{ member.user_username || '-' }}</div>
                          <input
                            v-model="member.member_note"
                            class="input mt-1 h-9 w-full min-w-[180px] text-xs"
                            placeholder="成员备注"
                          />
                        </td>
                        <td class="py-2">
                          <input
                            v-model.number="member.user_concurrency"
                            class="input h-9 w-24 text-xs"
                            type="number"
                            min="0"
                            step="1"
                            placeholder="0 不限"
                          />
                        </td>
                        <td class="py-2">
                          <input
                            v-model.number="member.pricing_factor"
                            class="input h-9 w-24 text-xs"
                            type="number"
                            min="0"
                            step="0.01"
                            :placeholder="tenantDefaultMemberRate.toFixed(3)"
                          />
                        </td>
                        <td class="py-2">
                          <div v-if="enterpriseGroupIDs.length" class="grid min-w-[220px] gap-1">
                            <label v-for="groupID in enterpriseGroupIDs" :key="groupID" class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-300">
                              <span class="w-32 shrink-0 truncate">{{ groupLabel(groupID) }}</span>
                              <input
                                v-model.number="member.group_rates[groupID]"
                                class="input h-8 w-24 text-xs"
                                type="number"
                                min="0.01"
                                step="0.001"
                                :placeholder="tenantMemberDefaultRate(groupID).toFixed(3)"
                              />
                            </label>
                          </div>
                          <span v-else class="text-xs text-gray-400">无企业专属分组</span>
                        </td>
                        <td class="py-2">{{ member.user_balance.toFixed(2) }}</td>
                        <td class="py-2">
                          <div class="flex flex-wrap gap-2">
                            <button class="btn btn-secondary btn-sm" @click="saveMember(member)">保存</button>
                            <button class="btn btn-secondary btn-sm" @click="grantBalance(member)">加余额</button>
                            <button class="btn btn-secondary btn-sm" @click="reclaimBalance(member)">回收</button>
                            <button class="btn btn-secondary btn-sm" @click="removeMember(member)">停用</button>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <div class="mt-3 flex flex-wrap items-center justify-between gap-2 border-t border-gray-100 pt-3 text-xs text-gray-500 dark:border-dark-700 dark:text-dark-300">
                  <span>共 {{ memberTotal }} 个成员，第 {{ memberPage }} / {{ memberPages }} 页</span>
                  <div class="flex items-center gap-2">
                    <button class="btn btn-secondary btn-sm" :disabled="memberPage <= 1" @click="changeMemberPage(memberPage - 1)">上一页</button>
                    <button class="btn btn-secondary btn-sm" :disabled="memberPage >= memberPages" @click="changeMemberPage(memberPage + 1)">下一页</button>
                  </div>
                </div>
              </div>

              <div class="grid gap-4 lg:grid-cols-2">
                <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                  <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">邀请码列表</div>
                  <div class="space-y-2">
                    <div v-for="invite in inviteCodes" :key="invite.id" class="rounded-lg border border-gray-100 px-3 py-2 dark:border-dark-700">
                      <div class="flex items-center justify-between gap-3">
                        <code class="text-sm font-semibold text-gray-900 dark:text-white">{{ invite.code }}</code>
                        <div class="flex items-center gap-2">
                          <span class="badge" :class="invite.status === 'active' ? 'badge-success' : 'badge-warning'">{{ invite.status }}</span>
                          <span class="text-xs text-gray-500 dark:text-dark-300">{{ invite.used_count }}/{{ invite.max_uses || '∞' }}</span>
                        </div>
                      </div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ invite.notes || '无备注' }}</div>
                      <div class="mt-2">
                        <button class="btn btn-secondary btn-sm" @click="toggleInvite(invite)">
                          {{ invite.status === 'active' ? '停用' : '启用' }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                  <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">额度台账</div>
                  <div class="space-y-2">
                    <div v-for="entry in ledger" :key="entry.id" class="rounded-lg border border-gray-100 px-3 py-2 text-sm dark:border-dark-700">
                      <div class="flex items-center justify-between gap-3">
                        <span class="font-medium text-gray-900 dark:text-white">{{ entry.direction }}</span>
                        <span class="text-xs text-gray-500 dark:text-dark-300">{{ formatDate(entry.created_at) }}</span>
                      </div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">
                        {{ entry.balance_before.toFixed(2) }} -> {{ entry.balance_after.toFixed(2) }} / {{ entry.amount.toFixed(2) }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
        </section>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import enterpriseAPI, { type EnterpriseMeResponse } from '@/api/enterprise'
import type { EnterpriseContext, EnterpriseGroupSummary, EnterpriseInviteCode, EnterpriseLedgerEntry, EnterpriseMembership } from '@/types'
import { useAppStore, useAuthStore } from '@/stores'

const appStore = useAppStore()
const authStore = useAuthStore()
const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const bindCode = ref('')
const memberSearch = ref('')
const memberPage = ref(1)
const memberPageSize = 20
const memberTotal = ref(0)

const me = reactive<EnterpriseMeResponse>({
  enterprise: null,
  tenant: null,
})

const members = ref<EnterpriseMembership[]>([])
const inviteCodes = ref<EnterpriseInviteCode[]>([])
const ledger = ref<EnterpriseLedgerEntry[]>([])
const groups = ref<EnterpriseGroupSummary[]>([])
const managerEnterprise = computed(() => isFullEnterpriseContext(me.enterprise) ? me.enterprise : null)

function isFullEnterpriseContext(value: EnterpriseMeResponse['enterprise']): value is EnterpriseContext {
  return !!value && 'pricing_floor_factor' in value
}

const memberForm = reactive({
  email: '',
  password: '',
  username: '',
  notes: '',
  member_note: '',
  concurrency: 0,
  initial_balance: 0,
  pricing_factor: 0,
  group_rates: {} as Record<number, number | undefined>,
})

const pricingDefaultsForm = reactive({
  member_default_pricing_factor: 0,
  member_default_concurrency: 0,
  member_group_rates: {} as Record<number, number | undefined>,
})

const inviteForm = reactive({
  code: '',
  max_uses: 0,
  expires_at: '',
  notes: '',
})

function showSuccess(message: string) {
  appStore.showSuccess(message)
}

function showError(error: unknown) {
  const detail = typeof error === 'object' && error && 'detail' in error ? String((error as { detail?: string }).detail || '') : ''
  const message = detail || (error instanceof Error ? error.message : '操作失败')
  appStore.showError(message)
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

const enterpriseGroupIDs = computed(() => {
  const ids = new Set<number>()
  const enterprise = managerEnterprise.value
  for (const id of me.tenant?.allowed_group_ids || enterprise?.allowed_group_ids || []) {
    if (Number(id) > 0) ids.add(Number(id))
  }
  for (const key of Object.keys(me.tenant?.group_rates || enterprise?.group_rates || {})) {
    const id = Number(key)
    if (id > 0) ids.add(id)
  }
  for (const key of Object.keys(me.tenant?.member_group_rates || enterprise?.member_group_rates || {})) {
    const id = Number(key)
    if (id > 0) ids.add(id)
  }
  return [...ids].sort((a, b) => a - b)
})

const enterpriseTotalBalance = computed(() => Number(me.tenant?.balance_quota_total ?? managerEnterprise.value?.balance_quota_total ?? 0))
const enterpriseSpentBalance = computed(() => Number(me.tenant?.balance_quota_spent ?? managerEnterprise.value?.balance_quota_spent ?? 0))
const enterpriseOverdraftLimit = computed(() => Number(me.tenant?.balance_overdraft_limit ?? managerEnterprise.value?.balance_overdraft_limit ?? 0))
const enterpriseAvailableBalance = computed(() => enterpriseTotalBalance.value + enterpriseOverdraftLimit.value - enterpriseSpentBalance.value)
const memberPages = computed(() => Math.max(1, Math.ceil(memberTotal.value / memberPageSize)))
const enterpriseBaseFloorRate = computed(() => Number(managerEnterprise.value?.pricing_floor_factor || me.tenant?.pricing_floor_factor || 1))
const tenantDefaultMemberRate = computed(() => {
  const value = Number(me.tenant?.member_default_pricing_factor ?? managerEnterprise.value?.member_default_pricing_factor ?? 0)
  if (Number.isFinite(value) && value > 0) {
    return value
  }
  return enterpriseBaseFloorRate.value
})
const tenantDefaultMemberConcurrency = computed(() => Number(me.tenant?.member_default_concurrency ?? managerEnterprise.value?.member_default_concurrency ?? 0) || 0)

function groupLabel(groupID: number): string {
  const group = groups.value.find((item) => item.id === groupID)
  return group ? `#${group.id} ${group.name}` : `#${groupID}`
}

function tenantGroupFloor(groupID: number): number {
  const value = me.tenant?.group_rates?.[groupID] ?? managerEnterprise.value?.group_rates?.[groupID]
  if (Number.isFinite(Number(value)) && Number(value) > 0) {
    return Number(value)
  }
  const group = groups.value.find((item) => item.id === groupID)
  if (Number.isFinite(Number(group?.rate_multiplier)) && Number(group?.rate_multiplier) > 0) {
    return Number(group?.rate_multiplier)
  }
  return Number(managerEnterprise.value?.pricing_floor_factor || me.tenant?.pricing_floor_factor || 1)
}

function tenantMemberDefaultRate(groupID: number): number {
  const value = me.tenant?.member_group_rates?.[groupID] ?? managerEnterprise.value?.member_group_rates?.[groupID]
  if (Number.isFinite(Number(value)) && Number(value) > 0) {
    return Number(value)
  }
  return tenantDefaultMemberRate.value || tenantGroupFloor(groupID)
}

function buildGroupRatesPayload(rates: Record<number, number | undefined>): Record<number, number | null> | undefined {
  if (!enterpriseGroupIDs.value.length) {
    return undefined
  }
  const payload: Record<number, number | null> = {}
  for (const groupID of enterpriseGroupIDs.value) {
    const value = Number(rates[groupID])
    payload[groupID] = Number.isFinite(value) && value > 0 ? value : null
  }
  return payload
}

function syncPricingDefaultsForm() {
  pricingDefaultsForm.member_default_pricing_factor = Number(me.tenant?.member_default_pricing_factor ?? managerEnterprise.value?.member_default_pricing_factor ?? 0) || 0
  pricingDefaultsForm.member_default_concurrency = tenantDefaultMemberConcurrency.value
  pricingDefaultsForm.member_group_rates = { ...(me.tenant?.member_group_rates || managerEnterprise.value?.member_group_rates || {}) }
}

async function loadMe() {
  const data = await enterpriseAPI.getMe()
  me.enterprise = data.enterprise || null
  me.tenant = data.tenant || null
  if (me.enterprise?.member_role && me.enterprise.member_role !== 'manager') {
    await authStore.refreshUser().catch(() => undefined)
    await router.replace('/dashboard')
    return
  }
  syncPricingDefaultsForm()
}

async function loadMembers() {
  if (!managerEnterprise.value) return
  const res = await enterpriseAPI.listMembers(memberPage.value, memberPageSize, { search: memberSearch.value.trim() || undefined })
  members.value = res.items.map((member) => ({
    ...member,
    group_rates: { ...(member.group_rates || {}) },
  }))
  memberTotal.value = res.total
}

async function applyMemberFilters() {
  memberPage.value = 1
  await loadMembers()
}

async function changeMemberPage(page: number) {
  memberPage.value = Math.min(Math.max(1, page), memberPages.value)
  await loadMembers()
}

async function loadInviteCodes() {
  if (!managerEnterprise.value) return
  const res = await enterpriseAPI.listInviteCodes(1, 100)
  inviteCodes.value = res.items
}

async function loadLedger() {
  if (!managerEnterprise.value) return
  const res = await enterpriseAPI.listLedger(1, 100)
  ledger.value = res.items
}

async function loadGroups() {
  if (!managerEnterprise.value) return
  groups.value = await enterpriseAPI.listGroups()
}

async function loadAll() {
  loading.value = true
  try {
    await loadMe()
    if (me.enterprise && !managerEnterprise.value) {
      return
    }
    await Promise.all([loadGroups(), loadMembers(), loadInviteCodes(), loadLedger()])
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function submitBindInvite() {
  submitting.value = true
  try {
    await enterpriseAPI.bindInviteCode(bindCode.value.trim())
    bindCode.value = ''
    showSuccess('企业邀请码绑定成功')
    await loadAll()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitCreateMember() {
  submitting.value = true
  try {
    await enterpriseAPI.createMember({
      email: memberForm.email.trim(),
      password: memberForm.password,
      username: memberForm.username.trim() || undefined,
      notes: memberForm.notes.trim() || undefined,
      concurrency: Number(memberForm.concurrency) || 0,
      member_note: memberForm.member_note.trim() || undefined,
      pricing_factor: Math.max(0, Number(memberForm.pricing_factor) || 0),
      pricing_scope: 'balance',
      group_rates: buildGroupRatesPayload(memberForm.group_rates),
      initial_balance: Number(memberForm.initial_balance) || 0,
    })
    showSuccess('企业成员已创建')
    memberForm.email = ''
    memberForm.password = ''
    memberForm.username = ''
    memberForm.notes = ''
    memberForm.member_note = ''
    memberForm.concurrency = 0
    memberForm.initial_balance = 0
    memberForm.pricing_factor = 0
    memberForm.group_rates = {}
    await Promise.all([loadMembers(), loadLedger(), loadMe()])
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitCreateInvite() {
  submitting.value = true
  try {
    await enterpriseAPI.createInviteCode({
      code: inviteForm.code.trim() || undefined,
      max_uses: Number(inviteForm.max_uses) || 0,
      expires_at: inviteForm.expires_at ? new Date(inviteForm.expires_at).toISOString() : undefined,
      notes: inviteForm.notes.trim() || undefined,
    })
    inviteForm.code = ''
    inviteForm.max_uses = 0
    inviteForm.expires_at = ''
    inviteForm.notes = ''
    showSuccess('邀请码已创建')
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitPricingDefaults() {
  submitting.value = true
  try {
    await enterpriseAPI.updatePricingDefaults({
      member_default_pricing_factor: Math.max(0, Number(pricingDefaultsForm.member_default_pricing_factor) || 0),
      member_default_concurrency: Math.max(0, Number(pricingDefaultsForm.member_default_concurrency) || 0),
      member_group_rates: buildGroupRatesPayload(pricingDefaultsForm.member_group_rates),
    })
    showSuccess('默认成员售价已保存')
    await loadMe()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function saveMember(member: EnterpriseMembership) {
  submitting.value = true
  try {
    await enterpriseAPI.updateMember(member.user_id, {
      member_note: member.member_note || undefined,
      pricing_factor: Math.max(0, Number(member.pricing_factor) || 0),
      pricing_scope: 'balance',
      concurrency: Math.max(0, Number(member.user_concurrency) || 0),
      group_rates: buildGroupRatesPayload(member.group_rates || {}),
    })
    showSuccess('成员信息已保存')
    await loadMembers()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function removeMember(member: EnterpriseMembership) {
  if (!window.confirm(`确认停用成员 ${member.user_email} 吗？`)) return
  try {
    await enterpriseAPI.updateMember(member.user_id, { status: 'disabled' })
    showSuccess('成员已停用')
    await loadMembers()
  } catch (error) {
    showError(error)
  }
}

async function grantBalance(member: EnterpriseMembership) {
  const raw = window.prompt(`给 ${member.user_email} 增加多少额度？`)
  if (!raw) return
  const amount = Number(raw)
  if (!Number.isFinite(amount) || amount <= 0) return
  try {
    await enterpriseAPI.adjustMemberBalance(member.user_id, { amount, operation: 'grant' })
    showSuccess('额度已发放')
    await Promise.all([loadMembers(), loadLedger(), loadMe()])
  } catch (error) {
    showError(error)
  }
}

async function reclaimBalance(member: EnterpriseMembership) {
  const raw = window.prompt(`从 ${member.user_email} 回收多少额度？`)
  if (!raw) return
  const amount = Number(raw)
  if (!Number.isFinite(amount) || amount <= 0) return
  try {
    await enterpriseAPI.adjustMemberBalance(member.user_id, { amount, operation: 'reclaim' })
    showSuccess('额度已回收')
    await Promise.all([loadMembers(), loadLedger(), loadMe()])
  } catch (error) {
    showError(error)
  }
}

async function toggleInvite(invite: EnterpriseInviteCode) {
  try {
    await enterpriseAPI.updateInviteCode(invite.id, {
      status: invite.status === 'active' ? 'disabled' : 'active',
    })
    showSuccess('邀请码状态已更新')
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  }
}

onMounted(loadAll)
</script>
