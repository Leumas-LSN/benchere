package api

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/Leumas-LSN/benchere/internal/stress"
	"github.com/go-chi/chi/v5"
)

type createJobRequest struct {
	Name           string         `json:"name"`
	ClientName     string         `json:"client_name"`
	Mode           string         `json:"mode"`
	ProxmoxNodes   []string       `json:"proxmox_nodes"`
	WorkersPerNode int            `json:"workers_per_node"`
	WorkerCPU      int            `json:"worker_cpu"`
	WorkerRAMMB    int            `json:"worker_ram_mb"`
	OSDiskGB       int            `json:"os_disk_gb"`
	DataDisks      int            `json:"data_disks"`
	DataDiskGB     int            `json:"data_disk_gb"`
	StoragePool    string         `json:"storage_pool,omitempty"`
	Profiles       []string       `json:"profiles"`
	StressConfig   *stress.Config `json:"stress_config,omitempty"`
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.DB.ListJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []db.Job{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.DB.GetJob(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (s *Server) createJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.ProxmoxNodes) == 0 {
		http.Error(w, "proxmox_nodes must be a non-empty list", http.StatusBadRequest)
		return
	}
	if req.WorkersPerNode < 1 {
		http.Error(w, "workers_per_node must be >= 1", http.StatusBadRequest)
		return
	}

	proxmoxURL, _ := s.DB.GetSetting("proxmox_url")
	proxmoxToken, _ := s.DB.GetSetting("proxmox_token")
	storagePool, _ := s.DB.GetSetting("storage_pool")
	if req.StoragePool != "" {
		storagePool = req.StoragePool
	}
	imageStorage, _ := s.DB.GetSetting("image_storage")

	// Validate that each requested node exists in the cluster.
	freshClient := proxmox.NewClient(proxmoxURL, proxmoxToken)
	liveNodes, err := freshClient.GetNodes(r.Context())
	if err != nil {
		http.Error(w, "list nodes: "+err.Error(), http.StatusBadGateway)
		return
	}
	liveSet := map[string]bool{}
	for _, n := range liveNodes {
		liveSet[n] = true
	}
	for _, n := range req.ProxmoxNodes {
		if !liveSet[n] {
			http.Error(w, fmt.Sprintf("unknown proxmox node: %q", n), http.StatusBadRequest)
			return
		}
	}

	if req.WorkerCPU == 0 {
		req.WorkerCPU = 2
	}
	if req.WorkerRAMMB == 0 {
		req.WorkerRAMMB = 2048
	}
	if req.OSDiskGB == 0 {
		req.OSDiskGB = 20
	}
	if req.DataDisks == 0 {
		req.DataDisks = 1
	}
	if req.DataDiskGB == 0 {
		req.DataDiskGB = 20
	}
	cfg := benchmark.JobConfig{
		Name:           req.Name,
		ClientName:     req.ClientName,
		Mode:           benchmark.Mode(req.Mode),
		ProxmoxNodes:   req.ProxmoxNodes,
		WorkersPerNode: req.WorkersPerNode,
		WorkerCPU:      req.WorkerCPU,
		WorkerRAMMB:    req.WorkerRAMMB,
		OSDiskGB:       req.OSDiskGB,
		DataDisks:      req.DataDisks,
		DataDiskGB:     req.DataDiskGB,
		StoragePool:    storagePool,
		CloudImageURL:  "http://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2",
		ImageStorage:   imageStorage,
		Profiles:       req.Profiles,
		StressConfig:   req.StressConfig,
	}

	job := benchmark.NewJob(cfg)
	if err := s.DB.CreateJob(job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orch := &benchmark.Orchestrator{
		DB:          s.DB,
		Proxmox:     freshClient,
		Ansible:     s.Orchestrator.Ansible,
		Stress:      s.Orchestrator.Stress,
		Hub:         s.Hub,
		SSHKey:      s.Orchestrator.SSHKey,
		ProfilesDir: s.Orchestrator.ProfilesDir,
		OutputDir:   s.Orchestrator.OutputDir,
	}

	go func() {
		_ = orch.RunExisting(context.Background(), job, cfg)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": job.ID})
}

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.DB.GetJob(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if job.Status == "done" || job.Status == "failed" || job.Status == "cancelled" {
		http.Error(w, "job already finished", http.StatusConflict)
		return
	}
	if err := s.Orchestrator.CancelJob(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listWorkers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workers, err := s.DB.ListWorkersByJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if workers == nil {
		workers = []db.Worker{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}

func (s *Server) clearHistory(w http.ResponseWriter, r *http.Request) {
	if err := s.DB.ClearHistory(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type profileResult struct {
	ProfileName        string  `json:"profile_name"`
	MaxIOPSRead        float64 `json:"max_iops_read"`
	MaxIOPSWrite       float64 `json:"max_iops_write"`
	AvgIOPSRead        float64 `json:"avg_iops_read"`
	AvgIOPSWrite       float64 `json:"avg_iops_write"`
	MaxThroughputRead  float64 `json:"max_throughput_read_mbps"`
	MaxThroughputWrite float64 `json:"max_throughput_write_mbps"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	SampleCount        int     `json:"sample_count"`
}

func (s *Server) getJobResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.DB.GetJob(id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	rows, err := s.DB.ListResultsByJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	byProfile := make(map[string]*profileResult)
	sumRead := make(map[string]float64)
	sumWrite := make(map[string]float64)
	sumLat := make(map[string]float64)
	for _, row := range rows {
		pr, ok := byProfile[row.ProfileName]
		if !ok {
			pr = &profileResult{ProfileName: row.ProfileName}
			byProfile[row.ProfileName] = pr
		}
		pr.SampleCount++
		if row.IOPSRead > pr.MaxIOPSRead {
			pr.MaxIOPSRead = row.IOPSRead
		}
		if row.IOPSWrite > pr.MaxIOPSWrite {
			pr.MaxIOPSWrite = row.IOPSWrite
		}
		if row.ThroughputReadMBps > pr.MaxThroughputRead {
			pr.MaxThroughputRead = row.ThroughputReadMBps
		}
		if row.ThroughputWriteMBps > pr.MaxThroughputWrite {
			pr.MaxThroughputWrite = row.ThroughputWriteMBps
		}
		sumRead[row.ProfileName] += row.IOPSRead
		sumWrite[row.ProfileName] += row.IOPSWrite
		sumLat[row.ProfileName] += row.LatencyAvgMs
	}
	for name, pr := range byProfile {
		n := float64(pr.SampleCount)
		pr.AvgIOPSRead = sumRead[name] / n
		pr.AvgIOPSWrite = sumWrite[name] / n
		pr.AvgLatencyMs = sumLat[name] / n
	}

	results := make([]profileResult, 0, len(byProfile))
	for _, pr := range byProfile {
		results = append(results, *pr)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ProfileName < results[j].ProfileName
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) exportJobCSV(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.DB.GetJob(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	rows, err := s.DB.ListResultsByJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("benchere_%s_%s.csv", job.Name, job.CreatedAt.Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"timestamp", "profile_name",
		"iops_read", "iops_write",
		"throughput_read_mbps", "throughput_write_mbps",
		"latency_avg_ms", "latency_p99_ms",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			row.ProfileName,
			fmt.Sprintf("%.2f", row.IOPSRead),
			fmt.Sprintf("%.2f", row.IOPSWrite),
			fmt.Sprintf("%.2f", row.ThroughputReadMBps),
			fmt.Sprintf("%.2f", row.ThroughputWriteMBps),
			fmt.Sprintf("%.4f", row.LatencyAvgMs),
			fmt.Sprintf("%.4f", row.LatencyP99Ms),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		log.Printf("exportJobCSV: flush error for job %s: %v", id, err)
	}
}
