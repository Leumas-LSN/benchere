<template>
  <article class="nch">
    <header class="nch-head">
      <span class="nch-tag">NODE</span>
      <span class="nch-name" :title="name">{{ name }}</span>
      <span v-if="status" class="nch-status" :class="statusClass">{{ status }}</span>
    </header>
    <svg viewBox="0 0 200 36" class="nch-spark" preserveAspectRatio="none">
      <polyline
        v-if="sparkPoints"
        :points="sparkPoints"
        fill="none"
        :stroke="cpuColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <line v-else x1="0" y1="34" x2="200" y2="34" stroke="currentColor" stroke-width="1" stroke-dasharray="2 2" opacity="0.2" />
    </svg>
    <footer class="nch-foot">
      <span class="nch-stat" :class="cpuTextClass">
        <span class="nch-stat-label">cpu</span>
        <span class="nch-stat-val">{{ cpu.toFixed(1) }}<span class="fg-faint">%</span></span>
      </span>
      <span class="nch-stat fg-secondary">
        <span class="nch-stat-label">ram</span>
        <span class="nch-stat-val">{{ ram.toFixed(0) }}<span class="fg-faint">%</span></span>
      </span>
      <span class="nch-stat fg-secondary">
        <span class="nch-stat-label">load</span>
        <span class="nch-stat-val">{{ load.toFixed(2) }}</span>
      </span>
    </footer>
  </article>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name:    { type: String, required: true },
  cpu:     { type: Number, default: 0 },
  ram:     { type: Number, default: 0 },
  load:    { type: Number, default: 0 },
  history: { type: Array,  default: () => [] },
  status:  { type: String, default: '' },
})

const cpuColor = computed(() =>
  props.cpu > 80 ? '#ef4444' : props.cpu > 60 ? '#f59e0b' : '#10b981'
)
const cpuTextClass = computed(() =>
  props.cpu > 80 ? 'text-red-600 dark:text-red-400' :
  props.cpu > 60 ? 'text-amber-600 dark:text-amber-400' :
                   'fg-secondary'
)
const statusClass = computed(() => {
  switch (props.status) {
    case 'online':  return 'nch-status-ok'
    case 'offline': return 'nch-status-fail'
    default:        return 'nch-status-neutral'
  }
})

const sparkPoints = computed(() => {
  const xs = (props.history || []).slice(-30)
  if (xs.length < 2) return ''
  const W = 200, H = 36, PAD = 2
  return xs.map((y, i) => {
    const x = (i / (xs.length - 1)) * W
    const ny = H - PAD - (Math.max(0, Math.min(100, y)) / 100) * (H - PAD * 2)
    return x.toFixed(1) + ',' + ny.toFixed(1)
  }).join(' ')
})
</script>

<style scoped>
.nch {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  padding: 0.55rem 0.75rem 0.55rem;
}
.nch-head {
  display: flex;
  align-items: baseline;
  gap: 0.45rem;
}
.nch-tag {
  font-size: 0.55rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.nch-name {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.82rem;
  font-weight: 500;
  color: var(--fg-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nch-status {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.55rem;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  padding: 0.05rem 0.35rem;
  border-radius: 0.2rem;
}
.nch-status-ok {
  background: rgba(22,163,74,0.10);
  color: rgb(22,163,74);
}
:root.dark .nch-status-ok {
  background: rgba(74,222,128,0.12);
  color: rgb(74,222,128);
}
.nch-status-fail {
  background: rgba(220,38,38,0.10);
  color: rgb(220,38,38);
}
:root.dark .nch-status-fail {
  background: rgba(248,113,113,0.15);
  color: rgb(248,113,113);
}
.nch-status-neutral {
  background: var(--surface-muted);
  color: var(--fg-muted);
}
.nch-spark {
  width: 100%;
  height: 36px;
  color: var(--fg-faint);
}
.nch-foot {
  display: flex;
  gap: 1rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
}
.nch-stat {
  display: inline-flex;
  align-items: baseline;
  gap: 0.25rem;
}
.nch-stat-label {
  font-size: 0.55rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--fg-faint);
}
.nch-stat-val {
  font-weight: 500;
}
</style>
