package ws

import "encoding/json"

type EventType string

const (
	EventElbenchoMetric   EventType = "elbencho_metric"
	EventProxmoxNode      EventType = "proxmox_node"
	EventProxmoxVM        EventType = "proxmox_vm"
	EventJobStatus        EventType = "job_status"
	EventProvisioningStep EventType = "provisioning_step"
)

type Event struct {
	Type    EventType       `json:"type"`
	JobID   string          `json:"job_id"`
	Payload json.RawMessage `json:"payload"`
}

type ElbenchoMetricPayload struct {
	ProfileName         string  `json:"profile_name"`
	IOPSRead            float64 `json:"iops_read"`
	IOPSWrite           float64 `json:"iops_write"`
	ThroughputReadMBps  float64 `json:"throughput_read_mbps"`
	ThroughputWriteMBps float64 `json:"throughput_write_mbps"`
	LatencyAvgMs        float64 `json:"latency_avg_ms"`
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
	Step     string  `json:"step"`     // "vm_creating", "vm_ready", "ansible_start", "ansible_done", "workers_ready"
	Detail   string  `json:"detail"`   // human-readable message
	Progress float64 `json:"progress"` // 0.0 to 1.0
	JobID    string  `json:"job_id"`
}

func MustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
