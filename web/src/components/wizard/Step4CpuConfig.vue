<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.cpuConfig.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.cpuConfig.subtitle') }}</p>
    </header>

    <section class="card space-y-4">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.cpuConfig.stressors') }}</h3>
      <p class="helper">{{ t('wizard.steps.cpuConfig.stressorsHint') }}</p>
      <div class="flex flex-wrap gap-2">
        <button
          v-for="s in ALLOWED_STRESSORS"
          :key="s"
          type="button"
          class="stressor-pill"
          :class="{ 'is-active': store.cpuConfig.stressors.includes(s) }"
          @click="toggle(s)"
        >
          <Icon v-if="store.cpuConfig.stressors.includes(s)" name="check" :size="11" stroke-width="3" />
          <span class="num">{{ s }}</span>
        </button>
      </div>
    </section>

    <section class="card grid grid-cols-1 md:grid-cols-2 gap-4">
      <div>
        <label class="label">{{ t('wizard.steps.cpuConfig.stressorWorkers') }}</label>
        <input
          v-model.number="store.cpuConfig.stressorWorkers"
          type="number"
          min="1"
          max="64"
          class="input"
        />
        <p class="helper">{{ t('wizard.steps.cpuConfig.stressorWorkersHint') }}</p>
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.cpuConfig.timeoutSec') }}</label>
        <input
          v-model.number="store.cpuConfig.timeoutSec"
          type="number"
          min="10"
          max="3600"
          class="input"
        />
        <p class="helper">{{ t('wizard.steps.cpuConfig.timeoutSecHint') }}</p>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()

// Hardcoded allowlist. The user cannot type a stressor name through this
// step, so injection into the stress-ng arg vector via the wizard surface
// is impossible. The backend C1 fix is tracked separately, but this UI
// hardening prevents the wizard itself from being a vector.
const ALLOWED_STRESSORS = [
  'cpu',
  'vm',
  'io',
  'hdd',
  'matrix',
  'cache',
  'pipe',
]

function toggle(name) {
  if (!ALLOWED_STRESSORS.includes(name)) return
  const i = store.cpuConfig.stressors.indexOf(name)
  if (i === -1) {
    store.cpuConfig.stressors.push(name)
  } else {
    store.cpuConfig.stressors.splice(i, 1)
  }
}

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)
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

.stressor-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--fg-secondary);
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  font-size: 12px;
  transition: background 120ms ease, color 120ms ease, border-color 120ms ease;
}

.stressor-pill:hover {
  background: var(--bg-elevated);
  border-color: var(--border-default);
}

.stressor-pill.is-active {
  background: #f97316;
  color: #ffffff;
  border-color: #f97316;
}
</style>
