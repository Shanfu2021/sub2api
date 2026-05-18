<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <div class="grid gap-3 lg:grid-cols-[1.35fr_1fr]">
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <div
            class="rounded-3xl bg-gradient-to-br from-primary-500 via-primary-500 to-primary-600 p-4 text-white shadow-lg shadow-primary-500/20"
          >
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-primary-100">{{ t('redeem.currentBalance') }}</p>
              <span class="rounded-xl bg-white/15 p-2">
                <Icon name="creditCard" size="md" class="text-white" />
              </span>
            </div>
            <p class="mt-3 text-3xl font-bold">$ {{ user?.balance?.toFixed(2) || '0.00' }}</p>
            <p class="mt-1 text-xs text-primary-100">余额实时更新，可直接用于接口调用</p>
          </div>

          <div
            class="rounded-3xl border border-sky-200 bg-white p-4 shadow-sm dark:border-sky-900/40 dark:bg-dark-900"
          >
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-dark-400">{{ t('redeem.concurrency') }}</p>
              <span class="rounded-xl bg-sky-100 p-2 dark:bg-sky-900/30">
                <Icon name="bolt" size="md" class="text-sky-600 dark:text-sky-300" />
              </span>
            </div>
            <p class="mt-3 text-3xl font-bold text-gray-900 dark:text-white">
              {{ user?.concurrency || 0 }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">当前账号可同时处理的请求数量</p>
          </div>

          <div
            class="rounded-3xl border border-emerald-200 bg-white p-4 shadow-sm dark:border-emerald-900/40 dark:bg-dark-900"
          >
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-dark-400">{{ t('redeem.needToBuyTitle') }}</p>
              <span class="rounded-xl bg-emerald-100 p-2 dark:bg-emerald-900/30">
                <Icon name="shoppingBag" size="md" class="text-emerald-600 dark:text-emerald-300" />
              </span>
            </div>
            <p class="mt-3 text-base font-semibold text-gray-900 dark:text-white">通用余额卡</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">支持链动小铺购买后回到本页兑换</p>
          </div>

          <div
            class="rounded-3xl border border-amber-200 bg-white p-4 shadow-sm dark:border-amber-900/40 dark:bg-dark-900"
          >
            <div class="flex items-center justify-between">
              <p class="text-sm font-medium text-gray-500 dark:text-dark-400">联系与帮助</p>
              <span class="rounded-xl bg-amber-100 p-2 dark:bg-amber-900/30">
                <Icon name="users" size="md" class="text-amber-600 dark:text-amber-300" />
              </span>
            </div>
            <p class="mt-3 text-base font-semibold text-gray-900 dark:text-white">
              {{ contactInfo || '客服 QQ：1198716953' }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">兑换异常或购买问题请直接联系客服处理</p>
          </div>
        </div>

        <div class="card overflow-hidden border-primary-200/70 shadow-sm dark:border-primary-800/40">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex items-center gap-3">
              <span class="rounded-2xl bg-primary-100 p-2 dark:bg-primary-900/30">
                <Icon name="gift" size="md" class="text-primary-600 dark:text-primary-300" />
              </span>
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">兑换码</h2>
                <p class="text-sm text-gray-500 dark:text-dark-400">
                  输入兑换码即可，系统会自动识别可兑换内容
                </p>
              </div>
            </div>
          </div>

          <div class="p-5">
            <form @submit.prevent="handleSmartSubmit" class="space-y-4">
              <div>
                <label for="smart-code" class="input-label">
                  兑换码
                </label>
                <div class="mt-1 flex flex-col gap-3 sm:flex-row">
                  <div class="relative flex-1">
                    <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                      <Icon name="gift" size="md" class="text-gray-400 dark:text-dark-500" />
                    </div>
                    <input
                      id="smart-code"
                      v-model="smartCode"
                      type="text"
                      required
                      :placeholder="smartCodePlaceholder"
                      :disabled="isSubmittingAny"
                      class="input h-12 pl-12 text-base"
                    />
                  </div>

                  <button
                    type="submit"
                    :disabled="!smartCode.trim() || isSubmittingAny"
                    class="btn btn-primary h-12 shrink-0 px-6"
                  >
                    <svg
                      v-if="isSubmittingAny"
                      class="-ml-1 mr-2 h-5 w-5 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    <Icon v-else name="checkCircle" size="md" class="mr-2" />
                    {{ submitButtonLabel }}
                  </button>
                </div>
                <div class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ t('redeem.redeemCodeHint') }}</span>
                </div>
              </div>

              <div class="flex flex-wrap items-center gap-3">
                <button
                  v-if="purchaseEnabled"
                  type="button"
                  class="btn btn-secondary py-2.5"
                  @click="handleBuyClick"
                >
                  <Icon name="externalLink" size="sm" class="mr-2" />
                  {{ t('redeem.buyNow') }}
                </button>
                <p
                  v-if="purchaseEnabled && purchaseHint"
                  class="text-xs text-gray-500 dark:text-dark-400"
                >
                  {{ purchaseHint }}
                </p>
              </div>
            </form>
          </div>
        </div>
      </div>

      <div
        v-if="purchaseEnabled && purchaseProducts.length > 0"
        class="card overflow-hidden border-sky-200 bg-sky-50/80 dark:border-sky-800/50 dark:bg-sky-900/20"
      >
        <div class="border-b border-sky-200/70 px-5 py-4 dark:border-sky-800/40 sm:px-6">
          <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div class="flex items-start gap-3">
              <div
                class="flex h-10 w-10 items-center justify-center rounded-xl bg-sky-100 dark:bg-sky-900/30"
              >
                <Icon name="shoppingBag" size="md" class="text-sky-600 dark:text-sky-400" />
              </div>
              <div>
                <h2 class="text-lg font-semibold text-sky-900 dark:text-sky-100">
                  {{ t('redeem.productSectionTitle') }}
                </h2>
                <p class="mt-1 text-sm text-sky-700 dark:text-sky-300">
                  {{ t('redeem.productSectionDesc') }}
                </p>
              </div>
            </div>

            <div
              class="rounded-2xl border border-sky-200/80 bg-white/80 px-4 py-3 dark:border-sky-800/40 dark:bg-sky-950/30"
            >
              <p class="text-sm font-semibold text-sky-900 dark:text-sky-100">
                {{ t('redeem.productStoreNoticeTitle') }}
              </p>
              <div class="mt-2 flex flex-wrap items-center gap-3">
                <p class="text-xs text-sky-700 dark:text-sky-300">
                  {{ t('redeem.productStoreNoticeDesc') }}
                </p>
                <a
                  :href="purchaseStoreUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="btn btn-secondary py-2"
                >
                  <Icon name="externalLink" size="sm" class="mr-2" />
                  {{ t('redeem.viewAllProducts') }}
                </a>
              </div>
            </div>
          </div>
        </div>

        <div class="grid gap-3 p-4 md:grid-cols-2 xl:grid-cols-4">
          <div
            v-for="product in purchaseProducts"
            :key="product.key"
            class="flex h-full flex-col rounded-2xl border border-sky-200/70 bg-white/95 p-4 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-sky-800/40 dark:bg-dark-900/80"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0">
                <p class="text-[11px] font-semibold uppercase tracking-[0.18em] text-sky-600 dark:text-sky-400">
                  {{ product.category }}
                </p>
                <h3 class="mt-1 truncate text-[15px] font-semibold leading-6 text-gray-900 dark:text-white">
                  {{ product.title }}
                </h3>
              </div>
              <span
                class="inline-flex shrink-0 rounded-full bg-sky-100 px-3 py-1 text-xs font-medium text-sky-700 dark:bg-sky-900/40 dark:text-sky-300"
              >
                {{ product.badge }}
              </span>
            </div>

            <p class="mt-2 text-sm font-semibold text-sky-900 dark:text-sky-100">
              {{ product.price }}
            </p>

            <p class="mt-2 min-h-[3.5rem] text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ product.description }}
            </p>

            <div class="mt-3 grid grid-cols-2 gap-2">
              <div
                v-for="spec in product.specs"
                :key="`${product.key}-${spec.label}`"
                class="rounded-xl bg-sky-50 px-3 py-2 dark:bg-sky-950/30"
              >
                <p class="text-[11px] font-medium uppercase tracking-wide text-sky-600 dark:text-sky-400">
                  {{ spec.label }}
                </p>
                <p class="mt-1 text-sm font-semibold text-sky-900 dark:text-sky-100">
                  {{ spec.value }}
                </p>
              </div>
            </div>

            <p v-if="product.note" class="mt-3 text-xs leading-5 text-sky-700 dark:text-sky-300">
              {{ product.note }}
            </p>

            <button
              type="button"
              class="btn btn-primary mt-4 w-full py-2.5"
              @click="openPurchaseProduct(product.url)"
            >
              <Icon name="externalLink" size="sm" class="mr-2" />
              {{ t('redeem.buyThisProduct') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Success Message -->
      <transition name="fade">
        <div
          v-if="redeemResult || promoResult"
          class="card border-emerald-200 bg-emerald-50 dark:border-emerald-800/50 dark:bg-emerald-900/20"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30"
              >
                <Icon name="checkCircle" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-emerald-800 dark:text-emerald-300">
                  {{ redeemResult ? t('redeem.redeemSuccess') : t('redeem.promoApplySuccess') }}
                </h3>
                <div class="mt-2 text-sm text-emerald-700 dark:text-emerald-400">
                  <p>{{ redeemResult?.message || promoResult?.message }}</p>
                  <div v-if="redeemResult" class="mt-3 space-y-1">
                    <p v-if="redeemResult.type === 'balance'" class="font-medium">
                      {{ t('redeem.added') }}: ${{ redeemResult.value.toFixed(2) }}
                    </p>
                    <p v-else-if="redeemResult.type === 'concurrency'" class="font-medium">
                      {{ t('redeem.added') }}: {{ redeemResult.value }}
                      {{ t('redeem.concurrentRequests') }}
                    </p>
                    <p v-else-if="redeemResult.type === 'subscription'" class="font-medium">
                      {{ t('redeem.subscriptionAssigned') }}
                      <span v-if="redeemResult.group_name"> - {{ redeemResult.group_name }}</span>
                      <span v-if="redeemResult.validity_days">
                        ({{
                          t('redeem.subscriptionDays', { days: redeemResult.validity_days })
                        }})</span
                      >
                    </p>
                    <p v-if="redeemResult.new_balance !== undefined">
                      {{ t('redeem.newBalance') }}:
                      <span class="font-semibold">${{ redeemResult.new_balance.toFixed(2) }}</span>
                    </p>
                    <p v-if="redeemResult.new_concurrency !== undefined">
                      {{ t('redeem.newConcurrency') }}:
                      <span class="font-semibold"
                        >{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</span
                      >
                    </p>
                  </div>
                  <div v-else-if="promoResult" class="mt-3 space-y-1">
                    <p v-if="promoResult.bonus_amount" class="font-medium">
                      {{ t('redeem.promoBonusApplied') }}: ${{ promoResult.bonus_amount.toFixed(2) }}
                    </p>
                    <p
                      v-if="promoResult.discount_factor && promoResult.discount_factor < 1"
                      class="font-medium"
                    >
                      {{ t('redeem.promoDiscountApplied', { factor: promoResult.discount_factor }) }}
                    </p>
                    <p v-if="promoResult.discount_label">
                      {{ promoResult.discount_label }}
                    </p>
                    <p v-if="promoResult.new_balance !== undefined">
                      {{ t('redeem.newBalance') }}:
                      <span class="font-semibold">${{ promoResult.new_balance.toFixed(2) }}</span>
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Error Message -->
      <transition name="fade">
        <div
          v-if="errorMessage"
          class="card border-red-200 bg-red-50 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-red-100 dark:bg-red-900/30"
              >
                <Icon
                  name="exclamationCircle"
                  size="md"
                  class="text-red-600 dark:text-red-400"
                />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-red-800 dark:text-red-300">
                  {{ t('redeem.redeemFailed') }}
                </h3>
                <p class="mt-2 text-sm text-red-700 dark:text-red-400">
                  {{ errorMessage }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <!-- Information Card -->
      <div class="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
        <div
          class="card border-primary-200 bg-primary-50 dark:border-primary-800/50 dark:bg-primary-900/20"
        >
          <div class="p-5">
            <div class="flex items-start gap-4">
              <div
                class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary-100 dark:bg-primary-900/30"
              >
                <Icon name="infoCircle" size="md" class="text-primary-600 dark:text-primary-400" />
              </div>
              <div class="flex-1">
                <h3 class="text-sm font-semibold text-primary-800 dark:text-primary-300">
                  {{ t('redeem.aboutCodes') }}
                </h3>
                <div class="mt-3 grid gap-2 sm:grid-cols-2">
                  <div class="rounded-2xl bg-white/70 p-3 text-sm text-primary-800 dark:bg-dark-900/40 dark:text-primary-100">
                    {{ t('redeem.codeRule1') }}
                  </div>
                  <div class="rounded-2xl bg-white/70 p-3 text-sm text-primary-800 dark:bg-dark-900/40 dark:text-primary-100">
                    {{ t('redeem.codeRule2') }}
                  </div>
                  <div class="rounded-2xl bg-white/70 p-3 text-sm text-primary-800 dark:bg-dark-900/40 dark:text-primary-100">
                    {{ t('redeem.codeRule4') }}
                  </div>
                  <div class="rounded-2xl bg-white/70 p-3 text-sm text-primary-800 dark:bg-dark-900/40 dark:text-primary-100">
                    {{ t('redeem.codeRule3') }}
                    <span
                      v-if="contactInfo"
                      class="mt-2 inline-flex items-center rounded-md bg-primary-200/60 px-2 py-0.5 text-xs font-medium text-primary-900 dark:bg-primary-800/40 dark:text-primary-100"
                    >
                      {{ contactInfo }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('redeem.recentActivity') }}
            </h2>
          </div>
          <div class="p-5">
            <!-- Loading State -->
            <div v-if="loadingHistory" class="flex items-center justify-center py-8">
              <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
            </div>

            <!-- History List -->
            <div v-else-if="history.length > 0" class="grid gap-3 sm:grid-cols-2">
              <div
                v-for="item in history"
                :key="item.id"
                class="rounded-2xl bg-gray-50 p-4 dark:bg-dark-800"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="flex items-center gap-4">
                    <div
                      :class="[
                        'flex h-10 w-10 items-center justify-center rounded-xl',
                        isBalanceType(item.type)
                          ? item.value >= 0
                            ? 'bg-emerald-100 dark:bg-emerald-900/30'
                            : 'bg-red-100 dark:bg-red-900/30'
                          : isSubscriptionType(item.type)
                            ? 'bg-purple-100 dark:bg-purple-900/30'
                            : item.value >= 0
                              ? 'bg-blue-100 dark:bg-blue-900/30'
                              : 'bg-orange-100 dark:bg-orange-900/30'
                      ]"
                    >
                      <Icon
                        v-if="isBalanceType(item.type)"
                        name="dollar"
                        size="md"
                        :class="
                          item.value >= 0
                            ? 'text-emerald-600 dark:text-emerald-400'
                            : 'text-red-600 dark:text-red-400'
                        "
                      />
                      <Icon
                        v-else-if="isSubscriptionType(item.type)"
                        name="badge"
                        size="md"
                        class="text-purple-600 dark:text-purple-400"
                      />
                      <Icon
                        v-else
                        name="bolt"
                        size="md"
                        :class="
                          item.value >= 0
                            ? 'text-blue-600 dark:text-blue-400'
                            : 'text-orange-600 dark:text-orange-400'
                        "
                      />
                    </div>
                    <div>
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ getHistoryItemTitle(item) }}
                      </p>
                      <p class="text-xs text-gray-500 dark:text-dark-400">
                        {{ formatDateTime(item.used_at) }}
                      </p>
                    </div>
                  </div>
                  <div class="text-right">
                    <p
                      :class="[
                        'text-sm font-semibold',
                        isBalanceType(item.type)
                          ? item.value >= 0
                            ? 'text-emerald-600 dark:text-emerald-400'
                            : 'text-red-600 dark:text-red-400'
                          : isSubscriptionType(item.type)
                            ? 'text-purple-600 dark:text-purple-400'
                            : item.value >= 0
                              ? 'text-blue-600 dark:text-blue-400'
                              : 'text-orange-600 dark:text-orange-400'
                      ]"
                    >
                      {{ formatHistoryValue(item) }}
                    </p>
                    <p
                      v-if="!isAdminAdjustment(item.type)"
                      class="font-mono text-xs text-gray-400 dark:text-dark-500"
                    >
                      {{ item.code.slice(0, 8) }}...
                    </p>
                    <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                      {{ t('redeem.adminAdjustment') }}
                    </p>
                  </div>
                </div>
                <p
                  v-if="item.notes"
                  class="mt-3 text-xs text-gray-500 dark:text-dark-400"
                  :title="item.notes"
                >
                  {{ item.notes }}
                </p>
              </div>
            </div>

            <!-- Empty State -->
            <div v-else class="empty-state py-8">
              <div
                class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
              >
                <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
              </div>
              <p class="text-sm text-gray-500 dark:text-dark-400">
                {{ t('redeem.historyWillAppear') }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, authAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

const smartCode = ref('')
const submitting = ref(false)
const promoSubmitting = ref(false)
const redeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
} | null>(null)
const promoResult = ref<{
  message: string
  bonus_amount?: number
  discount_factor?: number
  discount_label?: string
  new_balance?: number
} | null>(null)
const errorMessage = ref('')

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const contactInfo = ref('')
const purchaseUrl = ref('')
const purchaseEnabled = ref(false)
const purchaseStoreUrl = 'https://pay.ldxp.cn/shop/CN8U85FN'

type PurchaseProduct = {
  key: string
  category: string
  title: string
  description: string
  price: string
  specs: Array<{
    label: string
    value: string
  }>
  note?: string
  badge: string
  url: string
}

const isSubmittingAny = computed(() => submitting.value || promoSubmitting.value)
const smartCodePlaceholder = computed(() => '请输入兑换码')
const submitButtonLabel = computed(() => {
  if (submitting.value) return t('redeem.redeeming')
  if (promoSubmitting.value) return t('redeem.applyingPromo')
  return '立即识别并使用'
})

const purchaseHint = computed(() => {
  if (!purchaseEnabled.value) return ''
  return purchaseUrl.value ? t('redeem.buyExternalHint') : t('redeem.buyInternalHint')
})

const purchaseProducts = computed<PurchaseProduct[]>(() => {
  if (!purchaseEnabled.value) return []

  return [
    {
      key: 'balance-10',
      category: '通用余额',
      title: '10 刀通用余额卡',
      description: '适合首次体验和轻量调用，买完后直接来本页兑换即可。',
      price: '价格以小铺为准',
      specs: [
        { label: '到账', value: '10 USD' },
        { label: '类型', value: '通用余额卡' }
      ],
      note: '复制兑换码回到本页输入，到账后可直接使用。',
      badge: '余额',
      url: 'https://pay.ldxp.cn/item/a7zg2a'
    },
    {
      key: 'balance-50',
      category: '通用余额',
      title: '50 刀通用余额卡',
      description: '适合日常开发与稳定调用，买完后直接来本页兑换即可。',
      price: '价格以小铺为准',
      specs: [
        { label: '到账', value: '50 USD' },
        { label: '类型', value: '通用余额卡' }
      ],
      note: '复制兑换码回到本页输入，到账后可直接使用。',
      badge: '余额',
      url: 'https://pay.ldxp.cn/item/mwwwvc'
    },
    {
      key: 'balance-100',
      category: '通用余额',
      title: '100 刀通用余额卡',
      description: '适合高频开发、长期使用或项目备量，买完后直接来本页兑换即可。',
      price: '价格以小铺为准',
      specs: [
        { label: '到账', value: '100 USD' },
        { label: '类型', value: '通用余额卡' }
      ],
      note: '复制兑换码回到本页输入，到账后可直接使用。',
      badge: '余额',
      url: 'https://pay.ldxp.cn/item/mbwrl9'
    },
    {
      key: 'balance-1000',
      category: '通用余额',
      title: '1000 刀通用余额卡',
      description: '适合团队囤货和长期高频调用，买完后直接来本页兑换即可。',
      price: '价格以小铺为准',
      specs: [
        { label: '到账', value: '1000 USD' },
        { label: '类型', value: '通用余额卡' }
      ],
      note: '复制兑换码回到本页输入，到账后可直接使用。',
      badge: '余额',
      url: 'https://pay.ldxp.cn/item/3p7js8'
    }
  ]
})

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async () => {
  loadingHistory.value = true
  try {
    history.value = await redeemAPI.getHistory()
  } catch (error) {
    console.error('Failed to fetch history:', error)
  } finally {
    loadingHistory.value = false
  }
}

const refreshAfterRedeem = async (result: {
  type: string
}) => {
  await authStore.refreshUser()

  if (result.type === 'subscription') {
    try {
      await subscriptionStore.fetchActiveSubscriptions(true)
    } catch (error) {
      console.error('Failed to refresh subscriptions after redeem:', error)
      appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
    }
  }

  await fetchHistory()
}

const handleRedeemOnly = async (code: string) => {
  const result = await redeemAPI.redeem(code)
  redeemResult.value = result
  promoResult.value = null
  await refreshAfterRedeem(result)
  appStore.showSuccess(t('redeem.codeRedeemSuccess'))
}

const handlePromoOnly = async (code: string) => {
  const result = await redeemAPI.applyPromoCode(code)
  promoResult.value = result
  redeemResult.value = null
  await authStore.refreshUser()
  appStore.showSuccess(t('redeem.promoApplySuccess'))
}

const shouldFallbackToPromo = (error: any) => {
  const code = String(error?.reason || error?.code || error?.response?.data?.reason || error?.response?.data?.code || '')
  return code === 'REDEEM_CODE_NOT_FOUND'
}

const readErrorMessage = (error: any, fallback: string) => {
  return error?.message || error?.response?.data?.detail || fallback
}

const handleSmartSubmit = async () => {
  const code = smartCode.value.trim()
  if (!code) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null
  promoResult.value = null

  try {
    await handleRedeemOnly(code)
    smartCode.value = ''
  } catch (error: any) {
    if (shouldFallbackToPromo(error)) {
      submitting.value = false
      promoSubmitting.value = true
      try {
        await handlePromoOnly(code)
        smartCode.value = ''
        errorMessage.value = ''
      } catch (promoError: any) {
        errorMessage.value = readErrorMessage(promoError, t('redeem.failedToApplyPromo'))
        appStore.showError(errorMessage.value)
      } finally {
        promoSubmitting.value = false
      }
      return
    }

    errorMessage.value = readErrorMessage(error, t('redeem.failedToRedeem'))
    appStore.showError(errorMessage.value)
  } finally {
    submitting.value = false
  }
}

const handleBuyClick = () => {
  const targetUrl = purchaseUrl.value || purchaseStoreUrl
  if (targetUrl) {
    window.open(targetUrl, '_blank', 'noopener,noreferrer')
    return
  }
  router.push('/purchase')
}

const openPurchaseProduct = (url: string) => {
  if (url) {
    window.open(url, '_blank', 'noopener,noreferrer')
    return
  }
  handleBuyClick()
}

onMounted(async () => {
  fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
    purchaseUrl.value = settings.purchase_subscription_url || purchaseStoreUrl
    purchaseEnabled.value = Boolean(
      settings.purchase_subscription_enabled || settings.payment_enabled || purchaseUrl.value
    )
  } catch (error) {
    console.error('Failed to load contact info:', error)
    purchaseEnabled.value = true
    purchaseUrl.value = purchaseStoreUrl
  }
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
