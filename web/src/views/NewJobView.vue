<template>
  <div class="wizard-shell">
    <header class="wizard-topbar">
      <div class="flex items-center gap-3">
        <RouterLink to="/" class="benchere-link">
          <BenchereWordmark size="sm" />
        </RouterLink>
        <span class="breadcrumb">
          <Icon name="chevron_right" :size="14" class="breadcrumb-sep" />
          <span class="breadcrumb-text">{{ t('wizard.title') }} - {{ t('wizard.draft') }}</span>
        </span>
      </div>
      <div class="flex items-center gap-2">
        <button type="button" class="btn-secondary btn-sm" @click="onCancel">{{ t('wizard.cancel') }}</button>
        <button type="button" class="btn-secondary btn-sm" @click="onSaveDraft">
          <Icon name="check" :size="13" stroke-width="3" />
          {{ t('wizard.saveDraft') }}
        </button>
      </div>
    </header>

    <div v-if="draftRestored" class="draft-banner">
      <Icon name="info" :size="16" />
      <span class="flex-1">{{ t('wizard.draftRestored') }}</span>
      <button type="button" class="btn-sm btn-ghost" @click="onResetDraft">{{ t('wizard.draftReset') }}</button>
      <button type="button" class="btn-sm btn-ghost" @click="draftRestored = false">
        <Icon name="x" :size="14" />
      </button>
    </div>

    <div v-if="toast" class="toast">{{ toast }}</div>

    <div class="wizard-grid">
      <WizardSidebar />

      <main class="wizard-main">
        <component :is="currentStepComponent" />

        <footer class="wizard-footer">
          <button
            type="button"
            class="btn-ghost btn-lg"
            :disabled="store.currentStep === 1"
            @click="store.previous()"
          >
            <Icon name="chevron_left" :size="16" />
            {{ t('wizard.previous') }}
          </button>
          <button
            v-if="!isLastStep"
            type="button"
            class="btn-primary btn-lg"
            :disabled="!store.stepValid[store.currentStep - 1]"
            @click="store.next()"
          >
            {{ nextLabel }}
            <Icon name="chevron_right" :size="16" />
          </button>
        </footer>
      </main>

      <WizardSummary />
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, RouterLink } from 'vue-router'
import { useWizardStore } from '../stores/wizard.js'
import BenchereWordmark from '../components/BenchereWordmark.vue'
import Icon from '../components/Icon.vue'
import WizardSidebar from '../components/wizard/WizardSidebar.vue'
import WizardSummary from '../components/wizard/WizardSummary.vue'
import Step1Type from '../components/wizard/Step1Type.vue'
import Step2Cluster from '../components/wizard/Step2Cluster.vue'
import Step3Pool from '../components/wizard/Step3Pool.vue'
import Step4Profiles from '../components/wizard/Step4Profiles.vue'
import Step4CpuConfig from '../components/wizard/Step4CpuConfig.vue'
import Step5Workers from '../components/wizard/Step5Workers.vue'
import Step6Range from '../components/wizard/Step6Range.vue'
import Step7Review from '../components/wizard/Step7Review.vue'

const { t } = useI18n()
const router = useRouter()
const store = useWizardStore()
const draftRestored = ref(false)
const toast = ref('')

// Mapping of step keys (defined in stores/wizard.js STEP_DEFS) to the
// concrete components rendered in the main column.
const COMPONENTS = {
  type: Step1Type,
  cluster: Step2Cluster,
  pool: Step3Pool,
  profiles: Step4Profiles,
  cpuConfig: Step4CpuConfig,
  workers: Step5Workers,
  range: Step6Range,
  review: Step7Review,
}

const currentStepComponent = computed(() => {
  const key = store.stepKeys[store.currentStep - 1]
  return COMPONENTS[key] || Step1Type
})

const isLastStep = computed(() => store.currentStep === store.totalSteps)

// Label of the next step in the user's locale, used in the Suivant - X
// CTA on the right side of the footer. Falls back to a plain Suivant
// label when the next step has no specific i18n key.
const nextLabel = computed(() => {
  const nextKey = store.stepKeys[store.currentStep]
  if (!nextKey) return t('wizard.nextNoLabel')
  const title = t('wizard.steps.' + nextKey + '.title')
  return t('wizard.next', { nextStep: title })
})

onMounted(() => {
  if (store.loadDraft()) {
    // Only show the banner when there is an actual partial draft, not
    // the bare default state.
    if (store.type || store.cluster || store.pools.length > 0 || store.profiles.length > 0) {
      draftRestored.value = true
    }
  }
})

function showToast(msg) {
  toast.value = msg
  setTimeout(() => { toast.value = '' }, 2400)
}

function onCancel() {
  const dirty = store.type || store.cluster || store.pools.length > 0 || store.profiles.length > 0
  if (dirty && !confirm(t('wizard.cancelConfirm'))) return
  store.reset()
  router.push('/')
}

function onSaveDraft() {
  store.persistDraft()
  showToast(t('wizard.draftSaved'))
}

function onResetDraft() {
  store.reset()
  draftRestored.value = false
}

// Auto-fetch estimate when the user changes step. The summary panel
// runs its own debounced watcher, but advancing a step is a strong
// trigger that should refresh immediately.
watch(() => store.currentStep, () => {
  store.fetchEstimate()
})
</script>

<style scoped>
.wizard-shell {
  min-height: 100vh;
  background: var(--bg-canvas);
  display: flex;
  flex-direction: column;
}

.wizard-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.benchere-link {
  display: inline-flex;
  align-items: center;
  color: var(--fg-primary);
  text-decoration: none;
}

.breadcrumb {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.breadcrumb-sep { color: var(--fg-faint); }

.breadcrumb-text {
  font-size: 13px;
  color: var(--fg-secondary);
}

.draft-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 24px;
  background: var(--bg-brand-soft);
  border-bottom: 1px solid #fdba74;
  color: #9a3412;
  font-size: 13px;
}
html.dark .draft-banner {
  border-color: rgba(249, 115, 22, 0.35);
  color: #fdba74;
}

.toast {
  position: fixed;
  top: 70px;
  right: 24px;
  padding: 10px 16px;
  border-radius: 10px;
  background: #16a34a;
  color: #ffffff;
  font-size: 13px;
  z-index: 50;
  box-shadow: var(--shadow-pop);
}
html.dark .toast { background: #4ade80; color: #052e16; }

.wizard-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 240px 1fr 320px;
  min-height: 0;
}

@media (max-width: 1280px) {
  .wizard-grid {
    grid-template-columns: 1fr;
  }
}

.wizard-main {
  padding: 32px 36px;
  max-width: 900px;
  width: 100%;
}

.wizard-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 36px;
  padding-top: 20px;
  border-top: 1px solid var(--border-subtle);
}
</style>
