<template>
  <AgentDirectChildrenView kind="enterprises" :title="t('nav.agentDirectEnterprises')" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AgentDirectChildrenView from './AgentDirectChildrenView.vue'

const { t } = useI18n()
</script>
