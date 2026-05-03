<template>
  <div class="space-y-6">
    <header>
      <p class="section-eyebrow mb-1">{{ stepEyebrow }}</p>
      <h2 class="step-title">{{ t('wizard.steps.cluster.title') }}</h2>
      <p class="step-subtitle">{{ t('wizard.steps.cluster.subtitle') }}</p>
    </header>

    <div v-if="loading" class="text-sm fg-muted">{{ t('common.loading') }}</div>
    <div v-else-if="!store.cluster" class="alert-warn">
      <Icon name="alert" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1">
        <p class="font-semibold">{{ t('wizard.steps.cluster.empty') }}</p>
      </div>
      <RouterLink to="/settings" class="btn-sm btn inline-flex bg-white text-amber-800 hover:bg-amber-50 border border-amber-300 dark:bg-amber-900/30 dark:text-amber-200 dark:border-amber-700/50">
        {{ t('wizard.steps.cluster.configureCta') }}
        <Icon name="arrow_right" :size="14" />
      </RouterLink>
    </div>

    <div v-else class="cluster-card">
      <span class="cluster-icon">
        <Icon name="server" :size="22" />
      </span>
      <div class="min-w-0 flex-1">
        <p class="text-sm fg-muted">{{ t('wizard.steps.cluster.autoSelected') }}</p>
        <p class="text-lg font-semibold num fg-primary mt-0.5">{{ store.cluster }}</p>
      </div>
      <span class="pill-active">
        <Icon name="check" :size="13" stroke-width="3" />
      </span>
    </div>

    <p v-if="store.cluster" class="text-xs fg-muted">{{ t('wizard.steps.cluster.v2Hint') }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import { useWizardStore } from '../../stores/wizard.js'
import { useSettingsStore } from '../../stores/settings.js'
import Icon from '../Icon.vue'

const { t } = useI18n()
const store = useWizardStore()
const settingsStore = useSettingsStore()
const loading = ref(true)

const stepEyebrow = computed(() =>
  t('wizard.steps.eyebrow', { current: store.currentStep, total: store.totalSteps }),
)

onMounted(async () => {
  if (store.cluster) {
    loading.value = false
    return
  }
  try {
    const settings = await settingsStore.load()
    if (settings?.cluster_name) {
      store.cluster = settings.cluster_name
    } else if (settings?.proxmox_url) {
      // Fall back to a sanitized URL host so the user has SOMETHING
      // displayed even when they did not name the cluster.
      try {
        const u = new URL(settings.proxmox_url)
        store.cluster = u.hostname
      } catch (_) {
        store.cluster = settings.proxmox_url
      }
    }
  } catch (_) { /* leave cluster empty so the warning shows */ }
  loading.value = false
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

.cluster-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 18px 18px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-default);
  border-radius: 12px;
}

.cluster-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 10px;
  background: var(--bg-brand-soft);
  color: #f97316;
  flex-shrink: 0;
}

.pill-active {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #16a34a;
  color: #ffffff;
}
html.dark .pill-active { background: #4ade80; color: #052e16; }
</style>
