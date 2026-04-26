<template>
  <div class="page max-w-4xl">
    <PageHeader
      eyebrow="Nouveau benchmark"
      title="Configurer un job"
      description="Renseignez l'identité, dimensionnez les workers et choisissez le mode. Le provisioning démarre dès l'envoi du formulaire."
    />

    <!-- Onboarding -->
    <div v-if="!configured" class="alert-warn mb-6">
      <Icon name="alert" :size="18" class="mt-0.5 shrink-0" />
      <div class="flex-1">
        <p class="font-semibold">Proxmox n'est pas configuré</p>
        <p class="mt-0.5 opacity-80">
          Sans configuration valide (URL, token, node, storage), le provisioning des workers échouera.
        </p>
      </div>
      <RouterLink to="/settings" class="btn-sm btn inline-flex bg-white text-amber-800 hover:bg-amber-50 border border-amber-300 dark:bg-amber-900/30 dark:text-amber-200 dark:border-amber-700/50">
        Configurer
        <Icon name="arrow_right" :size="14" />
      </RouterLink>
    </div>

    <form @submit.prevent="submit" class="space-y-6">
      <!-- Identification -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">Identification</h2>
          <span class="card-title">Étape 1 / 4</span>
        </header>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="label">Nom du job</label>
            <input v-model="form.name" type="text" placeholder="benchmark-prod-01" class="input" required />
          </div>
          <div>
            <label class="label">Nom du client</label>
            <input v-model="form.client_name" type="text" placeholder="Acme Corp" class="input" required />
          </div>
        </div>
      </section>

      <!-- Workers -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">Workers</h2>
          <span class="card-title">Étape 2 / 4</span>
        </header>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label class="label">Nombre de workers</label>
            <input v-model.number="form.worker_count" type="number" min="1" max="20" class="input" required />
          </div>
          <div>
            <label class="label">vCPU / worker</label>
            <input v-model.number="form.worker_cpu" type="number" min="1" max="64" class="input" required />
          </div>
          <div>
            <label class="label">RAM / worker (MB)</label>
            <input v-model.number="form.worker_ram_mb" type="number" min="512" class="input" required />
          </div>
          <div>
            <label class="label">Disque OS (GB)</label>
            <input v-model.number="form.os_disk_gb" type="number" min="10" class="input" required />
          </div>
          <div>
            <label class="label">Disques data / worker</label>
            <input v-model.number="form.data_disks" type="number" min="0" max="8" class="input" />
          </div>
          <div>
            <label class="label">Taille data (GB)</label>
            <input v-model.number="form.data_disk_gb" type="number" min="1" class="input" />
          </div>
        </div>
      </section>

      <!-- Mode -->
      <section class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">Mode de benchmark</h2>
          <span class="card-title">Étape 3 / 4</span>
        </header>
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <label
            v-for="mode in modes"
            :key="mode.value"
            :class="[
              'group cursor-pointer rounded-xl border p-4 transition-all duration-150',
              form.mode === mode.value
                ? 'border-brand-500 bg-brand-50 dark:bg-brand-500/10 shadow-brand'
                : 'border-default hover:border-strong hover:bg-soft'
            ]"
          >
            <input type="radio" v-model="form.mode" :value="mode.value" class="sr-only" />
            <div class="flex items-start gap-3">
              <span
                :class="[
                  'w-9 h-9 rounded-lg flex items-center justify-center shrink-0',
                  form.mode === mode.value
                    ? 'bg-brand-500 text-white'
                    : 'bg-soft fg-secondary'
                ]"
              >
                <Icon :name="mode.icon" :size="18" />
              </span>
              <div class="min-w-0">
                <p class="text-sm font-semibold fg-primary">{{ mode.label }}</p>
                <p class="text-xs fg-muted mt-0.5">{{ mode.hint }}</p>
              </div>
            </div>
          </label>
        </div>
      </section>

      <!-- Profiles -->
      <section v-if="form.mode !== 'cpu'" class="card space-y-4">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">Profils elbencho</h2>
          <span class="card-title">Étape 4 / 4 · Stockage</span>
        </header>
        <div v-if="profiles.length === 0" class="text-sm fg-muted">Chargement des profils…</div>
        <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-2">
          <label
            v-for="p in profiles"
            :key="p.id"
            :class="[
              'flex items-start gap-3 cursor-pointer rounded-lg border p-3 transition-colors',
              form.profiles.includes(p.name)
                ? 'border-brand-500 bg-brand-50 dark:bg-brand-500/10'
                : 'border-default hover:border-strong'
            ]"
          >
            <input
              type="checkbox"
              :value="p.name"
              v-model="form.profiles"
              class="sr-only peer"
            />
            <span
              :class="[
                'w-4 h-4 rounded flex items-center justify-center shrink-0 mt-0.5 transition-all',
                form.profiles.includes(p.name)
                  ? 'bg-brand-500 text-white'
                  : 'border border-default bg-elevated'
              ]"
            >
              <Icon v-if="form.profiles.includes(p.name)" name="check" :size="11" stroke-width="3" />
            </span>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium fg-primary truncate">{{ p.name }}</p>
              <p v-if="p.description" class="text-xs fg-muted truncate">{{ p.description }}</p>
            </div>
          </label>
        </div>
      </section>

      <!-- stress-ng -->
      <section v-if="form.mode !== 'storage'" class="card space-y-5">
        <header class="flex items-center justify-between">
          <h2 class="text-sm font-semibold fg-primary">Paramètres stress-ng</h2>
          <span class="card-title">Étape 4 / 4 · CPU</span>
        </header>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="label">Workers stress-ng</label>
            <input v-model.number="form.stress_config.workers" type="number" min="1" class="input" />
          </div>
          <div>
            <label class="label">Timeout (secondes)</label>
            <input v-model.number="form.stress_config.timeout" type="number" min="10" class="input" />
          </div>
        </div>
        <div>
          <label class="label">Stressors</label>
          <div class="flex flex-wrap gap-2 mt-1">
            <label
              v-for="s in stressors"
              :key="s"
              :class="[
                'cursor-pointer text-xs px-3 h-8 inline-flex items-center gap-2 rounded-md border transition-colors',
                form.stress_config.stressors.includes(s)
                  ? 'border-brand-500 bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
                  : 'border-default fg-secondary hover:border-strong'
              ]"
            >
              <input type="checkbox" :value="s" v-model="form.stress_config.stressors" class="sr-only" />
              <Icon
                v-if="form.stress_config.stressors.includes(s)"
                name="check" :size="12" stroke-width="3"
              />
              <span class="font-mono">{{ s }}</span>
            </label>
          </div>
        </div>
      </section>

      <div v-if="error" class="alert-error">
        <Icon name="x_circle" :size="18" class="mt-0.5 shrink-0" />
        <span>{{ error }}</span>
      </div>

      <div class="flex items-center justify-between gap-3 pt-2">
        <RouterLink to="/" class="btn-ghost">Annuler</RouterLink>
        <button type="submit" class="btn-primary btn-lg" :disabled="submitting">
          <Spinner v-if="submitting" :size="16" />
          <Icon v-else name="play" :size="16" />
          {{ submitting ? 'Provisionnement…' : 'Provisionner et lancer' }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRouter, RouterLink } from 'vue-router'
import { useJobsStore } from '../stores/jobs.js'
import { useSettingsStore } from '../stores/settings.js'
import { api } from '../api/client.js'
import PageHeader from '../components/PageHeader.vue'
import Icon       from '../components/Icon.vue'
import Spinner    from '../components/Spinner.vue'

const router        = useRouter()
const jobsStore     = useJobsStore()
const settingsStore = useSettingsStore()

const settings   = ref(null)
const configured = computed(() => !!settings.value?.proxmox_url)

const modes = [
  { value: 'storage', icon: 'hard_drive', label: 'Stockage', hint: 'IOPS, débit, latence (elbencho)' },
  { value: 'cpu',     icon: 'cpu',        label: 'CPU',      hint: 'Charge CPU pure (stress-ng)' },
  { value: 'mixed',   icon: 'shuffle',    label: 'Mixte',    hint: 'Stockage + CPU en parallèle' },
]

const stressors = ['cpu', 'vm', 'io', 'hdd']

const profiles   = ref([])
const submitting = ref(false)
const error      = ref('')

const form = reactive({
  name:          '',
  client_name:   '',
  mode:          'storage',
  worker_count:  3,
  worker_cpu:    4,
  worker_ram_mb: 4096,
  os_disk_gb:    20,
  data_disks:    1,
  data_disk_gb:  50,
  profiles:      [],
  stress_config: { workers: 4, timeout: 120, stressors: ['cpu'] },
})

watch(() => form.mode, () => { error.value = '' })

onMounted(async () => {
  try { settings.value = await settingsStore.load() } catch (_) {}
  try { profiles.value = await api.listProfiles() ?? [] } catch (_) {}
})

async function submit() {
  error.value = ''
  if (form.mode !== 'cpu' && form.profiles.length === 0) {
    error.value = 'Sélectionnez au moins un profil elbencho.'
    return
  }
  const payload = {
    name:          form.name,
    client_name:   form.client_name,
    mode:          form.mode,
    worker_count:  form.worker_count,
    worker_cpu:    form.worker_cpu,
    worker_ram_mb: form.worker_ram_mb,
    os_disk_gb:    form.os_disk_gb,
    data_disks:    form.data_disks,
    data_disk_gb:  form.data_disk_gb,
    profiles:      form.mode === 'cpu' ? [] : form.profiles,
    stress_config: form.mode === 'storage' ? null : {
      ...form.stress_config,
      stressors: [...form.stress_config.stressors],
    },
  }
  submitting.value = true
  try {
    const id = await jobsStore.createJob(payload)
    router.push(`/dashboard/${id}`)
  } catch (e) {
    error.value = 'Erreur : ' + e.message
    submitting.value = false
  }
}
</script>
