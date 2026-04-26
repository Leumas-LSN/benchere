import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWsStore } from './ws.js'

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('_handleEvent — elbencho_metric', () => {
  it('updates current metrics and appends to history', () => {
    const store = useWsStore()
    store._handleEvent({
      type: 'elbencho_metric',
      job_id: 'j1',
      payload: {
        iops_read: 42000, iops_write: 1000,
        throughput_read_mbps: 165, throughput_write_mbps: 40,
        latency_avg_ms: 0.9, latency_p99_ms: 2.1,
        profile_name: '4k_70read',
      },
    })
    expect(store.elbenchoMetrics.iopsRead).toBe(42000)
    expect(store.elbenchoMetrics.throughputReadMbps).toBe(165)
    expect(store.elbenchoMetrics.latencyAvgMs).toBe(0.9)
    expect(store.elbenchoMetrics.profileName).toBe('4k_70read')
    expect(store.elbenchoMetrics.history.iopsRead).toHaveLength(1)
    expect(store.elbenchoMetrics.history.iopsRead[0]).toBe(42000)
  })

  it('caps history at 60 points', () => {
    const store = useWsStore()
    for (let i = 0; i < 65; i++) {
      store._handleEvent({
        type: 'elbencho_metric', job_id: 'j1',
        payload: { iops_read: i, iops_write: 0, throughput_read_mbps: 0,
                   throughput_write_mbps: 0, latency_avg_ms: 0, latency_p99_ms: 0, profile_name: '' },
      })
    }
    expect(store.elbenchoMetrics.history.iopsRead).toHaveLength(60)
    // Most recent value is 64
    expect(store.elbenchoMetrics.history.iopsRead[59]).toBe(64)
  })
})

describe('_handleEvent — proxmox_node', () => {
  it('updates node metrics by name', () => {
    const store = useWsStore()
    store._handleEvent({
      type: 'proxmox_node', job_id: 'j1',
      payload: { node_name: 'pve-01', cpu_pct: 78, ram_pct: 61, load_avg: 2.4 },
    })
    expect(store.nodeMetrics['pve-01'].cpuPct).toBe(78)
    expect(store.nodeMetrics['pve-01'].ramPct).toBe(61)
  })
})

describe('_handleEvent — proxmox_vm', () => {
  it('updates worker metrics by id', () => {
    const store = useWsStore()
    store._handleEvent({
      type: 'proxmox_vm', job_id: 'j1',
      payload: { worker_id: 'w-abc', cpu_pct: 95 },
    })
    expect(store.workerMetrics['w-abc'].cpuPct).toBe(95)
  })
})

describe('_handleEvent — job_status', () => {
  it('updates job status and phase', () => {
    const store = useWsStore()
    store._handleEvent({
      type: 'job_status', job_id: 'j1',
      payload: { status: 'running', phase: 'benchmarking' },
    })
    expect(store.jobStatus.status).toBe('running')
    expect(store.jobStatus.phase).toBe('benchmarking')
  })
})

describe('reset', () => {
  it('clears all metrics state', () => {
    const store = useWsStore()
    store._handleEvent({
      type: 'elbencho_metric', job_id: 'j1',
      payload: { iops_read: 100, iops_write: 0, throughput_read_mbps: 1, throughput_write_mbps: 0,
                 latency_avg_ms: 0.1, latency_p99_ms: 0.2, profile_name: 'test' },
    })
    store._handleEvent({
      type: 'proxmox_node', job_id: 'j1',
      payload: { node_name: 'pve-01', cpu_pct: 78, ram_pct: 61, load_avg: 2.4 },
    })
    store.reset()
    expect(store.elbenchoMetrics.iopsRead).toBe(0)
    expect(store.elbenchoMetrics.history.labels).toHaveLength(0)
    expect(Object.keys(store.nodeMetrics)).toHaveLength(0)
  })
})
