<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.profiles.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.profiles.subtitle') }}</p>
    </header>

    <div class="filters-row">
      <div class="filter-pills">
        <button
          v-for="f in filters"
          :key="f.key"
          type="button"
          class="filter-pill"
          :class="{ 'is-active': activeFilter === f.key }"
          @click="activeFilter = f.key"
        >{{ f.label }}</button>
      </div>
      <div class="search-wrap">
        <Icon name="search" :size="14" class="search-icon" />
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          :placeholder="t('wizard.steps.profiles.search')"
        />
      </div>
    </div>

    <div v-if="loading" class="text-sm fg-muted">{{ t('common.loading') }}</div>
    <div v-else-if="!filtered.length" class="text-sm fg-muted">
      {{ t('wizard.steps.profiles.empty') }}
    </div>
    <div v-else class="grid grid-cols-1 xl:grid-cols-2 gap-3">
      <ProfileCard
        v-for="p in filtered"
        :key="p.id"
        :name="p.name"
        :bs="bsOf(p)"
        :rw="rwOf(p)"
        :est-minutes="estMinutesOf(p)"
        :popular="isPopular(p.name)"
        :built-in="isBuiltIn(p)"
        :selected="store.profiles.includes(p.name)"
        @toggle="toggle(p.name)"
      />
    </div>

    <div v-if="store.profiles.length > 0" class="estimate-alert">
      <Icon name="zap" :size="16" class="shrink-0" />
      <span>{{ estimateLine }}</span>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import { api } from '../../api/client.js'
import Icon from '../Icon.vue'
import ProfileCard from './ProfileCard.vue'

const { t } = useI18n()
const store = useWizardStore()
const loading = ref(true)
const profiles = ref([])
const activeFilter = ref('all')
const searchQuery = ref('')

// Profile names that ship with default thresholds and broad coverage. The
// POPULAIRE pill highlights the canonical six industry-standard profiles
// from migration 012 so users land on a sensible default selection.
const POPULAR_NAMES = new Set([
  'oltp-4k-70-30',
  'sql-8k-70-30',
  'vdi-4k-20-80',
  'backup-256k-read',
  'backup-256k-write',
  'mixed-32k-50-50',
])

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

const filters = computed(() => [
  { key: 'all',        label: t('wizard.steps.profiles.filters.all') },
  { key: 'random',     label: t('wizard.steps.profiles.filters.random') },
  { key: 'sequential', label: t('wizard.steps.profiles.filters.sequential') },
  { key: 'mixed',      label: t('wizard.steps.profiles.filters.mixed') },
  { key: 'custom',     label: t('wizard.steps.profiles.filters.custom') },
])

onMounted(async () => {
  loading.value = true
  try {
    const list = await api.listProfiles() ?? []
    // Default to fio profiles, since elbencho is gated behind a Settings
    // toggle in v2.0+ and the wizard targets fio as the primary engine.
    profiles.value = list.filter(p => (p.engine || 'fio') === 'fio')
  } catch (_) { /* leave empty so the empty state shows */ }
  loading.value = false
})

function iniValue(body, key) {
  const prefix = key + '='
  for (const line of String(body || '').split('\n')) {
    const l = line.trim()
    if (l.startsWith(prefix)) {
      return l.slice(prefix.length).trim()
    }
  }
  return ''
}

function bsOf(p) { return iniValue(p.config_json, 'bs') }
function rwOf(p) { return iniValue(p.config_json, 'rw') }

function estMinutesOf(p) {
  const runtime = parseInt(iniValue(p.config_json, 'runtime'), 10) || 0
  const ramp = parseInt(iniValue(p.config_json, 'ramp_time'), 10) || 0
  if (runtime <= 0) return 0
  return Math.round((runtime + ramp) / 60)
}

function isPopular(name) { return POPULAR_NAMES.has(name) }
function isBuiltIn(p) { return p.is_builtin === true || p.is_builtin === 1 }

const filtered = computed(() => {
  let list = profiles.value.slice()
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(p => p.name.toLowerCase().includes(q))
  }
  switch (activeFilter.value) {
    case 'random':
      list = list.filter(p => {
        const rw = rwOf(p)
        return rw === 'randread' || rw === 'randwrite'
      })
      break
    case 'sequential':
      list = list.filter(p => {
        const rw = rwOf(p)
        return rw === 'read' || rw === 'write'
      })
      break
    case 'mixed':
      list = list.filter(p => rwOf(p) === 'randrw')
      break
    case 'custom':
      list = list.filter(p => !isBuiltIn(p))
      break
    case 'all':
    default:
      break
  }
  return list
})

function toggle(name) {
  const i = store.profiles.indexOf(name)
  if (i === -1) {
    store.profiles.push(name)
  } else {
    store.profiles.splice(i, 1)
  }
}

const estimateLine = computed(() => {
  const sec = store.estimate.wallclockSec || 0
  const mins = Math.max(1, Math.round(sec / 60))
  return t('wizard.steps.profiles.estimate', {
    wallclock: mins + 'm',
    count: store.profiles.length,
    workers: store.workers.count || 1,
  })
})
</script>

<style scoped>
.step-title {
  font-size: 28px;
  font-weight: 600;
  color: var(--fg-primary);
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.step-subtitle {
  margin-top: 6px;
  font-size: 14px;
  color: var(--fg-secondary);
  max-width: 60ch;
}

.filters-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.filter-pills {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 6px;
}

.filter-pill {
  font-size: 12px;
  padding: 6px 14px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--fg-secondary);
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}

.filter-pill:hover {
  background: var(--bg-elevated);
  border-color: var(--border-default);
}

.filter-pill.is-active {
  background: #f97316;
  color: #ffffff;
  border-color: #f97316;
}

.search-wrap {
  position: relative;
  flex: 0 0 240px;
  max-width: 240px;
}

.search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--fg-muted);
  pointer-events: none;
}

.search-input {
  width: 100%;
  height: 36px;
  padding: 0 12px 0 32px;
  border-radius: 9px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  color: var(--fg-primary);
  font-size: 13px;
}

.search-input:focus {
  outline: none;
  border-color: #f97316;
  box-shadow: 0 0 0 3px rgba(249, 115, 22, 0.18);
}

.estimate-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--bg-brand-soft);
  border: 1px solid #fdba74;
  color: #9a3412;
  font-size: 13px;
}
html.dark .estimate-alert {
  border-color: rgba(249, 115, 22, 0.35);
  color: #fdba74;
}
</style>
