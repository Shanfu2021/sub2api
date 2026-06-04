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
              min="1"
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

          <template #cell-agent_income="{ value }">
            <span class="text-sm font-semibold text-emerald-700 dark:text-emerald-300">{{ formatCurrency(Number(value || 0)) }}</span>
          </template>

          <template #cell-allocation="{ row }">
            <div class="grid min-w-[240px] grid-cols-2 gap-2">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedConcurrency') }}</span>
                <input
                  :data-test="`allocation-concurrency-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="1"
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
              <button
                :data-test="`save-allocation-${row.id}`"
                class="btn btn-primary btn-sm"
                :disabled="savingChildId === row.id"
                @click="saveAllocation(row)"
              >
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

              <button
                :data-test="`manage-groups-${row.id}`"
                class="btn btn-secondary btn-sm"
                :disabled="savingChildId === row.id"
                @click="openGroupDialog(row)"
              >
                <Icon name="grid" size="sm" />
                <span>{{ t('agentManagement.groups.manage') }}</span>
              </button>

              <button class="btn btn-secondary btn-sm text-red-600 dark:text-red-400" @click="askDelete(row)">
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
          @update:page="changePage"
          @update:page-size="changePageSize"
        />
      </template>
    </TablePageLayout>

    <ConfirmDialog
      :show="deleteDialog.show"
      :title="deleteDialogTitle"
      :message="deleteDialogMessage"
      :confirm-text="deleteDialogConfirmText"
      danger
      @confirm="confirmDelete"
      @cancel="deleteDialog.show = false"
    />
    <ConfirmDialog
      :show="upgradeDialog.show"
      :title="upgradeDialogTitle"
      :message="upgradeDialogMessage"
      :confirm-text="t('agentManagement.direct.upgrade')"
      @confirm="confirmUpgrade"
      @cancel="closeUpgradeDialog"
    />
    <AgentDirectUserCreateModal
      v-if="canCreateDirectUser"
      :show="showCreateUserModal"
      :loading="creatingUser"
      :default-concurrency="inviteDefaultsDraft.invite_default_concurrency"
      :default-rpm="inviteDefaultsDraft.invite_default_rpm"
      @close="showCreateUserModal = false"
      @submit="createDirectUser"
    />

    <BaseDialog
      :show="groupDialog.show"
      :title="groupDialogTitle"
      width="wide"
      @close="closeGroupDialog"
    >
      <div class="space-y-4" data-test="group-delegation-modal">
        <div v-if="groupDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="groupDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('agentManagement.groups.emptyDelegable') }}
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="groupRate in groupDialog.groups"
            :key="groupRate.group.id"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <input
                    :data-test="`group-assigned-${groupRate.group.id}`"
                    class="checkbox"
                    type="checkbox"
                    :checked="groupDraftFor(groupRate).assigned"
                    @change="updateGroupAssignedDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                  />
                  <h4 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                    {{ groupRate.group.name }}
                  </h4>
                  <span class="badge badge-gray">{{ sourceLabel(groupRate.source) }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.effectiveRate') }}: {{ groupRate.effective_rate }}
                </div>
              </div>

              <div class="grid w-full gap-3 sm:grid-cols-[minmax(140px,1fr)_auto] lg:w-auto lg:grid-cols-[160px_auto] lg:items-end">
                <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ t('agentManagement.groups.childRate') }}</span>
                  <input
                    :data-test="`group-rate-${groupRate.group.id}`"
                    class="input h-9"
                    type="number"
                    min="0.000001"
                    step="0.000001"
                    :disabled="!groupDraftFor(groupRate).assigned"
                    :value="groupDraftFor(groupRate).rate_multiplier"
                    @input="updateGroupRateDraft(groupRate.group.id, ($event.target as HTMLInputElement).value)"
                  />
                </label>

                <label class="flex min-h-9 items-center gap-2 text-xs text-gray-600 dark:text-dark-300">
                  <input
                    :data-test="`group-can-delegate-${groupRate.group.id}`"
                    class="checkbox"
                    type="checkbox"
                    :disabled="!groupDraftFor(groupRate).assigned"
                    :checked="groupDraftFor(groupRate).can_delegate"
                    @change="updateGroupCanDelegateDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ t('agentManagement.groups.allowChildDelegate') }}</span>
                </label>
              </div>
            </div>

            <div class="mt-3 flex flex-wrap justify-end gap-2">
              <button
                :data-test="`save-group-${groupRate.group.id}`"
                class="btn btn-primary btn-sm"
                :disabled="groupDialog.savingGroupId === groupRate.group.id"
                @click="saveGroupDelegation(groupRate)"
              >
                <Icon name="check" size="sm" />
                <span>{{ t('agentManagement.groups.saveDelegation') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>
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
  AgentChildGroupDelegationOption,
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
import BaseDialog from '@/components/common/BaseDialog.vue'
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
const deleteDialog = reactive<{ show: boolean; child: AgentManagedUser | null }>({ show: false, child: null })
const upgradeDialog = reactive<{ show: boolean; child: AgentManagedUser | null; targetRole: AgentUpgradeTargetRole | null }>({
  show: false,
  child: null,
  targetRole: null,
})
const groupDialog = reactive<{
  show: boolean
  child: AgentManagedUser | null
  loading: boolean
  savingGroupId: number | null
  groups: AgentChildGroupDelegationOption[]
}>({
  show: false,
  child: null,
  loading: false,
  savingGroupId: null,
  groups: [],
})
const groupDrafts = reactive<Record<number, { assigned: boolean; rate_multiplier: number; can_delegate: boolean }>>({})

const columns = computed<Column[]>(() => [
  { key: 'email', label: t('common.email') },
  { key: 'role', label: t('agentManagement.direct.role') },
  { key: 'balance', label: t('agentManagement.direct.balance') },
  ...(props.kind === 'agents' ? [{ key: 'agent_income', label: t('agentManagement.direct.agentIncome') }] : []),
  { key: 'allocation', label: t('agentManagement.direct.allocation') },
  { key: 'status', label: t('common.status') },
  { key: 'actions', label: t('common.actions') },
])

const upgradeTargets = computed<AgentUpgradeTargetRole[]>(() => {
  if (props.kind !== 'users') return []
  const role = authStore.user?.role
  if (role === 'admin') return ['agent_level1', 'enterprise']
  if (role === 'agent_level1') return ['enterprise']
  return []
})

const canCreateDirectUser = computed(() => props.kind === 'users')

const isAdmin = computed(() => authStore.user?.role === 'admin')
const isAgent = computed(() => authStore.user?.role === 'agent_level1')
const showInviteDefaultsForm = computed(() => props.kind === 'users' && (isAdmin.value || isAgent.value))
const isAdminUnlimitedCapacity = computed(() => isAdmin.value)

const directSubtitle = computed(() => {
  if (isAdmin.value) {
    return t('agentManagement.direct.adminSubtitle')
  }
  return t('agentManagement.direct.subtitle')
})

const deleteDialogKind = computed(() => deleteKind(deleteDialog.child))
const deleteDialogTitle = computed(() => {
  if (deleteDialogKind.value === 'admin_user') return t('agentManagement.direct.deleteUserTitle')
  if (deleteDialogKind.value === 'admin_agent') return t('agentManagement.direct.deleteAgentTitle')
  if (deleteDialogKind.value === 'admin_enterprise') return t('agentManagement.direct.deleteEnterpriseTitle')
  return t('agentManagement.direct.deleteChildTitle')
})
const deleteDialogMessage = computed(() => {
  const email = deleteDialog.child?.email || ''
  if (deleteDialogKind.value === 'admin_user') return t('agentManagement.direct.deleteUserConfirm', { email })
  if (deleteDialogKind.value === 'admin_agent') return t('agentManagement.direct.deleteAgentConfirm', { email })
  if (deleteDialogKind.value === 'admin_enterprise') return t('agentManagement.direct.deleteEnterpriseConfirm', { email })
  return t('agentManagement.direct.deleteChildConfirm', { email })
})
const deleteDialogConfirmText = computed(() => {
  if (deleteDialogKind.value === 'admin_user') return t('agentManagement.direct.deleteUser')
  if (deleteDialogKind.value === 'admin_agent') return t('agentManagement.direct.deleteAgent')
  if (deleteDialogKind.value === 'admin_enterprise') return t('agentManagement.direct.deleteEnterprise')
  return t('agentManagement.direct.deleteChild')
})
const upgradeDialogTitle = computed(() => t('agentManagement.direct.upgradeTitle'))
const upgradeDialogMessage = computed(() => t('agentManagement.direct.upgradeConfirm', {
  email: upgradeDialog.child?.email || '',
  role: upgradeDialog.targetRole ? roleLabel(upgradeDialog.targetRole) : '',
}))
const remainingConcurrencyText = computed(() => allocation.value?.unlimited_concurrency ? t('common.unlimited') : String(allocation.value?.remaining_concurrency ?? '-'))
const remainingRpmText = computed(() => allocation.value?.unlimited_rpm ? t('common.unlimited') : String(allocation.value?.remaining_rpm ?? '-'))
const groupDialogTitle = computed(() => t('agentManagement.groups.manageTitle', { email: groupDialog.child?.email || '' }))

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
  inviteDefaultsDraft.invite_default_concurrency = normalizedPositiveInt(defaults?.invite_default_concurrency ?? 1)
  inviteDefaultsDraft.invite_default_rpm = normalizedNonNegative(defaults?.invite_default_rpm ?? 1)
}

function quotaFor(child: AgentManagedUser): AgentAllocationUpdate {
  if (props.kind === 'agents' || props.kind === 'enterprises') {
    return {
      concurrency: normalizedPositiveInt(child.pool_concurrency ?? 1),
      rpm: child.pool_rpm || 0,
    }
  }
  return {
    concurrency: normalizedPositiveInt(child.concurrency ?? 1),
    rpm: child.rpm_limit || 0,
  }
}

function currentQuotaFor(child: AgentManagedUser): AgentAllocationUpdate {
  return quotaFor(child)
}

async function listChildren(): Promise<AgentDirectChildrenResponse> {
  const query = {
    ...(activeSearch.value ? { search: activeSearch.value } : {}),
    page: pagination.page,
    page_size: pagination.page_size,
  }
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
  pagination.page = 1
  await loadData()
}

async function changePage(page: number) {
  pagination.page = page
  await loadData()
}

async function changePageSize(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  await loadData()
}

function draftFor(child: AgentManagedUser): AgentAllocationUpdate {
  if (!drafts[child.id]) {
    drafts[child.id] = quotaFor(child)
  }
  return drafts[child.id]
}

function updateDraft(childId: number, key: keyof AgentAllocationUpdate, rawValue: string) {
  const parsed = key === 'concurrency' ? normalizedPositiveInt(rawValue) : normalizedNonNegative(rawValue)
  drafts[childId] = {
    ...(drafts[childId] || { concurrency: 1, rpm: 0 }),
    [key]: parsed,
  }
}

function normalizedPositiveInt(value: unknown): number {
  return Math.max(1, Number.parseInt(String(value ?? '1'), 10) || 1)
}

function normalizedNonNegative(value: unknown): number {
  return Math.max(0, Number.parseInt(String(value ?? '0'), 10) || 0)
}

function normalizedPositiveFloat(value: unknown): number {
  const parsed = Number.parseFloat(String(value ?? '0'))
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0
  }
  return parsed
}

function exceedsQuota(totalRemaining: number, unlimited: boolean | undefined, requested: number): boolean {
  if (unlimited) {
    return false
  }
  return requested >= totalRemaining
}

function exceedsRpmQuota(totalRemaining: number, unlimited: boolean | undefined, requested: number): boolean {
  if (unlimited) {
    return false
  }
  if (requested === 0) {
    return true
  }
  return requested >= totalRemaining
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
    exceedsRpmQuota(current.remaining_rpm, current.unlimited_rpm, requestedRPM)
}

function exceedsRemainingAllocationForExisting(child: AgentManagedUser, payload: AgentAllocationUpdate): boolean {
  if (isAdminUnlimitedCapacity.value) {
    return false
  }
  const current = allocation.value
  if (!current) {
    return false
  }
  const existing = currentQuotaFor(child)
  return exceedsQuota(current.remaining_concurrency + existing.concurrency, current.unlimited_concurrency, payload.concurrency) ||
    exceedsRpmQuota(current.remaining_rpm + (existing.rpm > 0 ? existing.rpm : 0), current.unlimited_rpm, payload.rpm)
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
    invite_default_concurrency: normalizedPositiveInt(inviteDefaultsDraft.invite_default_concurrency),
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
  const draft = draftFor(child)
  if (exceedsRemainingAllocationForExisting(child, draft)) {
    appStore.showError(t('agentManagement.direct.insufficientAllocation'))
    return
  }
  savingChildId.value = child.id
  try {
    allocation.value = await agentManagementAPI.updateAllocation(child.id, draft)
    appStore.showSuccess(t('agentManagement.direct.allocationSaved'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.allocationFailed'))
  } finally {
    savingChildId.value = null
  }
}

function upgrade(child: AgentManagedUser, targetRole: AgentUpgradeTargetRole) {
  upgradeDialog.child = child
  upgradeDialog.targetRole = targetRole
  upgradeDialog.show = true
}

function closeUpgradeDialog() {
  upgradeDialog.show = false
  upgradeDialog.child = null
  upgradeDialog.targetRole = null
}

async function confirmUpgrade() {
  if (!upgradeDialog.child || !upgradeDialog.targetRole) return
  const child = upgradeDialog.child
  const targetRole = upgradeDialog.targetRole
  closeUpgradeDialog()
  savingChildId.value = child.id
  try {
    const payload = { target_role: targetRole }
    if (targetRole === 'agent_level1' || targetRole === 'enterprise') {
      const quota = draftFor(child)
      if (exceedsRemainingAllocationForExisting(child, quota)) {
        appStore.showError(t('agentManagement.direct.insufficientAllocation'))
        return
      }
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

function sourceLabel(source: AgentChildGroupDelegationOption['source']): string {
  return t(`agentManagement.groups.sources.${source}`)
}

function clearGroupDrafts() {
  for (const key of Object.keys(groupDrafts)) {
    delete groupDrafts[Number(key)]
  }
}

function syncGroupDrafts(groups: AgentChildGroupDelegationOption[]) {
  clearGroupDrafts()
  for (const item of groups) {
    if (!item.can_delegate) continue
    groupDrafts[item.group.id] = {
      assigned: item.assigned,
      rate_multiplier: item.assigned ? item.child_rate_multiplier : item.effective_rate,
      can_delegate: item.assigned ? item.child_can_delegate : false,
    }
  }
}

function groupDraftFor(groupRate: AgentChildGroupDelegationOption) {
  if (!groupDrafts[groupRate.group.id]) {
    groupDrafts[groupRate.group.id] = {
      assigned: groupRate.assigned,
      rate_multiplier: groupRate.assigned ? groupRate.child_rate_multiplier : groupRate.effective_rate,
      can_delegate: groupRate.assigned ? groupRate.child_can_delegate : false,
    }
  }
  return groupDrafts[groupRate.group.id]
}

function updateGroupRateDraft(groupID: number, rawValue: string) {
  groupDrafts[groupID] = {
    ...(groupDrafts[groupID] || { assigned: true, rate_multiplier: 0, can_delegate: false }),
    rate_multiplier: normalizedPositiveFloat(rawValue),
  }
}

function updateGroupAssignedDraft(groupID: number, assigned: boolean) {
  groupDrafts[groupID] = {
    ...(groupDrafts[groupID] || { assigned: false, rate_multiplier: 0, can_delegate: false }),
    assigned,
  }
}

function updateGroupCanDelegateDraft(groupID: number, canDelegate: boolean) {
  groupDrafts[groupID] = {
    ...(groupDrafts[groupID] || { assigned: true, rate_multiplier: 0, can_delegate: false }),
    can_delegate: canDelegate,
  }
}

async function openGroupDialog(child: AgentManagedUser) {
  groupDialog.child = child
  groupDialog.show = true
  groupDialog.loading = true
  groupDialog.groups = []
  clearGroupDrafts()
  try {
    const groups = await agentManagementAPI.listChildGroupDelegationOptions(child.id)
    groupDialog.groups = groups.filter((item) => item.can_delegate && item.group.is_exclusive)
    syncGroupDrafts(groupDialog.groups)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    groupDialog.loading = false
  }
}

function closeGroupDialog() {
  groupDialog.show = false
  groupDialog.child = null
  groupDialog.groups = []
  groupDialog.savingGroupId = null
  clearGroupDrafts()
}

async function saveGroupDelegation(groupRate: AgentChildGroupDelegationOption) {
  if (!groupDialog.child) return
  const draft = groupDraftFor(groupRate)
  if (!draft.assigned) {
    await removeGroupDelegation(groupRate)
    return
  }
  if (draft.rate_multiplier <= 0) {
    appStore.showError(t('agentManagement.groups.invalidRate'))
    return
  }
  groupDialog.savingGroupId = groupRate.group.id
  try {
    await agentManagementAPI.setChildGroupDelegation(groupDialog.child.id, groupRate.group.id, {
      rate_multiplier: draft.rate_multiplier,
      can_delegate: draft.can_delegate,
    })
    appStore.showSuccess(t('agentManagement.groups.delegationSaved'))
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.delegationFailed'))
  } finally {
    groupDialog.savingGroupId = null
  }
}

async function removeGroupDelegation(groupRate: AgentChildGroupDelegationOption) {
  if (!groupDialog.child) return
  groupDialog.savingGroupId = groupRate.group.id
  try {
    await agentManagementAPI.removeChildGroupDelegation(groupDialog.child.id, groupRate.group.id)
    appStore.showSuccess(t('agentManagement.groups.delegationRemoved'))
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.removeFailed'))
  } finally {
    groupDialog.savingGroupId = null
  }
}

function askDelete(child: AgentManagedUser) {
  deleteDialog.child = child
  deleteDialog.show = true
}

type DeleteKind = 'admin_user' | 'admin_agent' | 'admin_enterprise' | 'rehome'

function deleteKind(child: AgentManagedUser | null): DeleteKind {
  if (!isAdmin.value || !child) return 'rehome'
  if (child.role === 'user') return 'admin_user'
  if (child.role === 'agent_level1') return 'admin_agent'
  if (child.role === 'enterprise') return 'admin_enterprise'
  return 'rehome'
}

function deleteActionLabel(child: AgentManagedUser): string {
  const kind = deleteKind(child)
  if (kind === 'admin_user') return t('agentManagement.direct.deleteUser')
  if (kind === 'admin_agent') return t('agentManagement.direct.deleteAgent')
  if (kind === 'admin_enterprise') return t('agentManagement.direct.deleteEnterprise')
  return t('agentManagement.direct.deleteChild')
}

async function confirmDelete() {
  if (!deleteDialog.child) return
  const child = deleteDialog.child
  deleteDialog.show = false
  savingChildId.value = child.id
  try {
    await agentManagementAPI.deleteDirectChild(child.id)
    appStore.showSuccess(deleteKind(child) !== 'rehome' ? t('agentManagement.direct.directDeleted') : t('agentManagement.direct.childDeleted'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.deleteChildFailed'))
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
