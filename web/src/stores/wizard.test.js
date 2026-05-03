import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWizardStore } from './wizard.js'

beforeEach(() => {
  setActivePinia(createPinia())
  localStorage.clear()
  vi.unstubAllGlobals()
})

describe('wizard step list', () => {
  it('falls back to storage list when type is empty', () => {
    const w = useWizardStore()
    expect(w.totalSteps).toBe(7)
  })

  it('uses 7 steps for storage mode', () => {
    const w = useWizardStore()
    w.type = 'storage'
    expect(w.totalSteps).toBe(7)
    expect(w.stepKeys).toEqual(['type', 'cluster', 'pool', 'profiles', 'workers', 'range', 'review'])
  })

  it('uses 5 steps for cpu mode', () => {
    const w = useWizardStore()
    w.type = 'cpu'
    expect(w.totalSteps).toBe(5)
    expect(w.stepKeys).toEqual(['type', 'cluster', 'workers', 'cpuConfig', 'review'])
  })

  it('uses 8 steps for mixed mode', () => {
    const w = useWizardStore()
    w.type = 'mixed'
    expect(w.totalSteps).toBe(8)
    expect(w.stepKeys[4]).toBe('cpuConfig')
  })
})

describe('wizard validity', () => {
  it('step type is invalid when type empty', () => {
    const w = useWizardStore()
    expect(w.stepValid[0]).toBe(false)
    w.type = 'storage'
    expect(w.stepValid[0]).toBe(true)
  })

  it('profiles step requires at least one profile', () => {
    const w = useWizardStore()
    w.type = 'storage'
    // step index 4 (1-based) = profiles
    expect(w.stepValid[3]).toBe(false)
    w.profiles.push('oltp-4k-70-30')
    expect(w.stepValid[3]).toBe(true)
  })

  it('workers step requires count and nodes', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.workers.count = 4
    expect(w.stepValid[4]).toBe(false) // no nodes selected yet
    w.workers.nodes.push('node-1')
    expect(w.stepValid[4]).toBe(true)
  })

  it('cpuConfig requires stressors and timeout', () => {
    const w = useWizardStore()
    w.type = 'cpu'
    // step index 4 (1-based) = cpuConfig (after type, cluster, workers)
    w.cpuConfig.stressors = []
    expect(w.stepValid[3]).toBe(false)
    w.cpuConfig.stressors = ['cpu']
    w.cpuConfig.timeoutSec = 60
    expect(w.stepValid[3]).toBe(true)
  })
})

describe('wizard navigation', () => {
  it('next() does not advance when current step invalid', () => {
    const w = useWizardStore()
    w.type = '' // step 1 invalid
    expect(w.currentStep).toBe(1)
    expect(w.next()).toBe(false)
    expect(w.currentStep).toBe(1)
  })

  it('next() advances when current step valid', () => {
    const w = useWizardStore()
    w.type = 'storage'
    expect(w.next()).toBe(true)
    expect(w.currentStep).toBe(2)
  })

  it('previous() decrements when not at first step', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.next()
    expect(w.currentStep).toBe(2)
    expect(w.previous()).toBe(true)
    expect(w.currentStep).toBe(1)
  })

  it('previous() does nothing at step 1', () => {
    const w = useWizardStore()
    expect(w.previous()).toBe(false)
    expect(w.currentStep).toBe(1)
  })

  it('goTo() forward only jumps when all prior steps valid', () => {
    const w = useWizardStore()
    w.type = 'storage'
    expect(w.goTo(4)).toBe(false) // cluster, pool empty
    expect(w.currentStep).toBe(1)
  })

  it('goTo() backward always succeeds', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.next()
    w.cluster = 'aqua'
    w.next()
    expect(w.currentStep).toBe(3)
    expect(w.goTo(1)).toBe(true)
    expect(w.currentStep).toBe(1)
  })
})

describe('wizard localStorage persistence', () => {
  it('persistDraft writes to localStorage', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.cluster = 'aqua'
    w.persistDraft()
    const raw = localStorage.getItem('benchere.wizard.draft')
    expect(raw).toBeTruthy()
    const data = JSON.parse(raw)
    expect(data.type).toBe('storage')
    expect(data.cluster).toBe('aqua')
  })

  it('loadDraft restores state from localStorage and migrates legacy single pool', () => {
    // The first wizard ship serialized a single pool as data.pool string.
    // loadDraft must migrate that into the new pools array transparently.
    localStorage.setItem('benchere.wizard.draft', JSON.stringify({
      type: 'mixed',
      cluster: 'aqua',
      pool: 'ceph-rbd',
      profiles: ['oltp-4k-70-30'],
      workers: { auto: false, count: 8, nodes: ['n1'], cpu: 8, ramMb: 8192, dataDisks: 2, dataDiskGb: 64 },
      range: { runtimeSec: 300, rampSec: 30, ioDepth: 64, thresholdsCustom: false, thresholds: {} },
      cpuConfig: { stressors: ['cpu', 'vm'], stressorWorkers: 8, timeoutSec: 120 },
      currentStep: 3,
      meta: { name: 'mybench', clientName: 'TestCo' },
    }))

    const w = useWizardStore()
    expect(w.loadDraft()).toBe(true)
    expect(w.type).toBe('mixed')
    expect(w.cluster).toBe('aqua')
    expect(w.pools).toEqual(['ceph-rbd'])
    expect(w.profiles).toEqual(['oltp-4k-70-30'])
    expect(w.workers.count).toBe(8)
    expect(w.cpuConfig.stressors).toEqual(['cpu', 'vm'])
    expect(w.currentStep).toBe(3)
    expect(w.meta.name).toBe('mybench')
  })

  it('loadDraft accepts the new pools array format', () => {
    localStorage.setItem('benchere.wizard.draft', JSON.stringify({
      type: 'storage',
      pools: ['ceph-rbd-fast', 'ceph-rbd-cap'],
    }))
    const w = useWizardStore()
    expect(w.loadDraft()).toBe(true)
    expect(w.pools).toEqual(['ceph-rbd-fast', 'ceph-rbd-cap'])
  })

  it('loadDraft returns false when no draft exists', () => {
    const w = useWizardStore()
    expect(w.loadDraft()).toBe(false)
  })

  it('clearDraft removes localStorage entry', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.persistDraft()
    expect(localStorage.getItem('benchere.wizard.draft')).toBeTruthy()
    w.clearDraft()
    expect(localStorage.getItem('benchere.wizard.draft')).toBeNull()
  })

  it('reset clears state and localStorage', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.cluster = 'aqua'
    w.persistDraft()
    w.reset()
    expect(w.type).toBe('')
    expect(w.cluster).toBe('')
    expect(localStorage.getItem('benchere.wizard.draft')).toBeNull()
  })
})

describe('wizard sublabels', () => {
  it('sublabel for profiles uses count', () => {
    const w = useWizardStore()
    w.profiles = ['a', 'b', 'c']
    expect(w.sublabelFor('profiles')).toBe('3 profils')
  })

  it('sublabel for workers shows auto + count', () => {
    const w = useWizardStore()
    w.workers.auto = true
    w.workers.count = 6
    expect(w.sublabelFor('workers')).toBe('auto . 6 workers')
  })

  it('sublabel for range distinguishes default from custom', () => {
    const w = useWizardStore()
    w.range.thresholdsCustom = false
    expect(w.sublabelFor('range')).toBe('defauts')
    w.range.thresholdsCustom = true
    expect(w.sublabelFor('range')).toBe('personnalises')
  })
})

describe('wizard buildJobPayload', () => {
  it('maps state to /api/jobs payload shape', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.cluster = 'aqua'
    w.pools = ['ceph-rbd']
    w.profiles = ['oltp-4k-70-30']
    w.workers.count = 4
    w.workers.nodes = ['node-1', 'node-2']
    w.workers.cpu = 4
    w.workers.ramMb = 4096
    w.workers.dataDisks = 1
    w.workers.dataDiskGb = 32
    w.meta.name = 'bench-test'
    w.meta.clientName = 'Acme'

    const p = w.buildJobPayload()
    expect(p.mode).toBe('storage')
    expect(p.engine).toBe('fio')
    expect(p.proxmox_nodes).toEqual(['node-1', 'node-2'])
    expect(p.workers_per_node).toBe(2) // ceil(4/2)
    expect(p.storage_pool).toBe('ceph-rbd')
    expect(p.profiles).toEqual(['oltp-4k-70-30'])
    expect(p.stress_config).toBeNull()
    // Single pool selected: job name is unchanged.
    expect(p.name).toBe('bench-test')
  })

  it('multi-pool: buildJobPayload(poolName) suffixes the job name', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.pools = ['ceph-rbd-fast', 'ceph-rbd-cap']
    w.profiles = ['oltp-4k-70-30']
    w.workers.count = 4
    w.workers.nodes = ['node-1']
    w.meta.name = 'bench-test'

    const p1 = w.buildJobPayload('ceph-rbd-fast')
    const p2 = w.buildJobPayload('ceph-rbd-cap')
    expect(p1.storage_pool).toBe('ceph-rbd-fast')
    expect(p2.storage_pool).toBe('ceph-rbd-cap')
    expect(p1.name).toBe('bench-test-ceph-rbd-fast')
    expect(p2.name).toBe('bench-test-ceph-rbd-cap')
  })

  it('cpu mode includes stress_config and empty profiles', () => {
    const w = useWizardStore()
    w.type = 'cpu'
    w.workers.nodes = ['node-1']
    w.workers.count = 1
    w.cpuConfig.stressors = ['cpu', 'vm']
    w.cpuConfig.stressorWorkers = 8
    w.cpuConfig.timeoutSec = 120
    const p = w.buildJobPayload()
    expect(p.profiles).toEqual([])
    expect(p.stress_config).not.toBeNull()
    expect(p.stress_config.stressors).toEqual(['cpu', 'vm'])
    expect(p.stress_config.timeout).toBe(120)
  })
})

describe('wizard pool sublabel and validity', () => {
  it('pool step is invalid when no pool selected', () => {
    const w = useWizardStore()
    w.type = 'storage'
    expect(w.stepValid[2]).toBe(false)
  })

  it('pool step is valid when at least one pool selected', () => {
    const w = useWizardStore()
    w.type = 'storage'
    w.pools = ['ceph-rbd']
    expect(w.stepValid[2]).toBe(true)
  })

  it('sublabel for pool shows single name when one selected', () => {
    const w = useWizardStore()
    w.pools = ['ceph-rbd-fast']
    expect(w.sublabelFor('pool')).toBe('ceph-rbd-fast')
  })

  it('sublabel for pool shows count when several selected', () => {
    const w = useWizardStore()
    w.pools = ['fast', 'cap', 'archive']
    expect(w.sublabelFor('pool')).toBe('3 pools')
  })
})
