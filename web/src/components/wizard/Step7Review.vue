<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.review.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.review.subtitle') }}</p>
    </header>

    <section class="card space-y-4">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.review.identity') }}</h3>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="label">{{ t('wizard.steps.review.jobName') }}</label>
          <input v-model="store.meta.name" type="text" class="input" placeholder="bench-prod-01" />
        </div>
        <div>
          <label class="label">{{ t('wizard.steps.review.clientName') }}</label>
          <input v-model="store.meta.clientName" type="text" class="input" placeholder="Acme" />
        </div>
      </div>
    </section>

    <section v-if="store.type !== 'cpu'" class="card space-y-2">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.review.target') }}</h3>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.cluster') }}</span>
        <span class="rv num">{{ store.cluster || '-' }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.pool') }}</span>
        <span class="rv num">{{ poolsLabel }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.nodes') }}</span>
        <span class="rv num">{{ store.workers.nodes.join(', ') || '-' }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.workers') }}</span>
        <span class="rv num">{{ store.workers.count }} ({{ store.workers.cpu }} vCPU . {{ store.workers.ramMb }} MB)</span>
      </div>
    </section>

    <section v-if="store.type !== 'cpu'" class="card space-y-2">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.review.workload') }}</h3>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.profiles') }}</span>
        <span class="rv num">{{ store.profiles.join(', ') || '-' }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.runtime') }}</span>
        <span class="rv num">{{ store.range.runtimeSec }}s</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.ramp') }}</span>
        <span class="rv num">{{ store.range.rampSec }}s</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.ioDepth') }}</span>
        <span class="rv num">{{ store.range.ioDepth }}</span>
      </div>
    </section>

    <section v-if="store.type !== 'storage'" class="card space-y-2">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.cpuConfig.title') }}</h3>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.cpuStressors') }}</span>
        <span class="rv num">{{ store.cpuConfig.stressors.join(', ') || '-' }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.cpuTimeout') }}</span>
        <span class="rv num">{{ store.cpuConfig.timeoutSec }}s</span>
      </div>
    </section>

    <section class="card space-y-2">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.review.cost') }}</h3>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.wallclock') }}</span>
        <span class="rv num" style="color: #f97316;">{{ wallclockHuman }}</span>
      </div>
      <div class="review-row">
        <span class="rl">{{ t('wizard.steps.review.bytes') }}</span>
        <span class="rv num">~{{ bytesGB }} GB</span>
      </div>
    </section>

    <div v-if="multiPoolJobs > 1" class="alert-info">
      <Icon name="info" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ t('wizard.steps.review.multipleJobs', { count: multiPoolJobs }) }}</span>
    </div>

    <div v-if="error" class="alert-error">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <div class="flex justify-end pt-2">
      <button
        type="button"
        class="btn-primary btn-lg"
        :disabled="submitting"
        @click="onLaunch"
      >
        <Icon v-if="!submitting" name="play" :size="16" />
        <span v-else class="spinner" />
        {{ submitting ? t('common.loading') : launchLabel }}
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()
const router = useRouter()
const submitting = ref(false)
const error = ref('')

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

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

// Comma-joined pool list for the Pool review row. Falls back to '-'
// when no pool is selected (e.g. cpu mode).
const poolsLabel = computed(() => {
  const list = store.pools || []
  return list.length ? list.join(', ') : '-'
})

// Number of jobs that submit() will create. One per selected pool when
// the user picked several; one job otherwise.
const multiPoolJobs = computed(() => {
  const list = store.pools || []
  return list.length > 1 ? list.length : 1
})

const launchLabel = computed(() =>
  multiPoolJobs.value > 1
    ? t('wizard.launchMany', { count: multiPoolJobs.value })
    : t('wizard.launch'),
)

async function onLaunch() {
  error.value = ''
  submitting.value = true
  try {
    const ids = await store.submit()
    // Single job: drop the user on the live dashboard for it. Multiple
    // jobs (one per pool): send to history so they can pick which to
    // watch, since only one can be foregrounded at a time.
    const list = Array.isArray(ids) ? ids : (ids ? [ids] : [])
    if (list.length === 1) {
      router.push('/dashboard/' + list[0])
    } else if (list.length > 1) {
      router.push('/history')
    } else {
      error.value = t('wizard.submitError', { msg: 'no job created' })
      submitting.value = false
    }
  } catch (e) {
    error.value = t('wizard.submitError', { msg: e.message || String(e) })
    submitting.value = false
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

.review-row {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 12px;
  font-size: 13px;
  padding: 4px 0;
}

.rl {
  color: var(--fg-muted);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.rv {
  color: var(--fg-primary);
  text-align: right;
  word-break: break-all;
  max-width: 70%;
}

.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 800ms linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
