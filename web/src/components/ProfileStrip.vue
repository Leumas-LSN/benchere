<template>
  <section class="ps">
    <header class="ps-head">
      <span class="ps-head-label">{{ heading }}</span>
      <span class="ps-head-counter">{{ doneCount }}/{{ totalCount }}</span>
    </header>
    <ol class="ps-list">
      <li
        v-for="(p, idx) in tiles"
        :key="p.key"
        class="ps-tile"
        :class="['ps-' + p.state]"
      >
        <span class="ps-tile-name" :title="p.full">{{ p.short }}</span>
        <span class="ps-tile-state">
          <span v-if="p.state === 'done'" class="ps-pill ps-pill-pass">{{ t('dashboard.profileStrip.statePass') }}</span>
          <span v-else-if="p.state === 'running'" class="ps-pill ps-pill-run">{{ t('dashboard.profileStrip.stateRunning') }}</span>
          <span v-else class="ps-pill ps-pill-queued">{{ t('dashboard.profileStrip.stateQueued') }}</span>
        </span>
        <div v-if="p.state === 'running'" class="ps-fill-track">
          <div class="ps-fill" :style="{ width: progressPct + '%' }"></div>
        </div>
      </li>
    </ol>
    <footer class="ps-foot">
      <span class="ps-foot-text">
        <span class="ps-foot-eta">{{ etaLabel }}</span>
        <span class="ps-foot-sep"> . </span>
        <span class="ps-foot-stat">{{ t('dashboard.profileStrip.done', { count: doneCount }) }}</span>
        <span v-if="hasRunning" class="ps-foot-sep"> . </span>
        <span v-if="hasRunning" class="ps-foot-stat">{{ t('dashboard.profileStrip.running', { count: 1 }) }}</span>
        <span v-if="queuedCount > 0" class="ps-foot-sep"> . </span>
        <span v-if="queuedCount > 0" class="ps-foot-stat">{{ t('dashboard.profileStrip.queued', { count: queuedCount }) }}</span>
      </span>
    </footer>
  </section>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWsStore } from '../stores/ws.js'

const { t } = useI18n()
const ws = useWsStore()

const props = defineProps({
  // Completed profile summaries (from wsStore.phaseSummaries). Each entry
  // must expose at least .profile_name. Order matters: earliest first.
  phaseSummaries: { type: Array, default: () => [] },
  // Name of the profile currently running (as broadcast on job_status or
  // storage_metric). Empty string when no profile is active.
  currentProfile: { type: String, default: '' },
  // Total number of profiles for the job, when known. When 0 we fall back
  // to a "best-effort" count (done + running). queued tiles only render
  // if totalProfiles is provided and exceeds done + running.
  totalProfiles:  { type: Number, default: 0 },
  // Estimated prefill seconds, forwarded for the bar fill heuristics.
  prefillEstimatedSeconds: { type: Number, default: 0 },
})

const heading = computed(() => t('dashboard.profileStrip.heading'))

const now = ref(Date.now())
let timer = null
onMounted(() => { timer = setInterval(() => { now.value = Date.now() }, 500) })
onUnmounted(() => { if (timer) clearInterval(timer) })

function shorten(name) {
  if (!name) return ''
  // Pick the most informative segment: drop a leading "ceph-" or "fio-"
  // prefix, then truncate to 8 chars to keep the strip dense.
  const cleaned = name.replace(/^(ceph|fio)-/, '')
  return cleaned.length > 8 ? cleaned.slice(0, 8) : cleaned
}

const hasRunning = computed(() => {
  const ph = props.currentProfile
  if (!ph) return false
  if (ph === 'prefill') return false
  return true
})

const doneCount = computed(() => props.phaseSummaries.length)

const totalCount = computed(() => {
  if (props.totalProfiles && props.totalProfiles > 0) return props.totalProfiles
  return doneCount.value + (hasRunning.value ? 1 : 0)
})

const queuedCount = computed(() => {
  const total = totalCount.value
  const running = hasRunning.value ? 1 : 0
  return Math.max(0, total - doneCount.value - running)
})

const tiles = computed(() => {
  const out = []
  // 1. done tiles
  for (let i = 0; i < props.phaseSummaries.length; i++) {
    const s = props.phaseSummaries[i]
    out.push({
      key: 'done-' + i + '-' + (s.profile_name || ''),
      full: s.profile_name || '',
      short: shorten(s.profile_name),
      state: 'done',
    })
  }
  // 2. running tile
  if (hasRunning.value) {
    out.push({
      key: 'run-' + props.currentProfile,
      full: props.currentProfile,
      short: shorten(props.currentProfile),
      state: 'running',
    })
  }
  // 3. queued placeholders (best effort)
  for (let i = 0; i < queuedCount.value; i++) {
    out.push({
      key: 'q-' + i,
      full: '',
      short: '',
      state: 'queued',
    })
  }
  return out
})

// Profile fill progress: % of runtime elapsed for the running profile.
// Mirrors the logic in PhaseProgress so the strip and any deep-dive show
// consistent numbers.
const progressPct = computed(() => {
  const phase = ws.jobStatus.phase
  if (!phase || phase === 'prefill') return 0
  const total = ws.jobStatus.runtimeSeconds
  if (!total) return 0
  const start = ws.profileStartedAt || 0
  if (!start) return 0
  const elapsed = Math.max(0, (now.value - start) / 1000)
  const pct = (elapsed / total) * 100
  return Math.max(0, Math.min(99, pct))
})

// ETA total = remaining of the running profile + per-profile mean for the
// queued ones. We use ws.jobStatus.runtimeSeconds when known, otherwise
// degrade to 0.
const etaLabel = computed(() => {
  const remainingRunning = (() => {
    if (!hasRunning.value) return 0
    const total = ws.jobStatus.runtimeSeconds
    if (!total) return 0
    const start = ws.profileStartedAt || 0
    if (!start) return total
    const elapsed = Math.max(0, (now.value - start) / 1000)
    return Math.max(0, total - elapsed)
  })()
  const perProfile = ws.jobStatus.runtimeSeconds || 0
  const seconds = Math.round(remainingRunning + queuedCount.value * perProfile)
  if (seconds <= 0) return t('dashboard.profileStrip.eta', { value: '--' })
  return t('dashboard.profileStrip.eta', { value: fmtMS(seconds) })
})

function fmtMS(s) {
  if (s >= 3600) {
    const h = Math.floor(s / 3600)
    const m = Math.round((s % 3600) / 60)
    return h + 'h ' + (m < 10 ? '0' + m : m) + 'm'
  }
  if (s >= 60) {
    const m = Math.floor(s / 60)
    const sec = Math.round(s % 60)
    return m + 'm ' + (sec < 10 ? '0' + sec : sec) + 's'
  }
  return s + 's'
}
</script>

<style scoped>
.ps {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  padding: 0.55rem 0.7rem 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}
.ps-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.5rem;
}
.ps-head-label {
  font-size: 0.65rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.ps-head-counter {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.72rem;
  font-variant-numeric: tabular-nums;
  color: var(--fg-secondary);
}
.ps-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  list-style: none;
  padding: 0;
  margin: 0;
}
.ps-tile {
  position: relative;
  flex: 1 1 7rem;
  min-width: 6rem;
  max-width: 11rem;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  padding: 0.4rem 0.55rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: 0.35rem;
  background: var(--surface-muted);
  overflow: hidden;
}
.ps-done {
  background: var(--surface-muted);
}
.ps-running {
  background: var(--surface-base);
  border-color: rgba(249, 115, 22, 0.4);
}
.ps-queued {
  background: var(--surface-muted);
  opacity: 0.65;
}
.ps-tile-name {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.74rem;
  color: var(--fg-primary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ps-queued .ps-tile-name::after {
  content: '...';
  color: var(--fg-faint);
}
.ps-tile-state {
  display: flex;
  align-items: center;
}
.ps-pill {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.55rem;
  font-weight: 700;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  padding: 0.05rem 0.35rem;
  border-radius: 0.2rem;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}
.ps-pill::before {
  content: '';
  width: 5px;
  height: 5px;
  border-radius: 999px;
  display: inline-block;
}
.ps-pill-pass {
  background: rgba(22,163,74,0.10);
  color: rgb(22,163,74);
}
:root.dark .ps-pill-pass {
  background: rgba(74,222,128,0.12);
  color: rgb(74,222,128);
}
.ps-pill-pass::before { background: rgb(22,163,74); }
:root.dark .ps-pill-pass::before { background: rgb(74,222,128); }

.ps-pill-run {
  background: rgba(249,115,22,0.12);
  color: #c2410c;
}
:root.dark .ps-pill-run {
  background: rgba(249,115,22,0.18);
  color: #fb923c;
}
.ps-pill-run::before { background: #f97316; }

.ps-pill-queued {
  background: var(--surface-muted);
  color: var(--fg-muted);
}
.ps-pill-queued::before { background: var(--fg-faint); }

.ps-fill-track {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 2px;
  background: rgba(249,115,22,0.12);
}
.ps-fill {
  height: 100%;
  background: #f97316;
  transition: width 0.4s ease;
}
.ps-foot {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.68rem;
  color: var(--fg-secondary);
  font-variant-numeric: tabular-nums;
}
.ps-foot-eta {
  color: var(--fg-primary);
  font-weight: 500;
}
.ps-foot-sep {
  color: var(--fg-faint);
}
</style>
