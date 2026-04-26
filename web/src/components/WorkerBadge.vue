<template>
  <div
    class="rounded-lg surface-muted p-3 text-sm transition-colors"
    :class="cardRing"
  >
    <div class="flex items-center gap-2.5 mb-2">
      <span
        class="w-1.5 h-1.5 rounded-full shrink-0"
        :class="[dotColor, animated ? 'animate-pulse-dot' : '']"
      ></span>
      <span class="font-medium fg-primary">{{ name }}</span>
      <span v-if="cpu !== undefined" class="text-xs num fg-muted ml-auto">
        {{ cpu.toFixed(1) }}<span class="fg-faint">%</span>
      </span>
    </div>

    <!-- Stat grid: CPU + RAM + Net + Disk -->
    <div v-if="hasMetrics" class="grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] num">
      <div class="flex items-center justify-between">
        <span class="fg-muted">CPU</span>
        <span class="fg-secondary">{{ (cpu ?? 0).toFixed(1) }}%</span>
      </div>
      <div class="flex items-center justify-between">
        <span class="fg-muted">RAM</span>
        <span class="fg-secondary">{{ (ram ?? 0).toFixed(1) }}%</span>
      </div>
      <div class="flex items-center justify-between" :title="`in ${formatRate(netIn)} · out ${formatRate(netOut)}`">
        <span class="fg-muted">Net</span>
        <span class="fg-secondary">↓{{ formatRateShort(netIn) }} ↑{{ formatRateShort(netOut) }}</span>
      </div>
      <div class="flex items-center justify-between" :title="`read ${formatRate(diskRead)} · write ${formatRate(diskWrite)}`">
        <span class="fg-muted">Disk</span>
        <span class="fg-secondary">R{{ formatRateShort(diskRead) }} W{{ formatRateShort(diskWrite) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:        { type: String,  required: true },
  status:      { type: String,  default: 'provisioning' },
  cpu:         { type: Number,  default: undefined },
  ram:         { type: Number,  default: undefined },
  netIn:       { type: Number,  default: 0 },
  netOut:      { type: Number,  default: 0 },
  diskRead:    { type: Number,  default: 0 },
  diskWrite:   { type: Number,  default: 0 },
})

const hasMetrics = computed(() => props.cpu !== undefined)

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

function formatRateShort(bps) {
  if (!bps || bps < 1024) return '0'
  const u = ['', 'K', 'M', 'G']
  let i = 0
  let v = bps
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return `${v >= 100 ? Math.round(v) : v.toFixed(1)}${u[i]}`
}

function formatRate(bps) {
  return `${formatRateShort(bps)}B/s`
}
</script>
