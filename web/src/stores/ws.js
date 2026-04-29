import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'

const MAX_HISTORY = 60
const MAX_EVENTS  = 500

export const useWsStore = defineStore('ws', () => {
  let socket = null
  const connected = ref(false)

  const elbenchoMetrics = reactive({
    iopsRead: 0, iopsWrite: 0,
    throughputReadMbps: 0, throughputWriteMbps: 0,
    latencyAvgMs: 0,
    profileName: '',
    history: {
      iopsRead: [],
      iopsWrite: [],
      throughputRead: [],
      throughputWrite: [],
      latency: [],
      labels: [],
    },
  })

  const nodeMetrics  = reactive({})  // { nodeName: { cpuPct, ramPct, loadAvg } }
  const workerMetrics = reactive({}) // { workerId: { cpuPct } }
  const jobStatus    = reactive({ status: '', phase: '', runtimeSeconds: 0 })

  // Provisioning timeline
  const provSteps    = ref([])
  const provProgress = ref(0)

  // Phase timing for the per-phase progress bar.
  // prefillStartedAt / profileStartedAt are wall-clock millis when the
  // current phase began. Reset to 0 when not applicable.
  const prefillStartedAt = ref(0)
  const profileStartedAt = ref(0)

  // Live event log (capped FIFO). Each entry: {t: ISO ts, type, line, payload}.
  // Newest pushed at the end; the LiveLogsPanel renders newest-first.
  const events = ref([])

  function _formatLogLine(type, payload) {
    switch (type) {
      case 'job_status': {
        const phase = payload.phase || ''
        if (phase) return `phase: ${phase}`
        return `status: ${payload.status || ''}`
      }
      case 'provisioning_step':
        return `provisioning: ${payload.step}${payload.detail ? ' - ' + payload.detail : ''}`
      case 'elbencho_metric': {
        const r  = (payload.iops_read || 0).toFixed(0)
        const w  = (payload.iops_write || 0).toFixed(0)
        const br = (payload.throughput_read_mbps || 0).toFixed(0)
        const bw = (payload.throughput_write_mbps || 0).toFixed(0)
        const la = (payload.latency_avg_ms || 0).toFixed(2)
        return `metric: ${payload.profile_name || ''} iops_r=${r} iops_w=${w} bw_r=${br}MB/s bw_w=${bw}MB/s lat_avg=${la}ms`
      }
      case 'proxmox_node':
        return `node: ${payload.node_name} cpu=${(payload.cpu_pct || 0).toFixed(1)}% ram=${(payload.ram_pct || 0).toFixed(1)}%`
      case 'proxmox_vm':
        return `vm: ${payload.worker_id?.slice(0, 8) || '?'} cpu=${(payload.cpu_pct || 0).toFixed(1)}% ram=${(payload.ram_pct || 0).toFixed(1)}%`
      default:
        return type
    }
  }

  function _pushEvent(type, payload) {
    const now = new Date()
    events.value.push({
      t: now,
      type,
      line: _formatLogLine(type, payload),
      payload,
    })
    if (events.value.length > MAX_EVENTS) {
      events.value.splice(0, events.value.length - MAX_EVENTS)
    }
  }

  function connect(jobId) {
    disconnect()
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    socket = new WebSocket(`${proto}//${window.location.host}/ws`)

    socket.onopen  = () => { connected.value = true }
    socket.onclose = () => { connected.value = false }
    socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (jobId && msg.job_id !== jobId) return
        _handleEvent(msg)
      } catch (_) { /* ignore malformed frames */ }
    }
  }

  function _handleEvent(msg) {
    const p = msg.payload
    _pushEvent(msg.type, p || {})
    switch (msg.type) {
      case 'elbencho_metric': {
        elbenchoMetrics.iopsRead            = p.iops_read
        elbenchoMetrics.iopsWrite           = p.iops_write
        elbenchoMetrics.throughputReadMbps  = p.throughput_read_mbps
        elbenchoMetrics.throughputWriteMbps = p.throughput_write_mbps
        elbenchoMetrics.latencyAvgMs        = p.latency_avg_ms
        elbenchoMetrics.profileName         = p.profile_name
        const now = new Date().toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
        elbenchoMetrics.history.iopsRead.push(p.iops_read || 0)
        elbenchoMetrics.history.iopsWrite.push(p.iops_write || 0)
        elbenchoMetrics.history.throughputRead.push(p.throughput_read_mbps || 0)
        elbenchoMetrics.history.throughputWrite.push(p.throughput_write_mbps || 0)
        elbenchoMetrics.history.latency.push(p.latency_avg_ms || 0)
        elbenchoMetrics.history.labels.push(now)
        if (elbenchoMetrics.history.labels.length > MAX_HISTORY) {
          elbenchoMetrics.history.iopsRead.shift()
          elbenchoMetrics.history.iopsWrite.shift()
          elbenchoMetrics.history.throughputRead.shift()
          elbenchoMetrics.history.throughputWrite.shift()
          elbenchoMetrics.history.latency.shift()
          elbenchoMetrics.history.labels.shift()
        }
        break
      }
      case 'proxmox_node':
        nodeMetrics[p.node_name] = { cpuPct: p.cpu_pct, ramPct: p.ram_pct, loadAvg: p.load_avg }
        break
      case 'proxmox_vm':
        workerMetrics[p.worker_id] = { cpuPct: p.cpu_pct, ramPct: p.ram_pct, netInBps: p.net_in_bps, netOutBps: p.net_out_bps, diskReadBps: p.disk_read_bps, diskWriteBps: p.disk_write_bps }
        break
      case 'job_status': {
        const previousPhase = jobStatus.phase
        jobStatus.status = p.status
        jobStatus.phase  = p.phase || ''
        jobStatus.runtimeSeconds = p.runtime_seconds || 0
        if (jobStatus.phase !== previousPhase) {
          if (jobStatus.phase === 'prefill') {
            prefillStartedAt.value = Date.now()
            profileStartedAt.value = 0
          } else if (jobStatus.phase && jobStatus.runtimeSeconds > 0) {
            profileStartedAt.value = Date.now()
            prefillStartedAt.value = 0
          } else {
            prefillStartedAt.value = 0
            profileStartedAt.value = 0
          }
        }
        break
      }
      case 'provisioning_step': {
        provProgress.value = p.progress
        const idx = provSteps.value.findIndex(s => s.step === p.step)
        if (idx >= 0) {
          provSteps.value[idx] = p
        } else {
          provSteps.value.push(p)
        }
        break
      }
    }
  }

  function disconnect() {
    socket?.close()
    socket = null
    connected.value = false
  }

  function resetProvSteps() {
    provSteps.value = []
    provProgress.value = 0
  }

  function reset() {
    elbenchoMetrics.iopsRead = 0
    elbenchoMetrics.iopsWrite = 0
    elbenchoMetrics.throughputReadMbps = 0
    elbenchoMetrics.throughputWriteMbps = 0
    elbenchoMetrics.latencyAvgMs = 0
    elbenchoMetrics.profileName = ''
    elbenchoMetrics.history.iopsRead.splice(0)
    elbenchoMetrics.history.iopsWrite.splice(0)
    elbenchoMetrics.history.throughputRead.splice(0)
    elbenchoMetrics.history.throughputWrite.splice(0)
    elbenchoMetrics.history.latency.splice(0)
    elbenchoMetrics.history.labels.splice(0)
    Object.keys(nodeMetrics).forEach(k => delete nodeMetrics[k])
    Object.keys(workerMetrics).forEach(k => delete workerMetrics[k])
    jobStatus.status = ''
    jobStatus.phase  = ''
    jobStatus.runtimeSeconds = 0
    provSteps.value = []
    provProgress.value = 0
    prefillStartedAt.value = 0
    profileStartedAt.value = 0
    events.value.splice(0)
  }

  // Newest-first list, derived. Components iterate this directly.
  const eventsNewestFirst = computed(() => events.value.slice().reverse())
  const eventCount = computed(() => events.value.length)

  return {
    connected, elbenchoMetrics, nodeMetrics, workerMetrics, jobStatus,
    provSteps, provProgress,
    prefillStartedAt, profileStartedAt,
    events, eventsNewestFirst, eventCount,
    connect, disconnect, reset, resetProvSteps, _handleEvent,
  }
})
