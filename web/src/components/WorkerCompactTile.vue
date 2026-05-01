<template>
  <div class="wt" :class="ringClass">
    <div class="wt-head">
      <span class="wt-dot" :class="dotClass"></span>
      <span class="wt-name">{{ name }}</span>
      <span v-if="saturated" class="wt-sat">SAT</span>
    </div>
    <div class="wt-bars">
      <div class="wt-bar">
        <span class="wt-bar-label">cpu</span>
        <div class="wt-bar-track"><div class="wt-bar-fill" :class="cpuColor" :style="{ width: Math.min(cpu, 100) + '%' }"></div></div>
        <span class="wt-bar-val" :class="cpuText">{{ cpu.toFixed(0) }}<span class="fg-faint">%</span></span>
      </div>
      <div class="wt-bar">
        <span class="wt-bar-label">ram</span>
        <div class="wt-bar-track"><div class="wt-bar-fill bg-sky-500" :style="{ width: Math.min(ram, 100) + '%' }"></div></div>
        <span class="wt-bar-val">{{ ram.toFixed(0) }}<span class="fg-faint">%</span></span>
      </div>
    </div>
    <div class="wt-io">
      <span class="wt-io-row"><span class="wt-io-label">net</span>↓{{ rate(netIn) }} ↑{{ rate(netOut) }}</span>
      <span class="wt-io-row"><span class="wt-io-label">dsk</span>R{{ rate(diskRead) }} W{{ rate(diskWrite) }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:        { type: String,  required: true },
  status:      { type: String,  default: 'provisioning' },
  cpu:         { type: Number,  default: 0 },
  ram:         { type: Number,  default: 0 },
  netIn:       { type: Number,  default: 0 },
  netOut:      { type: Number,  default: 0 },
  diskRead:    { type: Number,  default: 0 },
  diskWrite:   { type: Number,  default: 0 },
  saturation:  { type: Object,  default: null },
})

const saturated = computed(() => {
  if (!props.saturation) return false
  return Date.now() - (props.saturation.ts || 0) < 10000
})

const dotClass = computed(() => {
  switch (props.status) {
    case 'ready':
    case 'running': return 'bg-emerald-500'
    case 'failed':  return 'bg-red-500'
    case 'done':    return 'bg-ink-400'
    default:        return 'bg-amber-500'
  }
})

const ringClass = computed(() => {
  if (saturated.value) return 'wt-ring-red'
  if (props.status === 'running' || props.status === 'ready') return 'wt-ring-ok'
  return ''
})

const cpuColor = computed(() =>
  props.cpu > 80 ? 'bg-red-500' : props.cpu > 60 ? 'bg-amber-500' : 'bg-emerald-500'
)
const cpuText = computed(() =>
  props.cpu > 80 ? 'text-red-600 dark:text-red-400' :
  props.cpu > 60 ? 'text-amber-600 dark:text-amber-400' :
                   'fg-secondary'
)

function rate(bps) {
  if (!bps || bps < 1024) return '0'
  const u = ['', 'K', 'M', 'G']
  let i = 0; let v = bps
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return (v >= 100 ? Math.round(v) : v.toFixed(1)) + u[i]
}
</script>

<style scoped>
.wt {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  padding: 0.55rem 0.7rem;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}
.wt-ring-ok  { border-color: rgba(16,185,129,0.30); }
.wt-ring-red { border-color: rgba(239,68,68,0.45);  }
.wt-head {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.wt-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  flex-shrink: 0;
}
.wt-name {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.78rem;
  font-weight: 500;
  color: var(--fg-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.wt-sat {
  font-size: 0.55rem;
  font-weight: 700;
  letter-spacing: 0.07em;
  background: #ef4444;
  color: white;
  padding: 0.05rem 0.3rem;
  border-radius: 0.2rem;
}
.wt-bars {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}
.wt-bar {
  display: grid;
  grid-template-columns: 24px 1fr 38px;
  align-items: center;
  gap: 0.45rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.68rem;
}
.wt-bar-label {
  text-transform: uppercase;
  font-size: 0.55rem;
  letter-spacing: 0.05em;
  color: var(--fg-muted);
}
.wt-bar-track {
  height: 4px;
  background: var(--surface-muted);
  border-radius: 999px;
  overflow: hidden;
}
.wt-bar-fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.4s ease;
}
.wt-bar-val {
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-size: 0.7rem;
}
.wt-io {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--fg-secondary);
}
.wt-io-row {
  display: flex;
  align-items: baseline;
  gap: 0.4rem;
}
.wt-io-label {
  text-transform: uppercase;
  font-size: 0.55rem;
  letter-spacing: 0.05em;
  color: var(--fg-muted);
  min-width: 24px;
}
</style>
