<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('nav.enterpriseGroups') }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('enterpriseManagement.groups.subtitle') }}</p>
          </div>
          <button class="btn btn-secondary px-3" :disabled="loading" @click="loadGroups">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="groups" :loading="loading">
          <template #cell-name="{ row }">
            <div class="flex flex-col">
              <span class="font-medium text-gray-900 dark:text-white">{{ row.group.name }}</span>
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ row.group.platform }}</span>
            </div>
          </template>

          <template #cell-source="{ value }">
            <span class="badge badge-gray">{{ t(`agentManagement.groups.sources.${value}`) }}</span>
          </template>

          <template #cell-effective_rate="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
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
import type { EnterpriseGroupRate } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const groups = ref<EnterpriseGroupRate[]>([])

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name') },
  { key: 'source', label: t('agentManagement.groups.source') },
  { key: 'effective_rate', label: t('agentManagement.groups.effectiveRate') },
])

async function loadGroups() {
  loading.value = true
  try {
    groups.value = await enterpriseManagementAPI.listGroups()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('enterpriseManagement.groups.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadGroups)
</script>
