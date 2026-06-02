<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('nav.agentGroups') }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('agentManagement.groups.subtitle') }}</p>
          </div>
          <button class="btn btn-secondary px-3" :disabled="loading || inviteDefaultLoading" @click="loadData">
            <Icon name="refresh" size="sm" :class="loading || inviteDefaultLoading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <div class="flex h-full flex-col gap-6 overflow-y-auto">
          <section
            v-if="showInviteDefaultGroups"
            data-test="invite-default-groups-section"
            class="border-b border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="mb-4 flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('agentManagement.groups.inviteDefaultsTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.inviteDefaultsDescription') }}
                </p>
              </div>
              <span class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                {{ inviteDefaultAssignedCount }}/{{ inviteDefaultGroupOptions.length }}
              </span>
            </div>

            <div v-if="inviteDefaultLoading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('common.loading') }}
            </div>

            <div
              v-else-if="inviteDefaultGroupOptions.length === 0"
              class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
            >
              {{ t('agentManagement.groups.emptyDelegable') }}
            </div>

            <div v-else class="grid gap-3">
              <div
                v-for="groupRate in inviteDefaultGroupOptions"
                :key="groupRate.group.id"
                class="group relative overflow-hidden rounded-lg border-2 p-4 transition-all duration-200"
                :class="inviteDefaultDraftFor(groupRate).assigned
                  ? 'border-primary-400 bg-primary-50/50 shadow-sm dark:border-primary-500 dark:bg-primary-900/20'
                  : 'border-gray-200 bg-white hover:border-gray-300 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-dark-500'"
              >
                <div class="flex flex-col gap-4 lg:flex-row lg:items-center">
                  <div class="flex min-w-0 flex-1 items-center gap-4">
                    <div class="flex-shrink-0">
                      <input
                        :data-test="`invite-default-assigned-${groupRate.group.id}`"
                        class="checkbox"
                        type="checkbox"
                        :checked="inviteDefaultDraftFor(groupRate).assigned"
                        @change="updateInviteDefaultAssignedDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                      />
                    </div>

                    <div class="min-w-0 flex-1">
                      <div class="flex flex-wrap items-center gap-2">
                        <span class="truncate text-base font-semibold text-gray-900 dark:text-white">
                          {{ groupRate.group.name }}
                        </span>
                        <span class="inline-flex items-center rounded-full bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/40 dark:text-purple-300">
                          {{ t('admin.groups.exclusive') }}
                        </span>
                        <span :class="inviteDefaultDraftFor(groupRate).assigned ? 'badge badge-green' : 'badge badge-gray'">
                          {{ t('agentManagement.groups.defaultPropagation') }}:
                          {{ inviteDefaultDraftFor(groupRate).assigned ? t('common.yes') : t('common.no') }}
                        </span>
                      </div>
                      <div class="mt-1.5 flex flex-wrap items-center gap-3 text-sm">
                        <span class="inline-flex items-center gap-1 text-gray-500 dark:text-gray-400">
                          <PlatformIcon :platform="groupRate.group.platform" size="xs" />
                          <span>{{ groupRate.group.platform }}</span>
                        </span>
                        <span class="text-gray-300 dark:text-dark-500">/</span>
                        <span class="text-gray-500 dark:text-gray-400">
                          {{ t('agentManagement.groups.source') }}:
                          <span class="font-medium text-gray-700 dark:text-gray-300">{{ sourceLabel(groupRate.source) }}</span>
                        </span>
                        <span class="text-gray-300 dark:text-dark-500">/</span>
                        <span class="text-gray-500 dark:text-gray-400">
                          {{ t('agentManagement.groups.effectiveRate') }}:
                          <span class="font-medium text-gray-700 dark:text-gray-300">{{ groupRate.effective_rate }}x</span>
                        </span>
                      </div>
                    </div>
                  </div>

                  <div class="flex flex-col gap-3 sm:flex-row sm:items-center lg:flex-shrink-0">
                    <label class="flex items-center gap-3">
                      <span class="whitespace-nowrap text-sm font-medium text-gray-600 dark:text-gray-400">
                        {{ t('admin.users.customRate') }}
                      </span>
                      <input
                        :data-test="`invite-default-rate-${groupRate.group.id}`"
                        class="hide-spinner w-24 rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-medium transition-colors focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 dark:border-dark-500 dark:bg-dark-700 dark:focus:border-primary-500"
                        type="number"
                        min="0.001"
                        step="0.001"
                        :disabled="!inviteDefaultDraftFor(groupRate).assigned"
                        :placeholder="String(groupRate.effective_rate)"
                        :value="inviteDefaultDraftFor(groupRate).rate_multiplier"
                        @input="updateInviteDefaultRateDraft(groupRate.group.id, ($event.target as HTMLInputElement).value)"
                      />
                    </label>

                    <button
                      :data-test="`save-invite-default-${groupRate.group.id}`"
                      class="btn btn-primary btn-sm justify-center"
                      :disabled="savingInviteDefaultGroupId === groupRate.group.id"
                      @click="saveInviteDefaultGroup(groupRate)"
                    >
                      <Icon name="check" size="sm" />
                      <span>{{ t('common.save') }}</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </section>

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

            <template #cell-can_delegate="{ value }">
              <span :class="value ? 'badge badge-green' : 'badge badge-gray'">
                {{ value ? t('common.yes') : t('common.no') }}
              </span>
            </template>
          </DataTable>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore, useAuthStore } from '@/stores'
import type { AgentChildGroupDelegationOption, AgentGroupRate } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const loading = ref(false)
const inviteDefaultLoading = ref(false)
const savingInviteDefaultGroupId = ref<number | null>(null)
const groups = ref<AgentGroupRate[]>([])
const inviteDefaultGroupOptions = ref<AgentChildGroupDelegationOption[]>([])
const inviteDefaultDrafts = reactive<Record<number, { assigned: boolean; rate_multiplier: number }>>({})

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('common.name') },
  { key: 'source', label: t('agentManagement.groups.source') },
  { key: 'effective_rate', label: t('agentManagement.groups.effectiveRate') },
  { key: 'can_delegate', label: t('agentManagement.groups.canDelegate') },
])

const showInviteDefaultGroups = computed(() => authStore.isAdmin || authStore.isAgent)
const inviteDefaultAssignedCount = computed(() => (
  inviteDefaultGroupOptions.value.filter((item) => {
    const draft = inviteDefaultDrafts[item.group.id]
    return draft ? draft.assigned : item.assigned
  }).length
))

function normalizedPositiveFloat(value: unknown): number {
  const parsed = Number.parseFloat(String(value ?? '0'))
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 0
  }
  return parsed
}

function sourceLabel(source: AgentChildGroupDelegationOption['source']): string {
  return t(`agentManagement.groups.sources.${source}`)
}

function clearInviteDefaultDrafts() {
  for (const key of Object.keys(inviteDefaultDrafts)) {
    delete inviteDefaultDrafts[Number(key)]
  }
}

function syncInviteDefaultDrafts(options: AgentChildGroupDelegationOption[]) {
  clearInviteDefaultDrafts()
  for (const item of options) {
    inviteDefaultDrafts[item.group.id] = {
      assigned: item.assigned,
      rate_multiplier: item.assigned ? item.child_rate_multiplier : item.effective_rate,
    }
  }
}

function inviteDefaultDraftFor(groupRate: AgentChildGroupDelegationOption) {
  if (!inviteDefaultDrafts[groupRate.group.id]) {
    inviteDefaultDrafts[groupRate.group.id] = {
      assigned: groupRate.assigned,
      rate_multiplier: groupRate.assigned ? groupRate.child_rate_multiplier : groupRate.effective_rate,
    }
  }
  return inviteDefaultDrafts[groupRate.group.id]
}

function updateInviteDefaultAssignedDraft(groupId: number, assigned: boolean) {
  inviteDefaultDrafts[groupId] = {
    ...(inviteDefaultDrafts[groupId] || { assigned: false, rate_multiplier: 0 }),
    assigned,
  }
}

function updateInviteDefaultRateDraft(groupId: number, rawValue: string) {
  inviteDefaultDrafts[groupId] = {
    ...(inviteDefaultDrafts[groupId] || { assigned: true, rate_multiplier: 0 }),
    rate_multiplier: normalizedPositiveFloat(rawValue),
  }
}

async function loadGroups() {
  loading.value = true
  try {
    groups.value = await agentManagementAPI.listGroups()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadInviteDefaultGroups() {
  if (!showInviteDefaultGroups.value) {
    inviteDefaultGroupOptions.value = []
    clearInviteDefaultDrafts()
    return
  }

  inviteDefaultLoading.value = true
  try {
    const options = await agentManagementAPI.listInviteGroupDefaultOptions()
    inviteDefaultGroupOptions.value = options.filter((item) => item.can_delegate && item.group.is_exclusive)
    syncInviteDefaultDrafts(inviteDefaultGroupOptions.value)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    inviteDefaultLoading.value = false
  }
}

async function loadData() {
  await Promise.all([
    loadGroups(),
    loadInviteDefaultGroups(),
  ])
}

async function saveInviteDefaultGroup(groupRate: AgentChildGroupDelegationOption) {
  const draft = inviteDefaultDraftFor(groupRate)
  if (!draft.assigned) {
    await removeInviteDefaultGroup(groupRate)
    return
  }

  if (draft.rate_multiplier <= 0) {
    appStore.showError(t('agentManagement.groups.invalidRate'))
    return
  }

  savingInviteDefaultGroupId.value = groupRate.group.id
  try {
    await agentManagementAPI.setInviteGroupDefault(groupRate.group.id, {
      rate_multiplier: draft.rate_multiplier,
    })
    appStore.showSuccess(t('agentManagement.groups.inviteDefaultGroupSaved'))
    await loadInviteDefaultGroups()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.inviteDefaultGroupFailed'))
  } finally {
    savingInviteDefaultGroupId.value = null
  }
}

async function removeInviteDefaultGroup(groupRate: AgentChildGroupDelegationOption) {
  savingInviteDefaultGroupId.value = groupRate.group.id
  try {
    await agentManagementAPI.removeInviteGroupDefault(groupRate.group.id)
    appStore.showSuccess(t('agentManagement.groups.inviteDefaultGroupRemoved'))
    await loadInviteDefaultGroups()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.inviteDefaultGroupRemoveFailed'))
  } finally {
    savingInviteDefaultGroupId.value = null
  }
}

onMounted(() => {
  authStore.checkAuth()
  loadData()
})
</script>

<style scoped>
.hide-spinner::-webkit-outer-spin-button,
.hide-spinner::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.hide-spinner {
  -moz-appearance: textfield;
}
</style>
