<template>
  <div class="page">
    <PageHeader>
      <template #title>
        <span class="flex items-center gap-3">
          {{ job?.name ?? '...' }}
          <StatusBadge v-if="job?.status" :status="job.status" />
        </span>
      </template>
      <template #description>
        <span v-if="job">
          {{ job.client_name }}
          <span class="fg-faint">·</span>
          mode {{ job.mode }}
          <span class="fg-faint">·</span>
          {{ formatDate(job.created_at) }}
        </span>
      </template>
      <template #actions>
        <RouterLink to="/history" class="btn-secondary btn-sm">
          <Icon name="history" :size="14" />
          Historique
        </RouterLink>
        <template v-if="job?.status === 'done'">
          <a :href="api.exportCsvUrl(jobId)" download class="btn-secondary btn-sm">
            <Icon name="table" :size="14" /> CSV
          </a>
          <a :href="api.reportHtmlUrl(jobId, locale)" target="_blank" class="btn-secondary btn-sm">
            <Icon name="external" :size="14" /> HTML
          </a>
          <a :href="api.reportPdfUrl(jobId, locale)" download class="btn-primary btn-sm">
            <Icon name="download" :size="14" /> PDF
          </a>
        </template>
        <a
          v-if="canDownloadDebug"
          :href="api.debugBundleUrl(jobId)"
          download
          class="btn-secondary btn-sm"
          :title="t('job.debug.tooltip')"
        >
          <Icon name="download" :size="14" /> {{ t('job.debug.button') }}
        </a>
        <button
          v-else
          class="btn-secondary btn-sm opacity-50 cursor-not-allowed"
          disabled
          :title="t('job.debug.disabledTooltip')"
        >
          <Icon name="download" :size="14" /> {{ t('job.debug.button') }}
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <div v-if="job?.status === 'failed'" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1">
        <p class="font-semibold">{{ t('jobDetail.failedReason') }}</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-1 break-all opacity-90">
          {{ job.error_message }}
        </p>
      </div>
    </div>

    <!-- Stats -->
    <section class="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-6">
      <StatCard icon="clock"      label="Durée totale" :value="duration" tone="brand" />
      <StatCard
        :icon="modeIcon(job?.mode)"
        label="Mode" :value="job?.mode ?? '—'"
        tone="info"
      />
      <StatCard icon="layers" label="Profils testés" :value="results.length" tone="neutral" />
    </section>

    <!-- Results -->
    <section class="card-flush mb-6">
      <header class="card-header">
        <span class="card-title">{{ t('jobDetail.sections.profiles') }}</span>
        <span v-if="results.length" class="pill">{{ results.length }} profils</span>
      </header>
      <div v-if="results.length === 0">
        <EmptyState icon="bar_chart" title="Aucun résultat" description="Aucune donnée n'a été enregistrée pour ce job." />
      </div>
      <div v-else class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th>Profil</th>
              <th class="text-right">IOPS Read max</th>
              <th class="text-right">IOPS Write max</th>
              <th class="text-right">{{ t('jobDetail.columns.throughputRead') }}</th>
              <th class="text-right">{{ t('jobDetail.columns.throughputWrite') }}</th>
              <th class="text-right">Latence avg</th>
              <th class="text-center">Verdict</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in results" :key="r.profile_name">
              <td>
                <span class="code-inline">{{ r.profile_name }}</span>
              </td>
              <td class="text-right num">{{ formatIops(r.max_iops_read) }}</td>
              <td class="text-right num">{{ formatIops(r.max_iops_write) }}</td>
              <td class="text-right num">{{ formatMbps(r.max_throughput_read_mbps) }}</td>
              <td class="text-right num">{{ formatMbps(r.max_throughput_write_mbps) }}</td>
              <td class="text-right num">{{ formatMs(r.avg_latency_ms) }}</td>
              <td class="text-center">
                <span
                  v-if="verdict(r.profile_name, r) === 'pass'"
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold uppercase tracking-wide bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:ring-emerald-500/30"
                >
                  <Icon name="check" :size="11" stroke-width="3" />
                  Pass
                </span>
                <span
                  v-else-if="verdict(r.profile_name, r) === 'fail'"
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold uppercase tracking-wide bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-500/10 dark:text-red-300 dark:ring-red-500/30"
                >
                  <Icon name="x" :size="11" stroke-width="3" />
                  Fail
                </span>
                <span v-else class="text-xs fg-faint">—</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Workers -->
    <section class="card-flush">
      <header class="card-header">
        <span class="card-title">{{ t('jobDetail.sections.workers') }}</span>
        <span v-if="workers.length" class="pill">{{ workers.length }}</span>
      </header>
      <div class="p-5">
        <div v-if="workers.length === 0">
          <EmptyState icon="server" title="Aucun worker enregistré" description="Aucun détail de worker disponible." />
        </div>
        <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div
            v-for="(w, i) in workers"
            :key="w.id"
            class="surface-muted px-4 py-3 flex items-center gap-3"
          >
            <div class="w-9 h-9 rounded-lg bg-elevated fg-secondary flex items-center justify-center shrink-0">
              <Icon name="server" :size="16" />
            </div>
            <div class="min-w-0">
              <p class="text-sm font-semibold fg-primary">Worker {{ i + 1 }}</p>
              <p class="text-xs fg-muted num">VM {{ w.vm_id }} · {{ w.ip || 'IP inconnue' }}</p>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { useI18n } from "vue-i18n"
import { ref, computed, onMounted } from 'vue'
import { useRoute, RouterLink } from 'vue-router'
import { api } from '../api/client.js'
import PageHeader  from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import StatCard    from '../components/StatCard.vue'
import EmptyState  from '../components/EmptyState.vue'
import Icon        from '../components/Icon.vue'

const { t, locale } = useI18n()

const route   = useRoute()
const jobId   = route.params.id
const job     = ref(null)
const results = ref([])
const workers = ref([])
const profiles = ref([])
const error   = ref('')

const canDownloadDebug = computed(() => {
  const s = job.value?.status
  return s === 'done' || s === 'failed' || s === 'cancelled'
})

const duration = computed(() => {
  if (!job.value?.finished_at || !job.value?.created_at) return '—'
  const ms = new Date(job.value.finished_at) - new Date(job.value.created_at)
  const s = Math.floor(ms / 1000)
  const m = Math.floor(s / 60)
  const h = Math.floor(m / 60)
  if (h > 0) return `${h}h ${m % 60}m`
  if (m > 0) return `${m}m ${s % 60}s`
  return `${s}s`
})

function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d)) return '—'
  return d.toLocaleString('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}
function formatIops(n) {
  if (!n) return '—'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1000)      return (n / 1000).toFixed(1) + 'k'
  return n.toFixed(0)
}
function formatMbps(n) { return n == null ? '—' : n.toFixed(0) + ' MB/s' }
function formatMs(n)   { return n == null ? '—' : n.toFixed(2) + ' ms' }
function modeIcon(mode) {
  if (mode === 'cpu')     return 'cpu'
  if (mode === 'storage') return 'hard_drive'
  return 'shuffle'
}

function verdict(profileName, r) {
  const prof = profiles.value.find(p => p.name === profileName)
  if (!prof || !prof.thresholds_json) return null
  let t
  try { t = JSON.parse(prof.thresholds_json) } catch { return null }
  if (!t.min_iops_read && !t.min_iops_write && !t.max_latency_ms) return null
  const pass =
    (!t.min_iops_read  || r.max_iops_read  >= t.min_iops_read)  &&
    (!t.min_iops_write || r.max_iops_write >= t.min_iops_write) &&
    (!t.max_latency_ms || r.avg_latency_ms <= t.max_latency_ms)
  return pass ? 'pass' : 'fail'
}

onMounted(async () => {
  try {
    const [j, r, w, profs] = await Promise.all([
      api.getJob(jobId),
      api.getJobResults(jobId),
      api.listWorkers(jobId),
      api.listProfiles(),
    ])
    job.value      = j
    results.value  = r     ?? []
    workers.value  = w     ?? []
    profiles.value = profs ?? []
  } catch (e) {
    error.value = 'Impossible de charger les données du job.'
  }
})
</script>
