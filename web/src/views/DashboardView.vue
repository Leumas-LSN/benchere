<template>
  <div class="page-compact space-y-2.5">
    <!-- Identity row: title + status + meta + actions on a single line -->
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <div class="flex items-center gap-3 min-w-0">
        <h1 class="text-lg md:text-xl font-semibold tracking-tight fg-primary truncate">
          {{ job?.name ?? '...' }}
        </h1>
        <StatusBadge v-if="job?.status" :status="job.status" />
        <span v-if="job" class="text-sm fg-secondary truncate hidden sm:flex items-center gap-1.5">
          <span class="fg-faint">·</span>
          <span class="truncate">{{ job.client_name }}</span>
          <span class="fg-faint">·</span>
          <span class="font-mono text-xs">mode {{ job.mode }}</span>
          <span v-if="wsStore.jobStatus.phase" class="hidden md:inline-flex items-center gap-1.5">
            <span class="fg-faint">·</span>
            <span class="font-mono text-xs">{{ wsStore.jobStatus.phase }}</span>
          </span>
        </span>
      </div>
      <div class="flex items-center gap-2">
        <span class="inline-flex items-center gap-1.5 text-xs fg-secondary">
          <span
            class="w-1.5 h-1.5 rounded-full"
            :class="wsStore.connected ? 'bg-emerald-500 animate-pulse-dot' : 'bg-ink-400'"
          ></span>
          {{ wsStore.connected ? t('dashboard.wsLive') : t('dashboard.wsInactive') }}
        </span>
        <RouterLink to="/history" class="btn-secondary btn-sm">
          <Icon name="history" :size="14" />
          Historique
        </RouterLink>
        <RouterLink
          v-if="['done','failed','cancelled'].includes(job?.status)"
          :to="`/jobs/${jobId}`"
          class="btn-secondary btn-sm"
        >
          <Icon name="file_text" :size="14" />
          Resultats
        </RouterLink>
        <button
          v-if="isRunning"
          @click="cancel"
          class="btn-danger btn-sm"
          :disabled="cancelling"
        >
          <Spinner v-if="cancelling" :size="14" />
          <Icon v-else name="stop" :size="14" />
          Stop
        </button>
      </div>
    </header>

    <!-- Failure panel (compact) -->
    <div v-if="job?.status === 'failed'" class="alert-error">
      <Icon name="x_circle" :size="16" class="mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-sm">Le job a echoue</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-0.5 break-all opacity-90">
          {{ job.error_message }}
        </p>
      </div>
    </div>

    <!-- Phase progress strip: provisioning | prefill | profile -->
    <PhaseProgress
      v-if="showPhaseStrip"
      :prefill-estimated-seconds="prefillEstimatedSeconds"
    />

    <!-- KPI tiles row: 4 compact tiles, single row, dense -->
    <section class="grid grid-cols-2 md:grid-cols-4 gap-2.5">
      <div class="kpi-tile">
        <span class="kpi-label">IOPS Read</span>
        <span class="kpi-value text-brand-600 dark:text-brand-400">{{ formatIops(wsStore.elbenchoMetrics.iopsRead) }}</span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">IOPS Write</span>
        <span class="kpi-value text-brand-600 dark:text-brand-400">{{ formatIops(wsStore.elbenchoMetrics.iopsWrite) }}</span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">{{ t('dashboard.cards.throughputRead') }}</span>
        <span class="kpi-value text-sky-600 dark:text-sky-400">
          {{ (wsStore.elbenchoMetrics.throughputReadMbps || 0).toFixed(1) }}
          <span class="kpi-unit">MB/s</span>
        </span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">{{ t('dashboard.cards.throughputWrite') }}</span>
        <span class="kpi-value text-sky-600 dark:text-sky-400">
          {{ (wsStore.elbenchoMetrics.throughputWriteMbps || 0).toFixed(1) }}
          <span class="kpi-unit">MB/s</span>
        </span>
      </div>
    </section>

    <!-- Charts grid: 2x2 IOPS R/W + BW R/W. Charts auto-hide when their
         side has been zero across the entire visible window, so a read-only
         profile renders 1 IOPS chart instead of 4 - lighter on the browser
         and visually less noisy. The grid collapses to 1 column when only
         one chart is visible. -->
    <section
      class="grid gap-2.5"
      :class="visibleChartsCount > 1 ? 'grid-cols-1 lg:grid-cols-2' : 'grid-cols-1'"
    >
      <div v-if="hasReadActivity" class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('jobLive.charts.iopsRead') }}</span>
          <span class="text-xs fg-muted num">{{ wsStore.elbenchoMetrics.history.iopsRead.length }} pts</span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 230px;">
          <LineChart
            label="IOPS Read"
            :data="wsStore.elbenchoMetrics.history.iopsRead"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#f97316"
          />
        </div>
      </div>
      <div v-if="hasWriteActivity" class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('jobLive.charts.iopsWrite') }}</span>
          <span class="text-xs fg-muted num">{{ wsStore.elbenchoMetrics.history.iopsWrite.length }} pts</span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 230px;">
          <LineChart
            label="IOPS Write"
            :data="wsStore.elbenchoMetrics.history.iopsWrite"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#0ea5e9"
          />
        </div>
      </div>
      <div v-if="hasReadActivity" class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('jobLive.charts.throughputRead') }}</span>
          <span class="text-xs fg-muted num">{{ wsStore.elbenchoMetrics.history.throughputRead.length }} pts</span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 230px;">
          <LineChart
            label="Read MB/s"
            :data="wsStore.elbenchoMetrics.history.throughputRead"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#10b981"
          />
        </div>
      </div>
      <div v-if="hasWriteActivity" class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('jobLive.charts.throughputWrite') }}</span>
          <span class="text-xs fg-muted num">{{ wsStore.elbenchoMetrics.history.throughputWrite.length }} pts</span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 230px;">
          <LineChart
            label="Write MB/s"
            :data="wsStore.elbenchoMetrics.history.throughputWrite"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#a855f7"
          />
        </div>
      </div>
    </section>

    <!-- Latency strip: shorter, full-width -->
    <section class="card-flush">
      <header class="card-header">
        <span class="card-title">{{ t('jobLive.charts.latency') }}</span>
        <span class="text-xs fg-muted num">{{ (wsStore.elbenchoMetrics.latencyAvgMs || 0).toFixed(2) }} ms</span>
      </header>
      <div class="px-3 pb-2 pt-1.5" style="height: 130px;">
        <LineChart
          label="Latence"
          :data="wsStore.elbenchoMetrics.history.latency"
          :labels="wsStore.elbenchoMetrics.history.labels"
          color="#7c3aed"
        />
      </div>
    </section>

    <!-- Live logs panel (collapsible) -->
    <LiveLogsPanel />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useWsStore } from '../stores/ws.js'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import LineChart      from '../components/LineChart.vue'
import StatusBadge    from '../components/StatusBadge.vue'
import Icon           from '../components/Icon.vue'
import Spinner        from '../components/Spinner.vue'
import LiveLogsPanel  from '../components/LiveLogsPanel.vue'
import PhaseProgress  from '../components/PhaseProgress.vue'

const { t } = useI18n()

const route     = useRoute()
const wsStore   = useWsStore()
const jobsStore = useJobsStore()

const jobId      = route.params.id
const job        = ref(null)
const workers    = ref([])
const cancelling = ref(false)

const TERMINAL = new Set(['done', 'failed', 'cancelled'])

const isRunning = computed(() =>
  job.value?.status === 'running' || job.value?.status === 'provisioning'
)

const showPhaseStrip = computed(() => {
  const s = job.value?.status
  return s === 'provisioning' || s === 'running'
})

// Auto-hide a side (read or write) when its history has no signal yet OR
// stays consistently zero. This keeps the dashboard at 1 chart for mono-
// directional profiles (rand-4k-write, seq-256k-read, ...) and 4 charts
// only for mixed profiles. Threshold is "any sample > 0 in the window"
// so a single non-zero sample brings the chart back.
//
// Until at least 3 samples have arrived we show both sides so the user
// is never staring at an empty page right after the bench starts.
function sideHasActivity(iopsHistory, bwHistory) {
  const total = (iopsHistory?.length ?? 0)
  if (total < 3) return true
  for (let i = 0; i < total; i++) {
    if ((iopsHistory[i] || 0) > 0) return true
    if ((bwHistory[i]   || 0) > 0) return true
  }
  return false
}

const hasReadActivity = computed(() =>
  sideHasActivity(
    wsStore.elbenchoMetrics.history.iopsRead,
    wsStore.elbenchoMetrics.history.throughputRead,
  )
)
const hasWriteActivity = computed(() =>
  sideHasActivity(
    wsStore.elbenchoMetrics.history.iopsWrite,
    wsStore.elbenchoMetrics.history.throughputWrite,
  )
)
const visibleChartsCount = computed(() =>
  (hasReadActivity.value ? 2 : 0) + (hasWriteActivity.value ? 2 : 0)
)

// Estimated prefill total = data_disk_gb * num_workers * 10s.
// Falls back to 0 (PhaseProgress shows elapsed-only) when we cannot infer.
const prefillEstimatedSeconds = computed(() => {
  const gb = job.value?.data_disk_gb || 0
  const n  = workers.value.length || 0
  if (!gb || !n) return 0
  return gb * n * 10
})

function formatIops(n) {
  if (!n && n !== 0) return '—'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1000)      return (n / 1000).toFixed(1) + 'k'
  return n.toFixed(0)
}

async function cancel() {
  cancelling.value = true
  try { await jobsStore.cancelJob(jobId) }
  finally { cancelling.value = false }
}

watch(() => job.value?.status, (s) => { if (s !== 'provisioning') wsStore.resetProvSteps() })

let pollInterval = null

async function pollJob() {
  try {
    job.value = await api.getJob(jobId)
    const w = await jobsStore.listWorkers(jobId)
    if (w) workers.value = w
    if (TERMINAL.has(job.value?.status)) {
      clearInterval(pollInterval); pollInterval = null
    }
  } catch (_) {}
}

onMounted(async () => {
  wsStore.reset()
  wsStore.connect(jobId)
  await pollJob()
  pollInterval = setInterval(pollJob, 3000)
})

onUnmounted(() => {
  wsStore.disconnect()
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
.page-compact {
  padding: 0.5rem 1rem 0.75rem;
  max-width: 1600px;
  margin: 0 auto;
}
@media (min-width: 1280px) {
  .page-compact {
    padding: 0.5rem 1.5rem 0.75rem;
  }
}
.kpi-tile {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.55rem 0.85rem;
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
}
.kpi-label {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.kpi-value {
  font-family: "Geist Mono", ui-monospace, monospace;
  font-size: 1.35rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}
.kpi-unit {
  font-family: "Geist", system-ui, sans-serif;
  font-size: 0.7rem;
  font-weight: 500;
  color: var(--fg-muted);
  margin-left: 0.15rem;
}
</style>
