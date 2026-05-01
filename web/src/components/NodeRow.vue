<template>
  <div class="node-row">
    <span class="node-name">{{ name }}</span>
    <svg viewBox="0 0 100 16" class="node-spark" preserveAspectRatio="none">
      <polyline :points="sparkPoints" fill="none" :stroke="cpuColor" stroke-width="1.2" />
    </svg>
    <div class="node-stats">
      <span class="node-stat" :class="cpuTextColor">
        <span class="node-stat-label">cpu</span>
        <span class="node-stat-val">{{ cpu.toFixed(1) }}<span class="fg-faint">%</span></span>
      </span>
      <span class="node-stat fg-secondary">
        <span class="node-stat-label">ram</span>
        <span class="node-stat-val">{{ ram.toFixed(0) }}<span class="fg-faint">%</span></span>
      </span>
      <span class="node-stat fg-secondary">
        <span class="node-stat-label">load</span>
        <span class="node-stat-val">{{ load.toFixed(2) }}</span>
      </span>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:    { type: String, required: true },
  cpu:     { type: Number, default: 0 },
  ram:     { type: Number, default: 0 },
  load:    { type: Number, default: 0 },
  history: { type: Array,  default: () => [] },
})

const cpuColor = computed(() =>
  props.cpu > 80 ? '#ef4444' : props.cpu > 60 ? '#f59e0b' : '#10b981'
)
const cpuTextColor = computed(() =>
  props.cpu > 80 ? 'text-red-600 dark:text-red-400' :
  props.cpu > 60 ? 'text-amber-600 dark:text-amber-400' :
                   'fg-secondary'
)

const sparkPoints = computed(() => {
  const xs = props.history.slice(-30)
  if (xs.length < 2) return ''
  const W = 100, H = 16
  return xs.map((y, i) =>
    `${((i / (xs.length - 1)) * W).toFixed(1)},${(H - (Math.max(0, Math.min(100, y)) / 100) * H).toFixed(1)}`
  ).join(' ')
})
</script>

<style scoped>
.node-row {
  display: grid;
  grid-template-columns: 110px 1fr auto;
  align-items: center;
  gap: 0.75rem;
  padding: 0.45rem 0.7rem;
  border-bottom: 1px solid var(--border-subtle);
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.78rem;
}
.node-row:last-child {
  border-bottom: 0;
}
.node-name {
  font-weight: 500;
  color: var(--fg-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.node-spark {
  width: 100%;
  height: 16px;
  min-width: 60px;
}
.node-stats {
  display: flex;
  gap: 0.85rem;
  font-variant-numeric: tabular-nums;
}
.node-stat {
  display: inline-flex;
  align-items: baseline;
  gap: 0.25rem;
}
.node-stat-label {
  font-size: 0.6rem;
  text-transform: uppercase;
  color: var(--fg-faint);
  letter-spacing: 0.05em;
}
.node-stat-val {
  font-weight: 500;
}
</style>
