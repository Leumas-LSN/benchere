<template>
  <span :class="['inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-[11px] font-semibold uppercase tracking-wide whitespace-nowrap', tone]">
    <span
      class="w-1.5 h-1.5 rounded-full shrink-0"
      :class="[dot, animated ? 'animate-pulse-dot' : '']"
    ></span>
    {{ label }}
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t, te } = useI18n()

const props = defineProps({
  status: { type: String, required: true },
})

const ANIMATED = new Set(['running', 'provisioning', 'pending'])
const animated = computed(() => ANIMATED.has(props.status))

const tone = computed(() => {
  switch (props.status) {
    case 'done':
      return 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30'
    case 'failed':
      return 'bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-500/10 dark:text-red-300 dark:ring-red-500/30'
    case 'running':
      return 'bg-blue-50 text-blue-700 ring-1 ring-blue-200 dark:bg-blue-500/10 dark:text-blue-300 dark:ring-blue-500/30'
    case 'provisioning':
      return 'bg-amber-50 text-amber-800 ring-1 ring-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:ring-amber-500/30'
    case 'pending':
      return 'bg-ink-100 text-ink-700 ring-1 ring-ink-200 dark:bg-ink-500/10 dark:text-ink-300 dark:ring-ink-500/30'
    case 'cancelled':
      return 'bg-ink-100 text-ink-600 ring-1 ring-ink-200 dark:bg-ink-500/10 dark:text-ink-400 dark:ring-ink-500/30'
    case 'ready':
      return 'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30'
    default:
      return 'bg-ink-100 text-ink-700 ring-1 ring-ink-200 dark:bg-ink-500/10 dark:text-ink-300 dark:ring-ink-500/30'
  }
})

const dot = computed(() => {
  switch (props.status) {
    case 'done':
    case 'ready':         return 'bg-emerald-500'
    case 'failed':        return 'bg-red-500'
    case 'running':       return 'bg-blue-500'
    case 'provisioning':  return 'bg-amber-500'
    case 'pending':       return 'bg-ink-400'
    case 'cancelled':     return 'bg-ink-400'
    default:              return 'bg-ink-400'
  }
})

const label = computed(() => {
  const key = `status.${props.status}`
  return te(key) ? t(key) : props.status
})
</script>
