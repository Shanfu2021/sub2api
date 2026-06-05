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
            <button
              data-test="open-direct-group-batch"
              class="btn btn-secondary px-3"
              :disabled="loading"
              @click="openDirectGroupBatchDialog"
            >
              <Icon name="grid" size="sm" />
              <span>{{ t('agentManagement.groups.directBatchDeploy') }}</span>
            </button>
            <button
              data-test="open-direct-group-update"
              class="btn btn-secondary px-3"
              :disabled="loading"
              @click="openDirectGroupUpdateDialog"
            >
              <Icon name="cog" size="sm" />
              <span>{{ t('agentManagement.groups.directBatchUpdate') }}</span>
            </button>
            <button
              data-test="open-direct-group-reclaim"
              class="btn btn-secondary px-3 text-red-600 dark:text-red-400"
              :disabled="loading"
              @click="openDirectGroupReclaimDialog"
            >
              <Icon name="trash" size="sm" />
              <span>{{ t('agentManagement.groups.directBatchReclaim') }}</span>
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
                v-if="canSetAgentIncome(row)"
                :data-test="`set-agent-income-${row.id}`"
                class="btn btn-secondary btn-sm"
                :disabled="savingChildId === row.id"
                @click="openAgentIncomeDialog(row)"
              >
                <Icon name="dollar" size="sm" />
                <span>{{ t('agentManagement.direct.setAgentIncome') }}</span>
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
      :show="agentIncomeDialog.show"
      :title="t('agentManagement.direct.setAgentIncomeTitle')"
      @close="closeAgentIncomeDialog"
    >
      <div class="space-y-4" data-test="agent-income-modal">
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="text-sm font-medium text-gray-900 dark:text-white">
            {{ agentIncomeDialog.child?.email || '-' }}
          </div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('agentManagement.direct.currentAgentIncome') }}: {{ formatCurrency(Number(agentIncomeDialog.child?.agent_income || 0)) }}
          </div>
        </div>

        <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
          <span>{{ t('agentManagement.direct.targetAgentIncome') }}</span>
          <input
            v-model.number="agentIncomeDraft.agent_income"
            data-test="agent-income-input"
            class="input h-10"
            type="number"
            step="0.000001"
          />
        </label>

        <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
          <span>{{ t('agentManagement.direct.agentIncomeReason') }}</span>
          <input
            v-model="agentIncomeDraft.reason"
            data-test="agent-income-reason"
            class="input h-10"
            type="text"
          />
        </label>

        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary px-3" type="button" @click="closeAgentIncomeDialog">
            {{ t('common.cancel') }}
          </button>
          <button
            data-test="agent-income-submit"
            class="btn btn-primary px-3"
            type="button"
            :disabled="agentIncomeDialog.saving"
            @click="saveAgentIncome"
          >
            <Icon name="check" size="sm" />
            <span>{{ t('common.save') }}</span>
          </button>
        </div>
      </div>
    </BaseDialog>

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
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
            <div class="grid gap-3 lg:grid-cols-[1fr_auto] lg:items-end">
              <div class="flex flex-wrap items-center gap-3">
                <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                  <input
                    data-test="group-batch-all"
                    class="checkbox"
                    type="checkbox"
                    :checked="groupBatchAll"
                    @change="setGroupBatchAll(($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ t('agentManagement.groups.batchAll') }}</span>
                </label>
                <span class="text-sm text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.batchSelected', { count: selectedGroupBatchIDs.length }) }}
                </span>
              </div>

              <div class="flex flex-wrap items-end gap-3">
                <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ t('agentManagement.groups.batchRate') }}</span>
                  <input
                    v-model.number="groupBatchRate"
                    data-test="group-batch-rate"
                    class="input h-9 w-28"
                    type="number"
                    min="0.000001"
                    step="0.000001"
                  />
                </label>
                <label class="flex min-h-9 items-center gap-2 text-xs text-gray-600 dark:text-dark-300">
                  <input
                    v-model="groupBatchCanDelegate"
                    data-test="group-batch-can-delegate"
                    class="checkbox"
                    type="checkbox"
                  />
                  <span>{{ t('agentManagement.groups.allowChildDelegate') }}</span>
                </label>
                <button
                  data-test="apply-group-batch"
                  class="btn btn-primary h-9 px-3"
                  :disabled="groupBatchSaving"
                  @click="applyGroupDelegationBatch"
                >
                  <Icon name="check" size="sm" />
                  <span>{{ t('agentManagement.groups.applyBatch') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div
            v-for="groupRate in groupDialog.groups"
            :key="groupRate.group.id"
            class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
          >
            <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-3">
                  <div class="flex flex-wrap items-center gap-2 rounded-md border border-gray-200 bg-gray-50 px-2.5 py-1.5 dark:border-dark-700 dark:bg-dark-900/40">
                    <label class="flex items-center gap-1.5 text-xs font-medium text-gray-600 dark:text-dark-300">
                      <input
                        :data-test="`group-batch-select-${groupRate.group.id}`"
                        class="checkbox"
                        type="checkbox"
                        :checked="selectedGroupBatchIDs.includes(groupRate.group.id)"
                        :disabled="groupBatchAll"
                        @change="updateGroupBatchSelection(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                      />
                      <span>{{ t('agentManagement.groups.batchSelectLabel') }}</span>
                    </label>
                    <span class="h-4 w-px bg-gray-200 dark:bg-dark-600"></span>
                    <label class="flex items-center gap-1.5 text-xs font-medium text-gray-700 dark:text-dark-200">
                      <input
                        :data-test="`group-assigned-${groupRate.group.id}`"
                        class="checkbox"
                        type="checkbox"
                        :checked="groupDraftFor(groupRate).assigned"
                        @change="updateGroupAssignedDraft(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
                      />
                      <span>{{ t('agentManagement.groups.groupAssignLabel') }}</span>
                    </label>
                  </div>
                  <div class="min-w-0 flex flex-1 flex-wrap items-center gap-2">
                    <h4 class="truncate text-sm font-semibold text-gray-900 dark:text-white">
                      {{ groupRate.group.name }}
                    </h4>
                    <span class="badge badge-gray">{{ sourceLabel(groupRate.source) }}</span>
                  </div>
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

    <BaseDialog
      :show="directGroupBatchDialog.show"
      :title="t('agentManagement.groups.directBatchTitle')"
      width="wide"
      @close="closeDirectGroupBatchDialog"
    >
      <div class="space-y-4" data-test="direct-group-batch-modal">
        <div v-if="directGroupBatchDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="directGroupBatchDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('agentManagement.groups.emptyDelegable') }}
        </div>

        <div v-else class="space-y-4">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
            <div class="grid gap-3 lg:grid-cols-[minmax(260px,1fr)_auto] lg:items-end">
              <div class="flex flex-wrap items-center gap-3">
                <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                  <input
                    data-test="direct-group-batch-all"
                    class="checkbox"
                    type="checkbox"
                    :checked="directGroupBatchAll"
                    @change="setDirectGroupBatchAll(($event.target as HTMLInputElement).checked)"
                  />
                  <span>{{ t('agentManagement.groups.batchAll') }}</span>
                </label>
                <span class="text-sm text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.batchSelected', { count: selectedDirectGroupBatchIDs.length }) }}
                </span>
              </div>

              <div class="flex flex-wrap items-end gap-3">
                <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ t('agentManagement.groups.batchRate') }}</span>
                  <input
                    v-model.number="directGroupBatchRate"
                    data-test="direct-group-batch-rate"
                    class="input h-9 w-28"
                    type="number"
                    min="0.000001"
                    step="0.000001"
                  />
                </label>
                <label class="flex min-h-9 items-center gap-2 text-xs text-gray-600 dark:text-dark-300">
                  <input
                    v-model="directGroupBatchCanDelegate"
                    data-test="direct-group-batch-can-delegate"
                    class="checkbox"
                    type="checkbox"
                  />
                  <span>{{ t('agentManagement.groups.allowChildDelegate') }}</span>
                </label>
                <button
                  data-test="apply-direct-group-batch"
                  class="btn btn-primary h-9 px-3"
                  :disabled="directGroupBatchDialog.saving"
                  @click="applyDirectGroupBatch"
                >
                  <Icon name="check" size="sm" />
                  <span>{{ t('agentManagement.groups.applyBatch') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="grid gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700 sm:grid-cols-2">
            <label
              v-for="groupRate in directGroupBatchDialog.groups"
              :key="groupRate.group.id"
              class="flex items-center gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700"
            >
              <input
                :data-test="`direct-group-batch-select-${groupRate.group.id}`"
                class="checkbox"
                type="checkbox"
                :checked="selectedDirectGroupBatchIDs.includes(groupRate.group.id)"
                :disabled="directGroupBatchAll"
                @change="updateDirectGroupBatchSelection(groupRate.group.id, ($event.target as HTMLInputElement).checked)"
              />
              <span class="min-w-0 flex-1">
                <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ groupRate.group.name }}</span>
                <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">
                  {{ t('agentManagement.groups.effectiveRate') }}: {{ groupRate.effective_rate }}
                </span>
              </span>
            </label>
          </div>

          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="mb-3 grid gap-3 lg:grid-cols-[1fr_auto] lg:items-end">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.groups.deploySearch') }}</span>
                <input
                  v-model="directGroupBatchDialog.searchDraft"
                  data-test="direct-group-batch-search"
                  class="input h-9"
                  type="search"
                  :placeholder="t('common.search')"
                  @keyup.enter="applyDirectGroupBatchSearch"
                />
              </label>
              <button
                data-test="direct-group-batch-search-submit"
                class="btn btn-secondary h-9 px-3"
                type="button"
                :disabled="directGroupBatchDialog.childrenLoading || !directGroupBatchHasGroupSelection"
                @click="applyDirectGroupBatchSearch"
              >
                <Icon name="search" size="sm" />
                <span>{{ t('common.search') }}</span>
              </button>
            </div>

            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                <input
                  data-test="direct-group-batch-all-children"
                  class="checkbox"
                  type="checkbox"
                  :checked="directGroupBatchAllChildren"
                  :disabled="!directGroupBatchHasGroupSelection"
                  @change="setDirectGroupBatchAllChildren(($event.target as HTMLInputElement).checked)"
                />
                <span>{{ t('agentManagement.groups.batchAllDeployableChildren') }}</span>
              </label>
              <span class="text-sm text-gray-500 dark:text-dark-400">
                {{ t('agentManagement.groups.updateSelectedChildren', { count: selectedDirectGroupBatchChildIDs.length }) }}
              </span>
            </div>

            <div
              v-if="!directGroupBatchHasGroupSelection"
              class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
            >
              {{ t('agentManagement.groups.selectGroupBeforeChildren') }}
            </div>
            <div v-else-if="directGroupBatchDialog.childrenLoading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('common.loading') }}
            </div>
            <div
              v-else-if="directGroupBatchDialog.children.length === 0"
              class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
            >
              {{ t('agentManagement.groups.emptyDeployableChildren') }}
            </div>
            <div v-else class="max-h-72 space-y-2 overflow-y-auto pr-1">
              <label
                v-for="child in directGroupBatchDialog.children"
                :key="child.id"
                class="flex items-center gap-3 rounded-md border border-gray-200 px-3 py-2 dark:border-dark-700"
              >
                <input
                  :data-test="`direct-group-batch-child-${child.id}`"
                  class="checkbox"
                  type="checkbox"
                  :checked="selectedDirectGroupBatchChildIDs.includes(child.id)"
                  :disabled="directGroupBatchAllChildren"
                  @change="updateDirectGroupBatchChildSelection(child.id, ($event.target as HTMLInputElement).checked)"
                />
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ child.email }}</span>
                  <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">{{ child.username || '-' }}</span>
                </span>
              </label>
            </div>

            <Pagination
              v-if="directGroupBatchDialog.pagination.total > directGroupBatchDialog.pagination.page_size"
              :page="directGroupBatchDialog.pagination.page"
              :total="directGroupBatchDialog.pagination.total"
              :page-size="directGroupBatchDialog.pagination.page_size"
              :show-page-size-selector="false"
              @update:page="changeDirectGroupBatchPage"
              @update:page-size="changeDirectGroupBatchPageSize"
            />
          </div>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="directGroupUpdateDialog.show"
      :title="t('agentManagement.groups.directUpdateTitle')"
      width="wide"
      @close="closeDirectGroupUpdateDialog"
    >
      <div class="space-y-4" data-test="direct-group-update-modal">
        <div v-if="directGroupUpdateDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="directGroupUpdateDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('agentManagement.groups.emptyDelegable') }}
        </div>

        <div v-else class="space-y-4">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
            <div class="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto] lg:items-end">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.groups.updateGroup') }}</span>
                <select
                  v-model.number="directGroupUpdateDialog.selectedGroupId"
                  data-test="direct-group-update-group"
                  class="input h-9"
                  @change="onDirectGroupUpdateGroupChange"
                >
                  <option
                    v-for="groupRate in directGroupUpdateDialog.groups"
                    :key="groupRate.group.id"
                    :value="groupRate.group.id"
                  >
                    {{ groupRate.group.name }}
                  </option>
                </select>
              </label>

              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.groups.updateSearch') }}</span>
                <input
                  v-model="directGroupUpdateDialog.searchDraft"
                  data-test="direct-group-update-search"
                  class="input h-9"
                  type="search"
                  :placeholder="t('common.search')"
                  @keyup.enter="applyDirectGroupUpdateSearch"
                />
              </label>

              <button
                data-test="direct-group-update-search-submit"
                class="btn btn-secondary h-9 px-3"
                type="button"
                :disabled="directGroupUpdateDialog.childrenLoading"
                @click="applyDirectGroupUpdateSearch"
              >
                <Icon name="search" size="sm" />
                <span>{{ t('common.search') }}</span>
              </button>
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="grid gap-4 lg:grid-cols-[minmax(220px,1fr)_minmax(260px,1fr)]">
              <div class="space-y-3">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                    <input
                      data-test="direct-group-update-all"
                      class="checkbox"
                      type="checkbox"
                      :checked="directGroupUpdateAll"
                      @change="setDirectGroupUpdateAll(($event.target as HTMLInputElement).checked)"
                    />
                    <span>{{ t('agentManagement.groups.updateAllAssigned') }}</span>
                  </label>
                  <span class="text-sm text-gray-500 dark:text-dark-400">
                    {{ t('agentManagement.groups.updateSelectedChildren', { count: selectedDirectGroupUpdateChildIDs.length }) }}
                  </span>
                </div>

                <div v-if="directGroupUpdateDialog.childrenLoading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
                  {{ t('common.loading') }}
                </div>
                <div
                  v-else-if="directGroupUpdateDialog.children.length === 0"
                  class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
                >
                  {{ t('agentManagement.groups.emptyAssignedChildren') }}
                </div>
                <div v-else class="max-h-80 space-y-2 overflow-y-auto pr-1">
                  <label
                    v-for="child in directGroupUpdateDialog.children"
                    :key="child.id"
                    class="flex items-center gap-3 rounded-md border border-gray-200 px-3 py-2 dark:border-dark-700"
                  >
                    <input
                      :data-test="`direct-group-update-child-${child.id}`"
                      class="checkbox"
                      type="checkbox"
                      :checked="selectedDirectGroupUpdateChildIDs.includes(child.id)"
                      :disabled="directGroupUpdateAll"
                      @change="updateDirectGroupUpdateChildSelection(child.id, ($event.target as HTMLInputElement).checked)"
                    />
                    <span class="min-w-0 flex-1">
                      <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ child.email }}</span>
                      <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">
                        {{ child.username || '-' }}
                      </span>
                    </span>
                    <span class="text-xs font-medium text-gray-600 dark:text-dark-300">
                      {{ t('agentManagement.groups.currentRate') }}: {{ currentDirectGroupUpdateChildRate(child) }}
                    </span>
                  </label>
                </div>

                <Pagination
                  v-if="directGroupUpdateDialog.pagination.total > directGroupUpdateDialog.pagination.page_size"
                  :page="directGroupUpdateDialog.pagination.page"
                  :total="directGroupUpdateDialog.pagination.total"
                  :page-size="directGroupUpdateDialog.pagination.page_size"
                  :show-page-size-selector="false"
                  @update:page="changeDirectGroupUpdatePage"
                  @update:page-size="changeDirectGroupUpdatePageSize"
                />
              </div>

              <div class="space-y-3 rounded-md bg-gray-50 p-3 dark:bg-dark-900/40">
                <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                  <input
                    v-model="directGroupUpdateRateEnabled"
                    data-test="direct-group-update-rate-enabled"
                    class="checkbox"
                    type="checkbox"
                  />
                  <span>{{ t('agentManagement.groups.updateRateEnabled') }}</span>
                </label>
                <input
                  v-model.number="directGroupUpdateRate"
                  data-test="direct-group-update-rate"
                  class="input h-9"
                  type="number"
                  min="0.000001"
                  step="0.000001"
                  :disabled="!directGroupUpdateRateEnabled"
                />

                <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                  <input
                    v-model="directGroupUpdateCanDelegateEnabled"
                    data-test="direct-group-update-can-delegate-enabled"
                    class="checkbox"
                    type="checkbox"
                  />
                  <span>{{ t('agentManagement.groups.updateCanDelegateEnabled') }}</span>
                </label>
                <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
                  <input
                    v-model="directGroupUpdateCanDelegate"
                    data-test="direct-group-update-can-delegate"
                    class="checkbox"
                    type="checkbox"
                    :disabled="!directGroupUpdateCanDelegateEnabled"
                  />
                  <span>{{ t('agentManagement.groups.allowChildDelegate') }}</span>
                </label>

                <div class="flex justify-end gap-2 pt-2">
                  <button class="btn btn-secondary px-3" type="button" @click="closeDirectGroupUpdateDialog">
                    {{ t('common.cancel') }}
                  </button>
                  <button
                    data-test="apply-direct-group-update"
                    class="btn btn-primary px-3"
                    type="button"
                    :disabled="directGroupUpdateDialog.saving"
                    @click="applyDirectGroupUpdate"
                  >
                    <Icon name="check" size="sm" />
                    <span>{{ t('agentManagement.groups.applyExistingUpdate') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog
      :show="directGroupReclaimDialog.show"
      :title="t('agentManagement.groups.directReclaimTitle')"
      width="wide"
      @close="closeDirectGroupReclaimDialog"
    >
      <div class="space-y-4" data-test="direct-group-reclaim-modal">
        <div v-if="directGroupReclaimDialog.loading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="directGroupReclaimDialog.groups.length === 0"
          class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
        >
          {{ t('agentManagement.groups.emptyDelegable') }}
        </div>

        <div v-else class="space-y-4">
          <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
            <div class="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto] lg:items-end">
              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.groups.reclaimGroup') }}</span>
                <select
                  v-model.number="directGroupReclaimDialog.selectedGroupId"
                  data-test="direct-group-reclaim-group"
                  class="input h-9"
                  @change="onDirectGroupReclaimGroupChange"
                >
                  <option
                    v-for="groupRate in directGroupReclaimDialog.groups"
                    :key="groupRate.group.id"
                    :value="groupRate.group.id"
                  >
                    {{ groupRate.group.name }}
                  </option>
                </select>
              </label>

              <label class="flex flex-col gap-1 text-xs text-gray-500 dark:text-dark-400">
                <span>{{ t('agentManagement.groups.reclaimSearch') }}</span>
                <input
                  v-model="directGroupReclaimDialog.searchDraft"
                  data-test="direct-group-reclaim-search"
                  class="input h-9"
                  type="search"
                  :placeholder="t('common.search')"
                  @keyup.enter="applyDirectGroupReclaimSearch"
                />
              </label>

              <button
                data-test="direct-group-reclaim-search-submit"
                class="btn btn-secondary h-9 px-3"
                type="button"
                :disabled="directGroupReclaimDialog.childrenLoading"
                @click="applyDirectGroupReclaimSearch"
              >
                <Icon name="search" size="sm" />
                <span>{{ t('common.search') }}</span>
              </button>
            </div>
          </div>

          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <label class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200">
                <input
                  data-test="direct-group-reclaim-all"
                  class="checkbox"
                  type="checkbox"
                  :checked="directGroupReclaimAll"
                  @change="setDirectGroupReclaimAll(($event.target as HTMLInputElement).checked)"
                />
                <span>{{ t('agentManagement.groups.reclaimAllAssigned') }}</span>
              </label>
              <span class="text-sm text-gray-500 dark:text-dark-400">
                {{ t('agentManagement.groups.updateSelectedChildren', { count: selectedDirectGroupReclaimChildIDs.length }) }}
              </span>
            </div>

            <div v-if="directGroupReclaimDialog.childrenLoading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
              {{ t('common.loading') }}
            </div>
            <div
              v-else-if="directGroupReclaimDialog.children.length === 0"
              class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
            >
              {{ t('agentManagement.groups.emptyAssignedChildren') }}
            </div>
            <div v-else class="max-h-80 space-y-2 overflow-y-auto pr-1">
              <label
                v-for="child in directGroupReclaimDialog.children"
                :key="child.id"
                class="flex items-center gap-3 rounded-md border border-gray-200 px-3 py-2 dark:border-dark-700"
              >
                <input
                  :data-test="`direct-group-reclaim-child-${child.id}`"
                  class="checkbox"
                  type="checkbox"
                  :checked="selectedDirectGroupReclaimChildIDs.includes(child.id)"
                  :disabled="directGroupReclaimAll"
                  @change="updateDirectGroupReclaimChildSelection(child.id, ($event.target as HTMLInputElement).checked)"
                />
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ child.email }}</span>
                  <span class="mt-0.5 block text-xs text-gray-500 dark:text-dark-400">{{ child.username || '-' }}</span>
                </span>
                <span class="text-xs font-medium text-gray-600 dark:text-dark-300">
                  {{ t('agentManagement.groups.currentRate') }}: {{ currentDirectGroupReclaimChildRate(child) }}
                </span>
              </label>
            </div>

            <Pagination
              v-if="directGroupReclaimDialog.pagination.total > directGroupReclaimDialog.pagination.page_size"
              :page="directGroupReclaimDialog.pagination.page"
              :total="directGroupReclaimDialog.pagination.total"
              :page-size="directGroupReclaimDialog.pagination.page_size"
              :show-page-size-selector="false"
              @update:page="changeDirectGroupReclaimPage"
              @update:page-size="changeDirectGroupReclaimPageSize"
            />

            <div class="flex justify-end gap-2 pt-4">
              <button class="btn btn-secondary px-3" type="button" @click="closeDirectGroupReclaimDialog">
                {{ t('common.cancel') }}
              </button>
              <button
                data-test="apply-direct-group-reclaim"
                class="btn btn-primary px-3"
                type="button"
                :disabled="directGroupReclaimDialog.saving"
                @click="applyDirectGroupReclaim"
              >
                <Icon name="trash" size="sm" />
                <span>{{ t('agentManagement.groups.applyReclaim') }}</span>
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
  AgentDirectChildKind,
  AgentDirectChildrenResponse,
  AgentDirectUserCreateRequest,
  AgentGroupRate,
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
const agentIncomeDraft = reactive({
  agent_income: 0,
  reason: '',
})
const drafts = reactive<Record<number, AgentAllocationUpdate>>({})
const pagination = reactive({ total: 0, page: 1, page_size: 20, pages: 1 })
const deleteDialog = reactive<{ show: boolean; child: AgentManagedUser | null }>({ show: false, child: null })
const upgradeDialog = reactive<{ show: boolean; child: AgentManagedUser | null; targetRole: AgentUpgradeTargetRole | null }>({
  show: false,
  child: null,
  targetRole: null,
})
const agentIncomeDialog = reactive<{
  show: boolean
  child: AgentManagedUser | null
  saving: boolean
}>({
  show: false,
  child: null,
  saving: false,
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
const directGroupBatchDialog = reactive<{
  show: boolean
  loading: boolean
  saving: boolean
  childrenLoading: boolean
  groups: AgentChildGroupDelegationOption[]
  children: AgentManagedUser[]
  searchDraft: string
  search: string
  pagination: { total: number; page: number; page_size: number; pages: number }
}>({
  show: false,
  loading: false,
  saving: false,
  childrenLoading: false,
  groups: [],
  children: [],
  searchDraft: '',
  search: '',
  pagination: { total: 0, page: 1, page_size: 20, pages: 1 },
})
const directGroupUpdateDialog = reactive<{
  show: boolean
  loading: boolean
  saving: boolean
  childrenLoading: boolean
  groups: AgentChildGroupDelegationOption[]
  selectedGroupId: number
  children: AgentManagedUser[]
  searchDraft: string
  search: string
  pagination: { total: number; page: number; page_size: number; pages: number }
}>({
  show: false,
  loading: false,
  saving: false,
  childrenLoading: false,
  groups: [],
  selectedGroupId: 0,
  children: [],
  searchDraft: '',
  search: '',
  pagination: { total: 0, page: 1, page_size: 20, pages: 1 },
})
const directGroupReclaimDialog = reactive<{
  show: boolean
  loading: boolean
  saving: boolean
  childrenLoading: boolean
  groups: AgentChildGroupDelegationOption[]
  selectedGroupId: number
  children: AgentManagedUser[]
  searchDraft: string
  search: string
  pagination: { total: number; page: number; page_size: number; pages: number }
}>({
  show: false,
  loading: false,
  saving: false,
  childrenLoading: false,
  groups: [],
  selectedGroupId: 0,
  children: [],
  searchDraft: '',
  search: '',
  pagination: { total: 0, page: 1, page_size: 20, pages: 1 },
})
const groupDrafts = reactive<Record<number, { assigned: boolean; rate_multiplier: number; can_delegate: boolean }>>({})
const groupBatchAll = ref(false)
const groupBatchRate = ref(1)
const groupBatchCanDelegate = ref(false)
const groupBatchSaving = ref(false)
const selectedGroupBatchIDs = ref<number[]>([])
const directGroupBatchAllChildren = ref(true)
const selectedDirectGroupBatchChildIDs = ref<number[]>([])
const directGroupBatchAll = ref(false)
const directGroupBatchRate = ref(1)
const directGroupBatchCanDelegate = ref(false)
const selectedDirectGroupBatchIDs = ref<number[]>([])
const directGroupUpdateAll = ref(false)
const selectedDirectGroupUpdateChildIDs = ref<number[]>([])
const directGroupUpdateRateEnabled = ref(true)
const directGroupUpdateRate = ref(1)
const directGroupUpdateCanDelegateEnabled = ref(false)
const directGroupUpdateCanDelegate = ref(false)
const directGroupReclaimAll = ref(false)
const selectedDirectGroupReclaimChildIDs = ref<number[]>([])

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
const directChildKind = computed<AgentDirectChildKind>(() => props.kind)
const directGroupBatchSelectedGroupIDs = computed(() => directGroupBatchAll.value
  ? directGroupBatchDialog.groups.map((item) => item.group.id)
  : selectedDirectGroupBatchIDs.value
)
const directGroupBatchHasGroupSelection = computed(() => directGroupBatchSelectedGroupIDs.value.length > 0)

function paginationFromResult(result: AgentDirectChildrenResponse) {
  const source = result.pagination || {}
  return {
    total: Number(source.total ?? source.Total ?? result.items.length),
    page: Number(source.page ?? source.Page ?? 1),
    page_size: Number(source.page_size ?? source.PageSize ?? 20),
    pages: Number(source.pages ?? source.Pages ?? 1),
  }
}

function extractPagination(result: AgentDirectChildrenResponse) {
  const next = paginationFromResult(result)
  pagination.total = next.total
  pagination.page = next.page
  pagination.page_size = next.page_size
  pagination.pages = next.pages
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

function canSetAgentIncome(child: AgentManagedUser): boolean {
  return isAdmin.value && props.kind === 'agents' && child.role === 'agent_level1'
}

function openAgentIncomeDialog(child: AgentManagedUser) {
  agentIncomeDialog.child = child
  agentIncomeDialog.show = true
  agentIncomeDraft.agent_income = Number(child.agent_income || 0)
  agentIncomeDraft.reason = ''
}

function closeAgentIncomeDialog() {
  agentIncomeDialog.show = false
  agentIncomeDialog.child = null
  agentIncomeDialog.saving = false
}

async function saveAgentIncome() {
  if (!agentIncomeDialog.child) return
  const targetIncome = Number(agentIncomeDraft.agent_income)
  if (!Number.isFinite(targetIncome)) {
    appStore.showError(t('agentManagement.direct.agentIncomeInvalid'))
    return
  }
  const child = agentIncomeDialog.child
  agentIncomeDialog.saving = true
  try {
    await agentManagementAPI.setAgentIncome(child.id, {
      agent_income: targetIncome,
      reason: agentIncomeDraft.reason.trim(),
    })
    appStore.showSuccess(t('agentManagement.direct.agentIncomeSaved'))
    closeAgentIncomeDialog()
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.direct.agentIncomeFailed'))
  } finally {
    agentIncomeDialog.saving = false
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
  selectedGroupBatchIDs.value = []
  groupBatchAll.value = false
  groupBatchRate.value = groups[0]?.effective_rate ?? 1
  groupBatchCanDelegate.value = false
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

function setGroupBatchAll(all: boolean) {
  groupBatchAll.value = all
  if (all) {
    selectedGroupBatchIDs.value = []
  }
}

function updateGroupBatchSelection(groupID: number, selected: boolean) {
  const next = new Set(selectedGroupBatchIDs.value)
  if (selected) {
    next.add(groupID)
  } else {
    next.delete(groupID)
  }
  selectedGroupBatchIDs.value = Array.from(next)
}

function syncDirectGroupBatchGroups(groups: AgentGroupRate[]) {
  directGroupBatchDialog.groups = groups
    .filter((item) => item.can_delegate && item.group.is_exclusive)
    .map((item) => ({
      ...item,
      assigned: false,
      child_rate_multiplier: item.effective_rate,
      child_can_delegate: false,
    }))
  selectedDirectGroupBatchIDs.value = []
  directGroupBatchAll.value = false
  directGroupBatchRate.value = directGroupBatchDialog.groups[0]?.effective_rate ?? 1
  directGroupBatchCanDelegate.value = false
  directGroupBatchDialog.children = []
  directGroupBatchDialog.searchDraft = ''
  directGroupBatchDialog.search = ''
  directGroupBatchDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
}

async function setDirectGroupBatchAll(all: boolean) {
  directGroupBatchAll.value = all
  if (all) {
    selectedDirectGroupBatchIDs.value = []
  }
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAllChildren.value = true
  directGroupBatchDialog.pagination.page = 1
  await loadDirectGroupBatchChildren()
}

function setDirectGroupBatchAllChildren(all: boolean) {
  directGroupBatchAllChildren.value = all
  if (all) {
    selectedDirectGroupBatchChildIDs.value = []
  }
}

async function updateDirectGroupBatchSelection(groupID: number, selected: boolean) {
  const next = new Set(selectedDirectGroupBatchIDs.value)
  if (selected) {
    next.add(groupID)
  } else {
    next.delete(groupID)
  }
  selectedDirectGroupBatchIDs.value = Array.from(next)
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAllChildren.value = true
  directGroupBatchDialog.pagination.page = 1
  const groupRate = directGroupBatchDialog.groups.find((item) => item.group.id === groupID)
  if (selected && groupRate) {
    directGroupBatchRate.value = groupRate.effective_rate
  }
  await loadDirectGroupBatchChildren()
}

function updateDirectGroupBatchChildSelection(childID: number, selected: boolean) {
  const next = new Set(selectedDirectGroupBatchChildIDs.value)
  if (selected) {
    next.add(childID)
  } else {
    next.delete(childID)
  }
  selectedDirectGroupBatchChildIDs.value = Array.from(next)
}

async function openDirectGroupBatchDialog() {
  directGroupBatchDialog.show = true
  directGroupBatchDialog.loading = true
  directGroupBatchDialog.childrenLoading = false
  directGroupBatchDialog.groups = []
  directGroupBatchDialog.children = []
  directGroupBatchDialog.searchDraft = ''
  directGroupBatchDialog.search = ''
  directGroupBatchDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
  directGroupBatchAllChildren.value = true
  selectedDirectGroupBatchChildIDs.value = []
  try {
    syncDirectGroupBatchGroups(await agentManagementAPI.listGroups())
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupBatchDialog.loading = false
  }
}

function closeDirectGroupBatchDialog() {
  directGroupBatchDialog.show = false
  directGroupBatchDialog.loading = false
  directGroupBatchDialog.saving = false
  directGroupBatchDialog.childrenLoading = false
  directGroupBatchDialog.groups = []
  directGroupBatchDialog.children = []
  directGroupBatchDialog.searchDraft = ''
  directGroupBatchDialog.search = ''
  directGroupBatchDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
  selectedDirectGroupBatchIDs.value = []
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAll.value = false
  directGroupBatchAllChildren.value = true
}

async function applyDirectGroupBatchSearch() {
  directGroupBatchDialog.search = directGroupBatchDialog.searchDraft.trim()
  directGroupBatchDialog.pagination.page = 1
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAllChildren.value = true
  await loadDirectGroupBatchChildren()
}

async function changeDirectGroupBatchPage(page: number) {
  directGroupBatchDialog.pagination.page = page
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAllChildren.value = true
  await loadDirectGroupBatchChildren()
}

async function changeDirectGroupBatchPageSize(pageSize: number) {
  directGroupBatchDialog.pagination.page_size = pageSize
  directGroupBatchDialog.pagination.page = 1
  selectedDirectGroupBatchChildIDs.value = []
  directGroupBatchAllChildren.value = true
  await loadDirectGroupBatchChildren()
}

async function loadDirectGroupBatchChildren() {
  const groupIDs = directGroupBatchSelectedGroupIDs.value
  if (groupIDs.length === 0) {
    directGroupBatchDialog.children = []
    directGroupBatchDialog.pagination = { total: 0, page: 1, page_size: directGroupBatchDialog.pagination.page_size, pages: 1 }
    return
  }
  directGroupBatchDialog.childrenLoading = true
  try {
    const result = await agentManagementAPI.listDirectChildrenWithoutGroupDelegation(directChildKind.value, {
      group_ids: groupIDs,
      ...(directGroupBatchDialog.search ? { search: directGroupBatchDialog.search } : {}),
      page: directGroupBatchDialog.pagination.page,
      page_size: directGroupBatchDialog.pagination.page_size,
    })
    directGroupBatchDialog.children = result.items
    directGroupBatchDialog.pagination = paginationFromResult(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupBatchDialog.childrenLoading = false
  }
}

function syncDirectGroupUpdateGroups(groups: AgentGroupRate[]) {
  directGroupUpdateDialog.groups = groups
    .filter((item) => item.can_delegate && item.group.is_exclusive)
    .map((item) => ({
      ...item,
      assigned: false,
      child_rate_multiplier: item.effective_rate,
      child_can_delegate: false,
    }))
  directGroupUpdateDialog.selectedGroupId = directGroupUpdateDialog.groups[0]?.group.id ?? 0
  const first = directGroupUpdateDialog.groups[0]
  directGroupUpdateRate.value = first?.effective_rate ?? 1
  directGroupUpdateCanDelegate.value = false
}

function syncDirectGroupReclaimGroups(groups: AgentGroupRate[]) {
  directGroupReclaimDialog.groups = groups
    .filter((item) => item.can_delegate && item.group.is_exclusive)
    .map((item) => ({
      ...item,
      assigned: false,
      child_rate_multiplier: item.effective_rate,
      child_can_delegate: false,
    }))
  directGroupReclaimDialog.selectedGroupId = directGroupReclaimDialog.groups[0]?.group.id ?? 0
}

async function openDirectGroupUpdateDialog() {
  directGroupUpdateDialog.show = true
  directGroupUpdateDialog.loading = true
  directGroupUpdateDialog.childrenLoading = false
  directGroupUpdateDialog.groups = []
  directGroupUpdateDialog.children = []
  directGroupUpdateDialog.searchDraft = ''
  directGroupUpdateDialog.search = ''
  directGroupUpdateDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
  directGroupUpdateRateEnabled.value = true
  directGroupUpdateCanDelegateEnabled.value = false
  try {
    syncDirectGroupUpdateGroups(await agentManagementAPI.listGroups())
    if (directGroupUpdateDialog.selectedGroupId > 0) {
      await loadDirectGroupUpdateChildren()
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupUpdateDialog.loading = false
  }
}

async function openDirectGroupReclaimDialog() {
  directGroupReclaimDialog.show = true
  directGroupReclaimDialog.loading = true
  directGroupReclaimDialog.childrenLoading = false
  directGroupReclaimDialog.groups = []
  directGroupReclaimDialog.children = []
  directGroupReclaimDialog.searchDraft = ''
  directGroupReclaimDialog.search = ''
  directGroupReclaimDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
  try {
    syncDirectGroupReclaimGroups(await agentManagementAPI.listGroups())
    if (directGroupReclaimDialog.selectedGroupId > 0) {
      await loadDirectGroupReclaimChildren()
    }
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupReclaimDialog.loading = false
  }
}

function closeDirectGroupUpdateDialog() {
  directGroupUpdateDialog.show = false
  directGroupUpdateDialog.loading = false
  directGroupUpdateDialog.saving = false
  directGroupUpdateDialog.childrenLoading = false
  directGroupUpdateDialog.groups = []
  directGroupUpdateDialog.children = []
  directGroupUpdateDialog.selectedGroupId = 0
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
}

function closeDirectGroupReclaimDialog() {
  directGroupReclaimDialog.show = false
  directGroupReclaimDialog.loading = false
  directGroupReclaimDialog.saving = false
  directGroupReclaimDialog.childrenLoading = false
  directGroupReclaimDialog.groups = []
  directGroupReclaimDialog.children = []
  directGroupReclaimDialog.selectedGroupId = 0
  directGroupReclaimDialog.searchDraft = ''
  directGroupReclaimDialog.search = ''
  directGroupReclaimDialog.pagination = { total: 0, page: 1, page_size: 20, pages: 1 }
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
}

async function onDirectGroupUpdateGroupChange() {
  directGroupUpdateDialog.pagination.page = 1
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
  const groupRate = directGroupUpdateDialog.groups.find((item) => item.group.id === directGroupUpdateDialog.selectedGroupId)
  directGroupUpdateRate.value = groupRate?.effective_rate ?? 1
  directGroupUpdateCanDelegate.value = false
  await loadDirectGroupUpdateChildren()
}

async function onDirectGroupReclaimGroupChange() {
  directGroupReclaimDialog.pagination.page = 1
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
  await loadDirectGroupReclaimChildren()
}

async function applyDirectGroupUpdateSearch() {
  directGroupUpdateDialog.search = directGroupUpdateDialog.searchDraft.trim()
  directGroupUpdateDialog.pagination.page = 1
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
  await loadDirectGroupUpdateChildren()
}

async function applyDirectGroupReclaimSearch() {
  directGroupReclaimDialog.search = directGroupReclaimDialog.searchDraft.trim()
  directGroupReclaimDialog.pagination.page = 1
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
  await loadDirectGroupReclaimChildren()
}

async function changeDirectGroupUpdatePage(page: number) {
  directGroupUpdateDialog.pagination.page = page
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
  await loadDirectGroupUpdateChildren()
}

async function changeDirectGroupReclaimPage(page: number) {
  directGroupReclaimDialog.pagination.page = page
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
  await loadDirectGroupReclaimChildren()
}

async function changeDirectGroupUpdatePageSize(pageSize: number) {
  directGroupUpdateDialog.pagination.page_size = pageSize
  directGroupUpdateDialog.pagination.page = 1
  selectedDirectGroupUpdateChildIDs.value = []
  directGroupUpdateAll.value = false
  await loadDirectGroupUpdateChildren()
}

async function changeDirectGroupReclaimPageSize(pageSize: number) {
  directGroupReclaimDialog.pagination.page_size = pageSize
  directGroupReclaimDialog.pagination.page = 1
  selectedDirectGroupReclaimChildIDs.value = []
  directGroupReclaimAll.value = false
  await loadDirectGroupReclaimChildren()
}

async function loadDirectGroupUpdateChildren() {
  if (directGroupUpdateDialog.selectedGroupId <= 0) return
  directGroupUpdateDialog.childrenLoading = true
  try {
    const result = await agentManagementAPI.listDirectChildrenWithGroupDelegation(directChildKind.value, {
      group_id: directGroupUpdateDialog.selectedGroupId,
      ...(directGroupUpdateDialog.search ? { search: directGroupUpdateDialog.search } : {}),
      page: directGroupUpdateDialog.pagination.page,
      page_size: directGroupUpdateDialog.pagination.page_size,
    })
    directGroupUpdateDialog.children = result.items
    directGroupUpdateDialog.pagination = paginationFromResult(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupUpdateDialog.childrenLoading = false
  }
}

async function loadDirectGroupReclaimChildren() {
  if (directGroupReclaimDialog.selectedGroupId <= 0) return
  directGroupReclaimDialog.childrenLoading = true
  try {
    const result = await agentManagementAPI.listDirectChildrenWithGroupDelegation(directChildKind.value, {
      group_id: directGroupReclaimDialog.selectedGroupId,
      ...(directGroupReclaimDialog.search ? { search: directGroupReclaimDialog.search } : {}),
      page: directGroupReclaimDialog.pagination.page,
      page_size: directGroupReclaimDialog.pagination.page_size,
    })
    directGroupReclaimDialog.children = result.items
    directGroupReclaimDialog.pagination = paginationFromResult(result)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.loadFailed'))
  } finally {
    directGroupReclaimDialog.childrenLoading = false
  }
}

function setDirectGroupUpdateAll(all: boolean) {
  directGroupUpdateAll.value = all
  if (all) {
    selectedDirectGroupUpdateChildIDs.value = []
  }
}

function setDirectGroupReclaimAll(all: boolean) {
  directGroupReclaimAll.value = all
  if (all) {
    selectedDirectGroupReclaimChildIDs.value = []
  }
}

function updateDirectGroupUpdateChildSelection(childID: number, selected: boolean) {
  const next = new Set(selectedDirectGroupUpdateChildIDs.value)
  if (selected) {
    next.add(childID)
  } else {
    next.delete(childID)
  }
  selectedDirectGroupUpdateChildIDs.value = Array.from(next)
}

function updateDirectGroupReclaimChildSelection(childID: number, selected: boolean) {
  const next = new Set(selectedDirectGroupReclaimChildIDs.value)
  if (selected) {
    next.add(childID)
  } else {
    next.delete(childID)
  }
  selectedDirectGroupReclaimChildIDs.value = Array.from(next)
}

function currentDirectGroupUpdateChildRate(child: AgentManagedUser): string {
  const rate = child.group_rates?.[directGroupUpdateDialog.selectedGroupId]
  if (rate == null) {
    return '-'
  }
  return String(rate)
}

function currentDirectGroupReclaimChildRate(child: AgentManagedUser): string {
  const rate = child.group_rates?.[directGroupReclaimDialog.selectedGroupId]
  if (rate == null) {
    return '-'
  }
  return String(rate)
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
  groupBatchSaving.value = false
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

async function applyGroupDelegationBatch() {
  if (!groupDialog.child) return
  const rateMultiplier = normalizedPositiveFloat(groupBatchRate.value)
  if (rateMultiplier <= 0) {
    appStore.showError(t('agentManagement.groups.invalidRate'))
    return
  }
  if (!groupBatchAll.value && selectedGroupBatchIDs.value.length === 0) {
    appStore.showError(t('agentManagement.groups.batchSelectionRequired'))
    return
  }

  groupBatchSaving.value = true
  try {
    await agentManagementAPI.setChildGroupDelegationsBatch(groupDialog.child.id, {
      group_ids: groupBatchAll.value ? [] : selectedGroupBatchIDs.value,
      all: groupBatchAll.value,
      rate_multiplier: rateMultiplier,
      can_delegate: groupBatchCanDelegate.value,
    })
    appStore.showSuccess(t('agentManagement.groups.batchDelegationSaved'))
    const groups = await agentManagementAPI.listChildGroupDelegationOptions(groupDialog.child.id)
    groupDialog.groups = groups.filter((item) => item.can_delegate && item.group.is_exclusive)
    syncGroupDrafts(groupDialog.groups)
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.batchDelegationFailed'))
  } finally {
    groupBatchSaving.value = false
  }
}

async function applyDirectGroupBatch() {
  const rateMultiplier = normalizedPositiveFloat(directGroupBatchRate.value)
  if (rateMultiplier <= 0) {
    appStore.showError(t('agentManagement.groups.invalidRate'))
    return
  }
  if (!directGroupBatchAll.value && selectedDirectGroupBatchIDs.value.length === 0) {
    appStore.showError(t('agentManagement.groups.batchSelectionRequired'))
    return
  }
  if (!directGroupBatchAllChildren.value && selectedDirectGroupBatchChildIDs.value.length === 0) {
    appStore.showError(t('agentManagement.groups.childSelectionRequired'))
    return
  }

  directGroupBatchDialog.saving = true
  try {
    await agentManagementAPI.setDirectChildrenGroupDelegationsBatch(directChildKind.value, {
      group_ids: directGroupBatchAll.value ? [] : selectedDirectGroupBatchIDs.value,
      all: directGroupBatchAll.value,
      child_ids: directGroupBatchAllChildren.value ? [] : selectedDirectGroupBatchChildIDs.value,
      all_children: directGroupBatchAllChildren.value,
      rate_multiplier: rateMultiplier,
      can_delegate: directGroupBatchCanDelegate.value,
    })
    appStore.showSuccess(t('agentManagement.groups.directBatchSaved'))
    closeDirectGroupBatchDialog()
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.directBatchFailed'))
  } finally {
    directGroupBatchDialog.saving = false
  }
}

async function applyDirectGroupUpdate() {
  if (directGroupUpdateDialog.selectedGroupId <= 0) {
    appStore.showError(t('agentManagement.groups.updateGroupRequired'))
    return
  }
  if (!directGroupUpdateAll.value && selectedDirectGroupUpdateChildIDs.value.length === 0) {
    appStore.showError(t('agentManagement.groups.childSelectionRequired'))
    return
  }
  if (!directGroupUpdateRateEnabled.value && !directGroupUpdateCanDelegateEnabled.value) {
    appStore.showError(t('agentManagement.groups.updateFieldRequired'))
    return
  }

  const payload: {
    group_id: number
    child_ids: number[]
    all: boolean
    rate_multiplier?: number
    can_delegate?: boolean
  } = {
    group_id: directGroupUpdateDialog.selectedGroupId,
    child_ids: directGroupUpdateAll.value ? [] : selectedDirectGroupUpdateChildIDs.value,
    all: directGroupUpdateAll.value,
  }

  if (directGroupUpdateRateEnabled.value) {
    const rateMultiplier = normalizedPositiveFloat(directGroupUpdateRate.value)
    if (rateMultiplier <= 0) {
      appStore.showError(t('agentManagement.groups.invalidRate'))
      return
    }
    payload.rate_multiplier = rateMultiplier
  }
  if (directGroupUpdateCanDelegateEnabled.value) {
    payload.can_delegate = directGroupUpdateCanDelegate.value
  }

  directGroupUpdateDialog.saving = true
  try {
    const result = await agentManagementAPI.updateDirectChildrenExistingGroupDelegations(directChildKind.value, payload)
    appStore.showSuccess(t('agentManagement.groups.directUpdateSaved', {
      updated: result.updated_children,
      skipped: result.skipped_children,
    }))
    await loadDirectGroupUpdateChildren()
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.directUpdateFailed'))
  } finally {
    directGroupUpdateDialog.saving = false
  }
}

async function applyDirectGroupReclaim() {
  if (directGroupReclaimDialog.selectedGroupId <= 0) {
    appStore.showError(t('agentManagement.groups.reclaimGroupRequired'))
    return
  }
  if (!directGroupReclaimAll.value && selectedDirectGroupReclaimChildIDs.value.length === 0) {
    appStore.showError(t('agentManagement.groups.childSelectionRequired'))
    return
  }

  directGroupReclaimDialog.saving = true
  try {
    const result = await agentManagementAPI.reclaimDirectChildrenGroupDelegations(directChildKind.value, {
      group_id: directGroupReclaimDialog.selectedGroupId,
      child_ids: directGroupReclaimAll.value ? [] : selectedDirectGroupReclaimChildIDs.value,
      all: directGroupReclaimAll.value,
    })
    appStore.showSuccess(t('agentManagement.groups.directReclaimSaved', {
      removed: result.removed_children,
      skipped: result.skipped_children,
    }))
    selectedDirectGroupReclaimChildIDs.value = []
    directGroupReclaimAll.value = false
    await loadDirectGroupReclaimChildren()
    await loadData()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('agentManagement.groups.directReclaimFailed'))
  } finally {
    directGroupReclaimDialog.saving = false
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
