<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.type.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.type.subtitle') }}</p>
    </header>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <button
        v-for="opt in options"
        :key="opt.value"
        type="button"
        class="type-card"
        :class="{ 'is-selected': store.type === opt.value }"
        @click="select(opt.value)"
      >
        <span class="type-icon">
          <Icon :name="opt.icon" :size="24" />
        </span>
        <span class="type-label">{{ opt.label }}</span>
        <span class="type-hint">{{ opt.hint }}</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

const options = computed(() => [
  { value: 'storage', icon: 'hard_drive', label: t('wizard.steps.type.storage'), hint: t('wizard.steps.type.storageHint') },
  { value: 'cpu',     icon: 'cpu',         label: t('wizard.steps.type.cpu'),     hint: t('wizard.steps.type.cpuHint') },
  { value: 'mixed',   icon: 'shuffle',     label: t('wizard.steps.type.mixed'),   hint: t('wizard.steps.type.mixedHint') },
])

function select(value) {
  store.type = value
  // Reset profile selection so swapping mode does not carry incompatible
  // profiles (e.g., from storage to cpu).
  if (value === 'cpu') store.profiles = []
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

.type-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  padding: 20px 18px;
  border-radius: 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  text-align: left;
  cursor: pointer;
  transition: border-color 120ms ease, background 120ms ease;
}

.type-card:hover {
  border-color: var(--border-strong);
  background: var(--bg-muted);
}

.type-card.is-selected {
  border-color: #f97316;
  background: var(--bg-brand-soft);
}

.type-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: var(--bg-muted);
  color: var(--fg-secondary);
}

.type-card.is-selected .type-icon {
  background: #f97316;
  color: #ffffff;
}

.type-label {
  font-size: 16px;
  font-weight: 600;
  color: var(--fg-primary);
}

.type-hint {
  font-size: 13px;
  color: var(--fg-muted);
}
</style>
