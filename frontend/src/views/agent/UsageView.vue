<template>
  <AppLayout>
    <div class="space-y-5">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('agentManagement.usage.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('agentManagement.usage.subtitle') }}</p>
        </div>
        <button class="btn btn-secondary px-3" :disabled="loading" @click="refreshData">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span class="ml-1.5">{{ t('common.refresh') }}</span>
        </button>
      </div>

      <UsageStatsCards :stats="usageStats" />

      <section class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          <label class="block">
            <span class="input-label">{{ t('agentManagement.usage.timeRange') }}</span>
            <DateRangePicker
              v-model:start-date="startDate"
              v-model:end-date="endDate"
              @change="applyFilters"
            />
          </label>

          <label class="block">
            <span class="input-label">{{ t('agentManagement.usage.userFilter') }}</span>
            <select v-model.number="filters.user_id" class="input w-full" @change="applyFilters">
              <option :value="0">{{ t('agentManagement.usage.allUsers') }}</option>
              <option v-for="user in usageUsers" :key="user.id" :value="user.id">
                {{ user.email }} · {{ roleLabel(user.role) }}
              </option>
            </select>
          </label>

          <label class="block">
            <span class="input-label">{{ t('usage.model') }}</span>
            <input
              v-model.trim="filters.model"
              class="input w-full"
              :placeholder="t('agentManagement.usage.modelPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </label>

          <label class="block">
            <span class="input-label">{{ t('admin.usage.group') }}</span>
            <input
              v-model.number="filters.group_id"
              class="input w-full"
              min="1"
              type="number"
              :placeholder="t('agentManagement.usage.groupPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </label>

          <label class="block">
            <span class="input-label">{{ t('usage.type') }}</span>
            <select v-model="filters.request_type" class="input w-full" @change="applyFilters">
              <option value="">{{ t('admin.usage.allTypes') }}</option>
              <option value="sync">{{ t('usage.sync') }}</option>
              <option value="stream">{{ t('usage.stream') }}</option>
              <option value="ws_v2">{{ t('usage.ws') }}</option>
            </select>
          </label>
        </div>

        <div class="mt-4 flex flex-wrap justify-end gap-2">
          <button class="btn btn-secondary" @click="resetFilters">{{ t('common.reset') }}</button>
          <button class="btn btn-primary" :disabled="loading" @click="applyFilters">{{ t('common.search') }}</button>
        </div>
      </section>

      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs font-medium uppercase tracking-wide text-gray-500 dark:bg-dark-900/50 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3">{{ t('admin.usage.user') }}</th>
                <th class="px-4 py-3">{{ t('usage.model') }}</th>
                <th class="px-4 py-3">{{ t('admin.usage.group') }}</th>
                <th class="px-4 py-3">{{ t('usage.type') }}</th>
                <th class="px-4 py-3">{{ t('usage.tokens') }}</th>
                <th class="px-4 py-3">{{ t('usage.cost') }}</th>
                <th class="px-4 py-3">{{ t('usage.duration') }}</th>
                <th class="px-4 py-3">{{ t('usage.time') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-if="loading">
                <td colspan="8" class="px-4 py-10 text-center text-gray-500 dark:text-dark-400">
                  {{ t('common.loading') }}
                </td>
              </tr>
              <tr v-else-if="usageLogs.length === 0">
                <td colspan="8" class="px-4 py-10 text-center text-gray-500 dark:text-dark-400">
                  {{ t('usage.noRecords') }}
                </td>
              </tr>
              <template v-else>
                <tr v-for="row in usageLogs" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/40">
                  <td class="px-4 py-3">
                    <div class="font-medium text-gray-900 dark:text-white">{{ row.user?.email || userLabel(row.user_id) }}</div>
                    <div class="text-xs text-gray-500 dark:text-dark-400">#{{ row.user_id }}</div>
                  </td>
                  <td class="max-w-[220px] break-all px-4 py-3 font-medium text-gray-900 dark:text-white">
                    {{ row.model || '-' }}
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-dark-200">
                    <span v-if="row.group" class="inline-flex rounded bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-200">
                      {{ row.group.name }}
                    </span>
                    <span v-else>{{ row.group_id || '-' }}</span>
                  </td>
                  <td class="px-4 py-3">
                    <span class="inline-flex rounded bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                      {{ requestTypeLabel(row) }}
                    </span>
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-dark-200">
                    <div>{{ formatTokens(row.input_tokens + row.output_tokens + row.cache_read_tokens + row.cache_creation_tokens) }}</div>
                    <div class="text-xs text-gray-500 dark:text-dark-400">
                      {{ t('usage.in') }} {{ formatTokens(row.input_tokens) }} / {{ t('usage.out') }} {{ formatTokens(row.output_tokens) }}
                    </div>
                  </td>
                  <td class="px-4 py-3 font-medium text-emerald-700 dark:text-emerald-300">
                    {{ formatCurrency(row.actual_cost || 0) }}
                  </td>
                  <td class="px-4 py-3 text-gray-700 dark:text-dark-200">{{ formatDuration(row.duration_ms || 0) }}</td>
                  <td class="whitespace-nowrap px-4 py-3 text-gray-700 dark:text-dark-200">{{ formatDateTime(row.created_at) }}</td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>
      </section>

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
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatCurrency, formatDateTime } from '@/utils/format'
import type { AdminUsageQueryParams, AdminUsageStatsResponse } from '@/api/admin/usage'
import type { AgentManagedUser, AgentUsageLog, UsageRequestType, UserRole } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import UsageStatsCards from '@/components/admin/usage/UsageStatsCards.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const usageLogs = ref<AgentUsageLog[]>([])
const usageUsers = ref<AgentManagedUser[]>([])
const usageStats = ref<AdminUsageStatsResponse | null>(null)
const loading = ref(false)

const today = new Date()
const yesterday = new Date(today.getTime() - 24 * 60 * 60 * 1000)
const startDate = ref(formatDateInput(yesterday))
const endDate = ref(formatDateInput(today))

const filters = reactive<{
  user_id: number
  model: string
  group_id: number | ''
  request_type: UsageRequestType | ''
}>({
  user_id: 0,
  model: '',
  group_id: '',
  request_type: '',
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(20),
  total: 0,
  pages: 1,
})

const userLabels = computed(() => {
  const labels = new Map<number, string>()
  for (const user of usageUsers.value) {
    labels.set(user.id, user.email)
  }
  return labels
})

function formatDateInput(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function roleLabel(role: UserRole | string): string {
  return t(`admin.users.roles.${role}`)
}

function userLabel(userId: number): string {
  return userLabels.value.get(userId) || `#${userId}`
}

function requestTypeLabel(row: AgentUsageLog): string {
  const requestType = row.request_type || (row.stream ? 'stream' : 'sync')
  if (requestType === 'ws_v2') return t('usage.ws')
  if (requestType === 'stream') return t('usage.stream')
  if (requestType === 'sync') return t('usage.sync')
  return t('usage.unknown')
}

function formatDuration(ms: number): string {
  return ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`
}

function formatTokens(value: number): string {
  if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`
  if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`
  if (value >= 1e3) return `${(value / 1e3).toFixed(2)}K`
  return value.toLocaleString()
}

function buildParams(includePagination = true): AdminUsageQueryParams {
  const params: AdminUsageQueryParams = {
    start_date: startDate.value,
    end_date: endDate.value,
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  }
  if (includePagination) {
    params.page = pagination.page
    params.page_size = pagination.page_size
    params.sort_by = 'created_at'
    params.sort_order = 'desc'
  }
  if (filters.user_id > 0) params.user_id = filters.user_id
  if (filters.model.trim()) params.model = filters.model.trim()
  const groupId = Number(filters.group_id)
  if (Number.isFinite(groupId) && groupId > 0) params.group_id = groupId
  if (filters.request_type) params.request_type = filters.request_type
  return params
}

async function loadUsageUsers() {
  usageUsers.value = await agentManagementAPI.listUsageUsers()
}

async function loadUsage() {
  loading.value = true
  try {
    const [records, stats] = await Promise.all([
      agentManagementAPI.listUsage(buildParams(true)),
      agentManagementAPI.getUsageStats(buildParams(false)),
    ])
    usageLogs.value = records.items || []
    pagination.total = records.total || 0
    pagination.page = records.page || pagination.page
    pagination.page_size = records.page_size || pagination.page_size
    pagination.pages = records.pages || 1
    usageStats.value = stats
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.usage.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function refreshData() {
  await loadUsage()
}

async function applyFilters() {
  pagination.page = 1
  await loadUsage()
}

async function resetFilters() {
  filters.user_id = 0
  filters.model = ''
  filters.group_id = ''
  filters.request_type = ''
  pagination.page = 1
  await loadUsage()
}

async function handlePageChange(page: number) {
  pagination.page = page
  await loadUsage()
}

async function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  await loadUsage()
}

onMounted(async () => {
  try {
    await loadUsageUsers()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.usage.usersLoadFailed'))
  }
  await loadUsage()
})
</script>
