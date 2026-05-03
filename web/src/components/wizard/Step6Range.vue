<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.range.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.range.subtitle') }}</p>
    </header>

    <section class="card grid grid-cols-1 md:grid-cols-3 gap-4">
      <div>
        <label class="label">{{ t('wizard.steps.range.runtime') }}</label>
        <input v-model.number="store.range.runtimeSec" type="number" min="1" max="3600" class="input" />
        <p class="helper">{{ t('wizard.steps.range.runtimeHint') }}</p>
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.range.ramp') }}</label>
        <input v-model.number="store.range.rampSec" type="number" min="0" max="600" class="input" />
        <p class="helper">{{ t('wizard.steps.range.rampHint') }}</p>
      </div>
      <div>
        <label class="label">{{ t('wizard.steps.range.ioDepth') }}</label>
        <input v-model.number="store.range.ioDepth" type="number" min="1" max="256" class="input" />
        <p class="helper">{{ t('wizard.steps.range.ioDepthHint') }}</p>
      </div>
    </section>

    <section class="card space-y-4">
      <h3 class="text-sm font-semibold fg-primary">{{ t('wizard.steps.range.thresholdsHeader') }}</h3>
      <div class="flex gap-3">
        <button
          type="button"
          class="mode-card flex-1"
          :class="{ 'is-selected': !store.range.thresholdsCustom }"
          @click="store.range.thresholdsCustom = false"
        >
          <span class="mode-card-label">{{ t('wizard.steps.range.verdictAuto') }}</span>
          <span class="mode-card-hint">{{ t('wizard.steps.range.verdictAutoHint') }}</span>
        </button>
        <button
          type="button"
          class="mode-card flex-1"
          :class="{ 'is-selected': store.range.thresholdsCustom }"
          @click="store.range.thresholdsCustom = true"
        >
          <span class="mode-card-label">{{ t('wizard.steps.range.verdictCustom') }}</span>
          <span class="mode-card-hint">{{ t('wizard.steps.range.verdictCustomHint') }}</span>
        </button>
      </div>

      <div v-if="store.range.thresholdsCustom" class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2">
        <div>
          <label class="label">{{ t('wizard.steps.range.minIopsRead') }}</label>
          <input v-model.number="store.range.thresholds.minIopsRead" type="number" min="0" class="input" />
        </div>
        <div>
          <label class="label">{{ t('wizard.steps.range.minIopsWrite') }}</label>
          <input v-model.number="store.range.thresholds.minIopsWrite" type="number" min="0" class="input" />
        </div>
        <div>
          <label class="label">{{ t('wizard.steps.range.maxAvgLatencyMs') }}</label>
          <input v-model.number="store.range.thresholds.maxAvgLatencyMs" type="number" min="0" step="0.1" class="input" />
        </div>
        <div>
          <label class="label">{{ t('wizard.steps.range.maxP99LatencyMs') }}</label>
          <input v-model.number="store.range.thresholds.maxP99LatencyMs" type="number" min="0" step="0.1" class="input" />
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'

const { t } = useI18n()
const store = useWizardStore()

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
</style>
