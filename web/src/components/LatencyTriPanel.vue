<template>
  <section class="ltp">
    <header class="ltp-head">
      <span class="ltp-head-label">{{ t('dashboard.latencyTri.label') }}</span>
      <span class="ltp-head-unit">{{ unit }}</span>
    </header>
    <div class="ltp-grid">
      <div
        v-for="s in series"
        :key="s.id"
        class="ltp-cell"
      >
        <header class="ltp-cell-head">
          <span class="ltp-cell-name" :style="{ color: s.color }">{{ s.label }}</span>
          <span class="ltp-cell-val">{{ formatVal(s.value) }}</span>
        </header>
        <svg viewBox="0 0 120 28" class="ltp-spark" preserveAspectRatio="none">
          <polyline
            v-if="s.points"
            :points="s.points"
            fill="none"
            :stroke="s.color"
            stroke-width="1.4"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
          <line v-else x1="0" y1="26" x2="120" y2="26" :stroke="s.color" stroke-width="1" stroke-dasharray="2 2" opacity="0.3" />
        </svg>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  p50History:  { type: Array,  default: () => [] },
  p95History:  { type: Array,  default: () => [] },
  p99History:  { type: Array,  default: () => [] },
  p50Current:  { type: Number, default: 0 },
  p95Current:  { type: Number, default: 0 },
  p99Current:  { type: Number, default: 0 },
  unit:        { type: String, default: 'ms' },
})

// Stay aligned with the Tendances multi-series chart so the visuals do
// not contradict each other when both panels are visible.
const COLOR_P50 = '#22d3ee'
const COLOR_P95 = '#f59e0b'
const COLOR_P99 = '#ef4444'

function buildPoints(history) {
  const xs = history || []
  if (xs.length < 2) return ''
  let max = 0
  for (const v of xs) if (v > max) max = v
  if (max <= 0) return ''
  const W = 120, H = 28, PAD = 2
  return xs.map((y, i) => {
    const x = (i / (xs.length - 1)) * W
    const ny = H - PAD - (Math.max(0, y) / max) * (H - PAD * 2)
    return x.toFixed(1) + ',' + ny.toFixed(1)
  }).join(' ')
}

const series = computed(() => [
  {
    id: 'p50',
    label: 'P50',
    color: COLOR_P50,
    value: props.p50Current || 0,
    points: buildPoints(props.p50History),
  },
  {
    id: 'p95',
    label: 'P95',
    color: COLOR_P95,
    value: props.p95Current || 0,
    points: buildPoints(props.p95History),
  },
  {
    id: 'p99',
    label: 'P99',
    color: COLOR_P99,
    value: props.p99Current || 0,
    points: buildPoints(props.p99History),
  },
])

function formatVal(n) {
  if (!n && n !== 0) return '0.00'
  if (n >= 100) return n.toFixed(0)
  return n.toFixed(2)
}
</script>

<style scoped>
.ltp {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  padding: 0.55rem 0.7rem 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}
.ltp-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
}
.ltp-head-label {
  font-size: 0.65rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.ltp-head-unit {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.62rem;
  color: var(--fg-faint);
  font-variant-numeric: tabular-nums;
}
.ltp-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
}
.ltp-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.35rem 0.4rem 0.4rem;
  border: 1px solid var(--border-subtle);
  border-radius: 0.35rem;
  background: var(--surface-muted);
  min-height: 56px;
}
.ltp-cell-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.4rem;
}
.ltp-cell-name {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.62rem;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.ltp-cell-val {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.85rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--fg-primary);
}
.ltp-spark {
  width: 100%;
  height: 28px;
}
</style>
