<template>
  <div class="page-compact space-y-3">
    <!-- 1. Header -->
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

    <!-- 2. Failure banner -->
    <div v-if="job?.status === 'failed'" class="alert-error">
      <Icon name="x_circle" :size="16" class="mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-sm">Le job a echoue</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-0.5 break-all opacity-90">{{ job.error_message }}</p>
      </div>
    </div>

    <!-- 3. Phase progress -->
    <PhaseProgress v-if="showPhaseStrip" :prefill-estimated-seconds="prefillEstimatedSeconds" />

    <!-- 4. Performance charts. Three side by side on desktop, stack on mobile.
         Each card carries the current value inline in the header so we no
         longer need a separate KPI tile row. -->
    <section class="grid grid-cols-1 lg:grid-cols-3 gap-3">
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">IOPS</span>
          <span class="num text-xs">
            <span class="text-orange-500">{{ formatIops(wsStore.liveMetrics.iopsRead) }}<span class="fg-muted ml-0.5">r</span></span>
            <span class="fg-faint mx-1.5">&#xB7;</span>
            <span class="text-sky-500">{{ formatIops(wsStore.liveMetrics.iopsWrite) }}<span class="fg-muted ml-0.5">w</span></span>
          </span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 240px;">
          <UPlotMulti :series="iopsSeries" />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">Throughput (MB/s)</span>
          <span class="num text-xs">
            <span class="text-emerald-500">{{ (wsStore.liveMetrics.throughputReadMbps || 0).toFixed(0) }}<span class="fg-muted ml-0.5">r</span></span>
            <span class="fg-faint mx-1.5">&#xB7;</span>
            <span class="text-violet-500">{{ (wsStore.liveMetrics.throughputWriteMbps || 0).toFixed(0) }}<span class="fg-muted ml-0.5">w</span></span>
          </span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 240px;">
          <UPlotMulti :series="bwSeries" />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">Latence (ms, log)</span>
          <span class="num text-xs">
            <span class="fg-muted">p99</span>
            <span :class="p99Tone" class="ml-1.5 font-medium">{{ (wsStore.liveMetrics.latencyP99Ms || 0).toFixed(2) }}</span>
          </span>
        </header>
        <div class="px-3 pb-2 pt-1.5" style="height: 240px;">
          <UPlotMulti :series="latSeries" :log="true" />
        </div>
      </div>
    </section>

    <!-- 5. Infrastructure: workers + cluster proxmox side by side. -->
    <section class="grid grid-cols-1 lg:grid-cols-3 gap-3">
      <div v-if="workers.length" class="card-flush lg:col-span-2">
        <header class="card-header">
          <span class="card-title">Workers</span>
          <span class="pill num text-xs">{{ workers.length }}</span>
        </header>
        <div class="p-3 grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-2">
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
      </div>
      <div v-if="clusterNodes.length" class="card-flush">
        <header class="card-header">
          <span class="card-title">Cluster Proxmox</span>
          <span class="pill num text-xs">{{ clusterNodes.length }}</span>
        </header>
        <div class="p-3 space-y-2">
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
      </div>
    </section>

    <!-- 6. Live logs (collapsible) -->
    <LiveLogsPanel />

    <!-- 7. Phase summaries strip -->
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

const isRunning      = computed(() => job.value?.status === 'running' || job.value?.status === 'provisioning')
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

const p99Tone = computed(() => {
  const p = wsStore.liveMetrics.latencyP99Ms || 0
  if (p > 5)    return 'text-red-600 dark:text-red-400'
  if (p > 1)    return 'text-amber-600 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
})

const clusterNodes = computed(() =>
  Object.entries(wsStore.nodeMetrics).map(function(e) { return Object.assign({ name: e[0] }, e[1]) })
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
  // wsStore.connect() is jobId-aware (v2.0.5): reset is only triggered
  // when the jobId actually changes, so re-mounting the dashboard for
  // the same job preserves accumulated history.
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
</style>
