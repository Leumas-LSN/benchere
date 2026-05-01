<template>
  <div class="page">
    <PageHeader>
      <template #title>
        <span class="flex items-center gap-3">{{ job?.name ?? '...' }}<StatusBadge v-if="job?.status" :status="job.status" /></span>
      </template>
      <template #description>
        <span v-if="job">
          {{ job.client_name }} <span class="fg-faint">&#xB7;</span> mode {{ job.mode }}
          <span class="fg-faint">&#xB7;</span> {{ job.engine || 'fio' }}
          <span class="fg-faint">&#xB7;</span> {{ formatDate(job.created_at) }}
        </span>
      </template>
      <template #actions>
        <RouterLink to="/history" class="btn-secondary btn-sm"><Icon name="history" :size="14" />Historique</RouterLink>
        <template v-if="job?.status === 'done'">
          <a :href="api.exportCsvUrl(jobId)" download class="btn-secondary btn-sm"><Icon name="table" :size="14" />CSV</a>
          <a :href="api.reportHtmlUrl(jobId, locale)" target="_blank" class="btn-secondary btn-sm"><Icon name="external_link" :size="14" />HTML</a>
          <a :href="api.reportPdfUrl(jobId, locale)" download class="btn-primary btn-sm"><Icon name="download" :size="14" />PDF</a>
        </template>
        <a v-if="canDownloadDebug" :href="api.debugBundleUrl(jobId)" download class="btn-secondary btn-sm">
          <Icon name="download" :size="14" />Debug
        </a>
      </template>
    </PageHeader>

    <div v-if="job?.status === 'failed'" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1">
        <p class="font-semibold">Le job a echoue</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-1 break-all opacity-90">{{ job.error_message }}</p>
      </div>
    </div>

    <section class="grid grid-cols-3 gap-4 mb-6">
      <div class="rounded-lg border p-3 text-center" :style="{ borderColor: 'var(--border-subtle)' }">
        <p class="text-xs uppercase tracking-wide fg-muted">PASS</p>
        <p class="text-2xl font-semibold num text-emerald-600 dark:text-emerald-400">{{ counts.pass }}</p>
      </div>
      <div class="rounded-lg border p-3 text-center" :style="{ borderColor: 'var(--border-subtle)' }">
        <p class="text-xs uppercase tracking-wide fg-muted">FAIL</p>
        <p class="text-2xl font-semibold num text-red-600 dark:text-red-400">{{ counts.fail }}</p>
      </div>
      <div class="rounded-lg border p-3 text-center" :style="{ borderColor: 'var(--border-subtle)' }">
        <p class="text-xs uppercase tracking-wide fg-muted">N/A</p>
        <p class="text-2xl font-semibold num fg-secondary">{{ counts.na }}</p>
      </div>
    </section>

    <MethodologyPanel
      :engine="job?.engine || 'fio'"
      :workers="workers"
      :cluster="clusterNames"
      :storage-pool="job?.storage_pool || ''"
      :working-set-per-worker-g-b="(job?.data_disks || 0) * (job?.data_disk_gb || 0)"
      :working-set-total-g-b="((job?.data_disks || 0) * (job?.data_disk_gb || 0)) * workers.length"
      :runtime="profileRuntimeFromList(profiles, results)"
      :warmup="30"
    />

    <section class="card-flush mb-6">
      <header class="card-header"><span class="card-title">Resultats par profil</span><span class="pill">{{ results.length }}</span></header>
      <div class="overflow-x-auto">
        <table class="table text-xs">
          <thead>
            <tr>
              <th>Profil</th>
              <th class="text-right">IOPS R avg/max</th>
              <th class="text-right">IOPS W avg/max</th>
              <th class="text-right">BW R max</th>
              <th class="text-right">BW W max</th>
              <th class="text-right">p50</th>
              <th class="text-right">p95</th>
              <th class="text-right">p99</th>
              <th class="text-right">CV%</th>
              <th class="text-center">Verdict</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in results" :key="r.profile_name">
              <td><span class="code-inline">{{ r.profile_name }}</span></td>
              <td class="text-right num">{{ formatIops(r.iops_read_avg) }} / {{ formatIops(r.iops_read_max) }}</td>
              <td class="text-right num">{{ formatIops(r.iops_write_avg) }} / {{ formatIops(r.iops_write_max) }}</td>
              <td class="text-right num">{{ formatMbps(r.throughput_read_mbps_max) }}</td>
              <td class="text-right num">{{ formatMbps(r.throughput_write_mbps_max) }}</td>
              <td class="text-right num">{{ formatLatencyP50(r) }}</td>
              <td class="text-right num">{{ formatLatencyP95(r) }}</td>
              <td class="text-right num">{{ formatLatencyP99(r) }}</td>
              <td class="text-right num">{{ (r.iops_cv_pct||0).toFixed(1) }}%</td>
              <td class="text-center">
                <span v-if="verdict(r.profile_name, r) === 'pass'" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold uppercase tracking-wide bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30"><Icon name="check" :size="11" />Pass</span>
                <span v-else-if="verdict(r.profile_name, r) === 'fail'" :title="failReason(r.profile_name, r)" class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold uppercase tracking-wide bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-500/10 dark:text-red-300 dark:ring-red-500/30 cursor-help"><Icon name="x" :size="11" />Fail</span>
                <span v-else class="text-xs fg-faint">N/A</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="card-flush mb-6">
      <header class="card-header"><span class="card-title">Workers</span><span v-if="workers.length" class="pill">{{ workers.length }}</span></header>
      <div class="p-5 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <div v-for="(w, i) in workers" :key="w.id" class="surface-muted px-4 py-3 flex items-center gap-3 rounded-lg">
          <div class="w-9 h-9 rounded-lg bg-elevated fg-secondary flex items-center justify-center shrink-0"><Icon name="server" :size="16" /></div>
          <div class="min-w-0">
            <p class="text-sm font-semibold fg-primary">Worker {{ i + 1 }}</p>
            <p class="text-xs fg-muted num">VM {{ w.vm_id }} on {{ w.proxmox_node }} &#xB7; {{ w.ip || 'IP inconnue' }}</p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { ref, computed, onMounted } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { api } from '../api/client.js'
import PageHeader       from '../components/PageHeader.vue'
import StatusBadge      from '../components/StatusBadge.vue'
import Icon             from '../components/Icon.vue'
import MethodologyPanel from '../components/MethodologyPanel.vue'

const { locale } = useI18n()
const route   = useRoute()
const jobId   = route.params.id
const job     = ref(null)
const results = ref([])
const workers = ref([])
const profiles= ref([])
const error   = ref('')

const canDownloadDebug = computed(() => ['done','failed','cancelled'].includes(job.value?.status))

const counts = computed(() => {
  const out = { pass: 0, fail: 0, na: 0 }
  for (const r of results.value) {
    const v = verdict(r.profile_name, r)
    if (v === 'pass') out.pass++
    else if (v === 'fail') out.fail++
    else out.na++
  }
  return out
})

const clusterNames = computed(() => {
  const set = new Set(workers.value.map(function(w) { return w.proxmox_node }).filter(Boolean))
  return [...set]
})

function formatDate(iso) {
  if (!iso) return 'N/A'
  const d = new Date(iso)
  return isNaN(d) ? 'N/A' : d.toLocaleString('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
function formatIops(n) {
  if (!n && n !== 0) return 'N/A'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000)    return (n / 1000).toFixed(1) + 'k'
  return n.toFixed(0)
}
function formatMbps(n) { return n == null ? 'N/A' : n.toFixed(0) + ' MB/s' }
function formatMs(n) { return n == null ? 'N/A' : n.toFixed(2) + ' ms' }

// v2.1.1: latency columns adapt to the active leg of the workload. The
// fio parser only captures p50/p95/p99/p999 on the read leg and p99 on
// the write leg today, so for write-only profiles the read percentiles
// are correctly 0 (no read happened) but the user expects to see the
// write p99. We pick the write percentile when the profile is write-
// dominant. p50 and p95 stay blank for write-dominant runs until the
// parser is extended to capture write p50/p95 too (planned for v2.2).
function isWriteDominant(r) {
  return (r.iops_write_avg || 0) > (r.iops_read_avg || 0)
}
function formatLatencyP50(r) {
  if (isWriteDominant(r)) return r.latency_write_p50_ms ? r.latency_write_p50_ms.toFixed(2) + ' ms (W)' : 'N/A'
  return formatMs(r.latency_p50_ms)
}
function formatLatencyP95(r) {
  if (isWriteDominant(r)) return r.latency_write_p95_ms ? r.latency_write_p95_ms.toFixed(2) + ' ms (W)' : 'N/A'
  return formatMs(r.latency_p95_ms)
}
function formatLatencyP99(r) {
  if (isWriteDominant(r)) {
    if (r.latency_write_p99_ms) return r.latency_write_p99_ms.toFixed(2) + ' ms (W)'
    return 'N/A'
  }
  return formatMs(r.latency_p99_ms)
}

function verdict(profileName, r) {
  const prof = profiles.value.find(function(p) { return p.name === profileName })
  if (!prof || !prof.thresholds_json) return null
  let th
  try { th = JSON.parse(prof.thresholds_json) } catch (_) { return null }
  if (!th.min_iops_read && !th.min_iops_write && !th.max_p99_latency_ms && !th.max_avg_latency_ms) return null
  let pass = true
  if (th.min_iops_read   && (r.iops_read_max  < th.min_iops_read))    pass = false
  if (th.min_iops_write  && (r.iops_write_max < th.min_iops_write))   pass = false
  // Verdict uses the active-leg p99 too (v2.1.1).
  const effP99 = isWriteDominant(r) ? (r.latency_write_p99_ms || 0) : (r.latency_p99_ms || 0)
  if (th.max_p99_latency_ms && effP99 > th.max_p99_latency_ms) pass = false
  if (th.max_avg_latency_ms && (r.latency_avg_ms > th.max_avg_latency_ms)) pass = false
  return pass ? 'pass' : 'fail'
}

function failReason(profileName, r) {
  const prof = profiles.value.find(function(p) { return p.name === profileName })
  if (!prof || !prof.thresholds_json) return ''
  let th
  try { th = JSON.parse(prof.thresholds_json) } catch (_) { return '' }
  const out = []
  if (th.min_iops_read && r.iops_read_max < th.min_iops_read) out.push('iops_read_max ' + formatIops(r.iops_read_max) + ' < ' + formatIops(th.min_iops_read))
  if (th.min_iops_write && r.iops_write_max < th.min_iops_write) out.push('iops_write_max ' + formatIops(r.iops_write_max) + ' < ' + formatIops(th.min_iops_write))
  const effP99fail = isWriteDominant(r) ? (r.latency_write_p99_ms || 0) : (r.latency_p99_ms || 0)
  if (th.max_p99_latency_ms && effP99fail > th.max_p99_latency_ms) out.push('p99 ' + formatMs(effP99fail) + ' > ' + formatMs(th.max_p99_latency_ms))
  if (th.max_avg_latency_ms && r.latency_avg_ms > th.max_avg_latency_ms) out.push('avg ' + formatMs(r.latency_avg_ms) + ' > ' + formatMs(th.max_avg_latency_ms))
  return out.join(' | ')
}

function profileRuntimeFromList(profs, res) {
  if (!profs.length || !res.length) return 300
  const first = profs.find(function(p) { return p.name === res[0].profile_name })
  if (!first || !first.config_json) return 300
  const m = first.config_json.match(/runtime\s*=\s*(\d+)/m)
  return m ? parseInt(m[1], 10) : 300
}

onMounted(async function() {
  try {
    const [j, r, w, profs] = await Promise.all([
      api.getJob(jobId),
      api.getJobResults(jobId),
      api.listWorkers(jobId),
      api.listProfiles(),
    ])
    job.value      = j
    results.value  = r ?? []
    workers.value  = w ?? []
    profiles.value = profs ?? []
  } catch (e) {
    error.value = 'Impossible de charger les donnees du job.'
  }
})
</script>
