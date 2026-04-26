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
          <span v-if="wsStore.jobStatus.phase">
            <span class="fg-faint">·</span>
            phase <span class="font-mono">{{ wsStore.jobStatus.phase }}</span>
          </span>
          <span class="fg-faint">·</span>
          <span class="inline-flex items-center gap-1.5">
            <span
              class="w-1.5 h-1.5 rounded-full"
              :class="wsStore.connected ? 'bg-emerald-500 animate-pulse-dot' : 'bg-ink-400'"
            ></span>
            {{ wsStore.connected ? t('dashboard.wsLive') : t('dashboard.wsInactive') }}
          </span>
        </span>
      </template>
      <template #actions>
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
          Résultats
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
      </template>
    </PageHeader>

    <!-- Failure panel -->
    <div v-if="job?.status === 'failed'" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="font-semibold">Le job a échoué</p>
        <p v-if="job?.error_message" class="font-mono text-xs mt-1 break-all opacity-90">
          {{ job.error_message }}
        </p>
      </div>
    </div>

    <!-- Provisioning timeline -->
    <section
      v-if="job && job.status === 'provisioning'"
      class="card mb-6 space-y-4 animate-fade-in"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2.5">
          <span class="card-title">{{ t("newJob.submitting") }}</span>
        </div>
        <span class="num text-sm fg-secondary">{{ Math.round(wsStore.provProgress * 100) }}%</span>
      </div>
      <ProgressBar :value="wsStore.provProgress * 100" tone="brand" />
      <ol class="space-y-2.5 mt-2">
        <li
          v-for="s in wsStore.provSteps"
          :key="s.step"
          class="flex items-center gap-3 text-sm"
        >
          <span class="w-5 h-5 rounded-full flex items-center justify-center bg-emerald-100 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300 shrink-0">
            <Icon name="check" :size="12" stroke-width="3" />
          </span>
          <span class="fg-primary">{{ s.detail }}</span>
        </li>
        <li
          v-if="wsStore.provProgress < 1 && wsStore.provSteps.length > 0"
          class="flex items-center gap-3 text-sm"
        >
          <span class="w-5 h-5 rounded-full flex items-center justify-center bg-brand-100 text-brand-700 dark:bg-brand-500/20 dark:text-brand-300 shrink-0">
            <Spinner :size="11" />
          </span>
          <span class="fg-secondary italic">En cours…</span>
        </li>
      </ol>
    </section>

    <!-- Live metrics grid -->
    <section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      <StatCard
        icon="zap"
        label="IOPS Read"
        :value="formatIops(wsStore.elbenchoMetrics.iopsRead)"
        :hint="wsStore.elbenchoMetrics.profileName || t('dashboard.waiting')"
        tone="brand"
      />
      <StatCard
        icon="zap"
        label="IOPS Write"
        :value="formatIops(wsStore.elbenchoMetrics.iopsWrite)"
        :hint="wsStore.elbenchoMetrics.profileName || t('dashboard.waiting')"
        tone="brand"
      />
      <StatCard
        icon="hard_drive"
        label="Débit Read"
        :value="(wsStore.elbenchoMetrics.throughputReadMbps || 0).toFixed(1)"
        unit="MB/s"
        tone="info"
      />
      <StatCard
        icon="clock"
        label="Latence avg"
        :value="(wsStore.elbenchoMetrics.latencyAvgMs || 0).toFixed(2)"
        unit="ms"
        :hint="`p99 : ${(wsStore.elbenchoMetrics.latencyP99Ms || 0).toFixed(2)} ms`"
        :tone="wsStore.elbenchoMetrics.latencyAvgMs > 5 ? 'danger' : 'success'"
      />
    </section>

    <!-- Charts row -->
    <section class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">IOPS — temps réel</span>
          <span class="text-xs fg-muted num">{{ wsStore.elbenchoMetrics.history.iopsRead.length }} pts</span>
        </header>
        <div class="p-4" style="height: 220px;">
          <LineChart
            label="IOPS Read"
            :data="wsStore.elbenchoMetrics.history.iopsRead"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#f97316"
          />
        </div>
      </div>
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">Latence avg (ms)</span>
        </header>
        <div class="p-4" style="height: 220px;">
          <LineChart
            label="Latence"
            :data="wsStore.elbenchoMetrics.history.latency"
            :labels="wsStore.elbenchoMetrics.history.labels"
            color="#7c3aed"
          />
        </div>
      </div>
    </section>

    <!-- Cluster + Workers -->
    <section class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('dashboard.sections.cluster') }}</span>
          <span class="pill">{{ Object.keys(wsStore.nodeMetrics).length }} nodes</span>
        </header>
        <div class="px-5 py-2">
          <div v-if="Object.keys(wsStore.nodeMetrics).length === 0" class="py-6">
            <EmptyState icon="server" title="En attente des métriques" description="Les données du cluster apparaîtront ici." />
          </div>
          <div v-else class="divide-y" style="border-color: var(--border-subtle);">
            <NodeCard
              v-for="(metrics, node) in wsStore.nodeMetrics"
              :key="node"
              :name="node"
              :cpu="metrics.cpuPct"
              :ram="metrics.ramPct"
            />
          </div>
        </div>
      </div>

      <div class="card-flush">
        <header class="card-header">
          <span class="card-title">{{ t('dashboard.sections.workers') }}</span>
          <span class="pill">{{ workers.length }}</span>
        </header>
        <div class="p-5">
          <div v-if="workers.length === 0">
            <EmptyState
              v-if="job?.status === 'failed'"
              icon="server" title="{{ t('dashboard.workers.none') }}"
              description="{{ t('dashboard.workers.noneFailed') }}"
            />
            <EmptyState
              v-else
              icon="server" :title="t('dashboard.sections.provisioning')"
              description="{{ t('dashboard.workers.noneProvisioning') }}"
            />
          </div>
          <div v-else class="grid grid-cols-2 sm:grid-cols-3 gap-2">
            <WorkerBadge
              :ram="wsStore.workerMetrics[w.id]?.ramPct"
              :net-in="wsStore.workerMetrics[w.id]?.netInBps"
              :net-out="wsStore.workerMetrics[w.id]?.netOutBps"
              :disk-read="wsStore.workerMetrics[w.id]?.diskReadBps"
              :disk-write="wsStore.workerMetrics[w.id]?.diskWriteBps"
              v-for="(w, i) in workers"
              :key="w.id"
              :name="`Worker ${i + 1}`"
              :status="w.status"
              :cpu="wsStore.workerMetrics[w.id]?.cpuPct"
            />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useWsStore } from '../stores/ws.js'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import LineChart    from '../components/LineChart.vue'
import NodeCard     from '../components/NodeCard.vue'
import WorkerBadge  from '../components/WorkerBadge.vue'
import StatusBadge  from '../components/StatusBadge.vue'
import StatCard     from '../components/StatCard.vue'
import EmptyState   from '../components/EmptyState.vue'
import PageHeader   from '../components/PageHeader.vue'
import ProgressBar  from '../components/ProgressBar.vue'
import Icon         from '../components/Icon.vue'
import Spinner      from '../components/Spinner.vue'

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
