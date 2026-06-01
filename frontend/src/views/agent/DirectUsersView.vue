<template>
  <AgentDirectChildrenView kind="users" :title="t('nav.agentDirectUsers')" />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AgentDirectChildrenView from './AgentDirectChildrenView.vue'

const { t } = useI18n()
</script>
