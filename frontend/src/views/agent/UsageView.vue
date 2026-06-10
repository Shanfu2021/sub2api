<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('agentManagement.usage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('agentManagement.usage.subtitle') }}</p>
        </div>
      </div>

      <UsageStatsCards :stats="usageStats" />

      <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-wrap items-center gap-4">
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('agentManagement.usage.timeRange') }}:</span>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="onDateRangeChange"
            />
          </div>
        </div>
      </section>

      <UsageFilters
        v-model="filters"
        :start-date="startDate"
        :end-date="endDate"
        :exporting="exporting"
        :model-options="modelNameOptions"
        :show-cleanup="false"
        :show-api-key-filter="false"
        :show-account-filter="false"
        :search-users-fn="agentManagementAPI.searchUsageUsers"
        :search-api-keys-fn="agentManagementAPI.searchUsageApiKeys"
        :search-accounts-fn="agentManagementAPI.searchUsageAccounts"
        :load-groups-fn="loadAgentGroupOptions"
        @change="applyFilters"
        @refresh="refreshData"
        @reset="resetFilters"
        @export="exportToExcel"
      />

      <UsageTable
        :data="usageLogs"
        :loading="loading"
        :columns="visibleColumns"
        :server-side-sort="true"
        :default-sort-key="'created_at'"
        :default-sort-order="'desc'"
        :user-clickable="false"
        @sort="handleSort"
      />

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </AppLayout>
  <UsageExportProgress
    :show="exportProgress.show"
    :progress="exportProgress.progress"
    :current="exportProgress.current"
    :total="exportProgress.total"
    :estimated-time="exportProgress.estimatedTime"
    @cancel="cancelExport"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { saveAs } from 'file-saver'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatReasoningEffort } from '@/utils/format'
import { resolveUsageRequestType, requestTypeToLegacyStream } from '@/utils/usageRequestType'
import type { AdminUsageLog, SelectOption } from '@/types'
import type { AdminUsageQueryParams, AdminUsageStatsResponse } from '@/api/admin/usage'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import UsageStatsCards from '@/components/admin/usage/UsageStatsCards.vue'
import UsageFilters from '@/components/admin/usage/UsageFilters.vue'
import UsageTable from '@/components/admin/usage/UsageTable.vue'
import UsageExportProgress from '@/components/admin/usage/UsageExportProgress.vue'

const { t } = useI18n()
const appStore = useAppStore()

const usageStats = ref<AdminUsageStatsResponse | null>(null)
const usageLogs = ref<AdminUsageLog[]>([])
const loading = ref(false)
const exporting = ref(false)
let abortController: AbortController | null = null
let exportAbortController: AbortController | null = null

const exportProgress = reactive({ show: false, progress: 0, current: 0, total: 0, estimatedTime: '' })

const formatLocalDate = (date: Date) => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const defaultEnd = new Date()
const defaultStart = new Date(defaultEnd.getTime() - 24 * 60 * 60 * 1000)
const startDate = ref(formatLocalDate(defaultStart))
const endDate = ref(formatLocalDate(defaultEnd))

const filters = ref<AdminUsageQueryParams>({
  start_date: startDate.value,
  end_date: endDate.value,
  request_type: undefined,
  billing_type: null,
  billing_mode: undefined,
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc',
})

const visibleColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.usage.user'), sortable: false },
  { key: 'api_key', label: t('usage.apiKeyFilter'), sortable: false },
  { key: 'account', label: t('admin.usage.account'), sortable: false },
  { key: 'model', label: t('usage.model'), sortable: true },
  { key: 'reasoning_effort', label: t('usage.reasoningEffort'), sortable: false },
  { key: 'endpoint', label: t('usage.endpoint'), sortable: false },
  { key: 'group', label: t('admin.usage.group'), sortable: false },
  { key: 'stream', label: t('usage.type'), sortable: false },
  { key: 'billing_mode', label: t('admin.usage.billingMode'), sortable: false },
  { key: 'tokens', label: t('usage.tokens'), sortable: false },
  { key: 'cost', label: t('usage.cost'), sortable: false },
  { key: 'first_token', label: t('usage.firstToken'), sortable: false },
  { key: 'duration', label: t('usage.duration'), sortable: false },
  { key: 'created_at', label: t('usage.time'), sortable: true },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false },
])

const modelNameOptions = computed(() =>
  Array.from(new Set(usageLogs.value.map((item) => item.model).filter(Boolean))).sort()
)

function buildUsageListParams(page: number, pageSize: number, exactTotal: boolean): AdminUsageQueryParams {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  return {
    page,
    page_size: pageSize,
    exact_total: exactTotal,
    ...filters.value,
    start_date: startDate.value,
    end_date: endDate.value,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    stream: legacyStream === null ? undefined : legacyStream,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order,
  }
}

function buildStatsParams(force = false): AdminUsageQueryParams {
  const requestType = filters.value.request_type
  const legacyStream = requestType ? requestTypeToLegacyStream(requestType) : filters.value.stream
  return {
    ...filters.value,
    start_date: startDate.value,
    end_date: endDate.value,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    stream: legacyStream === null ? undefined : legacyStream,
    ...(force ? { nocache: 1 } : {}),
  }
}

async function loadAgentGroupOptions(): Promise<SelectOption[]> {
  const groups = await agentManagementAPI.listGroups()
  return groups.map((g) => ({ value: g.group.id, label: g.group.name || `#${g.group.id}` }))
}

async function loadLogs() {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  try {
    const res = await agentManagementAPI.listUsage(
      buildUsageListParams(pagination.page, pagination.page_size, false),
      { signal: controller.signal }
    )
    if (!controller.signal.aborted) {
      usageLogs.value = (res.items || []) as AdminUsageLog[]
      pagination.total = res.total || 0
    }
  } catch (error: any) {
    if (error?.name !== 'AbortError') {
      appStore.showError(error?.message || t('agentManagement.usage.loadFailed'))
    }
  } finally {
    if (abortController === controller) loading.value = false
  }
}

async function loadStats(force = false) {
  try {
    usageStats.value = await agentManagementAPI.getUsageStats(buildStatsParams(force))
  } catch (error: any) {
    appStore.showError(error?.message || t('agentManagement.usage.loadFailed'))
  }
}

function applyFilters() {
  pagination.page = 1
  void loadLogs()
  void loadStats()
}

function refreshData() {
  void loadLogs()
  void loadStats(true)
}

function resetFilters() {
  const end = new Date()
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
  startDate.value = formatLocalDate(start)
  endDate.value = formatLocalDate(end)
  filters.value = {
    start_date: startDate.value,
    end_date: endDate.value,
    request_type: undefined,
    billing_type: null,
    billing_mode: undefined,
  }
  applyFilters()
}

function onDateRangeChange(range: { startDate: string; endDate: string }) {
  startDate.value = range.startDate
  endDate.value = range.endDate
  filters.value = {
    ...filters.value,
    start_date: range.startDate,
    end_date: range.endDate,
  }
  applyFilters()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadLogs()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadLogs()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadLogs()
}

function getRequestTypeLabel(log: AdminUsageLog): string {
  const requestType = resolveUsageRequestType(log)
  if (requestType === 'ws_v2') return t('usage.ws')
  if (requestType === 'stream') return t('usage.stream')
  if (requestType === 'sync') return t('usage.sync')
  return t('usage.unknown')
}

function accountBilled(log: AdminUsageLog): number {
  const base = log.account_stats_cost != null ? log.account_stats_cost : (log.total_cost ?? 0)
  const result = base * (log.account_rate_multiplier ?? 1)
  return Number.isFinite(result) ? result : 0
}

function cancelExport() {
  exportAbortController?.abort()
}

async function exportToExcel() {
  if (exporting.value) return
  exporting.value = true
  exportProgress.show = true
  const controller = new AbortController()
  exportAbortController = controller
  try {
    let page = 1
    let total = pagination.total
    let exportedCount = 0
    const XLSX = await import('xlsx')
    const headers = [
      t('usage.time'), t('admin.usage.user'), t('usage.apiKeyFilter'),
      t('admin.usage.account'), t('usage.model'), t('usage.reasoningEffort'), t('admin.usage.group'),
      t('usage.inboundEndpoint'), t('usage.upstreamEndpoint'), t('usage.type'),
      t('admin.usage.inputTokens'), t('admin.usage.outputTokens'),
      t('admin.usage.cacheReadTokens'), t('admin.usage.cacheCreationTokens'),
      t('admin.usage.inputCost'), t('admin.usage.outputCost'),
      t('admin.usage.cacheReadCost'), t('admin.usage.cacheCreationCost'),
      t('usage.rate'), t('usage.original'), t('usage.userBilled'), t('usage.accountBilled'),
      t('usage.firstToken'), t('usage.duration'), t('admin.usage.requestId'), t('usage.userAgent')
    ]
    const ws = XLSX.utils.aoa_to_sheet([headers])
    while (true) {
      const res = await agentManagementAPI.listUsage(
        buildUsageListParams(page, 100, true),
        { signal: controller.signal }
      )
      if (controller.signal.aborted) break
      if (page === 1) {
        total = res.total || 0
        exportProgress.total = total
      }
      const rows = ((res.items || []) as AdminUsageLog[]).map((log) => [
        log.created_at,
        log.user?.email || '',
        log.api_key?.name || '',
        log.account?.name || '',
        log.model || '',
        formatReasoningEffort(log.reasoning_effort),
        log.group?.name || '',
        log.inbound_endpoint || '',
        log.upstream_endpoint || '',
        getRequestTypeLabel(log),
        log.input_tokens,
        log.output_tokens,
        log.cache_read_tokens,
        log.cache_creation_tokens,
        log.input_cost?.toFixed(6) || '0.000000',
        log.output_cost?.toFixed(6) || '0.000000',
        log.cache_read_cost?.toFixed(6) || '0.000000',
        log.cache_creation_cost?.toFixed(6) || '0.000000',
        log.rate_multiplier?.toPrecision(4) || '1.00',
        log.total_cost?.toFixed(6) || '0.000000',
        log.actual_cost?.toFixed(6) || '0.000000',
        accountBilled(log).toFixed(6),
        log.first_token_ms ?? '',
        log.duration_ms,
        log.request_id || '',
        log.user_agent || '',
      ])
      if (rows.length) XLSX.utils.sheet_add_aoa(ws, rows, { origin: -1 })
      exportedCount += rows.length
      exportProgress.current = exportedCount
      exportProgress.progress = total > 0 ? Math.min(100, Math.round(exportedCount / total * 100)) : 0
      if (exportedCount >= total || rows.length < 100) break
      page++
    }
    if (!controller.signal.aborted) {
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, 'Usage')
      const data = XLSX.write(wb, { bookType: 'xlsx', type: 'array' })
      saveAs(new Blob([data], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }), `agent_usage_${startDate.value}_to_${endDate.value}.xlsx`)
      appStore.showSuccess(t('usage.exportSuccess'))
    }
  } catch (error: any) {
    if (error?.name !== 'AbortError') {
      appStore.showError(error?.message || 'Export Failed')
    }
  } finally {
    if (exportAbortController === controller) {
      exportAbortController = null
      exporting.value = false
      exportProgress.show = false
    }
  }
}

onMounted(() => {
  void loadLogs()
  void loadStats()
})

onUnmounted(() => {
  abortController?.abort()
  exportAbortController?.abort()
})
</script>
