<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">企业管理</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">管理企业租户、额度池、成员、邀请码与台账。</p>
          </div>
          <div class="flex gap-2">
            <input v-model="search" class="input w-56" placeholder="搜索企业名称 / 编码" @keyup.enter="loadTenants" />
            <button class="btn btn-secondary" :disabled="loading" @click="loadTenants">刷新</button>
          </div>
        </div>

        <div class="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
          <div class="space-y-3">
            <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
              <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">新建 / 编辑企业</div>
              <div class="space-y-2">
                <input v-model="tenantForm.name" class="input" placeholder="企业名称" />
                <input v-model="tenantForm.code" class="input" placeholder="企业编码" :disabled="!!selectedTenant?.id" />
                <div class="grid grid-cols-2 gap-2">
	                  <input v-model="tenantForm.pricing_floor_factor" class="input" type="number" min="0.01" step="0.01" placeholder="最低倍率" />
	                  <select v-model="tenantForm.pricing_scope" class="input">
	                    <option value="balance">仅余额</option>
	                  </select>
	                </div>
                <input v-model="tenantForm.portal_host" class="input" placeholder="门户域名，可留空" />
                <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
                  <div class="mb-2 flex items-center justify-between gap-2">
                    <div class="text-xs font-medium text-gray-600 dark:text-dark-200">企业可用分组</div>
                    <button class="btn btn-secondary btn-sm" type="button" @click="loadGroups">刷新分组</button>
                  </div>
                  <div class="max-h-52 space-y-2 overflow-y-auto pr-1">
                    <label
                      v-for="group in groupOptions"
                      :key="group.id"
                      class="flex cursor-pointer items-start gap-2 rounded-lg border border-gray-100 p-2 text-xs hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                    >
                      <input v-model="tenantForm.allowed_group_ids" class="mt-1" type="checkbox" :value="group.id" />
                      <span class="min-w-0 flex-1">
                        <span class="block truncate font-medium text-gray-800 dark:text-dark-100">
                          #{{ group.id }} {{ group.name }}
                        </span>
                        <span class="mt-1 block text-gray-500 dark:text-dark-300">
                          {{ group.platform }} · {{ group.subscription_type === 'subscription' ? '订阅' : '余额' }} · {{ group.is_exclusive ? '专属' : '公开' }}
                        </span>
                      </span>
                    </label>
                    <div v-if="!groupOptions.length" class="text-xs text-gray-500 dark:text-dark-300">暂无可选分组</div>
                  </div>
                  <div v-if="selectedGroupLabels.length || missingGroupIDs.length" class="mt-3 flex flex-wrap gap-2">
                    <span
                      v-for="group in selectedGroupLabels"
                      :key="group.id"
                      class="rounded-full bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-200"
                    >
                      #{{ group.id }} {{ group.name }}
                    </span>
                    <span
                      v-for="id in missingGroupIDs"
                      :key="id"
                      class="rounded-full bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-200"
                    >
                      未加载分组 #{{ id }}
                    </span>
                  </div>
                  <p class="mt-2 text-xs text-gray-500 dark:text-dark-300">
                    企业成员可直接使用这里选中的分组，包括专属分组；未选择时不额外限制公开分组。
                  </p>
                </div>
                <select v-model="tenantForm.status" class="input">
                  <option value="active">启用</option>
                  <option value="disabled">停用</option>
                </select>
                <textarea v-model="tenantForm.notes" class="input min-h-[92px]" placeholder="备注"></textarea>
              </div>
              <div class="mt-3 flex gap-2">
                <button class="btn btn-primary flex-1" :disabled="submitting" @click="submitTenant">
                  {{ selectedTenant?.id ? '保存企业' : '创建企业' }}
                </button>
                <button class="btn btn-secondary" @click="resetTenantForm">清空</button>
              </div>
            </div>

            <div class="rounded-xl border border-gray-200 dark:border-dark-700">
              <button
                v-for="item in tenants"
                :key="item.id"
                class="w-full border-b border-gray-100 px-4 py-3 text-left last:border-b-0 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700/40"
                :class="selectedTenant?.id === item.id ? 'bg-primary-50 dark:bg-primary-900/20' : ''"
                @click="selectTenant(item)"
              >
                <div class="flex items-center justify-between gap-3">
                  <div class="min-w-0">
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.name }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-dark-300">{{ item.code }}</div>
                  </div>
                  <span class="badge" :class="item.status === 'active' ? 'badge-success' : 'badge-warning'">
                    {{ item.status }}
                  </span>
                </div>
                <div class="mt-2 text-xs text-gray-500 dark:text-dark-300">
                  已用 {{ item.balance_quota_used.toFixed(2) }} / 总额 {{ item.balance_quota_total.toFixed(2) }}
                </div>
              </button>
            </div>
          </div>

          <div v-if="selectedTenant" class="space-y-4">
            <div class="grid gap-4 md:grid-cols-4">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">企业额度</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_quota_total.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">已分发额度</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.balance_quota_used.toFixed(2) }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">企业管理员</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.manager_count }}</div>
              </div>
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="text-xs text-gray-500 dark:text-dark-300">成员数</div>
                <div class="mt-2 text-xl font-semibold text-gray-900 dark:text-white">{{ selectedTenant.member_count }}</div>
              </div>
            </div>

            <div class="grid gap-4 lg:grid-cols-2">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">调整企业额度池</div>
                <div class="grid grid-cols-[1fr_120px] gap-2">
                  <input v-model="quotaForm.amount" class="input" type="number" min="0.01" step="0.01" placeholder="额度" />
                  <select v-model="quotaForm.direction" class="input">
                    <option value="platform_grant">增加</option>
                    <option value="platform_reclaim">回收</option>
                  </select>
                </div>
                <textarea v-model="quotaForm.notes" class="input mt-2 min-h-[72px]" placeholder="备注"></textarea>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitQuota">提交额度调整</button>
              </div>

              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 text-sm font-medium text-gray-700 dark:text-dark-200">绑定现有用户到企业</div>
                <div class="space-y-2">
                  <input v-model="bindForm.user_id" class="input" type="number" min="1" placeholder="用户 ID" />
                  <div class="grid grid-cols-2 gap-2">
                    <select v-model="bindForm.member_role" class="input">
                      <option value="member">成员</option>
                      <option value="manager">管理员</option>
                    </select>
                    <input v-model="bindForm.pricing_factor" class="input" type="number" min="0.01" step="0.01" placeholder="倍率" />
                  </div>
                  <textarea v-model="bindForm.member_note" class="input min-h-[72px]" placeholder="成员备注"></textarea>
                </div>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitBindMember">绑定用户</button>
              </div>
            </div>

            <div class="grid gap-4 xl:grid-cols-2">
              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex items-center justify-between">
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">成员列表</div>
                  <button class="btn btn-secondary btn-sm" @click="loadMembers">刷新</button>
                </div>
                <div class="overflow-x-auto">
                  <table class="min-w-full text-sm">
                    <thead class="text-left text-gray-500 dark:text-dark-300">
                      <tr>
                        <th class="py-2">用户</th>
                        <th class="py-2">角色</th>
                        <th class="py-2">倍率</th>
                        <th class="py-2">余额</th>
                        <th class="py-2">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="member in members" :key="member.id" class="border-t border-gray-100 dark:border-dark-700">
                        <td class="py-2">
                          <div class="font-medium text-gray-900 dark:text-white">{{ member.user_email }}</div>
                          <div class="text-xs text-gray-500 dark:text-dark-300">{{ member.member_note || '-' }}</div>
                        </td>
                        <td class="py-2">{{ member.member_role }}</td>
                        <td class="py-2">{{ member.pricing_factor.toFixed(2) }}x</td>
                        <td class="py-2">{{ member.user_balance.toFixed(2) }}</td>
                        <td class="py-2">
                          <button class="btn btn-secondary btn-sm" @click="removeMember(member)">移除</button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>

              <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
                <div class="mb-3 flex items-center justify-between">
                  <div class="text-sm font-medium text-gray-700 dark:text-dark-200">邀请码</div>
                  <button class="btn btn-secondary btn-sm" @click="loadInviteCodes">刷新</button>
                </div>
                <div class="mb-3 grid grid-cols-[1fr_110px] gap-2">
                  <input v-model="inviteForm.code" class="input" placeholder="留空自动生成" />
                  <input v-model="inviteForm.max_uses" class="input" type="number" min="0" placeholder="次数" />
                </div>
                <input v-model="inviteForm.expires_at" class="input mb-2" type="datetime-local" />
                <textarea v-model="inviteForm.notes" class="input min-h-[72px]" placeholder="备注"></textarea>
                <button class="btn btn-primary mt-3" :disabled="submitting" @click="submitInvite">创建邀请码</button>
                <div class="mt-4 space-y-2">
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
            </div>

            <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
              <div class="mb-3 flex items-center justify-between">
                <div class="text-sm font-medium text-gray-700 dark:text-dark-200">额度台账</div>
                <button class="btn btn-secondary btn-sm" @click="loadLedger">刷新</button>
              </div>
              <div class="overflow-x-auto">
                <table class="min-w-full text-sm">
                  <thead class="text-left text-gray-500 dark:text-dark-300">
                    <tr>
                      <th class="py-2">时间</th>
                      <th class="py-2">方向</th>
                      <th class="py-2">金额</th>
                      <th class="py-2">前后</th>
                      <th class="py-2">备注</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="entry in ledger" :key="entry.id" class="border-t border-gray-100 dark:border-dark-700">
                      <td class="py-2 text-xs">{{ formatDate(entry.created_at) }}</td>
                      <td class="py-2">{{ entry.direction }}</td>
                      <td class="py-2">{{ entry.amount.toFixed(2) }}</td>
                      <td class="py-2">{{ entry.balance_before.toFixed(2) }} -> {{ entry.balance_after.toFixed(2) }}</td>
                      <td class="py-2">{{ entry.notes || '-' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import enterpriseAdminAPI from '@/api/admin/enterprise'
import groupsAPI from '@/api/admin/groups'
import type { AdminGroup, EnterpriseInviteCode, EnterpriseLedgerEntry, EnterpriseMembership, EnterpriseTenant } from '@/types'
import { useAppStore } from '@/stores'

const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const search = ref('')
const tenants = ref<EnterpriseTenant[]>([])
const selectedTenant = ref<EnterpriseTenant | null>(null)
const members = ref<EnterpriseMembership[]>([])
const inviteCodes = ref<EnterpriseInviteCode[]>([])
const ledger = ref<EnterpriseLedgerEntry[]>([])
const groups = ref<AdminGroup[]>([])

const tenantForm = reactive({
  name: '',
  code: '',
  status: 'active',
  notes: '',
  portal_host: '',
  pricing_floor_factor: 1,
  pricing_scope: 'balance',
  allowed_group_ids: [] as number[],
})

const quotaForm = reactive({
  amount: 0,
  direction: 'platform_grant',
  notes: '',
})

const bindForm = reactive({
  user_id: 0,
  member_role: 'member',
  member_note: '',
  pricing_factor: 1,
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

const groupOptions = computed(() =>
  [...groups.value].sort((a, b) => {
    if (a.platform !== b.platform) return a.platform.localeCompare(b.platform)
    return a.id - b.id
  })
)

const selectedGroupLabels = computed(() => {
  const byID = new Map(groups.value.map((group) => [group.id, group]))
  return tenantForm.allowed_group_ids
    .map((id) => byID.get(id))
    .filter((group): group is AdminGroup => !!group)
})

const missingGroupIDs = computed(() => {
  const known = new Set(groups.value.map((group) => group.id))
  return tenantForm.allowed_group_ids.filter((id) => !known.has(id))
})

function resetTenantForm() {
  tenantForm.name = ''
  tenantForm.code = ''
  tenantForm.status = 'active'
  tenantForm.notes = ''
  tenantForm.portal_host = ''
  tenantForm.pricing_floor_factor = 1
  tenantForm.pricing_scope = 'balance'
  tenantForm.allowed_group_ids = []
  selectedTenant.value = null
}

function fillTenantForm(item: EnterpriseTenant) {
  tenantForm.name = item.name
  tenantForm.code = item.code
  tenantForm.status = item.status
  tenantForm.notes = item.notes || ''
  tenantForm.portal_host = item.portal_host || ''
  tenantForm.pricing_floor_factor = item.pricing_floor_factor
  tenantForm.pricing_scope = item.pricing_scope || 'balance'
  tenantForm.allowed_group_ids = [...(item.allowed_group_ids || [])]
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

async function loadTenants() {
  loading.value = true
  try {
    const res = await enterpriseAdminAPI.listTenants(1, 100, { search: search.value.trim() || undefined })
    tenants.value = res.items
    if (!selectedTenant.value && res.items.length > 0) {
      await selectTenant(res.items[0])
    } else if (selectedTenant.value) {
      const next = res.items.find((item) => item.id === selectedTenant.value?.id)
      if (next) {
        selectedTenant.value = next
        fillTenantForm(next)
      }
    }
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    groups.value = await groupsAPI.getAll()
  } catch (error) {
    showError(error)
  }
}

async function selectTenant(item: EnterpriseTenant) {
  selectedTenant.value = item
  fillTenantForm(item)
  await Promise.all([loadMembers(), loadInviteCodes(), loadLedger()])
}

async function loadMembers() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listMembers(selectedTenant.value.id, 1, 100)
    members.value = res.items
  } catch (error) {
    showError(error)
  }
}

async function loadInviteCodes() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listInviteCodes(selectedTenant.value.id, 1, 100)
    inviteCodes.value = res.items
  } catch (error) {
    showError(error)
  }
}

async function loadLedger() {
  if (!selectedTenant.value) return
  try {
    const res = await enterpriseAdminAPI.listLedger(selectedTenant.value.id, 1, 100)
    ledger.value = res.items
  } catch (error) {
    showError(error)
  }
}

async function submitTenant() {
  submitting.value = true
  try {
    const payload = {
      name: tenantForm.name.trim(),
      code: tenantForm.code.trim(),
      status: tenantForm.status,
      notes: tenantForm.notes.trim(),
      portal_host: tenantForm.portal_host.trim(),
      pricing_floor_factor: Number(tenantForm.pricing_floor_factor) || 1,
      pricing_scope: tenantForm.pricing_scope,
      allowed_group_ids: [...tenantForm.allowed_group_ids],
    }
    if (selectedTenant.value?.id) {
      await enterpriseAdminAPI.updateTenant(selectedTenant.value.id, payload)
      showSuccess('企业已更新')
    } else {
      await enterpriseAdminAPI.createTenant(payload)
      showSuccess('企业已创建')
    }
    await loadTenants()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitQuota() {
  if (!selectedTenant.value) return
  submitting.value = true
  try {
    selectedTenant.value = await enterpriseAdminAPI.adjustQuota(selectedTenant.value.id, {
      amount: Number(quotaForm.amount),
      direction: quotaForm.direction,
      notes: quotaForm.notes.trim() || undefined,
    })
    showSuccess('额度池已调整')
    await Promise.all([loadTenants(), loadLedger()])
    quotaForm.amount = 0
    quotaForm.notes = ''
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function submitBindMember() {
  if (!selectedTenant.value) return
  submitting.value = true
  try {
    await enterpriseAdminAPI.bindMember(selectedTenant.value.id, {
      user_id: Number(bindForm.user_id),
      member_role: bindForm.member_role,
      member_note: bindForm.member_note,
      pricing_factor: Number(bindForm.pricing_factor) || 1,
      pricing_scope: 'balance',
      joined_via: 'manual_bind',
      joined_source: 'admin_bind',
    })
    showSuccess('成员已绑定')
    bindForm.user_id = 0
    bindForm.member_note = ''
    bindForm.pricing_factor = 1
    await Promise.all([loadMembers(), loadTenants()])
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function removeMember(member: EnterpriseMembership) {
  if (!selectedTenant.value) return
  if (!window.confirm(`确认移除成员 ${member.user_email} 吗？`)) return
  try {
    await enterpriseAdminAPI.deleteMember(selectedTenant.value.id, member.user_id)
    showSuccess('成员已移除')
    await Promise.all([loadMembers(), loadTenants()])
  } catch (error) {
    showError(error)
  }
}

async function submitInvite() {
  if (!selectedTenant.value) return
  submitting.value = true
  try {
    await enterpriseAdminAPI.createInviteCode(selectedTenant.value.id, {
      code: inviteForm.code.trim() || undefined,
      max_uses: Number(inviteForm.max_uses) || 0,
      expires_at: inviteForm.expires_at ? new Date(inviteForm.expires_at).toISOString() : undefined,
      notes: inviteForm.notes.trim() || undefined,
    })
    showSuccess('邀请码已创建')
    inviteForm.code = ''
    inviteForm.max_uses = 0
    inviteForm.expires_at = ''
    inviteForm.notes = ''
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  } finally {
    submitting.value = false
  }
}

async function toggleInvite(invite: EnterpriseInviteCode) {
  try {
    await enterpriseAdminAPI.updateInviteCode(invite.id, {
      status: invite.status === 'active' ? 'disabled' : 'active',
    })
    showSuccess('邀请码状态已更新')
    await loadInviteCodes()
  } catch (error) {
    showError(error)
  }
}

onMounted(() => {
  void Promise.all([loadTenants(), loadGroups()])
})
</script>
