<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('nav.enterpriseEmployees') }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.employees.subtitle') }}</p>
          </div>
          <button class="btn btn-secondary px-3" :disabled="loading" @click="loadEmployees">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="employees" :loading="loading">
          <template #cell-username="{ row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.username }}</span>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ row.email }}</span>
            </div>
          </template>

          <template #cell-balance="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
          </template>

          <template #cell-status="{ value }">
            <span :class="value === 'active' ? 'badge badge-green' : 'badge badge-gray'">
              {{ value === 'active' ? t('common.active') : t('common.disabled') }}
            </span>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { enterpriseManagementAPI } from '@/api/enterpriseManagement'
import { useAppStore } from '@/stores'
import type { Column } from '@/components/common/types'
import type { EnterpriseEmployee } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const employees = ref<EnterpriseEmployee[]>([])

const columns = computed<Column[]>(() => [
  { key: 'username', label: t('common.email') },
  { key: 'balance', label: t('common.balance') },
  { key: 'concurrency', label: t('agentManagement.direct.allocatedConcurrency') },
  { key: 'rpm_limit', label: t('agentManagement.direct.allocatedRpm') },
  { key: 'status', label: t('common.status') },
])

async function loadEmployees() {
  loading.value = true
  try {
    const response = await enterpriseManagementAPI.listEmployees()
    employees.value = response.items
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.employees.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadEmployees)
</script>
