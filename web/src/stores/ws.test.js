import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWsStore } from './ws.js'

describe('useWsStore v2', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('handles a storage_metric event', () => {
    const ws = useWsStore()
    ws._handleEvent({
      type: 'storage_metric',
      job_id: 'j1',
      payload: {
        engine: 'fio',
        profile_name: 'oltp-4k-70-30',
        iops_read: 80000, iops_write: 30000,
        throughput_read_mbps: 312.5, throughput_write_mbps: 117.2,
        latency_avg_ms: 0.6, latency_p50_ms: 0.5, latency_p95_ms: 1.2,
        latency_p99_ms: 2.0, latency_p999_ms: 4.1, latency_write_p99_ms: 1.8,
      },
    })
    expect(ws.liveMetrics.engine).toBe('fio')
    expect(ws.liveMetrics.profileName).toBe('oltp-4k-70-30')
    expect(ws.liveMetrics.iopsRead).toBe(80000)
    expect(ws.liveMetrics.latencyP99Ms).toBe(2.0)
    expect(ws.liveMetrics.history.iopsRead).toEqual([80000])
    expect(ws.liveMetrics.history.latencyP99).toEqual([2.0])
  })

  it('caps history to MAX_HISTORY', () => {
    const ws = useWsStore()
    for (let i = 0; i < 80; i++) {
      ws._handleEvent({ type: 'storage_metric', job_id: 'j', payload: { iops_read: i, engine: 'fio', profile_name: 'p' } })
    }
    expect(ws.liveMetrics.history.iopsRead.length).toBe(60)
  })

  it('aggregates phase summaries', () => {
    const ws = useWsStore()
    ws._handleEvent({ type: 'phase_summary', job_id: 'j', payload: { profile_name: 'oltp', iops_read_avg: 80000, lat_p99_ms: 2.3 } })
    expect(ws.phaseSummaries[0].profile_name).toBe('oltp')
  })

  it('records worker saturation', () => {
    const ws = useWsStore()
    ws._handleEvent({ type: 'worker_saturation', job_id: 'j', payload: { worker_id: 'w1', kind: 'cpu', value: 92, threshold: 80 } })
    expect(ws.workerMetrics['w1'].saturation.kind).toBe('cpu')
  })

  it('updates jobStatus on job_status event', () => {
    const ws = useWsStore()
    ws._handleEvent({ type: 'job_status', job_id: 'j', payload: { status: 'running', phase: 'oltp-4k-70-30', runtime_seconds: 300 } })
    expect(ws.jobStatus.status).toBe('running')
    expect(ws.jobStatus.phase).toBe('oltp-4k-70-30')
  })

  it('updates node metrics with history', () => {
    const ws = useWsStore()
    ws._handleEvent({ type: 'proxmox_node', job_id: 'j', payload: { node_name: 'pve-01', cpu_pct: 78, ram_pct: 61, load_avg: 2.4 } })
    expect(ws.nodeMetrics['pve-01'].cpuPct).toBe(78)
    expect(ws.nodeMetrics['pve-01'].history).toEqual([78])
  })

  it('reset clears all state', () => {
    const ws = useWsStore()
    ws._handleEvent({ type: 'storage_metric', job_id: 'j', payload: { iops_read: 100, engine: 'fio', profile_name: 'p' } })
    ws._handleEvent({ type: 'proxmox_node', job_id: 'j', payload: { node_name: 'pve-01', cpu_pct: 78, ram_pct: 61, load_avg: 2.4 } })
    ws.reset()
    expect(ws.liveMetrics.iopsRead).toBe(0)
    expect(ws.liveMetrics.history.labels).toHaveLength(0)
    expect(Object.keys(ws.nodeMetrics)).toHaveLength(0)
  })
})
