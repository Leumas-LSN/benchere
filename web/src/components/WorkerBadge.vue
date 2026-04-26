<template>
  <div
    class="flex items-center gap-2.5 px-3 py-2 rounded-lg surface-muted text-sm transition-colors"
    :class="cardRing"
  >
    <span
      class="w-1.5 h-1.5 rounded-full shrink-0"
      :class="[dotColor, animated ? 'animate-pulse-dot' : '']"
    ></span>
    <span class="font-medium fg-primary">{{ name }}</span>
    <span v-if="cpu !== undefined" class="text-xs num fg-muted ml-auto">
      {{ cpu.toFixed(1) }}<span class="fg-faint">%</span>
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:   { type: String,  required: true },
  status: { type: String,  default: 'provisioning' },
  cpu:    { type: Number,  default: undefined },
})

const animated = computed(() => ['ready', 'running', 'provisioning'].includes(props.status))

const dotColor = computed(() => {
  switch (props.status) {
    case 'ready':
    case 'running': return 'bg-emerald-500'
    case 'failed':  return 'bg-red-500'
    case 'done':    return 'bg-ink-400'
    default:        return 'bg-amber-500'
  }
})

const cardRing = computed(() => {
  switch (props.status) {
    case 'failed':  return 'ring-1 ring-red-200 dark:ring-red-500/30'
    case 'ready':
    case 'running': return 'ring-1 ring-emerald-200 dark:ring-emerald-500/30'
    default:        return ''
  }
})
</script>
