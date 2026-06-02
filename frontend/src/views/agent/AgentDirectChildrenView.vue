<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ title }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ directSubtitle }}</p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2">
              <input
                v-model="searchDraft"
                data-test="direct-child-search"
                class="input h-9 w-48"
                type="search"
                :placeholder="t('common.search')"
                @keyup.enter="applySearch"
              />
              <button
                data-test="direct-child-search-submit"
                class="btn btn-secondary px-3"
                :disabled="loading"
                @click="applySearch"
              >
                <Icon name="search" size="sm" />
              </button>
            </div>
            <button
              v-if="canCreateDirectUser"
              data-test="create-direct-user"
              class="btn btn-primary px-3"
              :disabled="loading"
              @click="showCreateUserModal = true"
            >
              <Icon name="userPlus" size="sm" />
              <span>{{ t('agentManagement.direct.createUser') }}</span>
            </button>
            <span
              v-if="isAdminUnlimitedCapacity"
              class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200"
            >
              {{ t('agentManagement.direct.adminUnlimitedCapacity') }}
            </span>
            <span v-else class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('agentManagement.direct.remainingConcurrency') }}: {{ remainingConcurrencyText }}
            </span>
            <span v-if="!isAdminUnlimitedCapacity" class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('agentManagement.direct.remainingRpm') }}: {{ remainingRpmText }}
            </span>
            <button class="btn btn-secondary px-3" :disabled="loading" @click="loadData">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
        <form
          v-if="showInviteDefaultsForm"
          data-test="invite-default-submit"
          class="mt-3 flex flex-wrap items-end gap-3 border-t border-gray-200 pt-3 dark:border-dark-700"
          @submit.prevent="saveInviteDefaults"
        >
          <label class="flex min-w-[150px] flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('agentManagement.direct.inviteDefaultConcurrency') }}</span>
            <input
              v-model.number="inviteDefaultsDraft.invite_default_concurrency"
              data-test="invite-default-concurrency"
              class="input h-9"
              type="number"
              min="0"
              step="1"
            />
          </label>
          <label class="flex min-w-[150px] flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('agentManagement.direct.inviteDefaultRpm') }}</span>
            <input
              v-model.number="inviteDefaultsDraft.invite_default_rpm"
              data-test="invite-default-rpm"
              class="input h-9"
              type="number"
              min="0"
              step="1"
            />
          </label>
          <button class="btn btn-secondary h-9 px-3" type="submit" :disabled="savingInviteDefaults || loading">
            <Icon name="check" size="sm" />
            <span>{{ t('agentManagement.direct.saveInviteDefaults') }}</span>
          </button>
        </form>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="children" :loading="loading">
          <template #cell-email="{ value, row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ row.username || '-' }}</span>
            </div>
          </template>

          <template #cell-role="{ value }">
            <span class="badge badge-gray">{{ roleLabel(value) }}</span>
          </template>

          <template #cell-balance="{ value }">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ formatCurrency(Number(value || 0)) }}</span>
          </template>

          <template #cell-allocation="{ row }">
            <div class="grid min-w-[240px] grid-cols-2 gap-2">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedConcurrency') }}</span>
                <input
                  :data-test="`allocation-concurrency-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  :value="draftFor(row).concurrency"
                  @input="updateDraft(row.id, 'concurrency', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedRpm') }}</span>
                <input
                  :data-test="`allocation-rpm-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  :value="draftFor(row).rpm"
                  @input="updateDraft(row.id, 'rpm', ($event.target as HTMLInputElement).value)"
                />
              </label>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center gap-2">
              <button class="btn btn-primary btn-sm" :disabled="savingChildId === row.id" @click="saveAllocation(row)">
                <Icon name="check" size="sm" />
                <span>{{ t('agentManagement.direct.saveAllocation') }}</span>
              </button>

              <button
                v-for="targetRole in upgradeTargets"
                :key="targetRole"
                :data-test="`upgrade-${targetRole}-${row.id}`"
                class="btn btn-secondary btn-sm"
                :disabled="savingChildId === row.id"
                @click="upgrade(row, targetRole)"
              >
                <Icon name="userPlus" size="sm" />
                <span>{{ roleLabel(targetRole) }}</span>
              </button>

              <button class="btn btn-secondary btn-sm text-red-600 dark:text-red-400" @click="askDetach(row)">
                <Icon name="trash" size="sm" />
                <span>{{ deleteActionLabel(row) }}</span>
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          :show-page-size-selector="false"
          @update:page="pagination.page = $event"
          @update:page-size="pagination.page_size = $event"
        />
      </template>
    </TablePageLayout>

    <ConfirmDialog
      :show="detachDialog.show"
      :title="deleteDialogTitle"
      :message="deleteDialogMessage"
      :confirm-text="deleteDialogConfirmText"
      danger
      @confirm="confirmDetach"
      @cancel="detachDialog.show = false"
    />
    <AgentDirectUserCreateModal
      v-if="canCreateDirectUser"
      :show="showCreateUserModal"
      :loading="creatingUser"
      @close="showCreateUserModal = false"
      @submit="createDirectUser"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore, useAuthStore } from '@/stores'
import { formatCurrency } from '@/utils/format'
import type {
  AgentAllocationSummary,
  AgentAllocationUpdate,
  AgentDirectChildrenResponse,
  AgentDirectUserCreateRequest,
  AgentInviteDefaultsUpdate,
  AgentManagedUser,
  AgentUpgradeTargetRole,
  UserRole
} from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AgentDirectUserCreateModal from '@/components/agent/AgentDirectUserCreateModal.vue'
import Icon from '@/components/icons/Icon.vue'

type ChildKind = 'users' | 'agents' | 'enterprises'

const props = defineProps<{
  kind: ChildKind
  title: string
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const savingChildId = ref<number | null>(null)
const creatingUser = ref(false)
const showCreateUserModal = ref(false)
const savingInviteDefaults = ref(false)
const searchDraft = ref('')
const activeSearch = ref('')
const children = ref<AgentManagedUser[]>([])
const allocation = ref<AgentAllocationSummary | null>(null)
const inviteDefaultsDraft = reactive<AgentInviteDefaultsUpdate>({
  invite_default_concurrency: 1,
  invite_default_rpm: 1,
})
const drafts = reactive<Record<number, AgentAllocationUpdate>>({})
const pagination = reactive({ total: 0, page: 1, page_size: 20, pages: 1 })
const detachDialog = reactive<{ show: boolean; child: AgentManagedUser | null }>({ show: false, child: null })

const columns = computed<Column[]>(() => [
  { key: 'email', label: t('common.email') },
  { key: 'role', label: t('agentManagement.direct.role') },
  { key: 'balance', label: t('agentManagement.direct.balance') },
  { key: 'allocation', label: t('agentManagement.direct.allocation') },
  { key: 'status', label: t('common.status') },
  { key: 'actions', label: t('common.actions') },
])

const upgradeTargets = computed<AgentUpgradeTargetRole[]>(() => {
  if (props.kind !== 'users') return []
  const role = authStore.user?.role
  if (role === 'admin') return ['agent_level1', 'enterprise']
  if (role === 'agent_level1') return ['agent_level2', 'enterprise']
  if (role === 'agent_level2') return ['enterprise']
  return []
})

const canCreateDirectUser = computed(() => props.kind === 'users')

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isAgent = computed(() => authStore.user?.role === 'agent_level1' || authStore.user?.role === 'agent_level2')
const showInviteDefaultsForm = computed(() => props.kind === 'users' && isAgent.value)
const isAdminUnlimitedCapacity = computed(() => isAdmin.value)

const directSubtitle = computed(() => {
  if (isAdmin.value) {
    return t('agentManagement.direct.adminSubtitle')
  }
  return t('agentManagement.direct.subtitle')
})

const deleteDialogIsTrueDelete = computed(() => isTrueDeleteDirectUser(detachDialog.child))
const deleteDialogTitle = computed(() => deleteDialogIsTrueDelete.value ? t('agentManagement.direct.deleteUserTitle') : t('agentManagement.direct.detachTitle'))
const deleteDialogMessage = computed(() => {
  const email = detachDialog.child?.email || ''
  return deleteDialogIsTrueDelete.value
    ? t('agentManagement.direct.deleteUserConfirm', { email })
    : t('agentManagement.direct.detachConfirm', { email })
})
const deleteDialogConfirmText = computed(() => deleteDialogIsTrueDelete.value ? t('agentManagement.direct.deleteUser') : t('agentManagement.direct.detach'))
const remainingConcurrencyText = computed(() => allocation.value?.unlimited_concurrency ? t('common.unlimited') : String(allocation.value?.remaining_concurrency ?? '-'))
const remainingRpmText = computed(() => allocation.value?.unlimited_rpm ? t('common.unlimited') : String(allocation.value?.remaining_rpm ?? '-'))

function extractPagination(result: AgentDirectChildrenResponse) {
  const source = result.pagination || {}
  pagination.total = Number(source.total ?? source.Total ?? result.items.length)
  pagination.page = Number(source.page ?? source.Page ?? 1)
  pagination.page_size = Number(source.page_size ?? source.PageSize ?? 20)
  pagination.pages = Number(source.pages ?? source.Pages ?? 1)
}

function syncDrafts(items: AgentManagedUser[]) {
  for (const child of items) {
    drafts[child.id] = quotaFor(child)
  }
}

function syncInviteDefaults(defaults?: AgentInviteDefaultsUpdate) {
  inviteDefaultsDraft.invite_default_concurrency = normalizedNonNegative(defaults?.invite_default_concurrency ?? 1)
  inviteDefaultsDraft.invite_default_rpm = normalizedNonNegative(defaults?.invite_default_rpm ?? 1)
}

function quotaFor(child: AgentManagedUser): AgentAllocationUpdate {
  if (props.kind === 'agents') {
    return {
      concurrency: child.pool_concurrency || 0,
      rpm: child.pool_rpm || 0,
    }
  }
  return {
    concurrency: child.concurrency || 0,
    rpm: child.rpm_limit || 0,
  }
}

async function listChildren(): Promise<AgentDirectChildrenResponse> {
  const query = activeSearch.value ? { search: activeSearch.value } : {}
  if (props.kind === 'agents') return agentManagementAPI.listDirectAgents(query)
  if (props.kind === 'enterprises') return agentManagementAPI.listDirectEnterprises(query)
  return agentManagementAPI.listDirectUsers(query)
}

async function loadData() {
  loading.value = true
  try {
    const [summary, result] = await Promise.all([
      agentManagementAPI.getSummary(),
      listChildren(),
    ])
    allocation.value = summary.allocation
    if (showInviteDefaultsForm.value) {
      syncInviteDefaults(summary.invite_defaults)
    }
    children.value = result.items
    syncDrafts(result.items)
    extractPagination(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function applySearch() {
  activeSearch.value = searchDraft.value.trim()
  await loadData()
}

function draftFor(child: AgentManagedUser): AgentAllocationUpdate {
  if (!drafts[child.id]) {
    drafts[child.id] = quotaFor(child)
  }
  return drafts[child.id]
}

function updateDraft(childId: number, key: keyof AgentAllocationUpdate, rawValue: string) {
  const parsed = normalizedNonNegative(rawValue)
  drafts[childId] = {
    ...(drafts[childId] || { concurrency: 0, rpm: 0 }),
    [key]: parsed,
  }
}

function normalizedNonNegative(value: unknown): number {
  return Math.max(0, Number.parseInt(String(value ?? '0'), 10) || 0)
}

function exceedsQuota(totalRemaining: number, unlimited: boolean | undefined, requested: number): boolean {
  if (unlimited) {
    return false
  }
  if (requested === 0) {
    return true
  }
  return requested > totalRemaining
}

function exceedsRemainingAllocation(payload: AgentDirectUserCreateRequest | AgentInviteDefaultsUpdate): boolean {
  if (isAdminUnlimitedCapacity.value) {
    return false
  }
  const current = allocation.value
  if (!current) {
    return false
  }
  const requestedConcurrency = 'allocated_concurrency' in payload ? payload.allocated_concurrency : payload.invite_default_concurrency
  const requestedRPM = 'allocated_rpm' in payload ? payload.allocated_rpm : payload.invite_default_rpm
  return exceedsQuota(current.remaining_concurrency, current.unlimited_concurrency, requestedConcurrency) ||
    exceedsQuota(current.remaining_rpm, current.unlimited_rpm, requestedRPM)
}

async function createDirectUser(payload: AgentDirectUserCreateRequest) {
  if (exceedsRemainingAllocation(payload)) {
    appStore.showError(t('agentManagement.direct.insufficientAllocation'))
    return
  }
  creatingUser.value = true
  try {
    await agentManagementAPI.createDirectUser(payload)
    appStore.showSuccess(t('agentManagement.direct.userCreated'))
    showCreateUserModal.value = false
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.createFailed'))
  } finally {
    creatingUser.value = false
  }
}

async function saveInviteDefaults() {
  const payload = {
    invite_default_concurrency: normalizedNonNegative(inviteDefaultsDraft.invite_default_concurrency),
    invite_default_rpm: normalizedNonNegative(inviteDefaultsDraft.invite_default_rpm),
  }
  if (exceedsRemainingAllocation(payload)) {
    appStore.showError(t('agentManagement.direct.insufficientAllocation'))
    return
  }
  savingInviteDefaults.value = true
  try {
    const profile = await agentManagementAPI.updateInviteDefaults(payload)
    syncInviteDefaults({
      invite_default_concurrency: profile.invite_default_concurrency,
      invite_default_rpm: profile.invite_default_rpm,
    })
    appStore.showSuccess(t('agentManagement.direct.inviteDefaultsSaved'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.inviteDefaultsFailed'))
  } finally {
    savingInviteDefaults.value = false
  }
}

async function saveAllocation(child: AgentManagedUser) {
  savingChildId.value = child.id
  try {
    allocation.value = await agentManagementAPI.updateAllocation(child.id, draftFor(child))
    appStore.showSuccess(t('agentManagement.direct.allocationSaved'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.allocationFailed'))
  } finally {
    savingChildId.value = null
  }
}

async function upgrade(child: AgentManagedUser, targetRole: AgentUpgradeTargetRole) {
  savingChildId.value = child.id
  try {
    const payload = { target_role: targetRole }
    if (targetRole === 'agent_level1' || targetRole === 'agent_level2') {
      const quota = draftFor(child)
      await agentManagementAPI.upgradeChild(child.id, {
        ...payload,
        pool_concurrency: quota.concurrency,
        pool_rpm: quota.rpm,
      })
    } else {
      await agentManagementAPI.upgradeChild(child.id, payload)
    }
    appStore.showSuccess(t('agentManagement.direct.upgradeSaved'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.upgradeFailed'))
  } finally {
    savingChildId.value = null
  }
}

function askDetach(child: AgentManagedUser) {
  detachDialog.child = child
  detachDialog.show = true
}

function isTrueDeleteDirectUser(child: AgentManagedUser | null): boolean {
  return isAdmin.value && props.kind === 'users' && child?.role === 'user'
}

function deleteActionLabel(child: AgentManagedUser): string {
  return isTrueDeleteDirectUser(child) ? t('agentManagement.direct.deleteUser') : t('agentManagement.direct.detach')
}

async function confirmDetach() {
  if (!detachDialog.child) return
  const child = detachDialog.child
  detachDialog.show = false
  savingChildId.value = child.id
  try {
    await agentManagementAPI.deleteDirectChild(child.id)
    appStore.showSuccess(isTrueDeleteDirectUser(child) ? t('agentManagement.direct.userDeleted') : t('agentManagement.direct.detached'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.detachFailed'))
  } finally {
    savingChildId.value = null
  }
}

function roleLabel(role: UserRole | string): string {
  return t(`admin.users.roles.${role}`)
}

onMounted(() => {
  authStore.checkAuth()
  loadData()
})
</script>
