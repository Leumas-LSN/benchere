<template>
  <aside class="wizard-sidebar">
    <p class="section-eyebrow mb-4">{{ t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }) }}</p>
    <ol class="space-y-1">
      <li v-for="(key, idx) in store.stepKeys" :key="key" class="wizard-step-row" :class="rowClass(idx + 1)">
        <button
          type="button"
          class="wizard-step-btn"
          :disabled="!canReach(idx + 1)"
          @click="store.goTo(idx + 1)"
        >
          <span class="wizard-step-marker" :class="markerClass(idx + 1)">
            <Icon v-if="isDone(idx + 1)" name="check" :size="13" stroke-width="3" />
            <span v-else class="num">{{ idx + 1 }}</span>
          </span>
          <span class="min-w-0 flex-1 text-left">
            <span class="block text-sm font-medium fg-primary">{{ stepLabel(key) }}</span>
            <span v-if="sublabel(key)" class="block text-xs num fg-muted truncate">{{ sublabel(key) }}</span>
          </span>
        </button>
      </li>
    </ol>
  </aside>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { useWizardStore } from '../../stores/wizard.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()

function stepLabel(key) {
  return t('wizard.steps.' + key + '.title')
}

function sublabel(key) {
  return store.sublabelFor(key)
}

function isDone(idx) {
  return idx < store.currentStep && store.stepValid[idx - 1]
}

function isCurrent(idx) {
  return idx === store.currentStep
}

function canReach(idx) {
  if (idx === store.currentStep) return true
  if (idx < store.currentStep) return true
  for (let i = 1; i < idx; i++) {
    if (!store.stepValid[i - 1]) return false
  }
  return true
}

function rowClass(idx) {
  if (isCurrent(idx)) return 'is-current'
  if (isDone(idx)) return 'is-done'
  return 'is-future'
}

function markerClass(idx) {
  if (isDone(idx)) return 'marker-done'
  if (isCurrent(idx)) return 'marker-current'
  return 'marker-future'
}
</script>

<style scoped>
.wizard-sidebar {
  padding: 24px 16px;
  border-right: 1px solid var(--border-subtle);
  min-width: 200px;
  max-width: 240px;
}

.wizard-step-row {
  position: relative;
  border-radius: 10px;
}

.wizard-step-row.is-current {
  background: var(--bg-brand-soft);
}

.wizard-step-row.is-current::before {
  content: '';
  position: absolute;
  left: -8px;
  top: 8px;
  bottom: 8px;
  width: 3px;
  background: #f97316;
  border-radius: 2px;
}

.wizard-step-btn {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  background: transparent;
  border: 0;
  border-radius: 10px;
  cursor: pointer;
  transition: background 120ms ease;
}

.wizard-step-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.wizard-step-btn:not(:disabled):hover {
  background: var(--bg-muted);
}

.wizard-step-row.is-current .wizard-step-btn:not(:disabled):hover {
  background: rgba(249, 115, 22, 0.10);
}

.wizard-step-marker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  margin-top: 1px;
}

.marker-done {
  background: #16a34a;
  color: #ffffff;
}
html.dark .marker-done {
  background: #4ade80;
  color: #052e16;
}

.marker-current {
  background: #f97316;
  color: #ffffff;
}

.marker-future {
  background: var(--bg-muted);
  color: var(--fg-muted);
  border: 1px solid var(--border-default);
}
</style>
