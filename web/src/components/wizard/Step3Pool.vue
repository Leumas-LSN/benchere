<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.pool.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.pool.subtitle') }}</p>
    </header>

    <div v-if="loading" class="text-sm fg-muted">{{ t('wizard.steps.pool.loading') }}</div>
    <div v-else-if="error" class="alert-error">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>
    <div v-else-if="!pools.length" class="text-sm fg-muted">
      {{ t('wizard.steps.pool.empty') }}
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <button
        v-for="p in pools"
        :key="p.id"
        type="button"
        class="pool-card"
        :class="{ 'is-selected': store.pool === p.id }"
        @click="select(p.id)"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="pool-name num">{{ p.id }}</span>
          <span class="pool-pill">{{ p.type }}</span>
        </div>
        <p v-if="p.content" class="pool-meta num">{{ p.content }}</p>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()
const loading = ref(true)
const error = ref('')
const pools = ref([])

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

onMounted(async () => {
  loading.value = true
  try {
    // Use the cluster-wide endpoint when no nodes selected yet, the
    // intersected endpoint once Workers step has narrowed the set.
    const url = store.workers.nodes.length
      ? '/api/proxmox/storages?nodes=' + encodeURIComponent(store.workers.nodes.join(','))
      : '/api/proxmox/storages'
    const r = await fetch(url)
    if (!r.ok) {
      error.value = await r.text()
      return
    }
    const list = (await r.json()) || []
    // Only pools that can host VM disks (content includes images).
    pools.value = list.filter(s => (s.content || '').split(',').map(c => c.trim()).includes('images'))
  } catch (e) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
})

function select(id) {
  store.pool = id
}
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

.pool-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 16px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 12px;
  text-align: left;
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
}

.pool-card:hover {
  border-color: var(--border-strong);
  background: var(--bg-muted);
}

.pool-card.is-selected {
  border-color: #f97316;
  background: var(--bg-brand-soft);
}

.pool-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg-primary);
}

.pool-meta {
  font-size: 12px;
  color: var(--fg-muted);
}

.pool-pill {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--bg-muted);
  color: var(--fg-secondary);
  border: 1px solid var(--border-subtle);
}
</style>
