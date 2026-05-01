package ws

import "encoding/json"

type EventType string

const (
	EventStorageMetric    EventType = "storage_metric"
	EventProxmoxNode      EventType = "proxmox_node"
	EventProxmoxVM        EventType = "proxmox_vm"
	EventJobStatus        EventType = "job_status"
	EventProvisioningStep EventType = "provisioning_step"
	EventLogLine          EventType = "log_line"
	EventPhaseSummary     EventType = "phase_summary"
	EventWorkerSaturation EventType = "worker_saturation"
)

type Event struct {
	Type    EventType       `json:"type"`
	JobID   string          `json:"job_id"`
	Payload json.RawMessage `json:"payload"`
}

// StorageMetricPayload is emitted once per status interval by the storage
// engine (fio or elbencho). Engine field carries the source so the frontend
// can render engine-specific notes (e.g. "elbencho lacks p50/p95"). Latency
// fields are zero when the engine does not provide them.
type StorageMetricPayload struct {
	Engine              string  `json:"engine"`
	ProfileName         string  `json:"profile_name"`
	IOPSRead            float64 `json:"iops_read"`
	IOPSWrite           float64 `json:"iops_write"`
	ThroughputReadMBps  float64 `json:"throughput_read_mbps"`
	ThroughputWriteMBps float64 `json:"throughput_write_mbps"`
	LatencyAvgMs        float64 `json:"latency_avg_ms"`
	LatencyReadAvgMs    float64 `json:"latency_read_avg_ms"`
	LatencyWriteAvgMs   float64 `json:"latency_write_avg_ms"`
	LatencyP50Ms        float64 `json:"latency_p50_ms"`
	LatencyP95Ms        float64 `json:"latency_p95_ms"`
	LatencyP99Ms        float64 `json:"latency_p99_ms"`
	LatencyP999Ms       float64 `json:"latency_p999_ms"`
	LatencyWriteP99Ms   float64 `json:"latency_write_p99_ms"`
}

type ProxmoxNodePayload struct {
	NodeName string  `json:"node_name"`
	CPUPct   float64 `json:"cpu_pct"`
	RAMPct   float64 `json:"ram_pct"`
	LoadAvg  float64 `json:"load_avg"`
}

type ProxmoxVMPayload struct {
	WorkerID     string  `json:"worker_id"`
	CPUPct       float64 `json:"cpu_pct"`
	RAMPct       float64 `json:"ram_pct"`
	NetInBps     float64 `json:"net_in_bps"`
	NetOutBps    float64 `json:"net_out_bps"`
	DiskReadBps  float64 `json:"disk_read_bps"`
	DiskWriteBps float64 `json:"disk_write_bps"`
}

// JobStatusPayload reports a job state transition. RuntimeSeconds carries the
// expected duration of the current phase when known (e.g. a profile bench
// runtime parsed from the profile config), and zero otherwise. Used by the
// frontend to drive a per-phase progress bar with an ETA.
type JobStatusPayload struct {
	Status         string `json:"status"`
	Phase          string `json:"phase,omitempty"`
	RuntimeSeconds int    `json:"runtime_seconds,omitempty"`
}

// ProvisioningStepPayload describes a provisioning step event.
type ProvisioningStepPayload struct {
	Step     string  `json:"step"`
	Detail   string  `json:"detail"`
	Progress float64 `json:"progress"`
	JobID    string  `json:"job_id"`
}

// LogLinePayload is the operator-grade narrative line. Source distinguishes
// orchestrator from ansible from fio from system messages so the frontend
// filter can scope the panel.
type LogLinePayload struct {
	Source string `json:"source"` // orch | ansible | fio | system
	Level  string `json:"level"`  // info | warn | error
	Text   string `json:"text"`
	TS     string `json:"ts"` // RFC3339
}

// PhaseSummaryPayload mirrors the db.PhaseSummary row, sent at end of each
// profile so the frontend can render the verdict strip without reloading.
type PhaseSummaryPayload struct {
	ProfileName            string  `json:"profile_name"`
	SamplesCount           int     `json:"samples_count"`
	IOPSReadAvg            float64 `json:"iops_read_avg"`
	IOPSReadMin            float64 `json:"iops_read_min"`
	IOPSReadMax            float64 `json:"iops_read_max"`
	IOPSWriteAvg           float64 `json:"iops_write_avg"`
	IOPSWriteMin           float64 `json:"iops_write_min"`
	IOPSWriteMax           float64 `json:"iops_write_max"`
	ThroughputReadMBpsAvg  float64 `json:"throughput_read_mbps_avg"`
	ThroughputReadMBpsMax  float64 `json:"throughput_read_mbps_max"`
	ThroughputWriteMBpsAvg float64 `json:"throughput_write_mbps_avg"`
	ThroughputWriteMBpsMax float64 `json:"throughput_write_mbps_max"`
	LatP50Ms               float64 `json:"lat_p50_ms"`
	LatP95Ms               float64 `json:"lat_p95_ms"`
	LatP99Ms               float64 `json:"lat_p99_ms"`
	LatP999Ms              float64 `json:"lat_p999_ms"`
	LatWriteP99Ms          float64 `json:"lat_write_p99_ms"`
	IOPSCVPct              float64 `json:"iops_cv_pct"`
	FinishedAt             string  `json:"finished_at"`
}

// WorkerSaturationPayload is emitted by the metrics poller when a worker
// has sustained kind > threshold for >= 5 consecutive samples. The frontend
// surfaces this as a warn badge on the worker tile so the user knows the
// guest may be the bottleneck, not the storage.
type WorkerSaturationPayload struct {
	WorkerID  string  `json:"worker_id"`
	Kind      string  `json:"kind"` // cpu | net | disk
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
}

func MustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
