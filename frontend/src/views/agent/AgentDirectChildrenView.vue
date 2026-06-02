<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex min-w-0 flex-col gap-1">
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ title }}</h1>
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ directSubtitle }}</p>
          </div>
          <div class="flex items-center gap-2">
            <button
              v-if="canCreateDirectUser"
              data-test="create-direct-user"
              class="btn btn-primary px-3"
              :disabled="loading"
              @click="showCreateUserForm = !showCreateUserForm"
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
              {{ t('agentManagement.direct.remainingConcurrency') }}: {{ allocation?.remaining_concurrency ?? '-' }}
            </span>
            <span v-if="!isAdminUnlimitedCapacity" class="rounded-md bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
              {{ t('agentManagement.direct.remainingRpm') }}: {{ allocation?.remaining_rpm ?? '-' }}
            </span>
            <button class="btn btn-secondary px-3" :disabled="loading" @click="loadData">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <form
          v-if="canCreateDirectUser && showCreateUserForm"
          data-test="create-direct-user-submit"
          class="mb-4 grid gap-3 border-b border-gray-200 pb-4 dark:border-dark-700 sm:grid-cols-2 lg:grid-cols-5"
          @submit.prevent="createDirectUser"
        >
          <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('common.email') }}</span>
            <input
              v-model="createUserForm.email"
              data-test="create-direct-user-email"
              class="input h-9"
              type="email"
              required
            />
          </label>
          <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.users.password') }}</span>
            <input
              v-model="createUserForm.password"
              data-test="create-direct-user-password"
              class="input h-9"
              type="text"
              required
              minlength="6"
            />
          </label>
          <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.users.username') }}</span>
            <input
              v-model="createUserForm.username"
              data-test="create-direct-user-username"
              class="input h-9"
              type="text"
            />
          </label>
          <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('agentManagement.direct.allocatedConcurrency') }}</span>
            <input
              v-model.number="createUserForm.allocated_concurrency"
              data-test="create-direct-user-concurrency"
              class="input h-9"
              type="number"
              min="0"
            />
          </label>
          <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('agentManagement.direct.allocatedRpm') }}</span>
            <input
              v-model.number="createUserForm.allocated_rpm"
              data-test="create-direct-user-rpm"
              class="input h-9"
              type="number"
              min="0"
            />
          </label>
          <div class="flex items-end gap-2 sm:col-span-2 lg:col-span-5">
            <button class="btn btn-primary btn-sm" type="submit" :disabled="creatingUser">
              <Icon name="check" size="sm" />
              <span>{{ t('common.create') }}</span>
            </button>
            <button class="btn btn-secondary btn-sm" type="button" :disabled="creatingUser" @click="showCreateUserForm = false">
              <Icon name="x" size="sm" />
              <span>{{ t('common.cancel') }}</span>
            </button>
          </div>
        </form>

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

          <template #cell-allocation="{ row }">
            <div class="grid min-w-[240px] grid-cols-2 gap-2">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedConcurrency') }}</span>
                <input
                  :data-test="`allocation-concurrency-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  :value="draftFor(row).allocated_concurrency"
                  @input="updateDraft(row.id, 'allocated_concurrency', ($event.target as HTMLInputElement).value)"
                />
              </label>
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.direct.allocatedRpm') }}</span>
                <input
                  :data-test="`allocation-rpm-${row.id}`"
                  class="input h-9"
                  type="number"
                  min="0"
                  :value="draftFor(row).allocated_rpm"
                  @input="updateDraft(row.id, 'allocated_rpm', ($event.target as HTMLInputElement).value)"
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
                <span>{{ t('agentManagement.direct.detach') }}</span>
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
      :title="t('agentManagement.direct.detachTitle')"
      :message="t('agentManagement.direct.detachConfirm', { email: detachDialog.child?.email || '' })"
      :confirm-text="t('agentManagement.direct.detach')"
      danger
      @confirm="confirmDetach"
      @cancel="detachDialog.show = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { agentManagementAPI } from '@/api/agentManagement'
import { useAppStore, useAuthStore } from '@/stores'
import type {
  AgentAllocationSummary,
  AgentAllocationUpdate,
  AgentDirectChildrenResponse,
  AgentDirectUserCreateRequest,
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
const showCreateUserForm = ref(false)
const children = ref<AgentManagedUser[]>([])
const allocation = ref<AgentAllocationSummary | null>(null)
const drafts = reactive<Record<number, AgentAllocationUpdate>>({})
const pagination = reactive({ total: 0, page: 1, page_size: 20, pages: 1 })
const detachDialog = reactive<{ show: boolean; child: AgentManagedUser | null }>({ show: false, child: null })
const createUserForm = reactive<Required<AgentDirectUserCreateRequest>>({
  email: '',
  password: '',
  username: '',
  allocated_concurrency: 0,
  allocated_rpm: 0,
})

const columns = computed<Column[]>(() => [
  { key: 'email', label: t('common.email') },
  { key: 'role', label: t('agentManagement.direct.role') },
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

const isAdminUnlimitedCapacity = computed(() => {
  return authStore.user?.role === 'admin' || allocation.value?.unlimited_capacity === true
})

const directSubtitle = computed(() => {
  if (isAdminUnlimitedCapacity.value) {
    return t('agentManagement.direct.adminSubtitle')
  }
  return t('agentManagement.direct.subtitle')
})

function extractPagination(result: AgentDirectChildrenResponse) {
  const source = result.pagination || {}
  pagination.total = Number(source.total ?? source.Total ?? result.items.length)
  pagination.page = Number(source.page ?? source.Page ?? 1)
  pagination.page_size = Number(source.page_size ?? source.PageSize ?? 20)
  pagination.pages = Number(source.pages ?? source.Pages ?? 1)
}

function syncDrafts(items: AgentManagedUser[]) {
  for (const child of items) {
    drafts[child.id] = {
      allocated_concurrency: child.allocated_concurrency,
      allocated_rpm: child.allocated_rpm,
    }
  }
}

async function listChildren(): Promise<AgentDirectChildrenResponse> {
  if (props.kind === 'agents') return agentManagementAPI.listDirectAgents()
  if (props.kind === 'enterprises') return agentManagementAPI.listDirectEnterprises()
  return agentManagementAPI.listDirectUsers()
}

async function loadData() {
  loading.value = true
  try {
    const [summary, result] = await Promise.all([
      agentManagementAPI.getSummary(),
      listChildren(),
    ])
    allocation.value = summary.allocation
    children.value = result.items
    syncDrafts(result.items)
    extractPagination(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.loadFailed'))
  } finally {
    loading.value = false
  }
}

function draftFor(child: AgentManagedUser): AgentAllocationUpdate {
  if (!drafts[child.id]) {
    drafts[child.id] = {
      allocated_concurrency: child.allocated_concurrency,
      allocated_rpm: child.allocated_rpm,
    }
  }
  return drafts[child.id]
}

function updateDraft(childId: number, key: keyof AgentAllocationUpdate, rawValue: string) {
  const parsed = Math.max(0, Number.parseInt(rawValue || '0', 10) || 0)
  drafts[childId] = {
    ...(drafts[childId] || { allocated_concurrency: 0, allocated_rpm: 0 }),
    [key]: parsed,
  }
}

function resetCreateUserForm() {
  Object.assign(createUserForm, {
    email: '',
    password: '',
    username: '',
    allocated_concurrency: 0,
    allocated_rpm: 0,
  })
}

async function createDirectUser() {
  creatingUser.value = true
  try {
    await agentManagementAPI.createDirectUser({
      email: createUserForm.email.trim(),
      password: createUserForm.password,
      username: createUserForm.username.trim(),
      allocated_concurrency: Math.max(0, Number(createUserForm.allocated_concurrency) || 0),
      allocated_rpm: Math.max(0, Number(createUserForm.allocated_rpm) || 0),
    })
    appStore.showSuccess(t('agentManagement.direct.userCreated'))
    resetCreateUserForm()
    showCreateUserForm.value = false
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.createFailed'))
  } finally {
    creatingUser.value = false
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
    await agentManagementAPI.upgradeChild(child.id, targetRole)
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

async function confirmDetach() {
  if (!detachDialog.child) return
  const child = detachDialog.child
  detachDialog.show = false
  savingChildId.value = child.id
  try {
    await agentManagementAPI.deleteDirectChild(child.id)
    appStore.showSuccess(t('agentManagement.direct.detached'))
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
