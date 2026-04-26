import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'

const MAX_HISTORY = 60

export const useWsStore = defineStore('ws', () => {
  let socket = null
  const connected = ref(false)

  const elbenchoMetrics = reactive({
    iopsRead: 0, iopsWrite: 0,
    throughputReadMbps: 0, throughputWriteMbps: 0,
    latencyAvgMs: 0, latencyP99Ms: 0,
    profileName: '',
    history: { iopsRead: [], iopsWrite: [], latency: [], labels: [] },
  })

  const nodeMetrics  = reactive({})  // { nodeName: { cpuPct, ramPct, loadAvg } }
  const workerMetrics = reactive({}) // { workerId: { cpuPct } }
  const jobStatus    = reactive({ status: '', phase: '' })

  // Provisioning timeline
  const provSteps    = ref([])
  const provProgress = ref(0)

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
    switch (msg.type) {
      case 'elbencho_metric': {
        elbenchoMetrics.iopsRead            = p.iops_read
        elbenchoMetrics.iopsWrite           = p.iops_write
        elbenchoMetrics.throughputReadMbps  = p.throughput_read_mbps
        elbenchoMetrics.throughputWriteMbps = p.throughput_write_mbps
        elbenchoMetrics.latencyAvgMs        = p.latency_avg_ms
        elbenchoMetrics.latencyP99Ms        = p.latency_p99_ms
        elbenchoMetrics.profileName         = p.profile_name
        const now = new Date().toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
        elbenchoMetrics.history.iopsRead.push(p.iops_read)
        elbenchoMetrics.history.iopsWrite.push(p.iops_write)
        elbenchoMetrics.history.latency.push(p.latency_avg_ms)
        elbenchoMetrics.history.labels.push(now)
        if (elbenchoMetrics.history.labels.length > MAX_HISTORY) {
          elbenchoMetrics.history.iopsRead.shift()
          elbenchoMetrics.history.iopsWrite.shift()
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
      case 'job_status':
        jobStatus.status = p.status
        jobStatus.phase  = p.phase
        break
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
    elbenchoMetrics.latencyP99Ms = 0
    elbenchoMetrics.profileName = ''
    elbenchoMetrics.history.iopsRead.splice(0)
    elbenchoMetrics.history.iopsWrite.splice(0)
    elbenchoMetrics.history.latency.splice(0)
    elbenchoMetrics.history.labels.splice(0)
    Object.keys(nodeMetrics).forEach(k => delete nodeMetrics[k])
    Object.keys(workerMetrics).forEach(k => delete workerMetrics[k])
    jobStatus.status = ''
    jobStatus.phase  = ''
    provSteps.value = []
    provProgress.value = 0
  }

  return {
    connected, elbenchoMetrics, nodeMetrics, workerMetrics, jobStatus,
    provSteps, provProgress,
    connect, disconnect, reset, resetProvSteps, _handleEvent,
  }
})
