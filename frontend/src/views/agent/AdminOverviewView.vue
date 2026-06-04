<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('agentManagement.overview.title') }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('agentManagement.overview.subtitle') }}</p>
          </div>
          <button class="btn btn-secondary px-3" :disabled="loading" @click="loadData">
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <div v-if="loading" class="py-12 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="items.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
          data-test="agent-overview-empty"
        >
          {{ t('agentManagement.overview.empty') }}
        </div>

        <div v-else class="space-y-4" data-test="agent-overview-list">
          <section
            v-for="item in items"
            :key="item.agent.id"
            class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800"
            data-test="agent-overview-card"
          >
            <div class="border-b border-gray-200 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/40">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <h2 class="truncate text-base font-semibold text-gray-900 dark:text-white">
                      {{ item.agent.email }}
                    </h2>
                    <span class="badge badge-gray">{{ roleLabel(item.agent.role) }}</span>
                  </div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                    {{ item.agent.username || '-' }} · ID {{ item.agent.id }}
                  </div>
                </div>
                <div class="grid grid-cols-3 gap-2 text-right text-xs sm:min-w-[360px]">
                  <div>
                    <div class="text-gray-500 dark:text-dark-400">{{ t('agentManagement.overview.poolConcurrency') }}</div>
                    <div class="font-semibold text-gray-900 dark:text-white">{{ item.agent.pool_concurrency }}</div>
                  </div>
                  <div>
                    <div class="text-gray-500 dark:text-dark-400">{{ t('agentManagement.overview.poolRpm') }}</div>
                    <div class="font-semibold text-gray-900 dark:text-white">{{ rpmText(item.agent.pool_rpm) }}</div>
                  </div>
                  <div>
                    <div class="text-gray-500 dark:text-dark-400">{{ t('agentManagement.direct.agentIncome') }}</div>
                    <div class="font-semibold text-emerald-700 dark:text-emerald-300">{{ formatCurrency(Number(item.agent.agent_income || 0)) }}</div>
                  </div>
                </div>
              </div>
            </div>

            <div class="grid gap-4 p-4 xl:grid-cols-2">
              <div class="space-y-2">
                <div class="flex items-center justify-between">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('agentManagement.overview.directUsers') }}</h3>
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ item.users.length }}</span>
                </div>
                <div v-if="item.users.length === 0" class="rounded-md border border-dashed border-gray-300 px-3 py-6 text-center text-xs text-gray-500 dark:border-dark-600 dark:text-dark-400">
                  {{ t('agentManagement.overview.noDirectUsers') }}
                </div>
                <div v-else class="overflow-x-auto rounded-md border border-gray-200 dark:border-dark-700">
                  <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                    <thead class="bg-gray-50 text-left text-xs font-medium text-gray-500 dark:bg-dark-900/50 dark:text-dark-400">
                      <tr>
                        <th class="px-3 py-2">{{ t('common.email') }}</th>
                        <th class="px-3 py-2">{{ t('agentManagement.direct.allocatedConcurrency') }}</th>
                        <th class="px-3 py-2">{{ t('agentManagement.direct.allocatedRpm') }}</th>
                        <th class="px-3 py-2">{{ t('agentManagement.direct.balance') }}</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                      <tr v-for="user in item.users" :key="user.id">
                        <td class="px-3 py-2">
                          <div class="font-medium text-gray-900 dark:text-white">{{ user.email }}</div>
                          <div class="text-xs text-gray-500 dark:text-dark-400">{{ user.username || '-' }}</div>
                        </td>
                        <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ user.concurrency }}</td>
                        <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ rpmText(user.rpm_limit) }}</td>
                        <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ formatCurrency(Number(user.balance || 0)) }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>

              <div class="space-y-2">
                <div class="flex items-center justify-between">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('agentManagement.overview.enterprises') }}</h3>
                  <span class="text-xs text-gray-500 dark:text-dark-400">{{ item.enterprises.length }}</span>
                </div>
                <div v-if="item.enterprises.length === 0" class="rounded-md border border-dashed border-gray-300 px-3 py-6 text-center text-xs text-gray-500 dark:border-dark-600 dark:text-dark-400">
                  {{ t('agentManagement.overview.noEnterprises') }}
                </div>
                <div v-else class="space-y-3">
                  <article
                    v-for="enterpriseNode in item.enterprises"
                    :key="enterpriseNode.enterprise.id"
                    class="rounded-md border border-gray-200 dark:border-dark-700"
                  >
                    <div class="flex flex-wrap items-center justify-between gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900/50">
                      <div>
                        <div class="font-medium text-gray-900 dark:text-white">{{ enterpriseNode.enterprise.email }}</div>
                        <div class="text-xs text-gray-500 dark:text-dark-400">
                          {{ enterpriseNode.enterprise.username || '-' }} · {{ t('agentManagement.overview.employeeCount', { count: enterpriseNode.employees.length }) }}
                        </div>
                      </div>
                      <div class="text-xs text-gray-600 dark:text-dark-300">
                        {{ t('agentManagement.overview.poolConcurrency') }} {{ enterpriseNode.enterprise.pool_concurrency }} · {{ t('agentManagement.overview.poolRpm') }} {{ rpmText(enterpriseNode.enterprise.pool_rpm) }}
                      </div>
                    </div>
                    <div v-if="enterpriseNode.employees.length === 0" class="px-3 py-4 text-center text-xs text-gray-500 dark:text-dark-400">
                      {{ t('agentManagement.overview.noEmployees') }}
                    </div>
                    <table v-else class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
                      <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                        <tr v-for="employee in enterpriseNode.employees" :key="employee.id">
                          <td class="px-3 py-2">
                            <div class="font-medium text-gray-900 dark:text-white">{{ employee.email }}</div>
                            <div class="text-xs text-gray-500 dark:text-dark-400">{{ employee.username || '-' }}</div>
                          </td>
                          <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ employee.concurrency }}</td>
                          <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ rpmText(employee.rpm_limit) }}</td>
                          <td class="px-3 py-2 text-gray-700 dark:text-dark-200">{{ formatCurrency(Number(employee.balance || 0)) }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </article>
                </div>
              </div>
            </div>
          </section>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore } from '@/stores'
import { formatCurrency } from '@/utils/format'
import type { AgentAdminTreeAgent, UserRole } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const items = ref<AgentAdminTreeAgent[]>([])

function roleLabel(role: UserRole | string): string {
  return t(`admin.users.roles.${role}`)
}

function rpmText(rpm: number | undefined): string {
  return Number(rpm || 0) === 0 ? t('common.unlimited') : String(rpm)
}

async function loadData() {
  loading.value = true
  try {
    const result = await agentManagementAPI.getAdminAgentTree()
    items.value = result.items
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.overview.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>
