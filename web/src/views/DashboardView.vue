<template>
  <div class="dash">

    <!-- 1. WALL header strip (single line) -->
    <header class="dash-header">
      <div class="dash-header-left">
        <span class="dash-conn">
          <span class="dash-conn-dot" :class="wsStore.connected ? 'dash-conn-on' : 'dash-conn-off'"></span>
          <span class="dash-conn-label">{{ wsStore.connected ? 'live' : 'offline' }}</span>
        </span>
        <span class="dash-sep">.</span>
        <span class="dash-wall">{{ t('dashboard.header.wall') }}</span>
        <template v-if="job">
          <span class="dash-sep">.</span>
          <span class="dash-meta-item dash-client" :title="job.client_name">{{ job.client_name }}</span>
          <span class="dash-sep">.</span>
          <span class="dash-meta-item dash-mono dash-jobid" :title="job.id">{{ shortJobId }}</span>
          <span class="dash-sep">.</span>
          <span class="dash-meta-item dash-mono">{{ elapsedLabel }}</span>
          <span class="dash-sep">.</span>
          <span class="dash-meta-item dash-mono">
            {{ t('dashboard.header.profileCounter', { current: profileCurrent, total: profileTotal }) }}
          </span>
          <span v-if="wsStore.jobStatus.phase" class="dash-sep">.</span>
          <span v-if="wsStore.jobStatus.phase" class="dash-mono dash-phase-tag">{{ phaseTagLabel }}</span>
          <StatusBadge v-if="job?.status" :status="job.status" class="dash-status" />
        </template>
      </div>
      <div class="dash-header-right">
        <RouterLink to="/history" class="btn-ghost btn-sm dash-btn-ghost">
          <Icon name="history" :size="13" /><span class="dash-btn-text">Historique</span>
        </RouterLink>
        <RouterLink v-if="['done','failed','cancelled'].includes(job?.status)" :to="'/jobs/' + jobId" class="btn-ghost btn-sm dash-btn-ghost">
          <Icon name="file_text" :size="13" /><span class="dash-btn-text">Resultats</span>
        </RouterLink>
        <a
          :href="bundleUrl"
          target="_blank"
          rel="noopener"
          class="btn-secondary btn-sm"
          :class="{ 'btn-disabled': !bundleAvailable }"
          :aria-disabled="!bundleAvailable"
          @click="onBundleClick"
        >
          <Icon name="download" :size="13" /><span class="dash-btn-text">{{ t('dashboard.header.bundle') }}</span>
        </a>
        <button v-if="isRunning" @click="cancel" class="btn-danger btn-sm" :disabled="cancelling">
          <Spinner v-if="cancelling" :size="13" /><Icon v-else name="stop" :size="13" /><span class="dash-btn-text">{{ t('dashboard.header.stop') }}</span>
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

    <!-- 3. KPI tiles with sparklines -->
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

    <!-- 4. Profile strip (left, ~65%) + Latency tri-panel (right, ~35%) -->
    <section class="dash-grid-profilelat">
      <ProfileStrip
        :phase-summaries="wsStore.phaseSummaries"
        :current-profile="currentProfileName"
        :total-profiles="totalProfiles"
        :prefill-estimated-seconds="prefillEstimatedSeconds"
      />
      <LatencyTriPanel
        :p50-history="wsStore.liveMetrics.history.latencyP50"
        :p95-history="wsStore.liveMetrics.history.latencyP95"
        :p99-history="wsStore.liveMetrics.history.latencyP99"
        :p50-current="wsStore.liveMetrics.latencyP50Ms || 0"
        :p95-current="wsStore.liveMetrics.latencyP95Ms || 0"
        :p99-current="wsStore.liveMetrics.latencyP99Ms || 0"
        unit="ms"
      />
    </section>

    <!-- 5. Workers grid (full width, dense up to 8 cols) -->
    <section class="dash-card">
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
    </section>

    <!-- 6. Cluster nodes row -->
    <section v-if="clusterNodes.length" class="dash-card">
      <header class="dash-card-head">
        <span class="dash-card-title">Cluster Proxmox</span>
        <span class="dash-card-pill">{{ clusterNodes.length }}</span>
      </header>
      <div class="dash-card-body dash-nodes-row">
        <NodeCardHorizontal
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

    <!-- 7. Tendances (collapsible, default closed) -->
    <details class="dash-collapsible">
      <summary class="dash-collapsible-summary">
        <Icon name="chevron_right" :size="14" class="dash-chevron" />
        <span class="dash-collapsible-title">{{ t('dashboard.tendances') }}</span>
      </summary>
      <div class="dash-collapsible-body dash-grid-charts">
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
      </div>
    </details>

    <!-- 8. Live logs (component already has its own collapse) -->
    <LiveLogsPanel />

    <!-- 9. Phase summaries strip -->
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
import { useI18n } from 'vue-i18n'
import { useWsStore } from '../stores/ws.js'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import StatusBadge        from '../components/StatusBadge.vue'
import Icon               from '../components/Icon.vue'
import Spinner            from '../components/Spinner.vue'
import LiveLogsPanel      from '../components/LiveLogsPanel.vue'
import UPlotMulti         from '../components/UPlotMulti.vue'
import KpiSpark           from '../components/KpiSpark.vue'
import NodeCardHorizontal from '../components/NodeCardHorizontal.vue'
import WorkerCompactTile  from '../components/WorkerCompactTile.vue'
import ProfileStrip       from '../components/ProfileStrip.vue'
import LatencyTriPanel    from '../components/LatencyTriPanel.vue'

const { t } = useI18n()
const route     = useRoute()
const wsStore   = useWsStore()
const jobsStore = useJobsStore()

const jobId      = route.params.id
const job        = ref(null)
const workers    = ref([])
const cancelling = ref(false)
const startTime  = ref(0)
const now        = ref(Date.now())

const TERMINAL = new Set(['done', 'failed', 'cancelled'])

const isRunning = computed(() => job.value?.status === 'running' || job.value?.status === 'provisioning')

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

// Profile strip data sourcing.
// We do not get the upfront profile name list from the Job API. We
// derive what we know: phaseSummaries gives us the completed profiles,
// jobStatus.phase or liveMetrics.profileName gives us the running one.
// totalProfiles stays best-effort: when the backend exposes a profile
// count later, plug it in via job.value.profiles_count.
const currentProfileName = computed(() => {
  const ph = wsStore.jobStatus.phase
  if (ph && ph !== 'prefill') return ph
  return wsStore.liveMetrics.profileName || ''
})
const totalProfiles = computed(() => 0)
const profileCurrent = computed(() => {
  return wsStore.phaseSummaries.length + (currentProfileName.value ? 1 : 0)
})
const profileTotal = computed(() => {
  if (totalProfiles.value > 0) return totalProfiles.value
  return profileCurrent.value
})

// Phase tag in the header. When running, show the high-level keyword
// (BENCH or PREFILL); the actual profile name is rendered in the strip.
const phaseTagLabel = computed(() => {
  const ph = wsStore.jobStatus.phase
  if (!ph) return ''
  if (ph === 'prefill') return 'PREFILL'
  return 'BENCH'
})

// Elapsed time since job started, in MM:SS or HHh MMm.
const elapsedLabel = computed(() => {
  if (!startTime.value) return '--:--'
  const sec = Math.floor((now.value - startTime.value) / 1000)
  if (sec < 0) return '--:--'
  if (sec >= 3600) {
    const h = Math.floor(sec / 3600)
    const m = Math.floor((sec % 3600) / 60)
    return h + 'h ' + String(m).padStart(2, '0') + 'm'
  }
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0')
})

const shortJobId = computed(() => {
  const id = jobId || ''
  return id.length > 8 ? 'job-' + id.slice(0, 8) : 'job-' + id
})

// Bundle download is only valid once the job is terminal: the API returns
// 409 while the job is still running. We let the user click anyway if
// they want, but visually disable the affordance.
const bundleAvailable = computed(() => TERMINAL.has(job.value?.status || ''))
const bundleUrl = computed(() => api.debugBundleUrl(jobId))
function onBundleClick(e) {
  if (!bundleAvailable.value) {
    e.preventDefault()
  }
}

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
let tickInterval = null
async function pollJob() {
  try {
    job.value = await api.getJob(jobId)
    const w = await jobsStore.listWorkers(jobId)
    if (w) workers.value = w
    if (job.value?.created_at && !startTime.value) {
      startTime.value = new Date(job.value.created_at).getTime()
    }
    if (TERMINAL.has(job.value?.status)) { clearInterval(pollInterval); pollInterval = null }
  } catch (_) {}
}

onMounted(async () => {
  wsStore.connect(jobId)
  await pollJob()
  pollInterval = setInterval(pollJob, 3000)
  tickInterval = setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  wsStore.disconnect()
  if (pollInterval) clearInterval(pollInterval)
  if (tickInterval) clearInterval(tickInterval)
})
</script>

<style scoped>
.dash {
  padding: 0.6rem 1rem 0.8rem;
  max-width: 1920px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}
@media (min-width: 1280px) {
  .dash { padding: 0.6rem 1.5rem 0.8rem; }
}

/* WALL header strip */
.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  padding: 0.4rem 0.65rem 0.4rem 0.75rem;
}
.dash-header-left {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 0;
  flex: 1;
  flex-wrap: wrap;
}
.dash-header-right {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}
.dash-conn {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.65rem;
  color: var(--fg-secondary);
  font-variant-numeric: tabular-nums;
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
.dash-conn-label {
  font-weight: 500;
}
.dash-wall {
  font-family: 'Geist', system-ui, sans-serif;
  font-weight: 700;
  letter-spacing: 0.12em;
  font-size: 0.72rem;
  color: #f97316;
}
.dash-sep {
  color: var(--fg-faint);
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.7rem;
}
.dash-meta-item {
  font-size: 0.74rem;
  color: var(--fg-secondary);
}
.dash-client {
  color: var(--fg-primary);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 18ch;
}
.dash-jobid {
  color: var(--fg-secondary);
}
.dash-mono {
  font-family: 'Geist Mono', ui-monospace, monospace;
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
}
.dash-phase-tag {
  background: var(--surface-muted);
  padding: 0.05rem 0.4rem;
  border-radius: 0.25rem;
  color: var(--fg-primary);
  font-weight: 500;
}
.dash-status {
  margin-left: 0.2rem;
}
.dash-btn-ghost {
  color: var(--fg-secondary);
}
.dash-btn-text {
  display: inline;
}
@media (max-width: 768px) {
  .dash-btn-text { display: none; }
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

/* Profile strip + latency tri-panel row */
.dash-grid-profilelat {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.6rem;
}
@media (min-width: 1024px) {
  .dash-grid-profilelat { grid-template-columns: 2fr 1fr; }
}
@media (min-width: 1280px) {
  .dash-grid-profilelat { grid-template-columns: 2.2fr 1fr; }
}

/* Charts row inside Tendances */
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

/* Cards (workers, cluster, summaries) */
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

/* Workers grid: progressive density up to 8 cols at 1080p+. */
.dash-workers-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.5rem;
}
@media (min-width: 768px) {
  .dash-workers-grid { grid-template-columns: repeat(4, 1fr); }
}
@media (min-width: 1024px) {
  .dash-workers-grid { grid-template-columns: repeat(6, 1fr); }
}
@media (min-width: 1280px) {
  .dash-workers-grid { grid-template-columns: repeat(8, 1fr); }
}

/* Cluster nodes flow: cards in a wrapping row. */
.dash-nodes-row {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.5rem;
}
@media (min-width: 768px) {
  .dash-nodes-row { grid-template-columns: repeat(2, 1fr); }
}
@media (min-width: 1024px) {
  .dash-nodes-row { grid-template-columns: repeat(3, 1fr); }
}
@media (min-width: 1280px) {
  .dash-nodes-row { grid-template-columns: repeat(4, 1fr); }
}

/* Tendances collapsible */
.dash-collapsible {
  border: 1px solid var(--border-subtle);
  border-radius: 0.5rem;
  background: var(--surface-base);
  overflow: hidden;
}
.dash-collapsible-summary {
  list-style: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.85rem;
  user-select: none;
  color: var(--fg-secondary);
  transition: background-color 0.15s ease;
}
.dash-collapsible-summary::-webkit-details-marker {
  display: none;
}
.dash-collapsible-summary:hover {
  background: var(--surface-muted);
}
.dash-collapsible-title {
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--fg-muted);
}
.dash-chevron {
  color: var(--fg-muted);
  transition: transform 0.15s ease;
}
.dash-collapsible[open] .dash-chevron {
  transform: rotate(90deg);
}
.dash-collapsible-body {
  padding: 0.55rem 0.6rem 0.6rem;
  border-top: 1px solid var(--border-subtle);
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

/* Disabled bundle button */
.btn-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
