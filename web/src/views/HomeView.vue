<template>
  <div class="page">
    <PageHeader
      :eyebrow="t('home.title')"
      :title="t('home.title')"
      :description="`Synthèse de l'infrastructure et des benchmarks. Mise à jour toutes les ${REFRESH_MS / 1000}s.`"
    >
      <template #actions>
        <button class="btn-secondary btn-sm" :disabled="loading" @click="loadOverview">
          <Spinner v-if="loading" :size="14" />
          <Icon v-else name="refresh" :size="14" />
          {{ t('common.refresh') }}
        </button>
        <RouterLink to="/jobs/new" class="btn-primary btn-sm">
          <Icon name="plus" :size="14" />
          Lancer un benchmark
        </RouterLink>
      </template>
    </PageHeader>

    <!-- Onboarding banner -->
    <div v-if="!configured" class="alert-warn mb-6">
      <Icon name="alert" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1">
        <p class="font-semibold">Proxmox n'est pas configuré</p>
        <p class="mt-0.5 opacity-80">Renseignez l'URL, le token et les storages avant de lancer un job.</p>
      </div>
      <RouterLink to="/settings" class="btn-sm btn inline-flex bg-white text-amber-800 hover:bg-amber-50 border border-amber-300 dark:bg-amber-900/30 dark:text-amber-200 dark:border-amber-700/50 dark:hover:bg-amber-800/40">
        Configurer
        <Icon name="arrow_right" :size="14" />
      </RouterLink>
    </div>

    <div v-if="error" class="alert-error mb-6">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span class="flex-1">{{ error }}</span>
    </div>

    <!-- KPI cards -->
    <section class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-7">
      <StatCard
        icon="activity"
        label="Jobs actifs"
        :value="data.active_jobs.length"
        tone="brand"
        hint="En cours d'exécution"
      />
      <StatCard
        icon="check_circle"
        label="Jobs terminés"
        :value="doneCount"
        tone="success"
        hint="Sur les 10 derniers"
      />
      <StatCard
        icon="server"
        label="Nodes Proxmox"
        :value="data.cluster.length"
        :tone="data.cluster.length ? 'info' : 'neutral'"
        :hint="onlineHint"
      />
      <StatCard
        icon="cpu"
        label="Charge CPU moy."
        :value="avgCpu"
        unit="%"
        tone="brand"
        :hint="data.cluster.length ? 'Moyenne cluster' : 'Aucune donnée'"
      />
    </section>

    <div class="grid grid-cols-1 xl:grid-cols-3 gap-6">
      <!-- Active runs -->
      <section class="card-flush xl:col-span-2">
        <header class="card-header">
          <div class="flex items-center gap-2.5">
            <span class="card-title">Runs en cours</span>
            <span v-if="data.active_jobs.length" class="pill">{{ data.active_jobs.length }}</span>
          </div>
        </header>

        <div v-if="data.active_jobs.length === 0">
          <EmptyState
            icon="activity"
            title="Aucun benchmark en cours"
            description="Lancez un nouveau job pour commencer le provisioning des workers."
          >
            <template #action>
              <RouterLink to="/jobs/new" class="btn-primary btn-sm">
                <Icon name="plus" :size="14" />
                Nouveau job
              </RouterLink>
            </template>
          </EmptyState>
        </div>
        <ul v-else class="divide-y" style="border-color: var(--border-subtle);">
          <li
            v-for="job in data.active_jobs"
            :key="job.id"
            class="flex items-center gap-4 px-5 py-4 hover:bg-soft transition-colors"
          >
            <div class="w-9 h-9 rounded-lg flex items-center justify-center bg-brand-50 text-brand-600 dark:bg-brand-500/10 dark:text-brand-400 shrink-0">
              <Icon :name="modeIcon(job.mode)" :size="18" />
            </div>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-semibold fg-primary truncate">{{ job.name }}</p>
              <p class="text-xs fg-muted truncate">{{ job.client_name }} · mode {{ job.mode }}</p>
            </div>
            <StatusBadge :status="job.status" />
            <RouterLink
              :to="`/dashboard/${job.id}`"
              class="btn-ghost btn-sm"
            >
              Live
              <Icon name="arrow_right" :size="14" />
            </RouterLink>
          </li>
        </ul>
      </section>

      <!-- Cluster -->
      <section class="card-flush">
        <header class="card-header">
          <span class="card-title">Cluster Proxmox</span>
          <span v-if="data.cluster.length" class="pill">{{ data.cluster.length }} nodes</span>
        </header>
        <div class="px-5 py-2">
          <div v-if="data.cluster.length === 0" class="py-6">
            <EmptyState
              icon="server"
              title="Aucune donnée cluster"
              description="Connectez Proxmox pour voir l'état des nodes."
            >
              <template #action>
                <RouterLink to="/settings" class="btn-secondary btn-sm">
                  <Icon name="cog" :size="14" />
                  Configurer
                </RouterLink>
              </template>
            </EmptyState>
          </div>
          <div v-else class="divide-y" style="border-color: var(--border-subtle);">
            <NodeCard
              v-for="node in data.cluster"
              :key="node.name"
              :name="node.name"
              :cpu="node.cpu_pct"
              :ram="node.ram_pct"
            />
          </div>
        </div>
      </section>
    </div>

    <!-- Recent jobs -->
    <section class="card-flush mt-6">
      <header class="card-header">
        <span class="card-title">Jobs récents</span>
        <RouterLink to="/history" class="btn-ghost btn-sm">
          Voir tout
          <Icon name="arrow_right" :size="14" />
        </RouterLink>
      </header>
      <div v-if="data.recent_jobs.length === 0">
        <EmptyState
          icon="history"
          title="Aucun job terminé"
          description="L'historique apparaîtra ici dès qu'un benchmark sera terminé."
        />
      </div>
      <div v-else class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th>Nom</th>
              <th>Client</th>
              <th>Mode</th>
              <th>Date</th>
              <th>Statut</th>
              <th class="text-right pr-5">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="job in data.recent_jobs" :key="job.id">
              <td class="font-medium fg-primary">{{ job.name }}</td>
              <td class="fg-secondary">{{ job.client_name }}</td>
              <td>
                <span class="pill">{{ job.mode }}</span>
              </td>
              <td class="num text-xs fg-muted">{{ formatDate(job.created_at) }}</td>
              <td><StatusBadge :status="job.status" /></td>
              <td class="text-right pr-5">
                <RouterLink
                  v-if="job.status === 'done'"
                  :to="`/jobs/${job.id}`"
                  class="btn-ghost btn-sm"
                >
                  Résultats
                  <Icon name="arrow_right" :size="14" />
                </RouterLink>
                <span v-else class="text-xs fg-faint">—</span>
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { api } from '../api/client.js'
import { useSettingsStore } from '../stores/settings.js'
import PageHeader   from '../components/PageHeader.vue'
import StatCard     from '../components/StatCard.vue'
import StatusBadge  from '../components/StatusBadge.vue'
import EmptyState   from '../components/EmptyState.vue'
import NodeCard     from '../components/NodeCard.vue'
import Icon         from '../components/Icon.vue'
import Spinner      from '../components/Spinner.vue'

const REFRESH_MS = 5000

const settingsStore = useSettingsStore()
const settings = ref(null)
const error    = ref('')
const loading  = ref(false)

const data = ref({ active_jobs: [], recent_jobs: [], cluster: [] })

const configured  = computed(() => !!settings.value?.proxmox_url)
const doneCount   = computed(() => data.value.recent_jobs.filter(j => j.status === 'done').length)
const onlineHint  = computed(() => {
  if (!data.value.cluster.length) return 'Aucun cluster détecté'
  const online = data.value.cluster.filter(n => n.online).length
  return `${online}/${data.value.cluster.length} en ligne`
})
const avgCpu = computed(() => {
  if (!data.value.cluster.length) return '—'
  const total = data.value.cluster.reduce((s, n) => s + (n.cpu_pct || 0), 0)
  return (total / data.value.cluster.length).toFixed(1)
})

let mounted = false
let timer = null

async function loadOverview() {
  loading.value = true
  try {
    const resp = await api.getOverview()
    if (resp && mounted) data.value = resp
    error.value = ''
  } catch (e) {
    error.value = "Impossible de charger les données (API inaccessible)."
  } finally {
    loading.value = false
  }
}

async function loadSettings() {
  try { settings.value = await settingsStore.load() } catch (_) {}
}

function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  if (isNaN(d)) return '—'
  return d.toLocaleString('fr-FR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function modeIcon(mode) {
  if (mode === 'cpu')     return 'cpu'
  if (mode === 'storage') return 'hard_drive'
  return 'shuffle'
}

onMounted(async () => {
  mounted = true
  await loadSettings()
  await loadOverview()
  timer = setInterval(loadOverview, REFRESH_MS)
})

onUnmounted(() => {
  mounted = false
  clearInterval(timer)
})
</script>
