<template>
  <section
    class="rounded-lg border px-4 py-2.5 flex items-center gap-4"
    :style="{ borderColor: 'var(--border-subtle)', background: 'var(--surface-base)' }"
  >
    <div class="flex-1 min-w-0">
      <header class="flex items-baseline justify-between gap-3 mb-1.5">
        <span class="flex items-baseline gap-2 min-w-0">
          <span class="text-xs uppercase tracking-wide fg-muted shrink-0">{{ heading }}</span>
          <span class="font-mono text-sm fg-primary truncate">{{ label }}</span>
        </span>
        <span class="font-mono text-xs fg-secondary tabular-nums whitespace-nowrap">
          {{ timing }}
        </span>
      </header>
      <ProgressBar :value="progress" :tone="tone" />
    </div>
  </section>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useWsStore } from '../stores/ws.js'
import ProgressBar from './ProgressBar.vue'

const props = defineProps({
  // Estimated total prefill duration in seconds. Computed by the parent
  // from data_disk_gb x num_workers x 10s. When 0, we degrade to an
  // elapsed-only display with an indeterminate-style 95% cap.
  prefillEstimatedSeconds: { type: Number, default: 0 },
})

const ws = useWsStore()
const now = ref(Date.now())
let timer = null

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now() }, 500)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// Mode selection. We only render when there is *something* to track:
//  - status=provisioning -> use wsStore.provProgress directly
//  - phase='prefill'     -> elapsed-only with optional estimate
//  - phase=<profile> with runtimeSeconds>0 -> profile mode
const mode = computed(() => {
  const s = ws.jobStatus.status
  const ph = ws.jobStatus.phase
  if (s === 'provisioning') return 'provisioning'
  if (s !== 'running')      return 'idle'
  if (ph === 'prefill')     return 'prefill'
  if (ph && ws.jobStatus.runtimeSeconds > 0) return 'profile'
  if (ph) return 'phase' // running, profile name but no runtime info -> elapsed only
  return 'idle'
})

const heading = computed(() => {
  switch (mode.value) {
    case 'provisioning': return 'Provisionnement'
    case 'prefill':      return 'Prefill'
    case 'profile':      return 'Bench'
    case 'phase':        return 'Phase'
    default:             return 'En attente'
  }
})

const label = computed(() => {
  if (mode.value === 'provisioning') {
    const last = ws.provSteps.length ? ws.provSteps[ws.provSteps.length - 1] : null
    return last?.detail || ws.jobStatus.phase || '...'
  }
  if (mode.value === 'prefill') return 'Allocation des data disks'
  if (ws.jobStatus.phase) return ws.jobStatus.phase
  return ''
})

const tone = computed(() => {
  if (mode.value === 'provisioning') return 'brand'
  if (mode.value === 'prefill')      return 'info'
  return 'brand'
})

function fmtMMSS(seconds) {
  if (!isFinite(seconds) || seconds < 0) seconds = 0
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

const elapsedPrefill = computed(() => {
  if (!ws.prefillStartedAt) return 0
  return Math.max(0, (now.value - ws.prefillStartedAt) / 1000)
})

const elapsedProfile = computed(() => {
  if (!ws.profileStartedAt) return 0
  return Math.max(0, (now.value - ws.profileStartedAt) / 1000)
})

const progress = computed(() => {
  if (mode.value === 'provisioning') {
    return Math.max(0, Math.min(100, ws.provProgress * 100))
  }
  if (mode.value === 'prefill') {
    if (props.prefillEstimatedSeconds > 0) {
      const pct = (elapsedPrefill.value / props.prefillEstimatedSeconds) * 100
      return Math.max(0, Math.min(95, pct))
    }
    // No estimate: show a stalled-low bar (5%) so the bar is visible but
    // does not pretend to know progress.
    return 5
  }
  if (mode.value === 'profile') {
    const pct = (elapsedProfile.value / ws.jobStatus.runtimeSeconds) * 100
    return Math.max(0, Math.min(99, pct))
  }
  return 0
})

const timing = computed(() => {
  if (mode.value === 'provisioning') {
    return `${Math.round(ws.provProgress * 100)}%`
  }
  if (mode.value === 'prefill') {
    if (props.prefillEstimatedSeconds > 0) {
      const eta = Math.max(0, props.prefillEstimatedSeconds - elapsedPrefill.value)
      return `${fmtMMSS(elapsedPrefill.value)} / ~${fmtMMSS(props.prefillEstimatedSeconds)}  ETA ${fmtMMSS(eta)}`
    }
    return fmtMMSS(elapsedPrefill.value)
  }
  if (mode.value === 'profile') {
    const total = ws.jobStatus.runtimeSeconds
    return `${fmtMMSS(elapsedProfile.value)} / ${fmtMMSS(total)}  ${Math.round((elapsedProfile.value / total) * 100)}%`
  }
  if (mode.value === 'phase') {
    return fmtMMSS(elapsedProfile.value || elapsedPrefill.value)
  }
  return ''
})
</script>
