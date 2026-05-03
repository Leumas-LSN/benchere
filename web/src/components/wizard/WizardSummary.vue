<template>
  <aside class="wizard-summary">
    <section class="card-flush mb-4">
      <header class="card-header">
        <span class="card-title">{{ t('wizard.summary.title') }}</span>
      </header>
      <div class="px-5 py-4 space-y-2.5 text-sm">
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.cluster') }}</span>
          <span class="row-value num">{{ store.cluster || '-' }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.pool') }}</span>
          <span class="row-value num">{{ poolsSummary }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.profiles') }}</span>
          <span class="row-value num">{{ store.profiles.length || 0 }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.workers') }}</span>
          <span class="row-value num">{{ store.workers.count || 0 }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.ioDepth') }}</span>
          <span class="row-value num">{{ store.range.ioDepth || 0 }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.runtime') }}</span>
          <span class="row-value num">{{ runtimeLabel }}</span>
        </div>
        <div class="row">
          <span class="row-label">{{ t('wizard.summary.verdict') }}</span>
          <span class="row-value">{{ verdictLabel }}</span>
        </div>
      </div>
    </section>

    <section class="card-flush">
      <header class="card-header">
        <span class="card-title">{{ t('wizard.cost.title') }}</span>
      </header>
      <div class="px-5 py-5">
        <p class="cost-headline num">{{ wallclockHuman }}</p>
        <p class="text-xs fg-muted mt-1">{{ t('wizard.cost.wallclock') }}</p>
        <p class="text-xs fg-muted mt-3 num">{{ t('wizard.cost.bytes', { gb: bytesGB }) }}</p>
      </div>
    </section>
  </aside>
</template>

<script setup>
import { computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'

const { t } = useI18n()
const store = useWizardStore()

const wallclockHuman = computed(() => {
  const s = store.estimate.wallclockSec || 0
  if (s <= 0) return '-'
  const minutes = Math.round(s / 60)
  if (minutes < 60) return '~' + minutes + 'm'
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return mins === 0 ? '~' + hours + 'h' : '~' + hours + 'h ' + mins + 'm'
})

const bytesGB = computed(() => {
  const b = store.estimate.bytesWritten || 0
  if (b <= 0) return 0
  return Math.round(b / (1024 * 1024 * 1024))
})

const runtimeLabel = computed(() => {
  const r = store.range.runtimeSec || 0
  return r > 0 ? r + 's' : '-'
})

const verdictLabel = computed(() =>
  store.range.thresholdsCustom
    ? t('wizard.summary.verdictCustom')
    : t('wizard.summary.verdictAuto'),
)

// Compact summary of selected pools: '-' when none, the single pool
// name when one, a count when several so the right rail stays narrow.
const poolsSummary = computed(() => {
  const list = store.pools || []
  if (!list.length) return '-'
  if (list.length === 1) return list[0]
  return list.length + ' pools'
})

// Debounce the estimate fetch so typing in workers count does not spam
// the backend. The wizard fetches on mount and after every relevant
// state change.
let debounceTimer = null
function scheduleFetch() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    store.fetchEstimate()
  }, 500)
}

watch(
  () => [
    store.type,
    store.workers.count,
    store.workers.dataDisks,
    store.workers.dataDiskGb,
    store.profiles.length,
    store.range.runtimeSec,
    store.range.rampSec,
    store.cpuConfig.timeoutSec,
  ],
  scheduleFetch,
)

onMounted(() => {
  store.fetchEstimate()
})
</script>

<style scoped>
.wizard-summary {
  width: 300px;
  padding: 24px 16px;
  border-left: 1px solid var(--border-subtle);
}

.row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
}

.row-label {
  color: var(--fg-muted);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.row-value {
  color: var(--fg-primary);
  font-size: 13px;
  text-align: right;
  word-break: break-all;
  max-width: 60%;
}

.cost-headline {
  font-size: 32px;
  font-weight: 600;
  line-height: 1;
  color: #f97316;
  letter-spacing: -0.02em;
}
</style>
