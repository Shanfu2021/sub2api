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
            <button
              data-test="import-enterprise-employees"
              class="btn btn-secondary px-3"
              :disabled="loading"
              @click="openImportDialog"
            >
              <Icon name="upload" size="sm" />
              <span>{{ t('enterpriseManagement.employees.import.title') }}</span>
            </button>
            <button
              data-test="initialize-employee-balances"
              class="btn btn-secondary px-3"
              :disabled="loading"
              @click="openBalanceInitDialog"
            >
              <Icon name="dollar" size="sm" />
              <span>{{ t('enterpriseManagement.employees.balanceInit.title') }}</span>
            </button>
            <button
              data-test="open-employee-group-defaults"
              class="btn btn-secondary px-3"
              :disabled="loading"
              @click="openDefaultGroupDialog"
            >
              <Icon name="grid" size="sm" />
              <span>{{ t('enterpriseManagement.employees.defaultGroups.title') }}</span>
            </button>
            <button class="btn btn-secondary px-3" :disabled="loading" @click="loadData">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
        <div class="mt-3 rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-xs text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200">
          <div class="font-medium">{{ t('enterpriseManagement.employees.import.formatTitle') }}</div>
          <pre class="mt-2 overflow-x-auto whitespace-pre rounded bg-white/70 p-3 font-mono text-[11px] leading-5 text-blue-900 dark:bg-dark-900/70 dark:text-blue-100">{{ importFormatExample }}</pre>
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
      :show="importDialog.show"
      :title="t('enterpriseManagement.employees.import.title')"
      width="wide"
      @close="closeImportDialog"
    >
      <div class="space-y-4" data-test="enterprise-employee-import-dialog">
        <div class="rounded-md border border-gray-200 bg-gray-50 px-4 py-3 text-xs text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300">
          <div class="font-medium text-gray-900 dark:text-white">{{ t('enterpriseManagement.employees.import.formatTitle') }}</div>
          <pre class="mt-2 overflow-x-auto whitespace-pre rounded bg-white p-3 font-mono text-[11px] leading-5 dark:bg-dark-900">{{ importFormatExample }}</pre>
          <p class="mt-2">{{ t('enterpriseManagement.employees.import.skipHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('enterpriseManagement.employees.import.file') }}</label>
          <input
            ref="importFileInput"
            data-test="employee-import-file"
            type="file"
            accept="application/json,.json"
            class="input"
            @change="handleImportFileChange"
          />
        </div>

        <div v-if="importDialog.fileName" class="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div class="rounded-md border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.employees.import.file') }}</div>
            <div class="mt-1 truncate text-sm font-medium text-gray-900 dark:text-white">{{ importDialog.fileName }}</div>
          </div>
          <div class="rounded-md border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.employees.import.validRows') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">{{ importDialog.records.length }}</div>
          </div>
          <div class="rounded-md border border-gray-200 p-3 dark:border-dark-700">
            <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.employees.import.requiredQuota') }}</div>
            <div class="mt-1 text-sm font-medium text-gray-900 dark:text-white">
              {{ t('agentManagement.direct.allocatedConcurrency') }} {{ importRequiredConcurrency }} / RPM {{ importRequiredRpm }}
            </div>
          </div>
        </div>

        <div v-if="importDialog.parseError" class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
          {{ importDialog.parseError }}
        </div>

        <div v-if="importDialog.result" class="rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-800 dark:border-green-900/60 dark:bg-green-950/30 dark:text-green-300">
          {{ t('enterpriseManagement.employees.import.result', {
            created: importDialog.result.created_count,
            skipped: importDialog.result.skipped_count
          }) }}
        </div>

        <div v-if="importDialog.result?.skipped?.length" class="max-h-48 overflow-auto rounded-md border border-gray-200 dark:border-dark-700">
          <table class="min-w-full text-left text-xs">
            <thead class="bg-gray-50 text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-3 py-2">{{ t('enterpriseManagement.employees.import.row') }}</th>
                <th class="px-3 py-2">{{ t('common.email') }}</th>
                <th class="px-3 py-2">{{ t('enterpriseManagement.employees.import.reason') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in importDialog.result.skipped" :key="`${item.row}-${item.email || ''}-${item.reason}`">
                <td class="px-3 py-2">{{ item.row }}</td>
                <td class="px-3 py-2">{{ item.email || '-' }}</td>
                <td class="px-3 py-2">{{ item.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="importingEmployees" @click="closeImportDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="importingEmployees || importDialog.records.length === 0" @click="importEmployees">
            {{ importingEmployees ? t('enterpriseManagement.employees.import.importing') : t('enterpriseManagement.employees.import.submit') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="balanceInitDialog.show"
      :title="t('enterpriseManagement.employees.balanceInit.title')"
      width="normal"
      @close="closeBalanceInitDialog"
    >
      <form id="employee-balance-init-form" class="space-y-4" data-test="employee-balance-init-form" @submit.prevent="initializeEmployeeBalances">
        <div>
          <label class="input-label">{{ t('enterpriseManagement.employees.balanceInit.target') }}</label>
          <input
            v-model.number="balanceInitDialog.balance"
            data-test="employee-balance-init-input"
            type="number"
            min="0"
            step="0.000001"
            class="input"
          />
        </div>
        <div class="rounded-md border border-gray-200 bg-gray-50 px-4 py-3 text-sm text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300">
          {{ t('enterpriseManagement.employees.balanceInit.hint') }}
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="initializingBalances" @click="closeBalanceInitDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="employee-balance-init-form" class="btn btn-primary" :disabled="initializingBalances">
            {{ initializingBalances ? t('common.saving') : t('enterpriseManagement.employees.balanceInit.submit') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="defaultGroupDialog.show"
      :title="t('enterpriseManagement.employees.defaultGroups.title')"
      width="wide"
      @close="closeDefaultGroupDialog"
    >
      <div class="space-y-4" data-test="enterprise-employee-default-groups-modal">
        <div class="rounded-md border border-blue-200 bg-blue-50 px-4 py-3 text-xs text-blue-800 dark:border-blue-900/60 dark:bg-blue-950/30 dark:text-blue-200">
          {{ t('enterpriseManagement.employees.defaultGroups.description') }}
        </div>

        <div v-if="defaultGroupDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="defaultGroupDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('enterpriseManagement.employees.emptyGroups') }}
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="groupRate in defaultGroupDialog.groups"
            :key="groupRate.group.id"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <input
                    :data-test="`employee-default-group-assigned-${groupRate.group.id}`"
                    class="checkbox"
                    type="checkbox"
                    :checked="defaultGroupDraftFor(groupRate).assigned"
                    @change="updateDefaultGroupDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                  />
                  <h4 class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ groupRate.group.name }}</h4>
                  <span class="badge badge-gray">{{ sourceLabel(groupRate.source) }}</span>
                </div>
                <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.effectiveRate') }}: {{ groupRate.effective_rate }}
                </div>
              </div>
              <button
                :data-test="`save-employee-default-group-${groupRate.group.id}`"
                class="btn btn-primary btn-sm"
                :disabled="defaultGroupDialog.savingGroupId === groupRate.group.id"
                @click="saveDefaultGroupAssignment(groupRate)"
              >
                <Icon name="check" size="sm" />
                <span>{{ t('common.save') }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
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
import { extractApiErrorCode, extractApiErrorMetadata, extractApiErrorMessage } from '@/utils/apiError'
import { formatCurrency } from '@/utils/format'
import type {
  EnterpriseAllocationSummary,
  EnterpriseEmployee,
  EnterpriseEmployeeAllocationUpdate,
  EnterpriseEmployeeBalanceInitializationResult,
  EnterpriseEmployeeCreateRequest,
  EnterpriseEmployeeGroupDefaultOption,
  EnterpriseEmployeeGroupOption,
  EnterpriseEmployeeImportRecord,
  EnterpriseEmployeeImportResult,
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
const importingEmployees = ref(false)
const initializingBalances = ref(false)
const savingEmployeeId = ref<number | null>(null)
const importFileInput = ref<HTMLInputElement | null>(null)
const searchDraft = ref('')
const activeSearch = ref('')
const employees = ref<EnterpriseEmployee[]>([])
const allocation = ref<EnterpriseAllocationSummary | null>(null)
const drafts = reactive<Record<number, EnterpriseEmployeeAllocationUpdate>>({})
const pagination = reactive({ total: 0, page: 1, page_size: 20, pages: 1 })
const createDialog = reactive({ show: false })
const importDialog = reactive<{
  show: boolean
  fileName: string
  records: EnterpriseEmployeeImportRecord[]
  parseError: string
  result: EnterpriseEmployeeImportResult | null
}>({
  show: false,
  fileName: '',
  records: [],
  parseError: '',
  result: null,
})
const balanceInitDialog = reactive({
  show: false,
  balance: 0,
})
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
const defaultGroupDialog = reactive<{
  show: boolean
  loading: boolean
  savingGroupId: number | null
  groups: EnterpriseEmployeeGroupDefaultOption[]
}>({
  show: false,
  loading: false,
  savingGroupId: null,
  groups: [],
})
const groupDrafts = reactive<Record<number, { assigned: boolean }>>({})
const defaultGroupDrafts = reactive<Record<number, { assigned: boolean }>>({})

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
const importRequiredConcurrency = computed(() => importDialog.records.reduce((sum, item) => sum + (typeof item.concurrency === 'number' && Number.isFinite(item.concurrency) && item.concurrency > 0 ? item.concurrency : 0), 0))
const importRequiredRpm = computed(() => importDialog.records.some((item) => typeof item.rpm === 'number' && item.rpm === 0)
  ? t('common.unlimited')
  : String(importDialog.records.reduce((sum, item) => sum + (typeof item.rpm === 'number' && Number.isFinite(item.rpm) && item.rpm > 0 ? item.rpm : 0), 0)))
const importFormatExample = `[
  {
    "email": "employee@example.com",
    "username": "employee",
    "password": "123456",
    "concurrency": 2,
    "rpm": 30
  }
]`

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

function openImportDialog() {
  importDialog.show = true
  importDialog.fileName = ''
  importDialog.records = []
  importDialog.parseError = ''
  importDialog.result = null
  if (importFileInput.value) {
    importFileInput.value.value = ''
  }
}

function closeImportDialog() {
  if (!importingEmployees.value) {
    importDialog.show = false
  }
}

function openBalanceInitDialog() {
  balanceInitDialog.balance = 0
  balanceInitDialog.show = true
}

function closeBalanceInitDialog() {
  if (!initializingBalances.value) {
    balanceInitDialog.show = false
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

function normalizeImportPayload(raw: unknown): EnterpriseEmployeeImportRecord[] {
  const source = Array.isArray(raw)
    ? raw
    : raw && typeof raw === 'object' && Array.isArray((raw as { employees?: unknown }).employees)
      ? (raw as { employees: unknown[] }).employees
      : null
  if (!source) {
    throw new Error(t('enterpriseManagement.employees.import.invalidJsonShape'))
  }
  return source.map((item) => item !== null && typeof item === 'object' && !Array.isArray(item)
    ? item as EnterpriseEmployeeImportRecord
    : {})
}

async function handleImportFileChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  importDialog.records = []
  importDialog.result = null
  importDialog.parseError = ''
  importDialog.fileName = file?.name || ''
  if (!file) return
  try {
    const text = await file.text()
    const parsed = JSON.parse(text)
    importDialog.records = normalizeImportPayload(parsed)
    if (importDialog.records.length === 0) {
      importDialog.parseError = t('enterpriseManagement.employees.import.emptyFile')
    }
  } catch (error) {
    importDialog.parseError = (error as { message?: string }).message || t('enterpriseManagement.employees.import.invalidJson')
  }
}

function importQuotaErrorMessage(error: unknown): string {
  if (extractApiErrorCode(error) !== 'ENTERPRISE_EMPLOYEE_IMPORT_QUOTA_EXCEEDED') {
    return extractApiErrorMessage(error, t('enterpriseManagement.employees.import.failed'))
  }
  const md = extractApiErrorMetadata(error) || {}
  return t('enterpriseManagement.employees.import.quotaExceeded', {
    currentConcurrency: String(md.available_concurrency ?? '-'),
    requiredConcurrency: String(md.required_concurrency ?? '-'),
    currentRpm: String(md.available_rpm ?? '-'),
    requiredRpm: String(md.required_rpm ?? '-'),
  })
}

async function importEmployees() {
  if (importDialog.records.length === 0) {
    appStore.showError(t('enterpriseManagement.employees.import.emptyFile'))
    return
  }
  importingEmployees.value = true
  try {
    const result = await enterpriseManagementAPI.importEmployees({ employees: importDialog.records })
    importDialog.result = result
    allocation.value = result.allocation
    appStore.showSuccess(t('enterpriseManagement.employees.import.result', {
      created: result.created_count,
      skipped: result.skipped_count,
    }))
    await loadData()
  } catch (error) {
    appStore.showError(importQuotaErrorMessage(error))
  } finally {
    importingEmployees.value = false
  }
}

function balanceInitErrorMessage(error: unknown): string {
  if (extractApiErrorCode(error) !== 'ENTERPRISE_EMPLOYEE_BALANCE_INIT_EXCEEDED') {
    return extractApiErrorMessage(error, t('enterpriseManagement.employees.balanceInit.failed'))
  }
  const md = extractApiErrorMetadata(error) || {}
  return t('enterpriseManagement.employees.balanceInit.exceeded', {
    current: formatCurrency(Number(md.current_balance ?? 0)),
    required: formatCurrency(Number(md.required_balance ?? 0)),
  })
}

function balanceInitSuccessMessage(result: EnterpriseEmployeeBalanceInitializationResult): string {
  return t('enterpriseManagement.employees.balanceInit.success', {
    count: result.employee_count,
    balance: formatCurrency(result.target_balance),
    required: formatCurrency(result.required_balance),
  })
}

async function initializeEmployeeBalances() {
  const balance = normalizedNonNegativeNumber(balanceInitDialog.balance)
  initializingBalances.value = true
  try {
    const result = await enterpriseManagementAPI.initializeEmployeeBalances({ balance })
    appStore.showSuccess(balanceInitSuccessMessage(result))
    balanceInitDialog.show = false
    await loadData()
  } catch (error) {
    appStore.showError(balanceInitErrorMessage(error))
  } finally {
    initializingBalances.value = false
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

function clearDefaultGroupDrafts() {
  for (const key of Object.keys(defaultGroupDrafts)) {
    delete defaultGroupDrafts[Number(key)]
  }
}

function syncDefaultGroupDrafts(groups: EnterpriseEmployeeGroupDefaultOption[]) {
  clearDefaultGroupDrafts()
  for (const item of groups) {
    defaultGroupDrafts[item.group.id] = { assigned: item.assigned }
  }
}

function defaultGroupDraftFor(groupRate: EnterpriseEmployeeGroupDefaultOption) {
  if (!defaultGroupDrafts[groupRate.group.id]) {
    defaultGroupDrafts[groupRate.group.id] = { assigned: groupRate.assigned }
  }
  return defaultGroupDrafts[groupRate.group.id]
}

function updateDefaultGroupDraft(groupID: number, assigned: boolean) {
  defaultGroupDrafts[groupID] = { assigned }
}

async function openDefaultGroupDialog() {
  defaultGroupDialog.show = true
  defaultGroupDialog.loading = true
  defaultGroupDialog.groups = []
  clearDefaultGroupDrafts()
  try {
    const groups = await enterpriseManagementAPI.listEmployeeGroupDefaultOptions()
    defaultGroupDialog.groups = groups.filter((item) => item.group.is_exclusive)
    syncDefaultGroupDrafts(defaultGroupDialog.groups)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.groups.loadFailed'))
  } finally {
    defaultGroupDialog.loading = false
  }
}

function closeDefaultGroupDialog() {
  defaultGroupDialog.show = false
  defaultGroupDialog.groups = []
  defaultGroupDialog.savingGroupId = null
  clearDefaultGroupDrafts()
}

async function saveDefaultGroupAssignment(groupRate: EnterpriseEmployeeGroupDefaultOption) {
  const draft = defaultGroupDraftFor(groupRate)
  defaultGroupDialog.savingGroupId = groupRate.group.id
  try {
    if (draft.assigned) {
      await enterpriseManagementAPI.setEmployeeGroupDefault(groupRate.group.id, { assigned: true })
      appStore.showSuccess(t('enterpriseManagement.employees.defaultGroups.saved'))
    } else {
      await enterpriseManagementAPI.removeEmployeeGroupDefault(groupRate.group.id)
      appStore.showSuccess(t('enterpriseManagement.employees.defaultGroups.removed'))
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.defaultGroups.failed'))
  } finally {
    defaultGroupDialog.savingGroupId = null
  }
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
