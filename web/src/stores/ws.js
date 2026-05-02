import { defineStore } from 'pinia'
import { ref, reactive, computed } from 'vue'
import { useJobsStore } from './jobs.js'

const MAX_HISTORY = 60
const MAX_EVENTS  = 500

// Exponential backoff schedule for WS reconnects, in milliseconds.
// Doubles up to a 30s ceiling. Resets to the first entry on successful open.
export const RECONNECT_BACKOFF_MS = [1000, 2000, 4000, 8000, 16000, 30000]
const TERMINAL_JOB_STATES = new Set(['done', 'failed', 'cancelled'])

export const useWsStore = defineStore('ws', () => {
  let socket = null
  let reconnectTimer = null
  let backoffIndex = 0
  const connected = ref(false)
  const reconnecting = ref(false)
  const lastEventTs = ref(0)

  // Live storage metrics, engine-agnostic. Replaces the old elbenchoMetrics
  // reactive object. Renamed in v2.0.0 along with the WS event.
  const liveMetrics = reactive({
    engine: '',
    profileName: '',
    iopsRead: 0, iopsWrite: 0,
    throughputReadMbps: 0, throughputWriteMbps: 0,
    latencyAvgMs: 0,
    latencyP50Ms: 0, latencyP95Ms: 0, latencyP99Ms: 0, latencyP999Ms: 0,
    latencyWriteP99Ms: 0,
    history: {
      iopsRead: [], iopsWrite: [],
      throughputRead: [], throughputWrite: [],
      latencyP50: [], latencyP95: [], latencyP99: [],
      labels: [],
    },
  })

  const nodeMetrics    = reactive({}) // { nodeName: { cpuPct, ramPct, loadAvg, history: [] } }
  const workerMetrics  = reactive({}) // { workerId: { cpuPct, ramPct, netInBps, ... } }
  const phaseSummaries = ref([])      // chronological list, append-on-event
  const jobStatus      = reactive({ status: '', phase: '', runtimeSeconds: 0 })

  const provSteps    = ref([])
  const provProgress = ref(0)
  const prefillStartedAt = ref(0)
  const profileStartedAt = ref(0)

  const events = ref([]) // { t, type, source, level, line, payload }

  function _formatLogLine(type, payload) {
    switch (type) {
      case 'job_status':
        return payload.phase ? ('phase: ' + payload.phase) : ('status: ' + (payload.status || ''))
      case 'provisioning_step':
        return 'provisioning: ' + payload.step + (payload.detail ? ' - ' + payload.detail : '')
      case 'storage_metric': {
        const r  = (payload.iops_read || 0).toFixed(0)
        const w  = (payload.iops_write || 0).toFixed(0)
        const la = (payload.latency_p99_ms || payload.latency_avg_ms || 0).toFixed(2)
        return 'metric[' + (payload.engine || '') + ']: ' + (payload.profile_name || '') + ' iops_r=' + r + ' iops_w=' + w + ' p99=' + la + 'ms'
      }
      case 'log_line':
        return payload.text
      case 'phase_summary':
        return 'summary: ' + (payload.profile_name || '') + ' iops_r_avg=' + (payload.iops_read_avg||0).toFixed(0) + ' p99=' + (payload.lat_p99_ms||0).toFixed(2) + 'ms cv=' + (payload.iops_cv_pct||0).toFixed(1) + '%'
      case 'worker_saturation':
        return 'saturation: worker=' + ((payload.worker_id || '?').slice(0,8)) + ' kind=' + (payload.kind || '') + ' value=' + (payload.value||0).toFixed(1)
      case 'proxmox_node':
        return 'node: ' + (payload.node_name || '') + ' cpu=' + (payload.cpu_pct||0).toFixed(1) + '% ram=' + (payload.ram_pct||0).toFixed(1) + '%'
      case 'proxmox_vm':
        return 'vm: ' + ((payload.worker_id || '?').slice(0,8)) + ' cpu=' + (payload.cpu_pct||0).toFixed(1) + '%'
      default:
        return type
    }
  }

  function _pushEvent(type, payload) {
    const now = new Date()
    const entry = {
      t: now,
      type,
      source: (payload && payload.source) ? payload.source : 'system',
      level: (payload && payload.level) ? payload.level : 'info',
      line: _formatLogLine(type, payload || {}),
      payload,
    }
    events.value.push(entry)
    if (events.value.length > MAX_EVENTS) {
      events.value.splice(0, events.value.length - MAX_EVENTS)
    }
  }

  // currentJobId tracks the job whose data is in liveMetrics/history. The
  // dashboard remounts on every navigation, but if we re-enter the same
  // job we want to keep what is already in memory instead of zeroing the
  // charts. Different job (or null): full reset.
  let currentJobId = null

  function _isCurrentJobTerminal() {
    if (!currentJobId) return false
    // Prefer the local jobStatus we already track from the WS feed: it is
    // authoritative for the current job and does not require pinia setup.
    if (jobStatus.status && TERMINAL_JOB_STATES.has(jobStatus.status)) return true
    try {
      const store = useJobsStore()
      const list = (store && store.jobs) ? store.jobs : []
      const arr = Array.isArray(list) ? list : (list.value || [])
      const job = arr.find(function(j) { return j && j.id === currentJobId })
      if (!job) return false
      return TERMINAL_JOB_STATES.has(job.status)
    } catch (_) {
      // If pinia is not active or jobs store not available, default to "not terminal"
      return false
    }
  }

  function _buildWsUrl() {
    const proto = (typeof window !== 'undefined' && window.location && window.location.protocol === 'https:') ? 'wss:' : 'ws:'
    const host  = (typeof window !== 'undefined' && window.location && window.location.host) ? window.location.host : 'localhost'
    let url = proto + '//' + host + '/ws'
    if (lastEventTs.value > 0) {
      url += (url.indexOf('?') === -1 ? '?' : '&') + 'since=' + lastEventTs.value
    }
    return url
  }

  function _scheduleReconnect(jobId) {
    if (_isCurrentJobTerminal()) {
      reconnecting.value = false
      return
    }
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    const idx = Math.min(backoffIndex, RECONNECT_BACKOFF_MS.length - 1)
    const delay = RECONNECT_BACKOFF_MS[idx]
    backoffIndex = idx + 1
    reconnecting.value = true
    reconnectTimer = setTimeout(function() {
      reconnectTimer = null
      _openSocket(jobId)
    }, delay)
  }

  function _openSocket(jobId) {
    socket = new WebSocket(_buildWsUrl())
    socket.onopen = function() {
      connected.value = true
      reconnecting.value = false
      backoffIndex = 0
    }
    socket.onclose = function() {
      connected.value = false
      socket = null
      // Only auto-reconnect if we still have a current job and it is not terminal.
      if (currentJobId) _scheduleReconnect(jobId)
    }
    socket.onmessage = function(event) {
      try {
        const msg = JSON.parse(event.data)
        if (jobId && msg.job_id !== jobId) return
        _handleEvent(msg)
      } catch (_) {}
    }
  }

  function connect(jobId) {
    disconnect()
    if (jobId !== currentJobId) {
      reset()
      currentJobId = jobId
    }
    backoffIndex = 0
    reconnecting.value = false
    _openSocket(jobId)
  }

  function _handleEvent(msg) {
    const p = msg.payload
    lastEventTs.value = Date.now()
    _pushEvent(msg.type, p || {})
    switch (msg.type) {
      case 'storage_metric': {
        liveMetrics.engine             = p.engine
        liveMetrics.profileName        = p.profile_name
        liveMetrics.iopsRead           = p.iops_read
        liveMetrics.iopsWrite          = p.iops_write
        liveMetrics.throughputReadMbps = p.throughput_read_mbps
        liveMetrics.throughputWriteMbps = p.throughput_write_mbps
        liveMetrics.latencyAvgMs       = p.latency_avg_ms
        liveMetrics.latencyP50Ms       = p.latency_p50_ms
        liveMetrics.latencyP95Ms       = p.latency_p95_ms
        liveMetrics.latencyP99Ms       = p.latency_p99_ms
        liveMetrics.latencyP999Ms      = p.latency_p999_ms
        liveMetrics.latencyWriteP99Ms  = p.latency_write_p99_ms
        const ts = new Date().toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
        const h = liveMetrics.history
        h.iopsRead.push(p.iops_read || 0)
        h.iopsWrite.push(p.iops_write || 0)
        h.throughputRead.push(p.throughput_read_mbps || 0)
        h.throughputWrite.push(p.throughput_write_mbps || 0)
        h.latencyP50.push(p.latency_p50_ms || 0)
        h.latencyP95.push(p.latency_p95_ms || 0)
        h.latencyP99.push(p.latency_p99_ms || 0)
        h.labels.push(ts)
        if (h.labels.length > MAX_HISTORY) {
          h.iopsRead.shift(); h.iopsWrite.shift()
          h.throughputRead.shift(); h.throughputWrite.shift()
          h.latencyP50.shift(); h.latencyP95.shift(); h.latencyP99.shift()
          h.labels.shift()
        }
        break
      }
      case 'proxmox_node': {
        const cur = nodeMetrics[p.node_name] || { cpuPct: 0, ramPct: 0, loadAvg: 0, history: [] }
        cur.cpuPct = p.cpu_pct
        cur.ramPct = p.ram_pct
        cur.loadAvg = p.load_avg
        cur.history.push(p.cpu_pct || 0)
        if (cur.history.length > 30) cur.history.shift()
        nodeMetrics[p.node_name] = cur
        break
      }
      case 'proxmox_vm': {
        const cur = workerMetrics[p.worker_id] || {}
        cur.cpuPct = p.cpu_pct
        cur.ramPct = p.ram_pct
        cur.netInBps = p.net_in_bps
        cur.netOutBps = p.net_out_bps
        cur.diskReadBps = p.disk_read_bps
        cur.diskWriteBps = p.disk_write_bps
        cur.cpuHistory = (cur.cpuHistory || []).concat(p.cpu_pct || 0).slice(-30)
        workerMetrics[p.worker_id] = cur
        break
      }
      case 'worker_saturation': {
        const cur = workerMetrics[p.worker_id] || {}
        cur.saturation = { kind: p.kind, value: p.value, threshold: p.threshold, ts: Date.now() }
        workerMetrics[p.worker_id] = cur
        break
      }
      case 'phase_summary': {
        phaseSummaries.value.push(p)
        if (phaseSummaries.value.length > 32) phaseSummaries.value.shift()
        break
      }
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
        // If the job just transitioned to a terminal state, stop trying to reconnect.
        if (TERMINAL_JOB_STATES.has(jobStatus.status)) {
          if (reconnectTimer) {
            clearTimeout(reconnectTimer)
            reconnectTimer = null
          }
          reconnecting.value = false
        }
        break
      }
      case 'provisioning_step': {
        provProgress.value = p.progress
        const idx = provSteps.value.findIndex(function(s) { return s.step === p.step })
        if (idx >= 0) provSteps.value[idx] = p
        else provSteps.value.push(p)
        break
      }
    }
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    reconnecting.value = false
    backoffIndex = 0
    currentJobId = null
    if (socket) {
      // Detach onclose first so it does not schedule another reconnect.
      socket.onclose = null
      socket.close()
    }
    socket = null
    connected.value = false
  }

  function resetProvSteps() {
    provSteps.value = []
    provProgress.value = 0
  }

  function reset() {
    Object.assign(liveMetrics, {
      engine: '', profileName: '',
      iopsRead: 0, iopsWrite: 0,
      throughputReadMbps: 0, throughputWriteMbps: 0,
      latencyAvgMs: 0,
      latencyP50Ms: 0, latencyP95Ms: 0, latencyP99Ms: 0, latencyP999Ms: 0,
      latencyWriteP99Ms: 0,
    })
    liveMetrics.history.iopsRead.splice(0)
    liveMetrics.history.iopsWrite.splice(0)
    liveMetrics.history.throughputRead.splice(0)
    liveMetrics.history.throughputWrite.splice(0)
    liveMetrics.history.latencyP50.splice(0)
    liveMetrics.history.latencyP95.splice(0)
    liveMetrics.history.latencyP99.splice(0)
    liveMetrics.history.labels.splice(0)
    Object.keys(nodeMetrics).forEach(function(k) { delete nodeMetrics[k] })
    Object.keys(workerMetrics).forEach(function(k) { delete workerMetrics[k] })
    phaseSummaries.value.splice(0)
    jobStatus.status = ''
    jobStatus.phase  = ''
    jobStatus.runtimeSeconds = 0
    provSteps.value = []
    provProgress.value = 0
    prefillStartedAt.value = 0
    profileStartedAt.value = 0
    events.value.splice(0)
    lastEventTs.value = 0
  }

  const eventsNewestFirst = computed(function() { return events.value.slice().reverse() })
  const eventCount = computed(function() { return events.value.length })

  // Test-only helpers. Not part of the public store contract but exposed so
  // unit tests can drive the reconnect logic without owning a real socket.
  function _hasReconnectTimer() { return reconnectTimer !== null }
  function _setCurrentJobId(id) { currentJobId = id }
  function _getBackoffIndex() { return backoffIndex }

  return {
    connected, reconnecting, lastEventTs,
    liveMetrics, nodeMetrics, workerMetrics, phaseSummaries, jobStatus,
    provSteps, provProgress,
    prefillStartedAt, profileStartedAt,
    events, eventsNewestFirst, eventCount,
    connect, disconnect, reset, resetProvSteps,
    _handleEvent, _scheduleReconnect, _openSocket,
    _hasReconnectTimer, _setCurrentJobId, _getBackoffIndex,
  }
})
