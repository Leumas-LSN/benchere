<template>
  <div class="page-compact space-y-2.5">
    <header class="flex items-center justify-between gap-4 flex-wrap">
      <div class="flex items-center gap-3 min-w-0">
        <h1 class="text-lg md:text-xl font-semibold tracking-tight fg-primary truncate">{{ job?.name ?? '...' }}</h1>
        <StatusBadge v-if="job?.status" :status="job.status" />
        <span v-if="job" class="text-sm fg-secondary truncate hidden sm:flex items-center gap-1.5">
          <span class="fg-faint">&#xB7;</span>
          <span class="truncate">{{ job.client_name }}</span>
          <span class="fg-faint">&#xB7;</span>
          <span class="font-mono text-xs">{{ job.engine || 'fio' }}</span>
          <span class="fg-faint">&#xB7;</span>
          <span class="font-mono text-xs">mode {{ job.mode }}</span>
          <span v-if="wsStore.jobStatus.phase" class="hidden md:inline-flex items-center gap-1.5">
            <span class="fg-faint">&#xB7;</span>
            <span class="font-mono text-xs">{{ wsStore.jobStatus.phase }}</span>
          </span>
        </span>
      </div>
      <div class="flex items-center gap-2">
        <span class="inline-flex items-center gap-1.5 text-xs fg-secondary">
          <span class="w-1.5 h-1.5 rounded-full" :class="wsStore.connected ? 'bg-emerald-500 animate-pulse-dot' : 'bg-ink-400'"></span>
          {{ wsStore.connected ? t('dashboard.wsLive') : t('dashboard.wsInactive') }}
        </span>
        <RouterLink to="/history" class="btn-secondary btn-sm"><Icon name="history" :size="14" />Historique</RouterLink>
        <RouterLink v-if="['done','failed','cancelled'].includes(job?.status)" :to="'/jobs/' + jobId" class="btn-secondary btn-sm">
          <Icon name="file_text" :size="14" />Resultats
        </RouterLink>
        <button v-if="isRunning" @click="cancel" class="btn-danger btn-sm" :disabled="cancelling">
          <Spinner v-if="cancelling" :size="14" /><Icon v-else name="stop" :size="14" />Stop
        </button>
      </div>
    </header>

    <div v-if="job?.status === 'failed'" class="alert-error">
      <Icon name="x_circle" :size="16" class="mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-sm">Le job a echoue</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-0.5 break-all opacity-90">{{ job.error_message }}</p>
      </div>
    </div>

    <PhaseProgress v-if="showPhaseStrip" :prefill-estimated-seconds="prefillEstimatedSeconds" />

    <section class="grid grid-cols-2 md:grid-cols-4 gap-2.5">
      <div class="kpi-tile">
        <span class="kpi-label">IOPS Read</span>
        <span class="kpi-value text-brand-600 dark:text-brand-400">{{ formatIops(wsStore.liveMetrics.iopsRead) }}</span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">IOPS Write</span>
        <span class="kpi-value text-brand-600 dark:text-brand-400">{{ formatIops(wsStore.liveMetrics.iopsWrite) }}</span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">Throughput R+W</span>
        <span class="kpi-value text-sky-600 dark:text-sky-400">
          {{ ((wsStore.liveMetrics.throughputReadMbps || 0) + (wsStore.liveMetrics.throughputWriteMbps || 0)).toFixed(0) }}
          <span class="kpi-unit">MB/s</span>
        </span>
      </div>
      <div class="kpi-tile">
        <span class="kpi-label">Latency p99</span>
        <span class="kpi-value" :class="(wsStore.liveMetrics.latencyP99Ms||0) > 5 ? 'text-red-600 dark:text-red-400' : 'text-violet-600 dark:text-violet-400'">
          {{ (wsStore.liveMetrics.latencyP99Ms || 0).toFixed(2) }}
          <span class="kpi-unit">ms</span>
        </span>
      </div>
    </section>

    <section class="grid grid-cols-1 lg:grid-cols-2 gap-2.5">
      <div class="card-flush">
        <header class="card-header"><span class="card-title">IOPS over time</span></header>
        <div class="px-3 pb-2 pt-1.5" style="height: 220px;">
          <UPlotMulti :series="iopsSeries" />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header"><span class="card-title">Throughput over time (MB/s)</span></header>
        <div class="px-3 pb-2 pt-1.5" style="height: 220px;">
          <UPlotMulti :series="bwSeries" />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header"><span class="card-title">Latency percentiles (ms, log)</span></header>
        <div class="px-3 pb-2 pt-1.5" style="height: 220px;">
          <UPlotMulti :series="latSeries" :log="true" />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header"><span class="card-title">Cluster CPU per node (%)</span></header>
        <div class="px-3 pb-2 pt-1.5" style="height: 220px;">
          <UPlotMulti :series="clusterCpuSeries" />
        </div>
      </div>
    </section>

    <section v-if="workers.length" class="card-flush">
      <header class="card-header"><span class="card-title">Workers</span><span class="pill num text-xs">{{ workers.length }}</span></header>
      <div class="p-3 grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-2.5">
        <WorkerLiveTile
          v-for="(w, i) in workers"
          :key="w.id"
          :name="'W' + (i+1) + ' \xB7 ' + ((w.ip||'').split('.').pop() || w.vm_id)"
          :status="w.status"
          :cpu="wsStore.workerMetrics[w.id]?.cpuPct || 0"
          :ram="wsStore.workerMetrics[w.id]?.ramPct || 0"
          :net-in="wsStore.workerMetrics[w.id]?.netInBps || 0"
          :net-out="wsStore.workerMetrics[w.id]?.netOutBps || 0"
          :disk-read="wsStore.workerMetrics[w.id]?.diskReadBps || 0"
          :disk-write="wsStore.workerMetrics[w.id]?.diskWriteBps || 0"
          :cpu-history="wsStore.workerMetrics[w.id]?.cpuHistory || []"
          :saturation="wsStore.workerMetrics[w.id]?.saturation || null"
        />
      </div>
    </section>

    <section v-if="clusterNodes.length" class="card-flush">
      <header class="card-header"><span class="card-title">Cluster Proxmox</span><span class="pill num text-xs">{{ clusterNodes.length }}</span></header>
      <div class="p-3 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2.5">
        <ClusterLiveCard
          v-for="n in clusterNodes"
          :key="n.name"
          :name="n.name"
          :cpu="n.cpuPct || 0"
          :ram="n.ramPct || 0"
          :load="n.loadAvg || 0"
          :history="n.history || []"
        />
      </div>
    </section>

    <LiveLogsPanel />

    <section v-if="wsStore.phaseSummaries.length" class="card-flush">
      <header class="card-header"><span class="card-title">Profils completes</span></header>
      <div class="p-3 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2.5">
        <div v-for="(s, i) in lastSummaries" :key="i" class="rounded-lg surface-muted p-3 text-sm">
          <p class="font-mono text-sm fg-primary truncate">{{ s.profile_name }}</p>
          <div class="grid grid-cols-2 gap-x-3 gap-y-1 mt-2 text-[11px] num">
            <div class="flex justify-between"><span class="fg-muted">iops r avg</span><span class="fg-secondary">{{ formatIops(s.iops_read_avg) }}</span></div>
            <div class="flex justify-between"><span class="fg-muted">iops w avg</span><span class="fg-secondary">{{ formatIops(s.iops_write_avg) }}</span></div>
            <div class="flex justify-between"><span class="fg-muted">p99</span><span class="fg-secondary">{{ (s.lat_p99_ms||0).toFixed(2) }}ms</span></div>
            <div class="flex justify-between"><span class="fg-muted">CV</span><span :class="(s.iops_cv_pct||0) > 10 ? 'text-amber-600' : 'fg-secondary'">{{ (s.iops_cv_pct||0).toFixed(1) }}%</span></div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, RouterLink } from 'vue-router'
import { useWsStore } from '../stores/ws.js'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import StatusBadge     from '../components/StatusBadge.vue'
import Icon            from '../components/Icon.vue'
import Spinner         from '../components/Spinner.vue'
import LiveLogsPanel   from '../components/LiveLogsPanel.vue'
import PhaseProgress   from '../components/PhaseProgress.vue'
import UPlotMulti      from '../components/UPlotMulti.vue'
import WorkerLiveTile  from '../components/WorkerLiveTile.vue'
import ClusterLiveCard from '../components/ClusterLiveCard.vue'

const { t } = useI18n()
const route     = useRoute()
const wsStore   = useWsStore()
const jobsStore = useJobsStore()

const jobId      = route.params.id
const job        = ref(null)
const workers    = ref([])
const cancelling = ref(false)

const TERMINAL = new Set(['done', 'failed', 'cancelled'])

const isRunning = computed(() => job.value?.status === 'running' || job.value?.status === 'provisioning')
const showPhaseStrip = computed(() => isRunning.value)

const iopsSeries = computed(() => [
  { label: 'read',  color: '#f97316', data: wsStore.liveMetrics.history.iopsRead.slice() },
  { label: 'write', color: '#0ea5e9', data: wsStore.liveMetrics.history.iopsWrite.slice() },
])
const bwSeries = computed(() => [
  { label: 'read',  color: '#10b981', data: wsStore.liveMetrics.history.throughputRead.slice() },
  { label: 'write', color: '#a855f7', data: wsStore.liveMetrics.history.throughputWrite.slice() },
])
const latSeries = computed(() => [
  { label: 'p50', color: '#22d3ee', data: wsStore.liveMetrics.history.latencyP50.slice() },
  { label: 'p95', color: '#f59e0b', data: wsStore.liveMetrics.history.latencyP95.slice() },
  { label: 'p99', color: '#ef4444', data: wsStore.liveMetrics.history.latencyP99.slice() },
])
const clusterNodes = computed(() => Object.entries(wsStore.nodeMetrics).map(function(e) { return Object.assign({ name: e[0] }, e[1]) }))
const palette = ['#f97316', '#10b981', '#0ea5e9', '#a855f7', '#ef4444', '#22d3ee']
const clusterCpuSeries = computed(() =>
  clusterNodes.value.map(function(n, i) {
    return { label: n.name, color: palette[i % palette.length], data: (n.history || []).slice() }
  })
)

const lastSummaries = computed(() => wsStore.phaseSummaries.slice(-3).reverse())

const prefillEstimatedSeconds = computed(() => {
  const gb = job.value?.data_disk_gb || 0
  const n  = workers.value.length || 0
  if (!gb || !n) return 0
  return gb * n * 10
})

function formatIops(n) {
  if (!n && n !== 0) return '0'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000)    return (n / 1000).toFixed(1) + 'k'
  return n.toFixed(0)
}

async function cancel() {
  cancelling.value = true
  try { await jobsStore.cancelJob(jobId) }
  finally { cancelling.value = false }
}

watch(function() { return job.value?.status }, function(s) { if (s !== 'provisioning') wsStore.resetProvSteps() })

let pollInterval = null
async function pollJob() {
  try {
    job.value = await api.getJob(jobId)
    const w = await jobsStore.listWorkers(jobId)
    if (w) workers.value = w
    if (TERMINAL.has(job.value?.status)) { clearInterval(pollInterval); pollInterval = null }
  } catch (_) {}
}

onMounted(async function() {
  wsStore.reset()
  wsStore.connect(jobId)
  await pollJob()
  pollInterval = setInterval(pollJob, 3000)
})

onUnmounted(function() {
  wsStore.disconnect()
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
.page-compact { padding: 0.5rem 1rem 0.75rem; max-width: 1600px; margin: 0 auto; }
@media (min-width: 1280px) { .page-compact { padding: 0.5rem 1.5rem 0.75rem; } }
.kpi-tile { display: flex; align-items: baseline; justify-content: space-between; gap: 0.75rem; padding: 0.55rem 0.85rem; border: 1px solid var(--border-subtle); border-radius: 0.5rem; background: var(--surface-base); }
.kpi-label { font-size: 0.7rem; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; color: var(--fg-muted); }
.kpi-value { font-family: 'Geist Mono', ui-monospace, monospace; font-size: 1.35rem; font-weight: 600; font-variant-numeric: tabular-nums; line-height: 1; }
.kpi-unit { font-family: 'Geist', system-ui, sans-serif; font-size: 0.7rem; font-weight: 500; color: var(--fg-muted); margin-left: 0.15rem; }
</style>
