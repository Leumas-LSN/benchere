<template>
  <div class="dash">

    <!-- 1. Identity bar -->
    <header class="dash-header">
      <div class="dash-header-left">
        <h1 class="dash-title">{{ job?.name ?? '...' }}</h1>
        <StatusBadge v-if="job?.status" :status="job.status" />
        <span v-if="job" class="dash-meta">
          <span>{{ job.client_name }}</span>
          <span class="dash-sep">/</span>
          <span class="dash-mono">{{ job.engine || 'fio' }}</span>
          <span class="dash-sep">/</span>
          <span class="dash-mono">{{ job.mode }}</span>
          <span v-if="wsStore.jobStatus.phase" class="dash-meta-phase">
            <span class="dash-sep">/</span>
            <span class="dash-mono dash-phase-tag">{{ wsStore.jobStatus.phase }}</span>
          </span>
        </span>
      </div>
      <div class="dash-header-right">
        <span class="dash-conn">
          <span class="dash-conn-dot" :class="wsStore.connected ? 'dash-conn-on' : 'dash-conn-off'"></span>
          {{ wsStore.connected ? 'live' : 'offline' }}
        </span>
        <RouterLink to="/history" class="btn-secondary btn-sm"><Icon name="history" :size="13" />Historique</RouterLink>
        <RouterLink v-if="['done','failed','cancelled'].includes(job?.status)" :to="'/jobs/' + jobId" class="btn-secondary btn-sm">
          <Icon name="file_text" :size="13" />Resultats
        </RouterLink>
        <button v-if="isRunning" @click="cancel" class="btn-danger btn-sm" :disabled="cancelling">
          <Spinner v-if="cancelling" :size="13" /><Icon v-else name="stop" :size="13" />Stop
        </button>
      </div>
    </header>

    <!-- 2. Failure (conditional) -->
    <div v-if="job?.status === 'failed'" class="alert-error dash-alert">
      <Icon name="x_circle" :size="15" class="mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold text-sm">Le job a echoue</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-0.5 break-all opacity-90">{{ job.error_message }}</p>
      </div>
    </div>

    <!-- 3. Phase progress -->
    <PhaseProgress v-if="showPhaseStrip" :prefill-estimated-seconds="prefillEstimatedSeconds" />

    <!-- 4. KPI tiles with sparklines -->
    <section class="dash-grid-kpi">
      <KpiSpark
        label="IOPS read"
        :value="wsStore.liveMetrics.iopsRead || 0"
        :history="wsStore.liveMetrics.history.iopsRead"
        tone="brand"
      />
      <KpiSpark
        label="IOPS write"
        :value="wsStore.liveMetrics.iopsWrite || 0"
        :history="wsStore.liveMetrics.history.iopsWrite"
        tone="sky"
      />
      <KpiSpark
        label="Throughput"
        suffix="r+w"
        :value="(wsStore.liveMetrics.throughputReadMbps || 0) + (wsStore.liveMetrics.throughputWriteMbps || 0)"
        unit="MB/s"
        :history="combinedThroughputHistory"
        tone="emerald"
        :format="formatBwTotal"
      />
      <KpiSpark
        label="Latency p99"
        unit="ms"
        :value="wsStore.liveMetrics.latencyP99Ms || 0"
        :history="wsStore.liveMetrics.history.latencyP99"
        tone="violet"
        :warn-above="5"
        :format="formatLatencyValue"
      />
    </section>

    <!-- 5. Trend charts row (3 columns) -->
    <section class="dash-grid-charts">
      <div class="dash-chart-card">
        <header class="dash-chart-head">
          <span class="dash-chart-title">IOPS over time</span>
          <span class="dash-chart-legend">
            <span class="dash-legend-dot" style="background:#f97316"></span>read
            <span class="dash-legend-dot ml-2" style="background:#0ea5e9"></span>write
          </span>
        </header>
        <div class="dash-chart-body">
          <UPlotMulti :series="iopsSeries" />
        </div>
      </div>
      <div class="dash-chart-card">
        <header class="dash-chart-head">
          <span class="dash-chart-title">Throughput MB/s</span>
          <span class="dash-chart-legend">
            <span class="dash-legend-dot" style="background:#10b981"></span>read
            <span class="dash-legend-dot ml-2" style="background:#a855f7"></span>write
          </span>
        </header>
        <div class="dash-chart-body">
          <UPlotMulti :series="bwSeries" />
        </div>
      </div>
      <div class="dash-chart-card">
        <header class="dash-chart-head">
          <span class="dash-chart-title">Latency ms (log)</span>
          <span class="dash-chart-legend">
            <span class="dash-legend-dot" style="background:#22d3ee"></span>p50
            <span class="dash-legend-dot ml-2" style="background:#f59e0b"></span>p95
            <span class="dash-legend-dot ml-2" style="background:#ef4444"></span>p99
          </span>
        </header>
        <div class="dash-chart-body">
          <UPlotMulti :series="latSeries" :log="true" />
        </div>
      </div>
    </section>

    <!-- 6. Infrastructure: workers (60%) + cluster nodes (40%) -->
    <section class="dash-grid-infra">
      <div class="dash-card dash-card-wide">
        <header class="dash-card-head">
          <span class="dash-card-title">Workers</span>
          <span class="dash-card-pill">{{ workers.length }}</span>
        </header>
        <div class="dash-card-body dash-workers-grid">
          <WorkerCompactTile
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
            :saturation="wsStore.workerMetrics[w.id]?.saturation || null"
          />
        </div>
      </div>
      <div class="dash-card dash-card-narrow">
        <header class="dash-card-head">
          <span class="dash-card-title">Cluster Proxmox</span>
          <span class="dash-card-pill">{{ clusterNodes.length }}</span>
        </header>
        <div class="dash-card-body dash-nodes-list">
          <NodeRow
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

    <!-- 7. Live logs (terminal-feel) -->
    <LiveLogsPanel />

    <!-- 8. Phase summaries strip -->
    <section v-if="wsStore.phaseSummaries.length" class="dash-card">
      <header class="dash-card-head">
        <span class="dash-card-title">Profils completes</span>
        <span class="dash-card-pill">{{ wsStore.phaseSummaries.length }}</span>
      </header>
      <div class="dash-card-body dash-summaries-grid">
        <div v-for="(s, i) in lastSummaries" :key="i" class="dash-summary-tile">
          <p class="dash-summary-name">{{ s.profile_name }}</p>
          <div class="dash-summary-stats">
            <div><span class="dash-summary-label">iops r</span><span class="dash-summary-val">{{ formatIops(s.iops_read_avg) }}</span></div>
            <div><span class="dash-summary-label">iops w</span><span class="dash-summary-val">{{ formatIops(s.iops_write_avg) }}</span></div>
            <div><span class="dash-summary-label">p99</span><span class="dash-summary-val">{{ (s.lat_p99_ms||0).toFixed(2) }} ms</span></div>
            <div><span class="dash-summary-label">cv</span><span class="dash-summary-val" :class="(s.iops_cv_pct||0) > 10 ? 'text-amber-600 dark:text-amber-400' : ''">{{ (s.iops_cv_pct||0).toFixed(1) }}%</span></div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { useWsStore } from '../stores/ws.js'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import StatusBadge        from '../components/StatusBadge.vue'
import Icon               from '../components/Icon.vue'
import Spinner            from '../components/Spinner.vue'
import LiveLogsPanel      from '../components/LiveLogsPanel.vue'
import PhaseProgress      from '../components/PhaseProgress.vue'
import UPlotMulti         from '../components/UPlotMulti.vue'
import KpiSpark           from '../components/KpiSpark.vue'
import NodeRow            from '../components/NodeRow.vue'
import WorkerCompactTile  from '../components/WorkerCompactTile.vue'

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
const combinedThroughputHistory = computed(() => {
  const r = wsStore.liveMetrics.history.throughputRead || []
  const w = wsStore.liveMetrics.history.throughputWrite || []
  const out = []
  const n = Math.max(r.length, w.length)
  for (let i = 0; i < n; i++) out.push((r[i] || 0) + (w[i] || 0))
  return out
})
const clusterNodes = computed(() =>
  Object.entries(wsStore.nodeMetrics).map((e) => Object.assign({ name: e[0] }, e[1]))
)
const lastSummaries = computed(() => wsStore.phaseSummaries.slice(-6).reverse())

const prefillEstimatedSeconds = computed(() => {
  const gb = job.value?.data_disk_gb || 0
  const n  = workers.value.length || 0
  if (!gb || !n) return 0
  return gb * n * 10
})

function formatIops(n) {
  if (!n && n !== 0) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1000)      return (n / 1000).toFixed(1) + 'k'
  return n.toFixed(0)
}
function formatBwTotal(n) {
  if (!n && n !== 0) return '0'
  if (n >= 1024) return (n / 1024).toFixed(2) + ' G'
  return n.toFixed(0)
}
function formatLatencyValue(n) {
  if (!n && n !== 0) return '0.00'
  if (n >= 100) return n.toFixed(0)
  return n.toFixed(2)
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
    if (TERMINAL.has(job.value?.status)) { clearInterval(pollInterval); pollInterval = null }
  } catch (_) {}
}

onMounted(async () => {
  wsStore.connect(jobId) // jobId-aware reset, see v2.0.5
  await pollJob()
  pollInterval = setInterval(pollJob, 3000)
})

onUnmounted(() => {
  wsStore.disconnect()
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
.dash {
  padding: 0.6rem 1rem 0.8rem;
  max-width: 1680px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}
@media (min-width: 1280px) {
  .dash { padding: 0.6rem 1.5rem 0.8rem; }
}

/* Header */
.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}
.dash-header-left {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  min-width: 0;
}
.dash-header-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.dash-title {
  font-size: 1.05rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--fg-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dash-meta {
  display: inline-flex;
  align-items: baseline;
  gap: 0.4rem;
  font-size: 0.78rem;
  color: var(--fg-secondary);
}
.dash-meta-phase {
  display: inline-flex;
  align-items: baseline;
  gap: 0.4rem;
}
.dash-sep {
  color: var(--fg-faint);
}
.dash-mono {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.7rem;
}
.dash-phase-tag {
  background: var(--surface-muted);
  padding: 0.05rem 0.35rem;
  border-radius: 0.25rem;
  color: var(--fg-primary);
}
.dash-conn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.7rem;
  color: var(--fg-secondary);
  font-family: 'Geist Mono', ui-monospace, monospace;
}
.dash-conn-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
}
.dash-conn-on {
  background: #10b981;
  box-shadow: 0 0 6px rgba(16,185,129,0.6);
}
.dash-conn-off {
  background: #737373;
}

.dash-alert {
  margin: 0;
}

/* KPI grid */
.dash-grid-kpi {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.55rem;
}
@media (min-width: 768px) {
  .dash-grid-kpi { grid-template-columns: repeat(4, 1fr); }
}

/* Charts row */
.dash-grid-charts {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.6rem;
}
@media (min-width: 1024px) {
  .dash-grid-charts { grid-template-columns: repeat(3, 1fr); }
}
.dash-chart-card {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.dash-chart-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.55rem 0.85rem 0.4rem;
  border-bottom: 1px solid var(--border-subtle);
}
.dash-chart-title {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.dash-chart-legend {
  display: inline-flex;
  align-items: center;
  gap: 0.15rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--fg-secondary);
}
.dash-legend-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 999px;
  margin-right: 0.2rem;
}
.dash-chart-body {
  padding: 0.45rem 0.7rem 0.55rem;
  height: 180px;
}

/* Infrastructure row */
.dash-grid-infra {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.6rem;
}
@media (min-width: 1024px) {
  .dash-grid-infra { grid-template-columns: 3fr 2fr; }
}
.dash-card {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  overflow: hidden;
}
.dash-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 0.85rem;
  border-bottom: 1px solid var(--border-subtle);
}
.dash-card-title {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.dash-card-pill {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.65rem;
  background: var(--surface-muted);
  color: var(--fg-secondary);
  padding: 0.05rem 0.4rem;
  border-radius: 0.25rem;
}
.dash-card-body {
  padding: 0.55rem 0.6rem;
}
.dash-workers-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.5rem;
}
@media (min-width: 768px) {
  .dash-workers-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (min-width: 1280px) {
  .dash-workers-grid { grid-template-columns: repeat(4, 1fr); }
}
.dash-nodes-list {
  padding: 0;
}

/* Phase summaries */
.dash-summaries-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.5rem;
}
@media (min-width: 768px) {
  .dash-summaries-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (min-width: 1280px) {
  .dash-summaries-grid { grid-template-columns: repeat(3, 1fr); }
}
.dash-summary-tile {
  background: var(--surface-muted);
  border-radius: 0.4rem;
  padding: 0.55rem 0.7rem;
}
.dash-summary-name {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.78rem;
  color: var(--fg-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dash-summary-stats {
  margin-top: 0.4rem;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.2rem 0.6rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.7rem;
}
.dash-summary-stats > div {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
}
.dash-summary-label {
  font-size: 0.6rem;
  color: var(--fg-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.dash-summary-val {
  color: var(--fg-secondary);
  font-variant-numeric: tabular-nums;
}
</style>
