<template>
  <div class="kpi-spark" :class="toneClass">
    <div class="kpi-spark-head">
      <span class="kpi-spark-label">{{ label }}</span>
      <span v-if="suffix" class="kpi-spark-suffix">{{ suffix }}</span>
    </div>
    <div class="kpi-spark-value">
      {{ display }}
      <span v-if="unit" class="kpi-spark-unit">{{ unit }}</span>
    </div>
    <svg viewBox="0 0 120 24" class="kpi-spark-svg" preserveAspectRatio="none">
      <polyline :points="points" fill="none" :stroke="strokeColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" />
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  label:   { type: String,  required: true },
  suffix:  { type: String,  default: '' },
  value:   { type: Number,  default: 0 },
  unit:    { type: String,  default: '' },
  history: { type: Array,   default: () => [] },
  format:  { type: Function, default: null },
  tone:    { type: String,  default: 'neutral' }, // neutral | brand | sky | emerald | violet | red
  warnAbove: { type: Number, default: 0 }, // when value > this, switch to red tone
})

const display = computed(() => {
  if (props.format) return props.format(props.value)
  if (props.value == null) return '0'
  if (props.value >= 1_000_000) return (props.value / 1_000_000).toFixed(1) + 'M'
  if (props.value >= 1000)      return (props.value / 1000).toFixed(1) + 'k'
  if (props.value >= 100)       return props.value.toFixed(0)
  return props.value.toFixed(2)
})

const effectiveTone = computed(() => {
  if (props.warnAbove > 0 && props.value > props.warnAbove) return 'red'
  return props.tone
})

const toneClass = computed(() => 'kpi-tone-' + effectiveTone.value)

const strokeColor = computed(() => {
  switch (effectiveTone.value) {
    case 'brand':   return '#f97316'
    case 'sky':     return '#0ea5e9'
    case 'emerald': return '#10b981'
    case 'violet':  return '#8b5cf6'
    case 'red':     return '#ef4444'
    default:        return '#737373'
  }
})

const points = computed(() => {
  const xs = props.history || []
  if (xs.length < 2) return ''
  let max = 0
  for (const v of xs) if (v > max) max = v
  if (max <= 0) return xs.map((_, i) => `${(i / (xs.length - 1)) * 120},22`).join(' ')
  const W = 120, H = 24
  const PAD = 1
  return xs.map((y, i) => {
    const x = (i / (xs.length - 1)) * W
    const ny = H - PAD - (Math.max(0, y) / max) * (H - PAD * 2)
    return `${x.toFixed(1)},${ny.toFixed(1)}`
  }).join(' ')
})
</script>

<style scoped>
.kpi-spark {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.7rem 0.9rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  min-height: 96px;
  position: relative;
  overflow: hidden;
}
.kpi-spark-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
}
.kpi-spark-label {
  font-size: 0.65rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.kpi-spark-suffix {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--fg-faint);
  font-variant-numeric: tabular-nums;
}
.kpi-spark-value {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 1.6rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1.05;
}
.kpi-spark-unit {
  font-family: 'Geist', system-ui, sans-serif;
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--fg-muted);
  margin-left: 0.2rem;
}
.kpi-spark-svg {
  width: 100%;
  height: 24px;
  margin-top: auto;
}
.kpi-tone-brand   .kpi-spark-value { color: #f97316; }
.kpi-tone-sky     .kpi-spark-value { color: #0ea5e9; }
.kpi-tone-emerald .kpi-spark-value { color: #10b981; }
.kpi-tone-violet  .kpi-spark-value { color: #8b5cf6; }
.kpi-tone-red     .kpi-spark-value { color: #ef4444; }
.kpi-tone-neutral .kpi-spark-value { color: var(--fg-primary); }
</style>
