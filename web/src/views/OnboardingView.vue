<template>
  <div class="page max-w-3xl">
    <header class="mb-8">
      <BenchereWordmark class="text-2xl mb-3" />
      <h1 class="text-2xl font-semibold fg-primary">{{ t('onboarding.title') }}</h1>
      <p class="fg-muted mt-1">{{ t('onboarding.subtitle') }}</p>
    </header>

    <!-- Progress -->
    <div class="flex items-center gap-2 mb-8">
      <div
        v-for="i in totalSteps"
        :key="i"
        :class="[
          'h-1 flex-1 rounded-full transition-all duration-300',
          i <= step ? 'bg-brand-500' : 'bg-soft'
        ]"
      ></div>
    </div>
    <p class="text-xs fg-muted mb-4 num">
      {{ t('onboarding.step', { n: step, total: totalSteps }) }}
    </p>

    <!-- STEP 1: Language -->
    <section v-if="step === 1" class="card space-y-6">
      <header>
        <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.languageStep.title') }}</h2>
        <p class="helper mt-1">{{ t('onboarding.languageStep.hint') }}</p>
      </header>
      <div class="grid grid-cols-2 gap-3">
        <button
          v-for="lang in languages"
          :key="lang.code"
          type="button"
          :class="[
            'rounded-xl border p-5 text-left transition-all',
            locale === lang.code
              ? 'border-brand-500 bg-brand-50 dark:bg-brand-500/10 shadow-brand'
              : 'border-default hover:border-strong'
          ]"
          @click="pickLanguage(lang.code)"
        >
          <div class="text-3xl mb-2">{{ lang.flag }}</div>
          <p class="font-semibold fg-primary">{{ lang.label }}</p>
          <p class="text-xs fg-muted">{{ lang.subtitle }}</p>
        </button>
      </div>
    </section>

    <!-- STEP 2: Hypervisor -->
    <section v-if="step === 2" class="card space-y-6">
      <header>
        <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.hypervisorStep.title') }}</h2>
        <p class="helper mt-1">{{ t('onboarding.hypervisorStep.hint') }}</p>
      </header>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <button
          v-for="h in hypervisors"
          :key="h.id"
          type="button"
          :disabled="!h.enabled"
          :class="[
            'relative rounded-xl border p-5 text-left transition-all',
            h.enabled
              ? (form.hypervisor === h.id
                  ? 'border-brand-500 bg-brand-50 dark:bg-brand-500/10 shadow-brand cursor-pointer'
                  : 'border-default hover:border-strong cursor-pointer')
              : 'border-default opacity-50 cursor-not-allowed'
          ]"
          @click="h.enabled && (form.hypervisor = h.id)"
        >
          <span
            v-if="!h.enabled"
            class="absolute top-2 right-2 text-[10px] px-2 py-0.5 rounded-full bg-soft fg-muted uppercase tracking-wider font-semibold"
          >{{ t('common.comingSoon') }}</span>
          <div class="flex items-start gap-4">
            <div class="w-12 h-12 rounded-lg flex items-center justify-center shrink-0 overflow-hidden bg-white">
              <img :src="h.logoSrc" :alt="t(h.labelKey)" class="w-10 h-10 object-contain" />
            </div>
            <div class="min-w-0 flex-1">
              <p class="font-semibold fg-primary">{{ t(h.labelKey) }}</p>
              <p class="text-xs fg-muted mt-1">{{ h.subtitle }}</p>
            </div>
          </div>
        </button>
      </div>
    </section>

    <!-- STEP 3: Cluster connection -->
    <section v-if="step === 3" class="card space-y-5">
      <header>
        <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.clusterStep.title') }}</h2>
        <p class="helper mt-1">{{ t('onboarding.clusterStep.hint') }}</p>
      </header>
      <div>
        <label class="label">{{ t('onboarding.clusterStep.apiUrl') }}</label>
        <input v-model="form.proxmox_url" type="url" placeholder="https://pve.example.com:8006" class="input" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="label">{{ t('onboarding.clusterStep.tokenId') }}</label>
          <input v-model="form.proxmox_token_id" type="text" placeholder="root@pam!benchere" class="input" autocomplete="off" />
        </div>
        <div>
          <label class="label">{{ t('onboarding.clusterStep.tokenSecret') }}</label>
          <input v-model="form.proxmox_token_secret" type="password" placeholder="••••••••-••••-••••-••••-••••••••••••" class="input" autocomplete="off" />
        </div>
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="label">{{ t('onboarding.clusterStep.node') }}</label>
          <input v-model="form.proxmox_node" type="text" placeholder="pve-01" class="input" />
          <p class="helper">{{ t('onboarding.clusterStep.nodeHint') }}</p>
        </div>
        <div>
          <label class="label">{{ t('onboarding.clusterStep.clusterName') }}</label>
          <input v-model="form.cluster_name" type="text" placeholder="prod-paris" class="input" />
          <p class="helper">{{ t('onboarding.clusterStep.clusterNameHint') }}</p>
        </div>
      </div>
    </section>

    <!-- STEP 4: Network -->
    <section v-if="step === 4" class="card space-y-5">
      <header>
        <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.networkStep.title') }}</h2>
        <p class="helper mt-1">{{ t('onboarding.networkStep.hint') }}</p>
      </header>
      <div>
        <label class="label">{{ t('onboarding.networkStep.bridge') }}</label>
        <input v-model="form.network_bridge" type="text" placeholder="vmbr0" class="input" />
      </div>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="label">{{ t('onboarding.networkStep.ipPool') }}</label>
          <input v-model="form.worker_ip_pool" type="text" placeholder="10.90.0.200-10.90.0.220" class="input" />
          <p class="helper">{{ t('onboarding.networkStep.ipPoolHint') }}</p>
        </div>
        <div>
          <label class="label">{{ t('onboarding.networkStep.cidr') }}</label>
          <input v-model="form.worker_cidr" type="text" inputmode="numeric" pattern="[0-9]{1,2}" placeholder="24" class="input" />
        </div>
      </div>
      <div>
        <label class="label">{{ t('onboarding.networkStep.gateway') }}</label>
        <input v-model="form.worker_gateway" type="text" placeholder="10.90.0.1" class="input" />
      </div>
    </section>

    <!-- STEP 5: SSH -->
    <section v-if="step === 5" class="card space-y-5">
      <header>
        <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.sshStep.title') }}</h2>
        <p class="helper mt-1">{{ t('onboarding.sshStep.hint') }}</p>
      </header>
      <div>
        <label class="label">{{ t('onboarding.sshStep.keyPath') }}</label>
        <input v-model="form.ssh_key_path" type="text" placeholder="/opt/benchere/id_rsa" class="input" />
      </div>
    </section>

    <!-- STEP 6: Done -->
    <section v-if="step === 6" class="card space-y-5 text-center py-10">
      <div class="w-16 h-16 mx-auto rounded-full bg-emerald-100 dark:bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 flex items-center justify-center">
        <Icon name="check" :size="28" stroke-width="3" />
      </div>
      <h2 class="text-lg font-semibold fg-primary">{{ t('onboarding.done.title') }}</h2>
      <p class="fg-muted">{{ t('onboarding.done.hint') }}</p>
      <RouterLink to="/" class="btn-primary inline-flex">{{ t('onboarding.done.cta') }}</RouterLink>
    </section>

    <!-- Error -->
    <div v-if="error" class="alert-error mt-4">
      <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
      <span>{{ error }}</span>
    </div>

    <!-- Navigation -->
    <div v-if="step < 6" class="flex items-center justify-between mt-6">
      <button
        type="button"
        class="btn-ghost"
        :disabled="step === 1"
        @click="step--"
      >
        <Icon name="arrow_left" :size="14" />
        {{ t('common.previous') }}
      </button>
      <button
        type="button"
        class="btn-primary"
        :disabled="!canAdvance || saving"
        @click="advance"
      >
        <Spinner v-if="saving" :size="14" />
        <span>{{ step === 5 ? t('common.finish') : t('common.next') }}</span>
        <Icon v-if="!saving" name="arrow_right" :size="14" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, RouterLink } from 'vue-router'
import { setLocale } from '../i18n/index.js'
import { useSettingsStore } from '../stores/settings.js'
import BenchereWordmark from '../components/BenchereWordmark.vue'
import Icon    from '../components/Icon.vue'
import Spinner from '../components/Spinner.vue'

const { t, locale } = useI18n()
const router = useRouter()
const settingsStore = useSettingsStore()

const totalSteps = 6
const step = ref(1)
const error = ref('')
const saving = ref(false)

const languages = [
  { code: 'fr', flag: '🇫🇷', label: 'Français',  subtitle: 'France, Belgique, Québec…' },
  { code: 'en', flag: '🇬🇧', label: 'English',  subtitle: 'United States, UK, global…' },
]

const hypervisors = [
  { id: 'proxmox',    labelKey: 'onboarding.hypervisorStep.proxmox',    subtitle: 'Proxmox VE 8.x and 9.x', logoSrc: '/hypervisor-logos/proxmox.png',     enabled: true  },
  { id: 'vsphere',    labelKey: 'onboarding.hypervisorStep.vsphere',    subtitle: 'VMware ESXi / vSphere',  logoSrc: '/hypervisor-logos/vsphere.jpg',     enabled: false },
  { id: 'hyperv',     labelKey: 'onboarding.hypervisorStep.hyperv',     subtitle: 'Microsoft Hyper-V',      logoSrc: '/hypervisor-logos/hyperv.png',      enabled: false },
  { id: 'azureLocal', labelKey: 'onboarding.hypervisorStep.azureLocal', subtitle: 'Azure Stack HCI',        logoSrc: '/hypervisor-logos/azure-local.png', enabled: false },
]

const form = reactive({
  hypervisor: 'proxmox',
  proxmox_url: '',
  proxmox_token_id: '',
  proxmox_token_secret: '',
  proxmox_node: '',
  cluster_name: '',
  network_bridge: '',
  worker_ip_pool: '',
  worker_cidr: '24',
  worker_gateway: '',
  ssh_key_path: '/opt/benchere/id_rsa',
})

const canAdvance = computed(() => {
  if (step.value === 1) return !!locale.value
  if (step.value === 2) return form.hypervisor === 'proxmox'
  if (step.value === 3) return form.proxmox_url && form.proxmox_token_id && form.proxmox_token_secret && form.proxmox_node
  if (step.value === 4) return form.network_bridge && form.worker_ip_pool && form.worker_cidr && form.worker_gateway
  if (step.value === 5) return !!form.ssh_key_path
  return true
})

function pickLanguage(code) {
  setLocale(code)
}

async function advance() {
  error.value = ''
  if (step.value < 5) {
    step.value++
    return
  }
  // Last step: persist all settings
  saving.value = true
  try {
    await settingsStore.save({
      proxmox_url:           form.proxmox_url,
      proxmox_token_id:      form.proxmox_token_id,
      proxmox_token_secret:  form.proxmox_token_secret,
      proxmox_node:          form.proxmox_node,
      cluster_name:          form.cluster_name,
      network_bridge:        form.network_bridge,
      worker_ip_pool:        form.worker_ip_pool,
      worker_cidr:           form.worker_cidr,
      worker_gateway:        form.worker_gateway,
      ssh_key_path:          form.ssh_key_path,
    })
    step.value = 6
  } catch (e) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}
</script>
