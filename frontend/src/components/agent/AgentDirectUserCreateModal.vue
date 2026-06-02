<template>
  <BaseDialog
    :show="show"
    :title="t('agentManagement.direct.createUser')"
    width="normal"
    @close="emit('close')"
  >
    <div data-test="create-direct-user-modal">
      <form id="agent-direct-user-create-form" data-test="create-direct-user-submit" class="space-y-5" @submit.prevent="submit">
        <div>
          <label class="input-label">{{ t('admin.users.email') }}</label>
          <input
            v-model="form.email"
            data-test="create-direct-user-email"
            type="email"
            required
            class="input"
            :placeholder="t('admin.users.enterEmail')"
          />
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.password') }}</label>
          <div class="flex gap-2">
            <div class="relative flex-1">
              <input
                v-model="form.password"
                data-test="create-direct-user-password"
                type="text"
                required
                minlength="6"
                class="input pr-10"
                :placeholder="t('admin.users.enterPassword')"
              />
            </div>
            <button type="button" class="btn btn-secondary px-3" @click="generateRandomPassword">
              <Icon name="refresh" size="md" />
            </button>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.users.username') }}</label>
          <input
            v-model="form.username"
            data-test="create-direct-user-username"
            type="text"
            class="input"
            :placeholder="t('admin.users.enterUsername')"
          />
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('agentManagement.direct.allocatedConcurrency') }}</label>
            <input
              v-model.number="form.allocated_concurrency"
              data-test="create-direct-user-concurrency"
              type="number"
              min="0"
              step="1"
              class="input"
            />
          </div>
          <div>
            <label class="input-label">{{ t('agentManagement.direct.allocatedRpm') }}</label>
            <input
              v-model.number="form.allocated_rpm"
              data-test="create-direct-user-rpm"
              type="number"
              min="0"
              step="1"
              class="input"
            />
          </div>
        </div>
      </form>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" form="agent-direct-user-create-form" class="btn btn-primary" :disabled="loading">
          {{ loading ? t('admin.users.creating') : t('common.create') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AgentDirectUserCreateRequest } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', payload: AgentDirectUserCreateRequest): void
}>()

const { t } = useI18n()

const form = reactive({
  email: '',
  password: '',
  username: '',
  allocated_concurrency: 0,
  allocated_rpm: 0,
})

function resetForm() {
  Object.assign(form, {
    email: '',
    password: '',
    username: '',
    allocated_concurrency: 0,
    allocated_rpm: 0,
  })
}

function numericValue(value: unknown): number {
  return Math.max(0, Number(value) || 0)
}

function submit() {
  emit('submit', {
    email: form.email.trim(),
    password: form.password,
    username: form.username.trim(),
    allocated_concurrency: numericValue(form.allocated_concurrency),
    allocated_rpm: numericValue(form.allocated_rpm),
  })
}

function generateRandomPassword() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let password = ''
  for (let i = 0; i < 16; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  form.password = password
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      resetForm()
    }
  }
)
</script>
