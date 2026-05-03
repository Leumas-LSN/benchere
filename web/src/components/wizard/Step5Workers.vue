<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.workers.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.workers.subtitle') }}</p>
    </header>

    <section class="card space-y-4">
      <div class="flex gap-3">
        <button
          type="button"
          class="mode-card flex-1"
          :class="{ 'is-selected': store.workers.auto }"
          @click="store.workers.auto = true"
        >
          <span class="mode-card-label">{{ t('wizard.steps.workers.auto') }}</span>
          <span class="mode-card-hint">{{ t('wizard.steps.workers.autoHint') }}</span>
        </button>
        <button
          type="button"
          class="mode-card flex-1"
          :class="{ 'is-selected': !store.workers.auto }"
          @click="store.workers.auto = false"
        >
          <span class="mode-card-label">{{ t('wizard.steps.workers.manual') }}</span>
          <span class="mode-card-hint">{{ t('wizard.steps.workers.manualHint') }}</span>
        </button>
      </div>
    </section>

    <section class="card space-y-4">
      <header class="flex items-center justify-between">
        <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.workers.nodes') }}</h3>
        <button
          v-if="availableNodes.length"
          type="button"
          class="text-xs"
          style="color: #f97316"
          @click="toggleAllNodes"
        >{{ allNodesSelected ? t('wizard.steps.workers.nodesNone') : t('wizard.steps.workers.nodesAll') }}</button>
      </header>
      <p class="helper">{{ t('wizard.steps.workers.nodesHint') }}</p>
      <div v-if="loadingNodes" class="text-sm fg-muted">{{ t('common.loading') }}</div>
      <div v-else-if="!availableNodes.length" class="text-sm fg-muted">
        {{ t('wizard.steps.workers.empty') }}
      </div>
      <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-2">
        <button
          v-for="n in availableNodes"
          :key="n.name"
          type="button"
          class="node-card"
          :class="{ 'is-selected': store.workers.nodes.includes(n.name) }"
          @click="toggleNode(n.name)"
        >
          <span class="checkbox" :class="{ 'is-checked': store.workers.nodes.includes(n.name) }">
            <Icon v-if="store.workers.nodes.includes(n.name)" name="check" :size="11" stroke-width="3" />
          </span>
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium num fg-primary truncate">{{ n.name }}</p>
            <p class="text-xs num fg-muted">CPU {{ n.cpu_pct.toFixed(0) }}% . RAM {{ n.ram_pct.toFixed(0) }}%</p>
          </div>
        </button>
      </div>
    </section>

    <section class="card grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="label">{{ t('wizard.steps.workers.count') }}</label>
        <input
          v-model.number="store.workers.count"
          type="number"
          min="1"
          max="40"
          class="input"
          :disabled="store.workers.auto"
        />
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.workers.cpu') }}</label>
        <input v-model.number="store.workers.cpu" type="number" min="1" max="64" class="input" />
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.workers.ramMb') }}</label>
        <input v-model.number="store.workers.ramMb" type="number" min="512" step="512" class="input" />
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.workers.osDiskGb') }}</label>
        <input v-model.number="store.workers.osDiskGb" type="number" min="10" max="200" class="input" />
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.workers.dataDisks') }}</label>
        <input v-model.number="store.workers.dataDisks" type="number" min="0" max="8" class="input" />
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.workers.dataDiskGb') }}</label>
        <input v-model.number="store.workers.dataDiskGb" type="number" min="1" max="1024" class="input" />
      </div>
    </section>

    <p v-if="store.workers.nodes.length > 0 && store.workers.count > 0" class="text-xs fg-muted num">
      {{ t('wizard.steps.workers.total', {
        total: store.workers.count,
        perNode: perNode,
        nodes: store.workers.nodes.length,
      }) }}
    </p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()
const availableNodes = ref([])
const loadingNodes = ref(true)

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

const allNodesSelected = computed(() =>
  availableNodes.value.length > 0
  && store.workers.nodes.length === availableNodes.value.length,
)

const perNode = computed(() => {
  if (store.workers.nodes.length === 0) return 0
  return Math.max(1, Math.ceil(store.workers.count / store.workers.nodes.length))
})

onMounted(async () => {
  loadingNodes.value = true
  try {
    const r = await fetch('/api/overview')
    if (r.ok) {
      const data = await r.json()
      availableNodes.value = (data.cluster || []).map(n => ({
        name: n.name,
        cpu_pct: typeof n.cpu_pct === 'number' ? n.cpu_pct : 0,
        ram_pct: typeof n.ram_pct === 'number' ? n.ram_pct : 0,
      }))
      // Auto-pick the default node when nothing chosen yet.
      if (store.workers.nodes.length === 0 && data.default_node) {
        const exists = availableNodes.value.find(n => n.name === data.default_node)
        if (exists) store.workers.nodes = [data.default_node]
      }
    }
  } catch (_) { /* leave empty */ }
  loadingNodes.value = false
})

// Auto mode keeps the count tracking the number of selected profiles or
// a sensible floor of 4.
watch(
  () => [store.workers.auto, store.profiles.length],
  () => {
    if (store.workers.auto) {
      store.workers.count = Math.max(4, store.profiles.length || 4)
    }
  },
  { immediate: true },
)

function toggleNode(name) {
  const i = store.workers.nodes.indexOf(name)
  if (i === -1) store.workers.nodes.push(name)
  else store.workers.nodes.splice(i, 1)
}

function toggleAllNodes() {
  if (allNodesSelected.value) {
    store.workers.nodes = []
  } else {
    store.workers.nodes = availableNodes.value.map(n => n.name)
  }
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

.mode-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 14px 16px;
  border-radius: 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
  text-align: left;
}

.mode-card:hover {
  border-color: var(--border-strong);
  background: var(--bg-muted);
}

.mode-card.is-selected {
  border-color: #f97316;
  background: var(--bg-brand-soft);
}

.mode-card-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--fg-primary);
}

.mode-card-hint {
  font-size: 12px;
  color: var(--fg-muted);
}

.node-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
  text-align: left;
}

.node-card:hover {
  border-color: var(--border-strong);
  background: var(--bg-muted);
}

.node-card.is-selected {
  border-color: #f97316;
  background: var(--bg-brand-soft);
}

.checkbox {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 5px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  color: var(--fg-onbrand);
  flex-shrink: 0;
}

.checkbox.is-checked {
  background: #f97316;
  border-color: #f97316;
}
</style>
