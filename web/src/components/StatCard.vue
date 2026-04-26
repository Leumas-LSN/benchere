<template>
  <div class="card flex items-start gap-4">
    <div
      v-if="icon"
      class="shrink-0 w-10 h-10 rounded-lg flex items-center justify-center"
      :class="iconWrap"
    >
      <Icon :name="icon" :size="20" />
    </div>
    <div class="flex-1 min-w-0">
      <p class="section-eyebrow">{{ label }}</p>
      <p class="mt-1 text-2xl font-semibold num truncate" :class="valueClass">
        {{ value }}
        <span v-if="unit" class="text-sm font-normal fg-muted ml-0.5">{{ unit }}</span>
      </p>
      <p v-if="hint" class="mt-1 text-xs fg-muted">{{ hint }}</p>
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = defineProps({
  label:      { type: String, required: true },
  value:      { type: [String, Number], required: true },
  unit:       { type: String, default: '' },
  hint:       { type: String, default: '' },
  icon:       { type: String, default: '' },
  tone:       { type: String, default: 'brand' }, // brand | neutral | success | danger | info
  valueClass: { type: String, default: 'fg-primary' },
})

const iconWrap = computed(() => {
  switch (props.tone) {
    case 'success':  return 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400'
    case 'danger':   return 'bg-red-50 text-red-600 dark:bg-red-500/10 dark:text-red-400'
    case 'info':     return 'bg-blue-50 text-blue-600 dark:bg-blue-500/10 dark:text-blue-400'
    case 'neutral':  return 'bg-ink-100 text-ink-600 dark:bg-ink-800 dark:text-ink-300'
    case 'brand':
    default:         return 'bg-brand-50 text-brand-600 dark:bg-brand-500/10 dark:text-brand-400'
  }
})
</script>
