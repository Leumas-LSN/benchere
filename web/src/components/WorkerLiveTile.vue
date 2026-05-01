<template>
  <div class="rounded-lg surface-muted p-3 text-sm transition-colors" :class="cardRing">
    <div class="flex items-center gap-2 mb-2">
      <span class="w-1.5 h-1.5 rounded-full shrink-0" :class="dotColor"></span>
      <span class="font-medium fg-primary truncate flex-1">{{ name }}</span>
      <span v-if="saturated" class="text-[10px] uppercase font-semibold tracking-wide bg-red-500 text-white px-1.5 py-0.5 rounded">SAT {{ saturation.kind }}</span>
    </div>
    <svg viewBox="0 0 60 16" class="w-full h-4 mb-1.5">
      <polyline :points="sparkPoints" fill="none" :stroke="cpuColor" stroke-width="1.2" />
    </svg>
    <div class="grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] num">
      <div class="flex justify-between"><span class="fg-muted">CPU</span><span :class="cpuTextColor">{{ (cpu||0).toFixed(1) }}%</span></div>
      <div class="flex justify-between"><span class="fg-muted">RAM</span><span class="fg-secondary">{{ (ram||0).toFixed(1) }}%</span></div>
      <div class="flex justify-between"><span class="fg-muted">Net</span><span class="fg-secondary">&#x2193;{{ rate(netIn) }} &#x2191;{{ rate(netOut) }}</span></div>
      <div class="flex justify-between"><span class="fg-muted">Disk</span><span class="fg-secondary">R{{ rate(diskRead) }} W{{ rate(diskWrite) }}</span></div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:      { type: String, required: true },
  status:    { type: String, default: 'provisioning' },
  cpu:       { type: Number, default: 0 },
  ram:       { type: Number, default: 0 },
  netIn:     { type: Number, default: 0 },
  netOut:    { type: Number, default: 0 },
  diskRead:  { type: Number, default: 0 },
  diskWrite: { type: Number, default: 0 },
  cpuHistory:{ type: Array,  default: () => [] },
  saturation:{ type: Object, default: null },
})

const saturated = computed(() => {
  if (!props.saturation) return false
  return Date.now() - (props.saturation.ts || 0) < 10000
})

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
  if (saturated.value) return 'ring-1 ring-red-300 dark:ring-red-500/40'
  if (props.status === 'running') return 'ring-1 ring-emerald-200 dark:ring-emerald-500/30'
  return ''
})

const cpuColor = computed(() => props.cpu > 80 ? '#ef4444' : '#f97316')
const cpuTextColor = computed(() => props.cpu > 80 ? 'text-red-600 dark:text-red-400' : 'fg-secondary')

const sparkPoints = computed(() => {
  const xs = props.cpuHistory.slice(-30)
  if (!xs.length) return ''
  const w = 60
  const h = 16
  return xs.map((y, i) => ((i / Math.max(xs.length - 1, 1)) * w).toFixed(1) + ',' + (h - (y / 100) * h).toFixed(1)).join(' ')
})

function rate(bps) {
  if (!bps || bps < 1024) return '0'
  const u = ['', 'K', 'M', 'G']
  let i = 0; let v = bps
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return (v >= 100 ? Math.round(v) : v.toFixed(1)) + u[i]
}
</script>
