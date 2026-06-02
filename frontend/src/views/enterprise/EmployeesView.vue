<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('nav.enterpriseEmployees') }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.employees.subtitle') }}</p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <div class="flex items-center gap-2">
              <input
                v-model="searchDraft"
                data-test="enterprise-employee-search"
                class="input h-9 w-48"
                type="search"
                :placeholder="t('common.search')"
                @keyup.enter="applySearch"
              />
              <button
                data-test="enterprise-employee-search-submit"
                class="btn btn-secondary px-3"
                :disabled="loading"
                @click="applySearch"
              >
                <Icon name="search" size="sm" />
              </button>
            </div>
            <span class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('agentManagement.direct.remainingConcurrency') }}: {{ remainingConcurrencyText }}
            </span>
            <span class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('agentManagement.direct.remainingRpm') }}: {{ remainingRpmText }}
            </span>
            <button
              data-test="create-enterprise-employee"
              class="btn btn-primary px-3"
              :disabled="loading"
              @click="openCreateDialog"
            >
              <Icon name="userPlus" size="sm" />
              <span>{{ t('enterpriseManagement.employees.create') }}</span>
            </button>
            <button class="btn btn-secondary px-3" :disabled="loading" @click="loadData">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="employees" :loading="loading">
          <template #cell-username="{ row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.username || row.email }}</span>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ row.email }}</span>
            </div>
          </template>

          <template #cell-balance="{ value }">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ formatCurrency(Number(value || 0)) }}</span>
          </template>

          <template #cell-allocation="{ row }">
            <div class="grid min-w-[280px] grid-cols-1 gap-2 sm:grid-cols-3">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('common.balance') }}</span>
                <input
                  :data-test="`employee-balance-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  step="0.000001"
                  :value="draftFor(row).balance"
                  @input="updateDraft(row.id, 'balance', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedConcurrency') }}</span>
                <input
                  :data-test="`employee-concurrency-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="1"
                  step="1"
                  :value="draftFor(row).concurrency"
                  @input="updateDraft(row.id, 'concurrency', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedRpm') }}</span>
                <input
                  :data-test="`employee-rpm-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  step="1"
                  :value="draftFor(row).rpm"
                  @input="updateDraft(row.id, 'rpm', ($event.target as HTMLInputElement).value)"
                />
              </label>
            </div>
          </template>

          <template #cell-status="{ value }">
            <span :class="value === 'active' ? 'badge badge-green' : 'badge badge-gray'">
              {{ value === 'active' ? t('common.active') : t('common.disabled') }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex flex-wrap items-center gap-2">
              <button
                :data-test="`employee-save-allocation-${row.id}`"
                class="btn btn-primary btn-sm"
                :disabled="savingEmployeeId === row.id"
                @click="saveAllocation(row)"
              >
                <Icon name="check" size="sm" />
                <span>{{ t('agentManagement.direct.saveAllocation') }}</span>
              </button>
              <button class="btn btn-secondary btn-sm" :disabled="savingEmployeeId === row.id" @click="openGroupDialog(row)">
                <Icon name="grid" size="sm" />
                <span>{{ t('agentManagement.groups.manage') }}</span>
              </button>
              <button class="btn btn-secondary btn-sm text-red-600 dark:text-red-400" :disabled="savingEmployeeId === row.id" @click="askDelete(row)">
                <Icon name="trash" size="sm" />
                <span>{{ t('common.delete') }}</span>
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

    <BaseDialog
      :show="createDialog.show"
      :title="t('enterpriseManagement.employees.create')"
      width="normal"
      @close="closeCreateDialog"
    >
      <form id="enterprise-employee-create-form" class="space-y-5" data-test="enterprise-employee-create-form" @submit.prevent="createEmployee">
        <div>
          <label class="input-label">{{ t('admin.users.email') }}</label>
          <input v-model="createForm.email" data-test="employee-create-email" type="email" required class="input" :placeholder="t('admin.users.enterEmail')" />
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.password') }}</label>
          <div class="flex gap-2">
            <input
              v-model="createForm.password"
              data-test="employee-create-password"
              type="text"
              required
              minlength="6"
              class="input"
              :placeholder="t('admin.users.enterPassword')"
            />
            <button type="button" class="btn btn-secondary px-3" @click="generateRandomPassword">
              <Icon name="refresh" size="md" />
            </button>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.username') }}</label>
          <input v-model="createForm.username" data-test="employee-create-username" type="text" class="input" :placeholder="t('admin.users.enterUsername')" />
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
          <div>
            <label class="input-label">{{ t('common.balance') }}</label>
            <input v-model.number="createForm.balance" data-test="employee-create-balance" type="number" min="0" step="0.000001" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('agentManagement.direct.allocatedConcurrency') }}</label>
            <input v-model.number="createForm.concurrency" data-test="employee-create-concurrency" type="number" min="1" step="1" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('agentManagement.direct.allocatedRpm') }}</label>
            <input v-model.number="createForm.rpm" data-test="employee-create-rpm" type="number" min="0" step="1" class="input" />
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="creatingEmployee" @click="closeCreateDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="enterprise-employee-create-form" class="btn btn-primary" :disabled="creatingEmployee">
            {{ creatingEmployee ? t('admin.users.creating') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="groupDialog.show"
      :title="groupDialogTitle"
      width="wide"
      @close="closeGroupDialog"
    >
      <div class="space-y-4" data-test="enterprise-employee-groups-modal">
        <div v-if="groupDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="groupDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('enterpriseManagement.employees.emptyGroups') }}
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="groupRate in groupDialog.groups"
            :key="groupRate.group.id"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <input
                    :data-test="`employee-group-assigned-${groupRate.group.id}`"
                    class="checkbox"
                    type="checkbox"
                    :checked="groupDraftFor(groupRate).assigned"
                    @change="updateGroupDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                  />
                  <h4 class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ groupRate.group.name }}</h4>
                  <span class="badge badge-gray">{{ sourceLabel(groupRate.source) }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.effectiveRate') }}: {{ groupRate.effective_rate }}
                </div>
              </div>
              <button
                :data-test="`save-employee-group-${groupRate.group.id}`"
                class="btn btn-primary btn-sm"
                :disabled="groupDialog.savingGroupId === groupRate.group.id"
                @click="saveGroupAssignment(groupRate)"
              >
                <Icon name="check" size="sm" />
                <span>{{ t('common.save') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="deleteDialog.show"
      :title="t('enterpriseManagement.employees.deleteTitle')"
      :message="t('enterpriseManagement.employees.deleteConfirm', { email: deleteDialog.employee?.email || '' })"
      :confirm-text="t('common.delete')"
      danger
      @confirm="confirmDelete"
      @cancel="deleteDialog.show = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { enterpriseManagementAPI } from '@/api/enterpriseManagement'
import { useAppStore } from '@/stores'
import { formatCurrency } from '@/utils/format'
import type {
  EnterpriseAllocationSummary,
  EnterpriseEmployee,
  EnterpriseEmployeeAllocationUpdate,
  EnterpriseEmployeeCreateRequest,
  EnterpriseEmployeeGroupOption,
  EnterpriseEmployeesResponse,
} from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const creatingEmployee = ref(false)
const savingEmployeeId = ref<number | null>(null)
const searchDraft = ref('')
const activeSearch = ref('')
const employees = ref<EnterpriseEmployee[]>([])
const allocation = ref<EnterpriseAllocationSummary | null>(null)
const drafts = reactive<Record<number, EnterpriseEmployeeAllocationUpdate>>({})
const pagination = reactive({ total: 0, page: 1, page_size: 20, pages: 1 })
const createDialog = reactive({ show: false })
const createForm = reactive<EnterpriseEmployeeCreateRequest>({
  email: '',
  password: '',
  username: '',
  balance: 0,
  concurrency: 1,
  rpm: 1,
})
const deleteDialog = reactive<{ show: boolean; employee: EnterpriseEmployee | null }>({
  show: false,
  employee: null,
})
const groupDialog = reactive<{
  show: boolean
  employee: EnterpriseEmployee | null
  loading: boolean
  savingGroupId: number | null
  groups: EnterpriseEmployeeGroupOption[]
}>({
  show: false,
  employee: null,
  loading: false,
  savingGroupId: null,
  groups: [],
})
const groupDrafts = reactive<Record<number, { assigned: boolean }>>({})

const columns = computed<Column[]>(() => [
  { key: 'username', label: t('common.email') },
  { key: 'balance', label: t('common.balance') },
  { key: 'allocation', label: t('agentManagement.direct.allocation') },
  { key: 'status', label: t('common.status') },
  { key: 'actions', label: t('common.actions') },
])

const remainingConcurrencyText = computed(() => allocation.value?.unlimited_concurrency ? t('common.unlimited') : String(allocation.value?.remaining_concurrency ?? '-'))
const remainingRpmText = computed(() => allocation.value?.unlimited_rpm ? t('common.unlimited') : String(allocation.value?.remaining_rpm ?? '-'))
const groupDialogTitle = computed(() => t('enterpriseManagement.employees.groupsTitle', { email: groupDialog.employee?.email || '' }))

function extractPagination(result: EnterpriseEmployeesResponse) {
  const source = result.pagination || {}
  pagination.total = Number(source.total ?? source.Total ?? result.items.length)
  pagination.page = Number(source.page ?? source.Page ?? 1)
  pagination.page_size = Number(source.page_size ?? source.PageSize ?? 20)
  pagination.pages = Number(source.pages ?? source.Pages ?? 1)
}

function quotaFor(employee: EnterpriseEmployee): EnterpriseEmployeeAllocationUpdate {
  return {
    balance: Number(employee.balance || 0),
    concurrency: normalizedPositiveInt(employee.concurrency || employee.allocated_concurrency || 1),
    rpm: Number(employee.rpm_limit || employee.allocated_rpm || 0),
  }
}

function currentQuotaFor(employee: EnterpriseEmployee): Pick<EnterpriseEmployeeAllocationUpdate, 'concurrency' | 'rpm'> {
  const quota = quotaFor(employee)
  return {
    concurrency: quota.concurrency,
    rpm: quota.rpm,
  }
}

function syncDrafts(items: EnterpriseEmployee[]) {
  for (const employee of items) {
    drafts[employee.id] = quotaFor(employee)
  }
}

async function loadData() {
  loading.value = true
  try {
    const query = {
      ...(activeSearch.value ? { search: activeSearch.value } : {}),
      page: pagination.page,
      page_size: pagination.page_size,
    }
    const [summary, result] = await Promise.all([
      enterpriseManagementAPI.getSummary(),
      enterpriseManagementAPI.listEmployees(query),
    ])
    allocation.value = summary.allocation
    employees.value = result.items
    syncDrafts(result.items)
    extractPagination(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.loadFailed'))
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

function draftFor(employee: EnterpriseEmployee): EnterpriseEmployeeAllocationUpdate {
  if (!drafts[employee.id]) {
    drafts[employee.id] = quotaFor(employee)
  }
  return drafts[employee.id]
}

function normalizedNonNegativeInt(value: unknown): number {
  return Math.max(0, Number.parseInt(String(value ?? '0'), 10) || 0)
}

function normalizedPositiveInt(value: unknown): number {
  return Math.max(1, Number.parseInt(String(value ?? '1'), 10) || 1)
}

function normalizedNonNegativeNumber(value: unknown): number {
  const parsed = Number.parseFloat(String(value ?? '0'))
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0
  }
  return parsed
}

function updateDraft(employeeId: number, key: keyof EnterpriseEmployeeAllocationUpdate, rawValue: string) {
  const current = drafts[employeeId] || { balance: 0, concurrency: 1, rpm: 0 }
  drafts[employeeId] = {
    ...current,
    [key]: key === 'balance' ? normalizedNonNegativeNumber(rawValue) : key === 'concurrency' ? normalizedPositiveInt(rawValue) : normalizedNonNegativeInt(rawValue),
  }
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

function exceedsRemainingAllocation(payload: Pick<EnterpriseEmployeeAllocationUpdate, 'concurrency' | 'rpm'>): boolean {
  const current = allocation.value
  if (!current) {
    return false
  }
  return exceedsQuota(current.remaining_concurrency, current.unlimited_concurrency, payload.concurrency) ||
    exceedsRpmQuota(current.remaining_rpm, current.unlimited_rpm, payload.rpm)
}

function exceedsRemainingAllocationForExisting(employee: EnterpriseEmployee, payload: Pick<EnterpriseEmployeeAllocationUpdate, 'concurrency' | 'rpm'>): boolean {
  const current = allocation.value
  if (!current) {
    return false
  }
  const existing = currentQuotaFor(employee)
  return exceedsQuota(current.remaining_concurrency + existing.concurrency, current.unlimited_concurrency, payload.concurrency) ||
    exceedsRpmQuota(current.remaining_rpm + (existing.rpm > 0 ? existing.rpm : 0), current.unlimited_rpm, payload.rpm)
}

function resetCreateForm() {
  Object.assign(createForm, {
    email: '',
    password: '',
    username: '',
    balance: 0,
    concurrency: 1,
    rpm: 1,
  })
}

function openCreateDialog() {
  resetCreateForm()
  createDialog.show = true
}

function closeCreateDialog() {
  if (!creatingEmployee.value) {
    createDialog.show = false
  }
}

async function createEmployee() {
  const payload: EnterpriseEmployeeCreateRequest = {
    email: createForm.email.trim(),
    password: createForm.password,
    username: String(createForm.username ?? '').trim(),
    balance: normalizedNonNegativeNumber(createForm.balance),
    concurrency: normalizedPositiveInt(createForm.concurrency),
    rpm: normalizedNonNegativeInt(createForm.rpm),
  }
  if (exceedsRemainingAllocation(payload)) {
    appStore.showError(t('enterpriseManagement.employees.insufficientAllocation'))
    return
  }
  creatingEmployee.value = true
  try {
    await enterpriseManagementAPI.createEmployee(payload)
    appStore.showSuccess(t('enterpriseManagement.employees.created'))
    createDialog.show = false
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.createFailed'))
  } finally {
    creatingEmployee.value = false
  }
}

async function saveAllocation(employee: EnterpriseEmployee) {
  const draft = draftFor(employee)
  const payload = {
    balance: normalizedNonNegativeNumber(draft.balance),
    concurrency: normalizedPositiveInt(draft.concurrency),
    rpm: normalizedNonNegativeInt(draft.rpm),
  }
  if (exceedsRemainingAllocationForExisting(employee, payload)) {
    appStore.showError(t('enterpriseManagement.employees.insufficientAllocation'))
    return
  }
  savingEmployeeId.value = employee.id
  try {
    allocation.value = await enterpriseManagementAPI.updateEmployeeAllocation(employee.id, payload)
    appStore.showSuccess(t('agentManagement.direct.allocationSaved'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.allocationFailed'))
  } finally {
    savingEmployeeId.value = null
  }
}

function askDelete(employee: EnterpriseEmployee) {
  deleteDialog.employee = employee
  deleteDialog.show = true
}

async function confirmDelete() {
  if (!deleteDialog.employee) return
  const employee = deleteDialog.employee
  deleteDialog.show = false
  savingEmployeeId.value = employee.id
  try {
    await enterpriseManagementAPI.deleteEmployee(employee.id)
    appStore.showSuccess(t('enterpriseManagement.employees.deleted'))
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.deleteFailed'))
  } finally {
    savingEmployeeId.value = null
    deleteDialog.employee = null
  }
}

function sourceLabel(source: EnterpriseEmployeeGroupOption['source']): string {
  return t(`agentManagement.groups.sources.${source}`)
}

function clearGroupDrafts() {
  for (const key of Object.keys(groupDrafts)) {
    delete groupDrafts[Number(key)]
  }
}

function syncGroupDrafts(groups: EnterpriseEmployeeGroupOption[]) {
  clearGroupDrafts()
  for (const item of groups) {
    groupDrafts[item.group.id] = { assigned: item.assigned }
  }
}

function groupDraftFor(groupRate: EnterpriseEmployeeGroupOption) {
  if (!groupDrafts[groupRate.group.id]) {
    groupDrafts[groupRate.group.id] = { assigned: groupRate.assigned }
  }
  return groupDrafts[groupRate.group.id]
}

function updateGroupDraft(groupID: number, assigned: boolean) {
  groupDrafts[groupID] = { assigned }
}

async function openGroupDialog(employee: EnterpriseEmployee) {
  groupDialog.employee = employee
  groupDialog.show = true
  groupDialog.loading = true
  groupDialog.groups = []
  clearGroupDrafts()
  try {
    const groups = await enterpriseManagementAPI.listEmployeeGroupOptions(employee.id)
    groupDialog.groups = groups.filter((item) => item.group.is_exclusive)
    syncGroupDrafts(groupDialog.groups)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.groups.loadFailed'))
  } finally {
    groupDialog.loading = false
  }
}

function closeGroupDialog() {
  groupDialog.show = false
  groupDialog.employee = null
  groupDialog.groups = []
  groupDialog.savingGroupId = null
  clearGroupDrafts()
}

async function saveGroupAssignment(groupRate: EnterpriseEmployeeGroupOption) {
  if (!groupDialog.employee) return
  const draft = groupDraftFor(groupRate)
  groupDialog.savingGroupId = groupRate.group.id
  try {
    if (draft.assigned) {
      await enterpriseManagementAPI.setEmployeeGroup(groupDialog.employee.id, groupRate.group.id, { assigned: true })
      appStore.showSuccess(t('enterpriseManagement.employees.groupSaved'))
    } else {
      await enterpriseManagementAPI.removeEmployeeGroup(groupDialog.employee.id, groupRate.group.id)
      appStore.showSuccess(t('enterpriseManagement.employees.groupRemoved'))
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.groupFailed'))
  } finally {
    groupDialog.savingGroupId = null
  }
}

function generateRandomPassword() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let password = ''
  for (let i = 0; i < 16; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  createForm.password = password
}

onMounted(loadData)
</script>
