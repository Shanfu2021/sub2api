<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('purchaseInfo.description') }}</p>
        </div>
        <button class="btn btn-secondary w-full justify-center px-3 sm:w-auto" :disabled="isBusy" @click="loadAll">
          <Icon name="refresh" size="sm" :class="isBusy ? 'animate-spin' : ''" />
        </button>
      </header>

      <section class="space-y-3">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.availableTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('purchaseInfo.availableDescription') }}</p>
          </div>
          <span v-if="!loadingVisible && visibleCards.length > 0" class="inline-flex h-7 items-center rounded-md bg-gray-100 px-2.5 text-sm font-medium text-gray-700 dark:bg-dark-700 dark:text-dark-200">
            {{ t('purchaseInfo.cardCount', { count: visibleCards.length }) }}
          </span>
        </div>

        <div v-if="loadingVisible" class="flex min-h-[180px] items-center justify-center rounded-lg border border-gray-200 bg-white px-4 py-8 text-sm text-gray-500 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
          <div class="flex items-center gap-2">
            <Icon name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('common.loading') }}</span>
          </div>
        </div>
        <div v-else-if="visibleCards.length === 0" class="flex min-h-[180px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
          <span class="mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
            <Icon name="link" size="md" />
          </span>
          <span>{{ t('purchaseInfo.emptyVisible') }}</span>
        </div>
        <div v-else class="grid gap-4 lg:grid-cols-2">
          <article
            v-for="card in visibleCards"
            :key="card.id"
            class="flex min-h-[300px] flex-col overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm transition hover:border-blue-200 hover:shadow-md dark:border-dark-700 dark:bg-dark-800 dark:hover:border-blue-900"
          >
            <div class="flex items-start gap-3 border-b border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
              <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-950/40 dark:text-blue-300">
                <Icon name="gift" size="md" />
              </span>
              <div class="min-w-0 flex-1">
                <h3 class="break-words text-base font-semibold text-gray-900 dark:text-white">{{ card.title }}</h3>
                <p v-if="card.description" class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-600 dark:text-dark-300">{{ card.description }}</p>
              </div>
            </div>
            <div class="flex flex-1 flex-col gap-4 p-4">
              <div v-if="card.purchase_url" class="rounded-md border border-gray-100 p-3 dark:border-dark-700">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <div class="mb-1 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
                      <Icon name="link" size="sm" />
                      <span>{{ t('purchaseInfo.purchaseUrl') }}</span>
                    </div>
                    <a
                      :href="card.purchase_url"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="block break-all text-sm text-gray-600 hover:text-blue-600 dark:text-dark-300 dark:hover:text-blue-300"
                    >
                      {{ card.purchase_url }}
                    </a>
                  </div>
                  <a
                    :href="card.purchase_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="btn btn-primary btn-sm w-full justify-center sm:w-auto sm:shrink-0"
                  >
                    <Icon name="externalLink" size="sm" />
                    <span>{{ t('purchaseInfo.openLink') }}</span>
                  </a>
                </div>
              </div>

              <div v-if="card.contact" class="rounded-md border border-gray-100 p-3 dark:border-dark-700">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div class="min-w-0">
                    <div class="mb-1 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
                      <Icon name="chat" size="sm" />
                      <span>{{ t('purchaseInfo.contact') }}</span>
                    </div>
                    <p class="break-all text-sm text-gray-600 dark:text-dark-300">{{ card.contact }}</p>
                  </div>
                  <button type="button" class="btn btn-secondary btn-sm w-full justify-center sm:w-auto sm:shrink-0" @click="copyContact(card.contact)">
                    <Icon name="copy" size="sm" />
                    <span>{{ t('purchaseInfo.copyContact') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section v-if="canManage" class="space-y-3 border-t border-gray-200 pt-6 dark:border-dark-700">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div class="min-w-0">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.manageTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('purchaseInfo.manageDescription') }}</p>
          </div>
          <button class="btn btn-primary w-full justify-center sm:w-auto" @click="openCreate">
            <Icon name="plus" size="sm" />
            <span>{{ t('purchaseInfo.addCard') }}</span>
          </button>
        </div>

        <div v-if="loadingManaged" class="flex min-h-[150px] items-center justify-center rounded-lg border border-gray-200 bg-white px-4 py-8 text-sm text-gray-500 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
          <div class="flex items-center gap-2">
            <Icon name="refresh" size="sm" class="animate-spin" />
            <span>{{ t('common.loading') }}</span>
          </div>
        </div>
        <div v-else-if="managedCards.length === 0" class="flex min-h-[150px] flex-col items-center justify-center rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
          <span class="mb-3 flex h-11 w-11 items-center justify-center rounded-lg bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-300">
            <Icon name="gift" size="md" />
          </span>
          <span>{{ t('purchaseInfo.emptyManaged') }}</span>
        </div>
        <div v-else class="grid gap-3 lg:grid-cols-2">
          <article
            v-for="card in managedCards"
            :key="card.id"
            class="flex min-h-[230px] flex-col rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="break-words text-base font-semibold text-gray-900 dark:text-white">{{ card.title }}</h3>
                  <span :class="card.enabled ? 'badge badge-green' : 'badge badge-gray'">
                    {{ card.enabled ? t('common.enabled') : t('common.disabled') }}
                  </span>
                  <span class="badge badge-gray">{{ t('purchaseInfo.sortOrderShort', { order: card.sort_order }) }}</span>
                </div>
                <p v-if="card.description" class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-gray-600 dark:text-dark-300">{{ card.description }}</p>
              </div>
              <div class="flex shrink-0 items-center gap-2">
                <button class="btn btn-secondary btn-sm flex-1 justify-center sm:flex-none" @click="openEdit(card)">
                  <Icon name="edit" size="sm" />
                  <span>{{ t('common.edit') }}</span>
                </button>
                <button class="btn btn-danger btn-sm flex-1 justify-center sm:flex-none" :disabled="deletingId === card.id" @click="deleteManagedCard(card)">
                  <Icon :name="deletingId === card.id ? 'refresh' : 'trash'" size="sm" :class="deletingId === card.id ? 'animate-spin' : ''" />
                  <span>{{ t('common.delete') }}</span>
                </button>
              </div>
            </div>

            <div class="mt-auto space-y-3 border-t border-gray-100 pt-4 dark:border-dark-700">
              <div v-if="card.purchase_url" class="min-w-0">
                <div class="mb-1 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
                  <Icon name="link" size="sm" />
                  <span>{{ t('purchaseInfo.purchaseUrl') }}</span>
                </div>
                <p class="break-all text-sm text-gray-500 dark:text-dark-400">{{ card.purchase_url }}</p>
              </div>
              <div v-if="card.contact" class="min-w-0">
                <div class="mb-1 flex items-center gap-2 text-sm font-medium text-gray-900 dark:text-white">
                  <Icon name="chat" size="sm" />
                  <span>{{ t('purchaseInfo.contact') }}</span>
                </div>
                <p class="break-all text-sm text-gray-500 dark:text-dark-400">{{ card.contact }}</p>
              </div>
            </div>
          </article>
        </div>
      </section>
    </div>

    <BaseDialog :show="showEditor" :title="editingCard ? t('purchaseInfo.editCard') : t('purchaseInfo.addCard')" @close="closeEditor">
      <form class="space-y-4" @submit.prevent="saveCard">
        <div>
          <label class="input-label">{{ t('purchaseInfo.cardTitle') }}</label>
          <input v-model="form.title" class="input" type="text" maxlength="100" required />
        </div>
        <div>
          <label class="input-label">{{ t('purchaseInfo.purchaseUrl') }}</label>
          <input v-model="form.purchase_url" class="input" type="url" maxlength="1024" placeholder="https://example.com" />
        </div>
        <div>
          <label class="input-label">{{ t('purchaseInfo.contact') }}</label>
          <input v-model="form.contact" class="input" type="text" maxlength="500" />
        </div>
        <div>
          <label class="input-label">{{ t('purchaseInfo.cardDescription') }}</label>
          <textarea v-model="form.description" class="input min-h-[96px]" maxlength="2000"></textarea>
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('purchaseInfo.sortOrder') }}</label>
            <input v-model.number="form.sort_order" class="input" type="number" />
          </div>
          <label class="flex items-center gap-3 pt-7">
            <input v-model="form.enabled" class="checkbox" type="checkbox" />
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">{{ t('common.enabled') }}</span>
          </label>
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button type="button" class="btn btn-secondary" @click="closeEditor">{{ t('common.cancel') }}</button>
          <button type="submit" class="btn btn-primary" :disabled="saving">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
            <span>{{ saving ? t('common.saving') : t('common.save') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { purchaseInfoAPI } from '@/api/purchaseInfo'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore, useAuthStore } from '@/stores'
import type { PurchaseInfoCard, PurchaseInfoCardInput } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const visibleCards = ref<PurchaseInfoCard[]>([])
const managedCards = ref<PurchaseInfoCard[]>([])
const loadingVisible = ref(false)
const loadingManaged = ref(false)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const showEditor = ref(false)
const editingCard = ref<PurchaseInfoCard | null>(null)

const canManage = computed(() => authStore.isAdmin || authStore.isAgent)
const isBusy = computed(() => loadingVisible.value || loadingManaged.value)

const form = reactive<PurchaseInfoCardInput>({
  title: '',
  description: '',
  purchase_url: '',
  contact: '',
  sort_order: 0,
  enabled: true,
})

function resetForm(card?: PurchaseInfoCard) {
  form.title = card?.title ?? ''
  form.description = card?.description ?? ''
  form.purchase_url = card?.purchase_url ?? ''
  form.contact = card?.contact ?? ''
  form.sort_order = card?.sort_order ?? 0
  form.enabled = card?.enabled ?? true
}

function openCreate() {
  editingCard.value = null
  resetForm()
  showEditor.value = true
}

function openEdit(card: PurchaseInfoCard) {
  editingCard.value = card
  resetForm(card)
  showEditor.value = true
}

function closeEditor() {
  if (saving.value) return
  showEditor.value = false
}

async function loadVisibleCards() {
  loadingVisible.value = true
  try {
    visibleCards.value = await purchaseInfoAPI.listVisibleCards()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('purchaseInfo.loadFailed'))
  } finally {
    loadingVisible.value = false
  }
}

async function loadManagedCards() {
  if (!canManage.value) return
  loadingManaged.value = true
  try {
    managedCards.value = await purchaseInfoAPI.listManagedCards()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('purchaseInfo.loadFailed'))
  } finally {
    loadingManaged.value = false
  }
}

async function loadAll() {
  await Promise.all([loadVisibleCards(), loadManagedCards()])
}

async function saveCard() {
  saving.value = true
  try {
    if (editingCard.value) {
      await purchaseInfoAPI.updateCard(editingCard.value.id, { ...form })
      appStore.showSuccess(t('purchaseInfo.updated'))
    } else {
      await purchaseInfoAPI.createCard({ ...form })
      appStore.showSuccess(t('purchaseInfo.created'))
    }
    showEditor.value = false
    await loadAll()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('purchaseInfo.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function deleteManagedCard(card: PurchaseInfoCard) {
  if (!window.confirm(t('purchaseInfo.deleteConfirm', { title: card.title }))) return
  deletingId.value = card.id
  try {
    await purchaseInfoAPI.deleteCard(card.id)
    appStore.showSuccess(t('purchaseInfo.deleted'))
    await loadAll()
  } catch (error) {
    appStore.showError((error as { message?: string }).message || t('purchaseInfo.deleteFailed'))
  } finally {
    deletingId.value = null
  }
}

async function copyContact(contact: string) {
  await copyToClipboard(contact, t('purchaseInfo.contactCopied'))
}

onMounted(loadAll)
</script>
