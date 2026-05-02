import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useWsStore, RECONNECT_BACKOFF_MS } from './ws.js'

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

// Reconnect / backoff tests. We do not exercise a real socket: each test
// calls the internal _scheduleReconnect and _openSocket helpers and stubs
// WebSocket with a constructor that records its url and exposes onopen/onclose.
describe('useWsStore reconnect with exponential backoff', () => {
  let originalWebSocket
  let socketsCreated

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    socketsCreated = []
    originalWebSocket = globalThis.WebSocket
    // Stub WebSocket: each instance is captured so the test can flip onopen / onclose.
    globalThis.WebSocket = function FakeWebSocket(url) {
      this.url = url
      this.onopen = null
      this.onclose = null
      this.onmessage = null
      this.close = vi.fn(() => {
        if (typeof this.onclose === 'function') this.onclose()
      })
      socketsCreated.push(this)
    }
  })

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket
    vi.useRealTimers()
  })

  it('schedules a reconnect 1s after onclose when job is not terminal', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    expect(ws.reconnecting).toBe(false)
    ws._scheduleReconnect('j1')
    expect(ws.reconnecting).toBe(true)
    expect(ws._hasReconnectTimer()).toBe(true)
    expect(socketsCreated).toHaveLength(0)
    vi.advanceTimersByTime(1000)
    expect(socketsCreated).toHaveLength(1)
  })

  it('follows the backoff schedule 1, 2, 4, 8, 16, 30 seconds capped', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    const expected = [1000, 2000, 4000, 8000, 16000, 30000, 30000]
    expected.forEach(function(delay, i) {
      ws._scheduleReconnect('j1')
      expect(ws._hasReconnectTimer()).toBe(true)
      vi.advanceTimersByTime(delay - 1)
      expect(socketsCreated.length).toBe(i)
      vi.advanceTimersByTime(1)
      expect(socketsCreated.length).toBe(i + 1)
      // Each created socket triggers onclose (via close()) to simulate continued
      // failure, which would normally schedule the next retry. We do that
      // explicitly here by calling _scheduleReconnect on the next iteration.
    })
    // Sanity: backoff schedule constant matches
    expect(RECONNECT_BACKOFF_MS).toEqual([1000, 2000, 4000, 8000, 16000, 30000])
  })

  it('does not reconnect when current job is in a terminal state', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    // Drive jobStatus.status to a terminal state via a job_status event.
    ws._handleEvent({ type: 'job_status', job_id: 'j1', payload: { status: 'done', phase: '', runtime_seconds: 600 } })
    ws._scheduleReconnect('j1')
    expect(ws._hasReconnectTimer()).toBe(false)
    expect(ws.reconnecting).toBe(false)
    vi.advanceTimersByTime(60000)
    expect(socketsCreated).toHaveLength(0)
  })

  it('does not reconnect when current job is failed or cancelled', () => {
    const cases = ['failed', 'cancelled']
    cases.forEach(function(status) {
      setActivePinia(createPinia())
      const ws = useWsStore()
      ws._setCurrentJobId('j1')
      ws._handleEvent({ type: 'job_status', job_id: 'j1', payload: { status: status, phase: '', runtime_seconds: 0 } })
      ws._scheduleReconnect('j1')
      expect(ws._hasReconnectTimer()).toBe(false)
    })
  })

  it('updates lastEventTs on each event handled', () => {
    const ws = useWsStore()
    expect(ws.lastEventTs).toBe(0)
    vi.setSystemTime(new Date(1700000000000))
    ws._handleEvent({ type: 'storage_metric', job_id: 'j', payload: { iops_read: 1, engine: 'fio', profile_name: 'p' } })
    expect(ws.lastEventTs).toBe(1700000000000)
    vi.setSystemTime(new Date(1700000005000))
    ws._handleEvent({ type: 'job_status', job_id: 'j', payload: { status: 'running', phase: 'oltp', runtime_seconds: 30 } })
    expect(ws.lastEventTs).toBe(1700000005000)
  })

  it('disconnect cancels the pending reconnect timer', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    ws._scheduleReconnect('j1')
    expect(ws._hasReconnectTimer()).toBe(true)
    expect(ws.reconnecting).toBe(true)
    ws.disconnect()
    expect(ws._hasReconnectTimer()).toBe(false)
    expect(ws.reconnecting).toBe(false)
    vi.advanceTimersByTime(60000)
    expect(socketsCreated).toHaveLength(0)
  })

  it('successful onopen resets backoff index to 0', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    // Drive backoff up by two failures.
    ws._scheduleReconnect('j1')
    vi.advanceTimersByTime(1000)
    ws._scheduleReconnect('j1')
    vi.advanceTimersByTime(2000)
    expect(ws._getBackoffIndex()).toBeGreaterThanOrEqual(2)
    // Now simulate a successful open on the most recent socket.
    const last = socketsCreated[socketsCreated.length - 1]
    if (typeof last.onopen === 'function') last.onopen()
    expect(ws._getBackoffIndex()).toBe(0)
    expect(ws.connected).toBe(true)
    expect(ws.reconnecting).toBe(false)
  })

  it('appends ?since=<lastEventTs> to the WS URL when reconnecting', () => {
    const ws = useWsStore()
    ws._setCurrentJobId('j1')
    vi.setSystemTime(new Date(1700000000000))
    ws._handleEvent({ type: 'storage_metric', job_id: 'j1', payload: { iops_read: 1, engine: 'fio', profile_name: 'p' } })
    ws._scheduleReconnect('j1')
    vi.advanceTimersByTime(1000)
    expect(socketsCreated).toHaveLength(1)
    expect(socketsCreated[0].url).toMatch(/\/ws\?since=1700000000000$/)
  })
})
