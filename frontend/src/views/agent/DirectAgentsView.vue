<template>
  <AgentDirectChildrenView kind="agents" :title="t('nav.agentDirectAgents')" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AgentDirectChildrenView from './AgentDirectChildrenView.vue'

const { t } = useI18n()
</script>
