package db

import "time"

type Job struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	ClientName   string     `json:"client_name"`
	Status       string     `json:"status"`
	Mode         string     `json:"mode"`
	Engine       string     `json:"engine"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

type Worker struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	VMID        int    `json:"vm_id"`
	ProxmoxNode string `json:"proxmox_node"`
	IP          string `json:"ip"`
	Status      string `json:"status"`
}

type BenchmarkProfile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Engine     string `json:"engine"`
	ConfigJSON string `json:"config_json"`
}

type Result struct {
	ID                  string
	JobID               string
	ProfileName         string
	Engine              string
	Timestamp           time.Time
	IOPSRead            float64
	IOPSWrite           float64
	ThroughputReadMBps  float64
	ThroughputWriteMBps float64
	LatencyAvgMs        float64
	LatencyP99Ms        float64 // 99th percentile latency in milliseconds
}

type ProxmoxSnapshot struct {
	ID        string
	JobID     string
	Timestamp time.Time
	NodeName  string
	CPUPct    float64
	RAMPct    float64
	LoadAvg   float64
}

type ProxmoxVMSnapshot struct {
	ID        string
	JobID     string
	Timestamp time.Time
	WorkerID  string
	CPUPct    float64
}
