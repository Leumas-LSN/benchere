<template>
  <div class="flex flex-col items-center justify-center text-center py-12 px-6">
    <div
      v-if="icon"
      class="w-12 h-12 rounded-xl flex items-center justify-center mb-4 bg-brand-50 text-brand-600 dark:bg-brand-500/10 dark:text-brand-400"
    >
      <Icon :name="icon" :size="22" />
    </div>
    <h3 v-if="title" class="text-base font-semibold fg-primary">{{ title }}</h3>
    <p v-if="description" class="mt-1 text-sm fg-muted max-w-sm">{{ description }}</p>
    <div v-if="$slots.action" class="mt-5">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup>
import Icon from './Icon.vue'

defineProps({
  icon:        { type: String, default: '' },
  title:       { type: String, default: '' },
  description: { type: String, default: '' },
})
</script>
