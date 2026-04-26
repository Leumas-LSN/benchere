<template>
  <div class="page">
    <PageHeader
      :eyebrow="t('history.title')"
      :title="t('history.title')"
      :description="t('history.description', jobs.length)"
    >
      <template #actions>
        <button class="btn-secondary btn-sm" :disabled="loading" @click="refresh">
          <Spinner v-if="loading" :size="14" />
          <Icon v-else name="refresh" :size="14" />
          {{ t('common.refresh') }}
        </button>
        <button
          v-if="hasHistory"
          class="btn-danger-ghost btn-sm"
          :disabled="clearing"
          @click="clearHistory"
        >
          <Spinner v-if="clearing" :size="14" />
          <Icon v-else name="trash" :size="14" />
          Vider l'historique
        </button>
      </template>
    </PageHeader>

    <div v-if="error" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <!-- Filter pills -->
    <div class="flex items-center gap-2 mb-5 flex-wrap">
      <button
        v-for="f in filters"
        :key="f.value"
        :class="[
          'px-3 h-8 text-xs font-medium rounded-md inline-flex items-center gap-1.5 border transition-colors',
          activeFilter === f.value
            ? 'border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
            : 'border-default fg-secondary hover:border-strong'
        ]"
        @click="activeFilter = f.value"
      >
        {{ f.label }}
        <span class="num text-[10px] opacity-70">{{ count(f.value) }}</span>
      </button>
    </div>

    <section class="card-flush">
      <div v-if="filteredJobs.length === 0 && !loading">
        <EmptyState
          icon="history"
          title="Aucun job à afficher"
          description="Aucun job ne correspond au filtre sélectionné."
        >
          <template #action>
            <RouterLink to="/jobs/new" class="btn-primary btn-sm">
              <Icon name="plus" :size="14" />
              Lancer un benchmark
            </RouterLink>
          </template>
        </EmptyState>
      </div>
      <div v-else class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th>Nom</th>
              <th>Client</th>
              <th>Date</th>
              <th>{{ t('history.columns.mode') }}</th>
              <th>{{ t('history.columns.status') }}</th>
              <th>{{ t('history.columns.duration') }}</th>
              <th class="text-right pr-5">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in filteredJobs" :key="job.id">
              <td class="font-medium fg-primary">{{ job.name }}</td>
              <td class="fg-secondary">{{ job.client_name }}</td>
              <td class="num text-xs fg-muted">{{ formatDate(job.created_at) }}</td>
              <td>
                <span class="pill flex items-center gap-1 w-fit">
                  <Icon :name="modeIcon(job.mode)" :size="11" />
                  {{ job.mode }}
                </span>
              </td>
              <td><StatusBadge :status="job.status" /></td>
              <td class="num text-xs fg-muted">{{ duration(job) }}</td>
              <td>
                <div class="flex items-center justify-end gap-1 pr-2">
                  <RouterLink
                    v-if="isActive(job)"
                    :to="`/dashboard/${job.id}`"
                    class="btn-ghost btn-sm"
                    title="Live"
                  >
                    <Icon name="activity" :size="14" />
                    <span class="hidden md:inline">Live</span>
                  </RouterLink>
                  <RouterLink
                    v-if="isFinished(job)"
                    :to="`/jobs/${job.id}`"
                    class="btn-ghost btn-sm"
                    title="Voir les résultats"
                  >
                    <Icon name="bar_chart" :size="14" />
                    <span class="hidden md:inline">{{ t('jobDetail.sections.results') }}</span>
                  </RouterLink>
                  <a
                    v-if="isFinished(job)"
                    :href="api.reportHtmlUrl(job.id, locale)"
                    target="_blank"
                    class="btn-ghost btn-sm"
                    title="Rapport HTML"
                  >
                    <Icon name="external" :size="14" />
                  </a>
                  <a
                    v-if="isFinished(job)"
                    :href="api.reportPdfUrl(job.id, locale)"
                    download
                    class="btn-ghost btn-sm"
                    title="Rapport PDF"
                  >
                    <Icon name="download" :size="14" />
                  </a>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { ref, computed, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { useJobsStore } from '../stores/jobs.js'
import { api } from '../api/client.js'
import PageHeader  from '../components/PageHeader.vue'
import StatusBadge from '../components/StatusBadge.vue'
import EmptyState  from '../components/EmptyState.vue'
import Icon        from '../components/Icon.vue'
import Spinner     from '../components/Spinner.vue'

const jobsStore = useJobsStore()
const error    = ref('')
const clearing = ref(false)
const activeFilter = ref('all')

const filters = [
  { value: 'all',     label: 'Tous' },
  { value: 'active',  label: 'Actifs' },
  { value: 'done',    label: 'Terminés' },
  { value: 'failed',  label: 'Échecs' },
]

const jobs    = computed(() => jobsStore.jobs)
const loading = computed(() => jobsStore.loading)

const hasHistory = computed(() =>
  jobs.value.some(j => ['done', 'failed', 'cancelled'].includes(j.status))
)

const sorted = computed(() =>
  [...jobs.value].sort((a, b) => new Date(b.created_at) - new Date(a.created_at))
)

const filteredJobs = computed(() => {
  switch (activeFilter.value) {
    case 'active': return sorted.value.filter(isActive)
    case 'done':   return sorted.value.filter(j => j.status === 'done')
    case 'failed': return sorted.value.filter(j => ['failed', 'cancelled'].includes(j.status))
    default:       return sorted.value
  }
})

function count(filter) {
  switch (filter) {
    case 'active': return jobs.value.filter(isActive).length
    case 'done':   return jobs.value.filter(j => j.status === 'done').length
    case 'failed': return jobs.value.filter(j => ['failed', 'cancelled'].includes(j.status)).length
    default:       return jobs.value.length
  }
}

async function clearHistory() {
  if (!confirm('Supprimer tous les jobs done / failed / cancelled ?')) return
  clearing.value = true
  error.value = ''
  try {
    await api.clearHistory()
    await jobsStore.fetchJobs()
  } catch (e) {
    error.value = 'Erreur : ' + e.message
  } finally {
    clearing.value = false
  }
}

async function refresh() {
  error.value = ''
  try { await jobsStore.fetchJobs() }
  catch (e) { error.value = 'Impossible de charger les jobs : ' + e.message }
}

function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d)) return '—'
  return d.toLocaleString('fr-FR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function duration(job) {
  if (!job.finished_at) return isActive(job) ? 'En cours' : '—'
  const ms = new Date(job.finished_at) - new Date(job.created_at)
  const s = Math.floor(ms / 1000)
  const m = Math.floor(s / 60)
  const h = Math.floor(m / 60)
  if (h > 0) return `${h}h ${m % 60}m`
  if (m > 0) return `${m}m ${s % 60}s`
  return `${s}s`
}

function modeIcon(mode) {
  if (mode === 'cpu')     return 'cpu'
  if (mode === 'storage') return 'hard_drive'
  return 'shuffle'
}

function isActive(job)   { return job.status === 'running' || job.status === 'provisioning' }
function isFinished(job) { return ['done', 'failed', 'cancelled'].includes(job.status) }

onMounted(refresh)
</script>
