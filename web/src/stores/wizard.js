// Wizard store: drives the multi-step New Job flow.
//
// Steps adapt to the selected mode:
//   - storage: 7 steps (type, cluster, pool, profiles, workers, range, review)
//   - cpu:     5 steps (type, cluster, workers, cpuConfig, review)
//   - mixed:   8 steps (type, cluster, pool, profiles, cpuConfig, workers, range, review)
//
// Drafts are persisted to localStorage on every state change so users
// who navigate away or refresh do not lose their work. The draft is
// scoped per browser, NOT per server, since v1 has no auth.
import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { api } from '../api/client.js'

const STORAGE_KEY = 'benchere.wizard.draft'

// Step keys appear in the order they are presented for each mode.
// The first key 'type' is shared and lets the user pick which mode
// branches the wizard, before the mode-specific keys come in.
const STEP_DEFS = {
  storage: ['type', 'cluster', 'pool', 'profiles', 'workers', 'range', 'review'],
  cpu:     ['type', 'cluster', 'workers', 'cpuConfig', 'review'],
  mixed:   ['type', 'cluster', 'pool', 'profiles', 'cpuConfig', 'workers', 'range', 'review'],
}

function defaultWorkers() {
  return {
    auto: true,
    count: 4,
    perNode: 0,
    nodes: [],
    cpu: 4,
    ramMb: 4096,
    osDiskGb: 20,
    dataDisks: 1,
    dataDiskGb: 32,
  }
}

function defaultRange() {
  return {
    runtimeSec: 120,
    rampSec: 30,
    ioDepth: 32,
    thresholdsCustom: false,
    thresholds: {
      minIopsRead: 0,
      minIopsWrite: 0,
      maxAvgLatencyMs: 0,
      maxP99LatencyMs: 0,
    },
  }
}

function defaultCpuConfig() {
  return {
    stressors: ['cpu'],
    stressorWorkers: 4,
    timeoutSec: 60,
  }
}

export const useWizardStore = defineStore('wizard', () => {
  const type = ref('')
  const cluster = ref('')
  const pool = ref('')
  const profiles = ref([])
  const workers = ref(defaultWorkers())
  const range = ref(defaultRange())
  const cpuConfig = ref(defaultCpuConfig())
  const currentStep = ref(1)
  const estimate = ref({ wallclockSec: 0, bytesWritten: 0 })
  const meta = ref({ name: '', clientName: '' })

  // The active step key list. Falls back to the storage list before the
  // user picks a type so the sidebar can render placeholders.
  const stepKeys = computed(() => {
    const t = type.value || 'storage'
    return STEP_DEFS[t] || STEP_DEFS.storage
  })

  const totalSteps = computed(() => stepKeys.value.length)

  function stepKeyAt(index) {
    return stepKeys.value[index - 1] || ''
  }

  // Per-step validity. Index = step number, value = true when the user
  // can advance. The wizard host reads these to enable the Next button
  // and to gate goTo() jumps.
  const stepValid = computed(() => {
    return stepKeys.value.map((key) => {
      switch (key) {
        case 'type': return !!type.value
        case 'cluster': return !!cluster.value
        case 'pool': return !!pool.value
        case 'profiles': return profiles.value.length > 0
        case 'workers':
          return workers.value.count > 0 && workers.value.nodes.length > 0
        case 'range':
          return range.value.runtimeSec > 0 && range.value.rampSec >= 0 && range.value.ioDepth > 0
        case 'cpuConfig':
          return cpuConfig.value.stressors.length > 0 && cpuConfig.value.timeoutSec > 0
        case 'review':
          return true
        default:
          return true
      }
    })
  })

  // Sublabel shown in the sidebar under each step row. Short, mono,
  // surfaces the locked-in value once the step is past.
  function sublabelFor(stepKey) {
    switch (stepKey) {
      case 'type':
        return type.value || ''
      case 'cluster':
        return cluster.value || ''
      case 'pool':
        return pool.value || ''
      case 'profiles':
        if (!profiles.value.length) return ''
        return profiles.value.length === 1
          ? '1 profil'
          : profiles.value.length + ' profils'
      case 'workers': {
        if (!workers.value.count) return ''
        const head = workers.value.auto ? 'auto' : 'manuel'
        return head + ' . ' + workers.value.count + ' workers'
      }
      case 'range':
        return range.value.thresholdsCustom ? 'personnalises' : 'defauts'
      case 'cpuConfig':
        if (!cpuConfig.value.stressors.length) return ''
        return cpuConfig.value.stressors.join(', ')
      default:
        return ''
    }
  }

  // Navigation. Forward motion requires the current step to be valid.
  // Backward motion is always allowed. goTo allows arbitrary jumps but
  // every step prior to the destination must already be valid (you
  // cannot skip past a missing required value).
  function next() {
    if (!stepValid.value[currentStep.value - 1]) return false
    if (currentStep.value < totalSteps.value) {
      currentStep.value++
      return true
    }
    return false
  }
  function previous() {
    if (currentStep.value > 1) {
      currentStep.value--
      return true
    }
    return false
  }
  function goTo(stepIndex) {
    if (stepIndex < 1 || stepIndex > totalSteps.value) return false
    if (stepIndex <= currentStep.value) {
      currentStep.value = stepIndex
      return true
    }
    // Forward jump: every step before the destination must be valid.
    for (let i = 1; i < stepIndex; i++) {
      if (!stepValid.value[i - 1]) return false
    }
    currentStep.value = stepIndex
    return true
  }

  function reset() {
    type.value = ''
    cluster.value = ''
    pool.value = ''
    profiles.value = []
    workers.value = defaultWorkers()
    range.value = defaultRange()
    cpuConfig.value = defaultCpuConfig()
    currentStep.value = 1
    estimate.value = { wallclockSec: 0, bytesWritten: 0 }
    meta.value = { name: '', clientName: '' }
    clearDraft()
  }

  function persistDraft() {
    try {
      const payload = {
        type: type.value,
        cluster: cluster.value,
        pool: pool.value,
        profiles: profiles.value,
        workers: workers.value,
        range: range.value,
        cpuConfig: cpuConfig.value,
        currentStep: currentStep.value,
        meta: meta.value,
      }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
    } catch (_) {
      // localStorage may throw in private browsing or when full; the
      // draft is a convenience, not data integrity, so swallow.
    }
  }

  function loadDraft() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) return false
      const data = JSON.parse(raw)
      if (!data || typeof data !== 'object') return false
      if (typeof data.type === 'string') type.value = data.type
      if (typeof data.cluster === 'string') cluster.value = data.cluster
      if (typeof data.pool === 'string') pool.value = data.pool
      if (Array.isArray(data.profiles)) profiles.value = data.profiles.slice()
      if (data.workers && typeof data.workers === 'object') {
        workers.value = { ...defaultWorkers(), ...data.workers }
      }
      if (data.range && typeof data.range === 'object') {
        range.value = {
          ...defaultRange(),
          ...data.range,
          thresholds: { ...defaultRange().thresholds, ...(data.range.thresholds || {}) },
        }
      }
      if (data.cpuConfig && typeof data.cpuConfig === 'object') {
        cpuConfig.value = { ...defaultCpuConfig(), ...data.cpuConfig }
      }
      if (typeof data.currentStep === 'number' && data.currentStep >= 1) {
        currentStep.value = Math.min(data.currentStep, totalSteps.value)
      }
      if (data.meta && typeof data.meta === 'object') {
        meta.value = { ...meta.value, ...data.meta }
      }
      return true
    } catch (_) {
      return false
    }
  }

  function clearDraft() {
    try { localStorage.removeItem(STORAGE_KEY) } catch (_) {}
  }

  // Hits POST /api/jobs/estimate with the current state. Quietly does
  // nothing when the wizard is too empty to estimate, so callers can
  // fire-and-forget on every step change.
  async function fetchEstimate() {
    const payload = {
      mode: type.value || 'storage',
      workers: workers.value.count || 1,
      data_disks: workers.value.dataDisks || 0,
      data_disk_gb: workers.value.dataDiskGb || 0,
      profiles: profiles.value.slice(),
      runtime_sec: range.value.runtimeSec || 0,
      ramp_sec: range.value.rampSec || 0,
      stress_timeout_sec: cpuConfig.value.timeoutSec || 0,
    }
    try {
      const res = await fetch('/api/jobs/estimate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) return null
      const data = await res.json()
      estimate.value = {
        wallclockSec: data.wallclock_sec || 0,
        bytesWritten: data.bytes_written || 0,
      }
      return estimate.value
    } catch (_) {
      return null
    }
  }

  // Maps the wizard state to the createJob payload accepted by
  // /api/jobs. The wizard supports a single pool (the mockup's design
  // decision); multi-pool is the legacy NewJobView feature and stays
  // available there until the wizard fully replaces it.
  function buildJobPayload() {
    const stress = (type.value === 'cpu' || type.value === 'mixed') ? {
      workers: cpuConfig.value.stressorWorkers || 1,
      timeout: cpuConfig.value.timeoutSec || 60,
      stressors: cpuConfig.value.stressors.slice(),
    } : null

    // workers.count is the cluster-wide total. Distribute across the
    // selected nodes by setting workers_per_node = ceil(count / nodes).
    const nodeCount = workers.value.nodes.length || 1
    const perNode = Math.max(1, Math.ceil((workers.value.count || 1) / nodeCount))

    return {
      name: meta.value.name || 'benchmark',
      client_name: meta.value.clientName || '',
      mode: type.value || 'storage',
      engine: 'fio',
      proxmox_nodes: workers.value.nodes.slice(),
      workers_per_node: perNode,
      worker_cpu: workers.value.cpu || 4,
      worker_ram_mb: workers.value.ramMb || 4096,
      os_disk_gb: workers.value.osDiskGb || 20,
      data_disks: workers.value.dataDisks || 0,
      data_disk_gb: workers.value.dataDiskGb || 0,
      storage_pool: pool.value || '',
      profiles: type.value === 'cpu' ? [] : profiles.value.slice(),
      stress_config: stress,
    }
  }

  async function submit() {
    const payload = buildJobPayload()
    const result = await api.createJob(payload)
    clearDraft()
    return result.id
  }

  // Auto-persist on every change. Use deep watch since most state is
  // nested objects.
  watch(
    [type, cluster, pool, profiles, workers, range, cpuConfig, currentStep, meta],
    () => persistDraft(),
    { deep: true },
  )

  return {
    type, cluster, pool, profiles, workers, range, cpuConfig,
    currentStep, estimate, meta,
    stepKeys, stepKeyAt, totalSteps, stepValid,
    sublabelFor,
    next, previous, goTo, reset,
    persistDraft, loadDraft, clearDraft,
    fetchEstimate, submit, buildJobPayload,
  }
})
