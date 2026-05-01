package db

import "time"

type Job struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	ClientName      string     `json:"client_name"`
	Status          string     `json:"status"`
	Mode            string     `json:"mode"`
	Engine          string     `json:"engine"`
	ErrorMessage    string     `json:"error_message,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	// v2.0.0: hardware fields for methodology rendering
	WorkerCPU       int    `json:"worker_cpu,omitempty"`
	WorkerRAMMB     int    `json:"worker_ram_mb,omitempty"`
	DataDisks       int    `json:"data_disks,omitempty"`
	DataDiskGB      int    `json:"data_disk_gb,omitempty"`
	StoragePool     string `json:"storage_pool,omitempty"`
	ProxmoxNodesCSV string `json:"proxmox_nodes_csv,omitempty"`
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
	LatencyP99Ms        float64

	// v2.0.0: extended percentile picture for fio. Zero when engine
	// does not provide them (elbencho fills only avg + read p99).
	LatencyReadAvgMs  float64
	LatencyWriteAvgMs float64
	LatencyP50Ms      float64
	LatencyP95Ms      float64
	LatencyP999Ms     float64
	LatencyWriteP99Ms float64
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

type PhaseSummary struct {
	ID                     string
	JobID                  string
	ProfileName            string
	SamplesCount           int
	IOPSReadAvg            float64
	IOPSReadMin            float64
	IOPSReadMax            float64
	IOPSWriteAvg           float64
	IOPSWriteMin           float64
	IOPSWriteMax           float64
	ThroughputReadMBpsAvg  float64
	ThroughputReadMBpsMax  float64
	ThroughputWriteMBpsAvg float64
	ThroughputWriteMBpsMax float64
	LatP50Ms               float64
	LatP95Ms               float64
	LatP99Ms               float64
	LatP999Ms              float64
	LatWriteP99Ms          float64
	IOPSCVPct              float64 // coefficient of variation = stddev/mean x 100
	FinishedAt             time.Time
}
