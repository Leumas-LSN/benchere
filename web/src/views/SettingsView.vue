<template>
  <div class="page max-w-3xl">
    <PageHeader
      :eyebrow="t('settings.title')"
      :title="t('settings.title')"
      :description="t('settings.description')"
    />

    <form @submit.prevent="save" class="space-y-6">
      <!-- Connection -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">{{ t('settings.sections.connection') }}</h2>
          <span class="card-title">{{ t('settings.sections.connectionRight') }}</span>
        </header>
        <div>
          <label class="label">{{ t('settings.fields.apiUrl') }}</label>
          <input v-model="form.proxmox_url" type="url" placeholder="https://pve.example.com:8006" class="input" />
          <p class="helper">{{ t('settings.fields.apiUrlHint') }}</p>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="label">{{ t('settings.fields.tokenId') }}</label>
            <input v-model="form.proxmox_token_id" type="text" placeholder="root@pam!benchere" class="input" autocomplete="off" />
          </div>
          <div>
            <label class="label">{{ t('settings.fields.tokenSecret') }}</label>
            <div class="relative">
              <input
                v-model="form.proxmox_token_secret"
                :type="showSecret ? 'text' : 'password'"
                placeholder="••••••••-••••-••••-••••-••••••••••••"
                class="input pr-10"
                autocomplete="off"
              />
              <button
                type="button"
                class="absolute right-2 top-1/2 -translate-y-1/2 w-7 h-7 inline-flex items-center justify-center rounded fg-muted hover:fg-primary"
                :aria-label="showSecret ? 'Masquer' : 'Afficher'"
                @click="showSecret = !showSecret"
              >
                <Icon :name="showSecret ? 'eye_off' : 'eye'" :size="14" />
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Topology -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">{{ t('settings.sections.topology') }}</h2>
          <span class="card-title">{{ t('settings.sections.topologyRight') }}</span>
        </header>
        <div>
          <label class="label">{{ t('settings.fields.node') }}</label>
          <input v-model="form.proxmox_node" type="text" placeholder="pve-01" class="input" />
          <p class="helper">{{ t('settings.fields.nodeHint') }}</p>
        </div>
        <div>
          <label class="label">{{ t('settings.fields.imageStorage') }}</label>
          <p class="helper -mt-1 mb-2">
            {{ t('settings.fields.imageStorageHint') }}
            <span class="code-inline">dir</span>.
          </p>
          <select v-if="storages.length" v-model="form.image_storage" class="select">
            <option value="">{{ t('settings.actions.defaultLocal') }}</option>
            <option v-for="s in storages" :key="s.id" :value="s.id">
              {{ s.id }} ({{ s.type }})
            </option>
          </select>
          <input
            v-else
            v-model="form.image_storage"
            type="text"
            placeholder="local"
            class="input"
          />
        </div>

        <div>
          <label class="label">{{ t('settings.fields.bridge') }}</label>
          <p class="helper -mt-1 mb-2">
            {{ t('settings.fields.bridgeHint') }}
          </p>
          <div class="flex gap-2">
            <select v-if="bridges.length" v-model="form.network_bridge" class="select flex-1">
              <option value="">{{ t('settings.actions.pickBridge') }}</option>
              <option v-for="b in bridges" :key="b.Name" :value="b.Name" :disabled="!b.Active">
                {{ b.Name }}{{ b.Address ? " (" + b.Address + ")" : "" }}{{ !b.Active ? " — inactif" : "" }}
              </option>
            </select>
            <input
              v-else
              v-model="form.network_bridge"
              type="text"
              placeholder="vmbr0"
              class="input flex-1"
            />
            <button
              type="button"
              class="btn-secondary whitespace-nowrap"
              :disabled="scanningBridges"
              @click="scanBridges"
            >
              <Spinner v-if="scanningBridges" :size="14" />
              <Icon v-else name="search" :size="14" />
              {{ t('settings.actions.scan') }}
            </button>
          </div>
          <p v-if="bridgesError" class="helper text-red-600 dark:text-red-400 mt-2">{{ bridgesError }}</p>
        </div>
      </section>

      <!-- Worker network -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">{{ t('settings.sections.workerNetwork') }}</h2>
          <span class="card-title">{{ t('settings.sections.workerNetworkRight') }}</span>
        </header>
        <p class="helper -mt-2">
          {{ t('settings.fields.workerNetworkLead') }} <span class="code-inline">qemu-guest-agent</span> 
        </p>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="label">{{ t('settings.fields.ipPool') }}</label>
            <input v-model="form.worker_ip_pool" type="text" placeholder="10.90.0.200-10.90.0.220" class="input" />
            <p class="helper">{{ t('settings.fields.ipPoolHint') }}</p>
          </div>
          <div>
            <label class="label">{{ t('settings.fields.cidr') }}</label>
            <input v-model="form.worker_cidr" type="text" inputmode="numeric" pattern="[0-9]{1,2}" placeholder="24" class="input" />
            <p class="helper">{{ t('settings.fields.cidrHint') }}</p>
          </div>
        </div>
        <div>
          <label class="label">{{ t('settings.fields.gateway') }}</label>
          <input v-model="form.worker_gateway" type="text" placeholder="10.90.0.1" class="input" />
          <p class="helper">{{ t('settings.fields.gatewayHint') }}</p>
        </div>
      </section>

      <!-- SSH -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">{{ t('settings.sections.ssh') }}</h2>
          <span class="card-title">{{ t('settings.sections.sshRight') }}</span>
        </header>
        <div>
          <label class="label">{{ t('settings.fields.sshKeyPath') }}</label>
          <input v-model="form.ssh_key_path" type="text" placeholder="/opt/benchere/id_rsa" class="input" />
          <p class="helper">{{ t('settings.fields.sshKeyPathHint') }}</p>
        </div>
      </section>

      <!-- Backends toggle section -->
      <section class="card space-y-3">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">{{ t('settings.sections.engines') }}</h2>
        </header>
        <label class="flex items-start gap-3 cursor-pointer">
          <input type="checkbox" v-model="form.enable_legacy_backends" class="mt-1 form-check accent-brand-500" />
          <div class="flex-1">
            <p class="text-sm fg-primary font-medium">{{ t('settings.fields.enableLegacy') }}</p>
            <p class="helper">{{ t('settings.fields.enableLegacyHint') }}</p>
          </div>
        </label>
      </section>

      <!-- Actions + feedback -->
      <div class="flex flex-col-reverse sm:flex-row sm:items-center sm:justify-between gap-3">
        <button type="button" class="btn-secondary" :disabled="testing" @click="testConnection">
          <Spinner v-if="testing" :size="14" />
          <Icon v-else name="shield" :size="14" />
          {{ t('settings.actions.testConnection') }}
        </button>
        <button type="submit" class="btn-primary" :disabled="saving">
          <Spinner v-if="saving" :size="14" />
          <Icon v-else name="check" :size="14" />
          {{ t('common.save') }}
        </button>
      </div>

      <div v-if="message" :class="messageClass">
        <Icon :name="isError ? 'x_circle' : 'check_circle'" :size="18" class="mt-0.5 shrink-0" />
        <span>{{ message }}</span>
      </div>
    </form>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { ref, reactive, computed, onMounted } from 'vue'
import { useSettingsStore } from '../stores/settings.js'
import PageHeader from '../components/PageHeader.vue'
import Icon       from '../components/Icon.vue'
import Spinner    from '../components/Spinner.vue'

const { t } = useI18n()

const settingsStore = useSettingsStore()

const form = reactive({
  proxmox_url:          '',
  proxmox_token_id:     '',
  proxmox_token_secret: '',
  proxmox_node:         '',
  image_storage:    '',
  network_bridge:   '',
  worker_ip_pool:   '',
  worker_cidr:      '24',
  worker_gateway:   '',
  ssh_key_path:          '/opt/benchere/id_rsa',
  enable_legacy_backends: false,
})

const saving     = ref(false)
const testing    = ref(false)
const scanning   = ref(false)
const showSecret = ref(false)
const message    = ref('')
const isError    = ref(false)
const storages   = ref([])
const scanError  = ref('')
const bridges    = ref([])
const scanningBridges = ref(false)
const bridgesError = ref('')

const messageClass = computed(() => isError.value ? 'alert-error' : 'alert-success')

onMounted(async () => {
  try {
    const s = await settingsStore.load()
    if (s) {
      Object.assign(form, s)
      form.enable_legacy_backends = s.enable_legacy_backends === true || s.enable_legacy_backends === 'true'
    }
  } catch (_) { /* first visit */ }
})

async function save() {
  saving.value = true
  message.value = ''
  try {
    await settingsStore.save({ ...form })
    message.value = 'Paramètres sauvegardés.'
    isError.value = false
  } catch (e) {
    message.value = 'Erreur : ' + e.message
    isError.value = true
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  testing.value = true
  message.value = ''
  try {
    const result = await settingsStore.test()
    if (result.ok) {
      message.value = `Connexion OK — nodes : ${result.nodes.join(', ')}`
      isError.value = false
    } else {
      message.value = 'Connexion échouée : ' + result.error
      isError.value = true
    }
  } catch (e) {
    message.value = 'Connexion échouée : ' + e.message
    isError.value = true
  } finally {
    testing.value = false
  }
}

async function scanBridges() {
  scanningBridges.value = true
  bridgesError.value = ""
  try {
    bridges.value = await settingsStore.scanBridges()
    if (!bridges.value.length) bridgesError.value = t("settings.feedback.noBridges")
  } catch (e) {
    bridgesError.value = "Erreur : " + e.message
  } finally {
    scanningBridges.value = false
  }
}

async function scanStorages() {
  scanning.value = true
  scanError.value = ''
  try {
    storages.value = await settingsStore.scanStorages()
    if (!storages.value.length) scanError.value = t('settings.feedback.noStorages')
  } catch (e) {
    scanError.value = 'Erreur : ' + e.message
  } finally {
    scanning.value = false
  }
}
</script>
