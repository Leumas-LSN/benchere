package api

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
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
	Engine         string         `json:"engine"`
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

	// Engine selection. Default to fio (v1.11.0+). Validate against the
	// known set so a typo cannot run a phantom engine.
	engine := req.Engine
	if engine == "" {
		engine = string(benchmark.EngineFIO)
	}
	if engine != string(benchmark.EngineFIO) && engine != string(benchmark.EngineElbencho) {
		http.Error(w, fmt.Sprintf("invalid engine: %q (must be fio or elbencho)", engine), http.StatusBadRequest)
		return
	}

	// Engine + toggle gate. v2.0.0+ rejects elbencho jobs unless the
	// operator has flipped enable_legacy_backends in Settings.
	if engine == string(benchmark.EngineElbencho) {
		v, _ := s.DB.GetSetting("enable_legacy_backends")
		if v != "true" {
			http.Error(w, "elbencho engine is disabled. Enable legacy backends in Settings to use it.", http.StatusBadRequest)
			return
		}
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
		Engine:         benchmark.Engine(engine),
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
		JobsDir:     s.Orchestrator.JobsDir,
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
	ProfileName            string  `json:"profile_name"`
	Engine                 string  `json:"engine"`
	SamplesCount           int     `json:"sample_count"`
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
	LatP50Ms               float64 `json:"latency_p50_ms"`
	LatP95Ms               float64 `json:"latency_p95_ms"`
	LatP99Ms               float64 `json:"latency_p99_ms"`
	LatP999Ms              float64 `json:"latency_p999_ms"`
	LatWriteP99Ms          float64 `json:"latency_write_p99_ms"`
	LatencyAvgMs           float64 `json:"latency_avg_ms"` // legacy
	IOPSCVPct              float64 `json:"iops_cv_pct"`
	// Backwards-compat fields some old frontend bits may still read
	MaxIOPSRead        float64 `json:"max_iops_read"`
	MaxIOPSWrite       float64 `json:"max_iops_write"`
	MaxThroughputRead  float64 `json:"max_throughput_read_mbps"`
	MaxThroughputWrite float64 `json:"max_throughput_write_mbps"`
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

	// Prefer the persisted phase summary when available (post-run); fall
	// back to recomputing from results when the job is still streaming
	// or for old jobs that predate v2.0.0.
	summaries, _ := s.DB.ListPhaseSummariesByJob(id)
	if len(summaries) > 0 {
		results := make([]profileResult, 0, len(summaries))
		for _, sum := range summaries {
			// Pull engine and the legacy avg latency from the results table.
			// phase_summaries was kept percentile-only by design, but the API
			// must surface both for verdict thresholds and CSV export.
			var engine string
			var avgLat sql.NullFloat64
			_ = s.DB.QueryRow(
				"SELECT COALESCE(MAX(engine),''), AVG(latency_avg_ms) FROM results WHERE job_id=? AND profile_name=?",
				id, sum.ProfileName,
			).Scan(&engine, &avgLat)
			if engine == "" {
				engine = "fio"
			}
			pr := profileResult{
				ProfileName: sum.ProfileName, Engine: engine, SamplesCount: sum.SamplesCount,
				IOPSReadAvg: sum.IOPSReadAvg, IOPSReadMin: sum.IOPSReadMin, IOPSReadMax: sum.IOPSReadMax,
				IOPSWriteAvg: sum.IOPSWriteAvg, IOPSWriteMin: sum.IOPSWriteMin, IOPSWriteMax: sum.IOPSWriteMax,
				ThroughputReadMBpsAvg:  sum.ThroughputReadMBpsAvg, ThroughputReadMBpsMax: sum.ThroughputReadMBpsMax,
				ThroughputWriteMBpsAvg: sum.ThroughputWriteMBpsAvg, ThroughputWriteMBpsMax: sum.ThroughputWriteMBpsMax,
				LatP50Ms: sum.LatP50Ms, LatP95Ms: sum.LatP95Ms, LatP99Ms: sum.LatP99Ms, LatP999Ms: sum.LatP999Ms,
				LatWriteP99Ms: sum.LatWriteP99Ms, IOPSCVPct: sum.IOPSCVPct,
				MaxIOPSRead: sum.IOPSReadMax, MaxIOPSWrite: sum.IOPSWriteMax,
				MaxThroughputRead: sum.ThroughputReadMBpsMax, MaxThroughputWrite: sum.ThroughputWriteMBpsMax,
			}
			if avgLat.Valid {
				pr.LatencyAvgMs = avgLat.Float64
			}
			results = append(results, pr)
		}
		sort.Slice(results, func(i, j int) bool { return results[i].ProfileName < results[j].ProfileName })
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
		return
	}

	// Fallback: live aggregation across results table.
	rows, err := s.DB.ListResultsByJob(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	byProfile := make(map[string]*profileResult)
	iopsReadByProfile := make(map[string][]float64)
	for _, row := range rows {
		pr, ok := byProfile[row.ProfileName]
		if !ok {
			pr = &profileResult{ProfileName: row.ProfileName, Engine: row.Engine}
			byProfile[row.ProfileName] = pr
		}
		pr.SamplesCount++
		if row.IOPSRead > pr.IOPSReadMax {
			pr.IOPSReadMax = row.IOPSRead
			pr.MaxIOPSRead = row.IOPSRead
		}
		if pr.IOPSReadMin == 0 || row.IOPSRead < pr.IOPSReadMin {
			pr.IOPSReadMin = row.IOPSRead
		}
		if row.IOPSWrite > pr.IOPSWriteMax {
			pr.IOPSWriteMax = row.IOPSWrite
			pr.MaxIOPSWrite = row.IOPSWrite
		}
		if pr.IOPSWriteMin == 0 || row.IOPSWrite < pr.IOPSWriteMin {
			pr.IOPSWriteMin = row.IOPSWrite
		}
		if row.ThroughputReadMBps > pr.ThroughputReadMBpsMax {
			pr.ThroughputReadMBpsMax = row.ThroughputReadMBps
			pr.MaxThroughputRead = row.ThroughputReadMBps
		}
		if row.ThroughputWriteMBps > pr.ThroughputWriteMBpsMax {
			pr.ThroughputWriteMBpsMax = row.ThroughputWriteMBps
			pr.MaxThroughputWrite = row.ThroughputWriteMBps
		}
		// Latency last-wins (steady-state preferred).
		if row.LatencyP50Ms > 0 {
			pr.LatP50Ms = row.LatencyP50Ms
		}
		if row.LatencyP95Ms > 0 {
			pr.LatP95Ms = row.LatencyP95Ms
		}
		if row.LatencyP99Ms > 0 {
			pr.LatP99Ms = row.LatencyP99Ms
		}
		if row.LatencyP999Ms > 0 {
			pr.LatP999Ms = row.LatencyP999Ms
		}
		if row.LatencyWriteP99Ms > 0 {
			pr.LatWriteP99Ms = row.LatencyWriteP99Ms
		}
		pr.LatencyAvgMs += row.LatencyAvgMs
		iopsReadByProfile[row.ProfileName] = append(iopsReadByProfile[row.ProfileName], row.IOPSRead)
	}
	for name, pr := range byProfile {
		n := float64(pr.SamplesCount)
		if n == 0 {
			continue
		}
		pr.LatencyAvgMs = pr.LatencyAvgMs / n
		// Compute averages across rows.
		var sumR, sumW, sumTR, sumTW float64
		for _, row := range rows {
			if row.ProfileName != name {
				continue
			}
			sumR += row.IOPSRead
			sumW += row.IOPSWrite
			sumTR += row.ThroughputReadMBps
			sumTW += row.ThroughputWriteMBps
		}
		pr.IOPSReadAvg = sumR / n
		pr.IOPSWriteAvg = sumW / n
		pr.ThroughputReadMBpsAvg = sumTR / n
		pr.ThroughputWriteMBpsAvg = sumTW / n
		// CV%
		if pr.IOPSReadAvg > 0 {
			samples := iopsReadByProfile[name]
			var sumSq float64
			for _, x := range samples {
				d := x - pr.IOPSReadAvg
				sumSq += d * d
			}
			if len(samples) > 1 {
				stdv := math.Sqrt(sumSq / float64(len(samples)-1))
				pr.IOPSCVPct = stdv / pr.IOPSReadAvg * 100
			}
		}
	}
	results := make([]profileResult, 0, len(byProfile))
	for _, pr := range byProfile {
		results = append(results, *pr)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ProfileName < results[j].ProfileName })
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
		"timestamp", "profile_name", "engine",
		"iops_read", "iops_write",
		"throughput_read_mbps", "throughput_write_mbps",
		"latency_avg_ms",
		"latency_read_avg_ms", "latency_write_avg_ms",
		"latency_p50_ms", "latency_p95_ms", "latency_p99_ms",
		"latency_p999_ms", "latency_write_p99_ms",
	})
	for _, row := range rows {
		_ = cw.Write([]string{
			row.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			row.ProfileName, row.Engine,
			fmt.Sprintf("%.2f", row.IOPSRead),
			fmt.Sprintf("%.2f", row.IOPSWrite),
			fmt.Sprintf("%.2f", row.ThroughputReadMBps),
			fmt.Sprintf("%.2f", row.ThroughputWriteMBps),
			fmt.Sprintf("%.4f", row.LatencyAvgMs),
			fmt.Sprintf("%.4f", row.LatencyReadAvgMs),
			fmt.Sprintf("%.4f", row.LatencyWriteAvgMs),
			fmt.Sprintf("%.4f", row.LatencyP50Ms),
			fmt.Sprintf("%.4f", row.LatencyP95Ms),
			fmt.Sprintf("%.4f", row.LatencyP99Ms),
			fmt.Sprintf("%.4f", row.LatencyP999Ms),
			fmt.Sprintf("%.4f", row.LatencyWriteP99Ms),
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		log.Printf("exportJobCSV: flush error for job %s: %v", id, err)
	}
}
