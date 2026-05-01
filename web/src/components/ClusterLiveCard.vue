<template>
  <div class="rounded-lg surface-muted p-3 text-sm">
    <div class="flex items-center gap-2 mb-1.5">
      <span class="font-mono font-medium fg-primary truncate flex-1">{{ name }}</span>
      <span class="text-[11px] num fg-muted">load {{ load.toFixed(2) }}</span>
    </div>
    <svg viewBox="0 0 80 18" class="w-full h-5 mb-1.5">
      <polyline :points="sparkPoints" fill="none" :stroke="cpuColor" stroke-width="1.2" />
    </svg>
    <div class="grid grid-cols-2 gap-2 text-[11px] num">
      <div>
        <div class="flex justify-between"><span class="fg-muted">CPU</span><span :class="cpuTextColor">{{ cpu.toFixed(1) }}%</span></div>
        <div class="h-1 rounded-full bg-soft overflow-hidden mt-0.5">
          <div class="h-full rounded-full transition-[width] duration-500" :class="cpuBarColor" :style="{ width: Math.min(cpu, 100) + '%' }"></div>
        </div>
      </div>
      <div>
        <div class="flex justify-between"><span class="fg-muted">RAM</span><span class="fg-secondary">{{ ram.toFixed(1) }}%</span></div>
        <div class="h-1 rounded-full bg-soft overflow-hidden mt-0.5">
          <div class="h-full rounded-full bg-sky-500 transition-[width] duration-500" :style="{ width: Math.min(ram, 100) + '%' }"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({
  name: { type: String, required: true },
  cpu:  { type: Number, default: 0 },
  ram:  { type: Number, default: 0 },
  load: { type: Number, default: 0 },
  history: { type: Array, default: () => [] },
})
const cpuColor    = computed(() => props.cpu > 80 ? '#ef4444' : props.cpu > 60 ? '#f59e0b' : '#10b981')
const cpuBarColor = computed(() => props.cpu > 80 ? 'bg-red-500' : props.cpu > 60 ? 'bg-amber-500' : 'bg-emerald-500')
const cpuTextColor = computed(() => props.cpu > 80 ? 'text-red-600 dark:text-red-400' : 'fg-secondary')
const sparkPoints = computed(() => {
  const xs = props.history.slice(-30)
  if (!xs.length) return ''
  const w = 80, h = 18
  return xs.map((y, i) => ((i / Math.max(xs.length - 1, 1)) * w).toFixed(1) + ',' + (h - (y / 100) * h).toFixed(1)).join(' ')
})
</script>
