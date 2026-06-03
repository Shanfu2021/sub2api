<template>
  <AppLayout>
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div class="min-w-0">
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('purchaseInfo.description') }}</p>
        </div>
        <button class="btn btn-secondary px-3" :disabled="loadingVisible || loadingManaged" @click="loadAll">
          <Icon name="refresh" size="sm" :class="loadingVisible || loadingManaged ? 'animate-spin' : ''" />
        </button>
      </header>

      <section>
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.availableTitle') }}</h2>
        </div>

        <div v-if="loadingVisible" class="rounded-md border border-gray-200 bg-white px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="visibleCards.length === 0" class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
          {{ t('purchaseInfo.emptyVisible') }}
        </div>
        <div v-else class="grid gap-4 md:grid-cols-2">
          <article
            v-for="card in visibleCards"
            :key="card.id"
            class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="flex min-w-0 items-start justify-between gap-3">
              <div class="min-w-0">
                <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ card.title }}</h3>
                <p v-if="card.description" class="mt-2 whitespace-pre-wrap text-sm text-gray-600 dark:text-dark-300">{{ card.description }}</p>
              </div>
            </div>
            <div class="mt-4 flex flex-wrap items-center gap-2">
              <a
                v-if="card.purchase_url"
                :href="card.purchase_url"
                target="_blank"
                rel="noopener noreferrer"
                class="btn btn-primary btn-sm"
              >
                <Icon name="externalLink" size="sm" />
                <span>{{ t('purchaseInfo.openLink') }}</span>
              </a>
              <span v-if="card.contact" class="inline-flex min-w-0 items-center gap-2 rounded-md bg-gray-100 px-2.5 py-1 text-sm text-gray-700 dark:bg-dark-700 dark:text-dark-200">
                <Icon name="chat" size="sm" />
                <span class="break-all">{{ card.contact }}</span>
              </span>
            </div>
          </article>
        </div>
      </section>

      <section v-if="canManage" class="border-t border-gray-200 pt-6 dark:border-dark-700">
        <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('purchaseInfo.manageTitle') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('purchaseInfo.manageDescription') }}</p>
          </div>
          <button class="btn btn-primary" @click="openCreate">
            <Icon name="plus" size="sm" />
            <span>{{ t('purchaseInfo.addCard') }}</span>
          </button>
        </div>

        <div v-if="loadingManaged" class="rounded-md border border-gray-200 bg-white px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="managedCards.length === 0" class="rounded-md border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
          {{ t('purchaseInfo.emptyManaged') }}
        </div>
        <div v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800">
          <div
            v-for="card in managedCards"
            :key="card.id"
            class="flex flex-col gap-3 border-b border-gray-100 p-4 last:border-b-0 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ card.title }}</span>
                <span :class="card.enabled ? 'badge badge-green' : 'badge badge-gray'">
                  {{ card.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
                <span class="badge badge-gray">{{ t('purchaseInfo.sortOrderShort', { order: card.sort_order }) }}</span>
              </div>
              <p v-if="card.purchase_url" class="mt-1 break-all text-sm text-gray-500 dark:text-dark-400">{{ card.purchase_url }}</p>
              <p v-if="card.contact" class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ card.contact }}</p>
            </div>
            <div class="flex flex-shrink-0 items-center gap-2">
              <button class="btn btn-secondary btn-sm" @click="openEdit(card)">
                <Icon name="edit" size="sm" />
                <span>{{ t('common.edit') }}</span>
              </button>
              <button class="btn btn-danger btn-sm" :disabled="deletingId === card.id" @click="deleteManagedCard(card)">
                <Icon name="trash" size="sm" />
                <span>{{ t('common.delete') }}</span>
              </button>
            </div>
          </div>
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
import { useAppStore, useAuthStore } from '@/stores'
import type { PurchaseInfoCard, PurchaseInfoCardInput } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const visibleCards = ref<PurchaseInfoCard[]>([])
const managedCards = ref<PurchaseInfoCard[]>([])
const loadingVisible = ref(false)
const loadingManaged = ref(false)
const saving = ref(false)
const deletingId = ref<number | null>(null)
const showEditor = ref(false)
const editingCard = ref<PurchaseInfoCard | null>(null)

const canManage = computed(() => authStore.isAdmin || authStore.isAgent)

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

onMounted(loadAll)
</script>
