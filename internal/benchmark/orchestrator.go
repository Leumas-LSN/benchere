package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Leumas-LSN/benchere/internal/ansible"
	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/elbencho"
	"github.com/Leumas-LSN/benchere/internal/fio"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/Leumas-LSN/benchere/internal/stress"
	"github.com/Leumas-LSN/benchere/internal/ws"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// Version is the binary version stamp, mirrored from cmd/benchere/main.go's
// Version var so artifact markers can record which build wrote them. Set at
// startup via benchmark.Version = main.Version. Defaults to "dev".
var Version = "dev"

type Orchestrator struct {
	DB          *db.DB
	Proxmox     *proxmox.Client
	Ansible     *ansible.Runner
	Stress      *stress.Runner
	Hub         *ws.Hub
	SSHKey      string
	ProfilesDir string
	OutputDir   string

	// JobsDir is the per-job artifact root, conventionally
	// /var/lib/benchere/jobs/. The orchestrator writes raw stdout/stderr,
	// captured CSVs, ansible logs, worker sysinfo and provisioning logs to
	// JobsDir/{jobID}/... so the debug bundle endpoint can read them later.
	// Empty disables artifact capture entirely.
	JobsDir string

	// provLog, when non-nil, receives one line per provisioning step in
	// addition to the WebSocket broadcast. Closed at end of RunExisting.
	provLog *os.File
}

// jobDir returns the per-job artifact directory. Returns empty when JobsDir
// is unset, callers must treat that as a no-op.
func (o *Orchestrator) jobDir(jobID string) string {
	if o.JobsDir == "" {
		return ""
	}
	return filepath.Join(o.JobsDir, jobID)
}

func (o *Orchestrator) Run(ctx context.Context, cfg JobConfig) error {
	job := NewJob(cfg)
	if err := o.DB.CreateJob(job); err != nil {
		return err
	}
	return o.RunExisting(ctx, job, cfg)
}

// persistMetrics drains a metric channel onto the DB and broadcasts each
// sample on the WebSocket. The shape is identical between engines (only
// the source of truth differs) so the same goroutine handles both.
//
// The loop exits only when the channel is closed by its source. We do
// not exit on ctx.Done because the upstream cancellation already causes
// the producer to exit and close the channel, and racing the close with
// ctx.Done can drop buffered metrics. fio in --client mode sends every
// status snapshot in a tight burst at the end of cmd.Wait, so a premature
// ctx.Done branch was discarding most of them before they hit the DB.
func (o *Orchestrator) persistMetrics(ctx context.Context, jobID, profileName, engine string, ch <-chan storageMetric) {
	_ = ctx // retained for signature stability; cancellation is handled upstream
	for m := range ch {
		r := db.Result{
			ID: uuid.NewString(), JobID: jobID, ProfileName: profileName,
			Engine:    engine,
			Timestamp: m.Timestamp, IOPSRead: m.IOPSRead, IOPSWrite: m.IOPSWrite,
			ThroughputReadMBps: m.ThroughputReadMBps, ThroughputWriteMBps: m.ThroughputWriteMBps,
			LatencyAvgMs: m.LatencyAvgMs, LatencyP99Ms: m.LatencyP99Ms,
		}
		_ = o.DB.InsertResult(r)
		o.emit(jobID, ws.EventElbenchoMetric, ws.ElbenchoMetricPayload{
			ProfileName:         profileName,
			IOPSRead:            m.IOPSRead,
			IOPSWrite:           m.IOPSWrite,
			ThroughputReadMBps:  m.ThroughputReadMBps,
			ThroughputWriteMBps: m.ThroughputWriteMBps,
			LatencyAvgMs:        m.LatencyAvgMs,
		})
	}
}

// storageMetric is the engine-agnostic shape used by persistMetrics.
type storageMetric struct {
	Timestamp           time.Time
	IOPSRead            float64
	IOPSWrite           float64
	ThroughputReadMBps  float64
	ThroughputWriteMBps float64
	LatencyAvgMs        float64
	LatencyP99Ms        float64
}

func storageMetricFromElbencho(m elbencho.Metric) storageMetric {
	return storageMetric{
		Timestamp:           m.Timestamp,
		IOPSRead:            m.IOPSRead,
		IOPSWrite:           m.IOPSWrite,
		ThroughputReadMBps:  m.ThroughputReadMBps,
		ThroughputWriteMBps: m.ThroughputWriteMBps,
		LatencyAvgMs:        m.LatencyAvgMs,
	}
}

func storageMetricFromFio(m fio.Metric) storageMetric {
	return storageMetric{
		Timestamp:           m.Timestamp,
		IOPSRead:            m.IOPSRead,
		IOPSWrite:           m.IOPSWrite,
		ThroughputReadMBps:  m.ThroughputReadMBps,
		ThroughputWriteMBps: m.ThroughputWriteMBps,
		LatencyAvgMs:        m.LatencyAvgMs,
		LatencyP99Ms:        m.LatencyP99Ms,
	}
}

func (o *Orchestrator) emit(jobID string, t ws.EventType, payload interface{}) {
	o.Hub.Broadcast(ws.Event{Type: t, JobID: jobID, Payload: ws.MustMarshal(payload)})
}

func (o *Orchestrator) emitProvStep(jobID, step, detail string, progress float64) {
	o.emit(jobID, ws.EventProvisioningStep, ws.ProvisioningStepPayload{
		Step:     step,
		Detail:   detail,
		Progress: progress,
		JobID:    jobID,
	})
	if o.provLog != nil {
		fmt.Fprintf(o.provLog, "%s %s progress=%.2f %s\n",
			time.Now().UTC().Format(time.RFC3339), step, progress, detail)
	}
}

// jsonMarshalIndent is a tiny shim so RunExisting does not pull encoding/json
// import directly via the orchestrator file (cycles already minimal here).
func jsonMarshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func (o *Orchestrator) fail(jobID string, err error) error {
	log.Printf("[job %s] FAILED: %v", jobID, err)
	_ = o.DB.FailJob(jobID, err.Error())
	o.emit(jobID, ws.EventJobStatus, ws.JobStatusPayload{Status: "failed"})
	return err
}

// cleanup stops and deletes the VMs of the given workers, on whatever node
// each worker lives on (worker.ProxmoxNode). Used both by the defer in
// RunExisting and by CancelJob / RecoverOrphanedJobs.
func (o *Orchestrator) cleanup(ctx context.Context, workers []db.Worker) {
	for _, w := range workers {
		if w.VMID == 0 || w.ProxmoxNode == "" {
			continue
		}
		_ = o.Proxmox.StopVM(ctx, w.ProxmoxNode, w.VMID)
		_ = o.Proxmox.DeleteVM(ctx, w.ProxmoxNode, w.VMID)
		_ = o.DB.UpdateWorkerStatus(w.ID, "done")
	}
}

func buildTargets(dataDisks int) []string {
	// Use Proxmox/QEMU stable by-id paths instead of /dev/sd* which can
	// reorder across boots when multiple disks are present (the kernel
	// probes virtio-scsi LUNs in nondeterministic order, so /dev/sdb
	// might be the OS disk on one worker and the data disk on another).
	// scsi0 is the OS disk, scsi1+ are the data disks added by CreateVM.
	targets := make([]string, dataDisks)
	for i := 0; i < dataDisks; i++ {
		targets[i] = fmt.Sprintf("/dev/disk/by-id/scsi-0QEMU_QEMU_HARDDISK_drive-scsi%d", i+1)
	}
	return targets
}

func readFile(path string) string {
	b, _ := os.ReadFile(path)
	return strings.TrimSpace(string(b))
}

// RunExisting runs an already-persisted job (skipping job creation).
// Used by the API handler which creates the job first to return the ID synchronously.
func (o *Orchestrator) RunExisting(ctx context.Context, job db.Job, cfg JobConfig) error {
	// Set up the per-job artifact directory and persist the JobConfig as the
	// first thing in the run. This means the debug bundle has the exact
	// in-memory config used by the orchestrator, including fields that are
	// not in the db.Job row (worker sizing, profiles, storage pool).
	jd := o.jobDir(job.ID)
	var mkdirErr error
	if jd != "" {
		mkdirErr = os.MkdirAll(jd, 0o755)
	}
	log.Printf("[artifact] runexisting_start: jd=%q mkdir_err=%v", jd, mkdirErr)
	if jd != "" {
		if mkdirErr != nil {
			log.Printf("[job %s] mkdir jobdir: %v", job.ID, mkdirErr)
		} else {
			markerPath := filepath.Join(jd, "INIT.txt")
			markerContent := fmt.Sprintf("job=%s started=%s binary_version=%s engine=%s\n",
				job.ID, time.Now().UTC().Format(time.RFC3339), Version, cfg.Engine)
			if err := os.WriteFile(markerPath, []byte(markerContent), 0o644); err != nil {
				log.Printf("[artifact] init marker write failed: %v", err)
			} else {
				log.Printf("[artifact] init marker written at %s", markerPath)
			}
			if data, err := jsonMarshalIndent(cfg); err == nil {
				_ = os.WriteFile(filepath.Join(jd, "config.json"), data, 0o644)
			}
			provLog, err := os.OpenFile(filepath.Join(jd, "provisioning.log"),
				os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err == nil {
				o.provLog = provLog
				defer func() {
					if o.provLog != nil {
						o.provLog.Close()
						o.provLog = nil
					}
				}()
			}
		}
	}

	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "provisioning", Phase: "creating_vms"})
	if err := o.DB.UpdateJobStatus(job.ID, "provisioning"); err != nil {
		return o.fail(job.ID, err)
	}

	if len(cfg.ProxmoxNodes) == 0 {
		return o.fail(job.ID, fmt.Errorf("no proxmox nodes selected"))
	}
	if cfg.WorkersPerNode <= 0 {
		return o.fail(job.ID, fmt.Errorf("workers_per_node must be >= 1"))
	}

	// 1. Cloud image - must be importable on each selected node. With a
	//    shared storage (Ceph/NFS) one EnsureCloudImage is enough; with a
	//    local-per-node image storage we must import on each node. We loop:
	//    EnsureCloudImage is idempotent on shared storage (instant return).
	imageStorage := cfg.ImageStorage
	if imageStorage == "" {
		imageStorage = "local"
	}
	var imageVol string
	for _, node := range cfg.ProxmoxNodes {
		v, err := o.Proxmox.EnsureCloudImage(ctx, node, imageStorage, cfg.CloudImageURL)
		if err != nil {
			return o.fail(job.ID, fmt.Errorf("cloud image on node %s: %w", node, err))
		}
		imageVol = v // same volid string regardless of node for shared storages
	}

	networkBridge, _ := o.DB.GetSetting("network_bridge")

	totalWorkers := len(cfg.ProxmoxNodes) * cfg.WorkersPerNode
	specs := SplitWorkers(cfg.ProxmoxNodes, cfg.WorkersPerNode)

	// 2. Static IP allocation (aggregated across all selected nodes).
	var staticIPs []net.IP
	var pool IPPool
	if poolStr, _ := o.DB.GetSetting("worker_ip_pool"); poolStr != "" {
		cidrStr, _ := o.DB.GetSetting("worker_cidr")
		cidr, _ := strconv.Atoi(cidrStr)
		gateway, _ := o.DB.GetSetting("worker_gateway")
		p, err := ParseIPPool(poolStr, cidr, gateway)
		if err != nil {
			return o.fail(job.ID, fmt.Errorf("worker network: %w", err))
		}
		pool = p
		staticIPs, err = AllocateIPs(ctx, o.Proxmox, cfg.ProxmoxNodes, pool, totalWorkers)
		if err != nil {
			return o.fail(job.ID, err)
		}
	}

	// 3. Allocate VMIDs serially (avoid races on Proxmox NextVMID).
	for i := range specs {
		vmid, err := o.Proxmox.NextVMID(ctx)
		if err != nil {
			return o.fail(job.ID, fmt.Errorf("allocate vmid: %w", err))
		}
		specs[i].VMID = vmid
	}

	// 4. Pre-insert worker rows so cleanup can find them on cancellation.
	createdWorkers := make([]db.Worker, len(specs))
	for i, spec := range specs {
		w := db.Worker{
			ID:          uuid.NewString(),
			JobID:       job.ID,
			VMID:        spec.VMID,
			ProxmoxNode: spec.Node,
			Status:      "provisioning",
		}
		if err := o.DB.CreateWorker(w); err != nil {
			return o.fail(job.ID, fmt.Errorf("create worker row: %w", err))
		}
		createdWorkers[i] = w
	}
	defer func() { o.cleanup(context.Background(), createdWorkers) }()

	o.emitProvStep(job.ID, "vm_creating", "Creation des machines virtuelles...", 0.05)

	// 5. Parallel CreateVM + StartVM + WaitForSSH with concurrency cap = 8.
	g, gctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, 8)
	workerIPs := make([]string, len(specs))
	var doneCount int64

	for i := range specs {
		i := i
		spec := specs[i]
		g.Go(func() error {
			sem <- struct{}{}
			defer func() { <-sem }()

			ipConfig := ""
			var assignedIP string
			if len(staticIPs) > 0 {
				assignedIP = staticIPs[i].String()
				ipConfig = pool.IPConfig(staticIPs[i])
			}

			if _, err := o.Proxmox.CreateVM(gctx, proxmox.VMCreateParams{
				Node:        spec.Node,
				VMID:        spec.VMID,
				Name:        fmt.Sprintf("benchere-worker-%d", spec.Index+1),
				Cores:       cfg.WorkerCPU,
				MemoryMB:    cfg.WorkerRAMMB,
				OSDiskGB:    cfg.OSDiskGB,
				DataDisks:   cfg.DataDisks,
				DataDiskGB:  cfg.DataDiskGB,
				StoragePool: cfg.StoragePool,
				SSHKey:      readFile(o.SSHKey + ".pub"),
				ImagePath:   imageVol,
				Bridge:      networkBridge,
				IPConfig:    ipConfig,
			}); err != nil {
				return fmt.Errorf("create vm %d on %s: %w", spec.VMID, spec.Node, err)
			}
			if err := o.Proxmox.StartVM(gctx, spec.Node, spec.VMID); err != nil {
				return fmt.Errorf("start vm %d on %s: %w", spec.VMID, spec.Node, err)
			}

			var ip string
			if assignedIP != "" {
				if err := o.Proxmox.WaitForSSH(gctx, assignedIP); err != nil {
					return fmt.Errorf("wait ssh %s: %w", assignedIP, err)
				}
				ip = assignedIP
			} else {
				discovered, err := o.Proxmox.WaitForIP(gctx, spec.Node, spec.VMID)
				if err != nil {
					return fmt.Errorf("wait ip vm %d: %w", spec.VMID, err)
				}
				if err := o.Proxmox.InjectSSHKey(gctx, spec.Node, spec.VMID, readFile(o.SSHKey+".pub")); err != nil {
					return fmt.Errorf("inject ssh key vm %d: %w", spec.VMID, err)
				}
				ip = discovered
			}

			_ = o.DB.UpdateWorkerIP(createdWorkers[i].ID, ip)
			workerIPs[i] = ip
			n := atomic.AddInt64(&doneCount, 1)
			o.emitProvStep(job.ID, "vm_creating",
				fmt.Sprintf("VM %d/%d creee", n, totalWorkers),
				0.05+0.35*float64(n)/float64(totalWorkers),
			)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return o.fail(job.ID, err)
	}

	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "provisioning", Phase: "ansible"})
	o.emitProvStep(job.ID, "ansible_start", "Deploiement Ansible (elbencho + fio + stress-ng)...", 0.60)
	targets := make([]ansible.WorkerTarget, len(workerIPs))
	for i, ip := range workerIPs {
		targets[i] = ansible.WorkerTarget{IP: ip}
	}
	ansibleOutDir := ""
	ajd := o.jobDir(job.ID)
	if ajd != "" {
		ansibleOutDir = filepath.Join(ajd, "ansible")
	}
	log.Printf("[artifact] ansible_dir: jd=%q mkdir_err=%v", ajd, "n/a")
	if err := o.Ansible.ProvisionWorkers(ctx, targets, ansibleOutDir); err != nil {
		diag := ""
		if len(workerIPs) > 0 {
			diag = sshDiagnose(o.SSHKey, workerIPs[0])
		}
		return o.fail(job.ID, fmt.Errorf("ansible: %w\n\n--- ssh -vvv to %s ---\n%s", err, workerIPs[0], diag))
	}
	o.emitProvStep(job.ID, "ansible_done", "Ansible termine, verification des workers...", 0.90)
	for _, w := range createdWorkers {
		_ = o.DB.UpdateWorkerStatus(w.ID, "ready")
	}
	o.emitProvStep(job.ID, "workers_ready", "Tous les workers sont prets. Demarrage du benchmark...", 1.0)

	_ = o.DB.UpdateJobStatus(job.ID, "running")

	workers, _ := o.DB.ListWorkersByJob(job.ID)
	metricsCtx, cancelMetrics := context.WithCancel(ctx)
	go PollProxmoxMetrics(metricsCtx, job.ID, o.Proxmox, o.DB, o.Hub, workers)
	defer cancelMetrics()

	if cfg.Mode == ModeCPU || cfg.Mode == ModeMixed {
		if cfg.StressConfig != nil {
			if err := o.Stress.StartOnWorkers(ctx, workerIPs, *cfg.StressConfig); err != nil {
				return o.fail(job.ID, err)
			}
		}
	}

	if cfg.Mode == ModeStorage || cfg.Mode == ModeMixed {
		engine := cfg.Engine
		if engine == "" {
			engine = EngineFIO
		}

		artifactDir := ""
		ejd := o.jobDir(job.ID)
		var subdirErr error
		if ejd != "" {
			artifactDir = filepath.Join(ejd, string(engine))
			subdirErr = os.MkdirAll(artifactDir, 0o755)
		}
		log.Printf("[artifact] %s_dir: jd=%q mkdir_err=%v", engine, ejd, subdirErr)
		if ejd != "" {
			if subdirErr != nil {
				log.Printf("[job %s] mkdir engine dir: %v", job.ID, subdirErr)
				artifactDir = ""
			} else {
				if engine == EngineFIO {
					fio.CaptureVersion(ctx, artifactDir)
				} else {
					elbencho.CaptureVersion(ctx, artifactDir)
				}
			}
		}

		switch engine {
		case EngineFIO:
			if err := o.runFIOPhase(ctx, job, cfg, workerIPs, artifactDir); err != nil {
				return o.fail(job.ID, err)
			}
		default:
			if err := o.runElbenchoPhase(ctx, job, cfg, workerIPs, artifactDir); err != nil {
				return o.fail(job.ID, err)
			}
		}
	}

	if cfg.Mode == ModeCPU || cfg.Mode == ModeMixed {
		_ = o.Stress.StopOnWorkers(ctx, workerIPs)
	}

	// Capture per-worker sysinfo and Proxmox config before the deferred
	// cleanup destroys the VMs. Bounded by a 2 minute deadline so a single
	// hung worker cannot block the job from finishing. Best-effort: any
	// failure is logged to {jobDir}/errors.log and the job still ends ok.
	// Re-read worker rows from DB so we get the IPs persisted by the
	// provisioning goroutine.
	{
		artifactsCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		freshWorkers, err := o.DB.ListWorkersByJob(job.ID)
		if err != nil || len(freshWorkers) == 0 {
			freshWorkers = createdWorkers
		}
		o.CollectWorkerArtifacts(artifactsCtx, job.ID, freshWorkers)
		cancel()
	}

	_ = o.DB.FinishJob(job.ID, "done")
	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "done"})
	return nil
}

// runElbenchoPhase preserves the v1.10.x storage flow (prefill + per profile
// elbencho run with TailCSV streaming). Engine selector branches into here
// when cfg.Engine == "elbencho".
func (o *Orchestrator) runElbenchoPhase(ctx context.Context, job db.Job, cfg JobConfig, workerIPs []string, artifactDir string) error {
	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "running", Phase: "prefill"})
	o.emitProvStep(job.ID, "prefill_start",
		fmt.Sprintf("Prefill des data disks (%d GB/worker) pour eviter les zero-block reads...", cfg.DataDiskGB), 1.0)
	prefillTargets := buildTargets(cfg.DataDisks)
	prefillStart := time.Now()
	if err := elbencho.Prefill(ctx, workerIPs, prefillTargets, cfg.DataDiskGB, artifactDir); err != nil {
		return fmt.Errorf("prefill: %w", err)
	}
	prefillDur := time.Since(prefillStart)
	expectedBytes := int64(cfg.DataDiskGB) * 1024 * 1024 * 1024 * int64(len(workerIPs))
	log.Printf("[prefill] completed in %s, expected %d bytes total (%d GB x %d workers)",
		prefillDur, expectedBytes, cfg.DataDiskGB, len(workerIPs))
	if prefillDur < 10*time.Second {
		log.Printf("[prefill] WARNING: completed too fast for %d GB on %d workers, allocation may be incomplete",
			cfg.DataDiskGB, len(workerIPs))
	}
	o.emitProvStep(job.ID, "prefill_done", "Prefill termine, demarrage du benchmark...", 1.0)

	for _, profileName := range cfg.Profiles {
		o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "running", Phase: profileName})

		profile, err := o.DB.GetProfileByNameAndEngine(profileName, "elbencho")
		if err != nil {
			return fmt.Errorf("profile %s: %w", profileName, err)
		}

		elbenchoCfg, err := elbencho.ProfileToConfig(profile.ConfigJSON)
		if err != nil {
			return fmt.Errorf("profile %s config: %w", profileName, err)
		}
		profileFile := filepath.Join(o.ProfilesDir, profileName+".elbencho")
		if err := os.WriteFile(profileFile, []byte(elbenchoCfg), 0644); err != nil {
			return err
		}

		liveCSV := filepath.Join(o.OutputDir, fmt.Sprintf("live_%s_%s.csv", job.ID, profileName))
		finalCSV := filepath.Join(o.OutputDir, fmt.Sprintf("results_%s_%s.csv", job.ID, profileName))

		metricCtx, cancelMetricCtx := context.WithCancel(ctx)
		elbCh := make(chan elbencho.Metric, 100)
		convertedCh := make(chan storageMetric, 100)
		go elbencho.TailCSV(metricCtx, liveCSV, elbCh)
		go func() {
			defer close(convertedCh)
			for m := range elbCh {
				convertedCh <- storageMetricFromElbencho(m)
			}
		}()
		persistDone := make(chan struct{})
		go func() {
			o.persistMetrics(metricCtx, job.ID, profileName, "elbencho", convertedCh)
			close(persistDone)
		}()

		runTargets := buildTargets(cfg.DataDisks)
		err = elbencho.Run(ctx, elbencho.RunConfig{
			Hosts:       workerIPs,
			ConfigFile:  profileFile,
			Targets:     runTargets,
			LiveCSVPath: liveCSV,
			CSVPath:     finalCSV,
			Label:       profileName,
			OutputDir:   artifactDir,
		})
		// elbencho.Run returned, but TailCSV is still blocked on metricCtx.
		// Cancel it so the tail goroutine exits and closes its channel,
		// then wait for persistMetrics to drain the buffer before moving
		// on to the next profile.
		cancelMetricCtx()
		<-persistDone
		if err != nil {
			return err
		}
	}
	return nil
}

// runFIOPhase runs prefill via fio then iterates over the selected profiles.
// fio's --client/--server semantics mean --size is per-job/per-filename, so
// no multiplication by host or target count is needed (unlike elbencho).
func (o *Orchestrator) runFIOPhase(ctx context.Context, job db.Job, cfg JobConfig, workerIPs []string, artifactDir string) error {
	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "running", Phase: "prefill"})
	o.emitProvStep(job.ID, "prefill_start",
		fmt.Sprintf("Prefill fio des data disks (%d GB/disk) pour eviter les zero-block reads...", cfg.DataDiskGB), 1.0)
	prefillTargets := buildTargets(cfg.DataDisks)
	prefillStart := time.Now()
	if err := fio.Prefill(ctx, workerIPs, prefillTargets, cfg.DataDiskGB, artifactDir); err != nil {
		return fmt.Errorf("prefill: %w", err)
	}
	prefillDur := time.Since(prefillStart)
	log.Printf("[prefill][fio] completed in %s for %d GB on %d workers x %d targets",
		prefillDur, cfg.DataDiskGB, len(workerIPs), len(prefillTargets))
	if prefillDur < 5*time.Second {
		log.Printf("[prefill][fio] WARNING: completed too fast, allocation may be incomplete")
	}
	o.emitProvStep(job.ID, "prefill_done", "Prefill termine, demarrage du benchmark...", 1.0)

	for _, profileName := range cfg.Profiles {
		o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "running", Phase: profileName})

		profile, err := o.DB.GetProfileByNameAndEngine(profileName, "fio")
		if err != nil {
			return fmt.Errorf("profile %s (fio): %w", profileName, err)
		}

		jobfile, err := fio.BuildJobfile(profileName, profile.ConfigJSON, prefillTargets)
		if err != nil {
			return fmt.Errorf("build jobfile %s: %w", profileName, err)
		}
		// keep jobfile around for the bundle; runner copies it into artifactDir.
		// remove only after the run finishes.
		defer os.Remove(jobfile)

		metricCtx, cancelMetricCtx := context.WithCancel(ctx)
		fioCh := make(chan fio.Metric, 200)
		convertedCh := make(chan storageMetric, 200)

		go func() {
			defer close(convertedCh)
			for m := range fioCh {
				convertedCh <- storageMetricFromFio(m)
			}
		}()
		persistDone := make(chan struct{})
		go func() {
			o.persistMetrics(metricCtx, job.ID, profileName, "fio", convertedCh)
			close(persistDone)
		}()

		err = fio.Run(ctx, fio.RunConfig{
			Hosts:             workerIPs,
			Jobfile:           jobfile,
			Label:             profileName,
			OutputDir:         artifactDir,
			StatusIntervalSec: 2,
		}, fioCh)
		// fio.Run returned: fioCh is closed. Wait for the converter to
		// drain it into convertedCh (which it then closes), then for
		// persistMetrics to drain convertedCh into the DB. Only after
		// both are done is it safe to cancel metricCtx and move on.
		<-persistDone
		cancelMetricCtx()
		if err != nil {
			return err
		}
	}
	return nil
}

// RecoverOrphanedJobs is called at startup to clean up jobs left in a non-terminal
// state by a previous binary crash. It stops/deletes their VMs and marks them failed.
func (o *Orchestrator) RecoverOrphanedJobs(ctx context.Context) {
	jobs, err := o.DB.ListJobs()
	if err != nil {
		log.Printf("[recovery] list jobs: %v", err)
		return
	}
	for _, job := range jobs {
		if job.Status == "done" || job.Status == "failed" || job.Status == "cancelled" {
			continue
		}
		log.Printf("[recovery] orphaned job %s (status=%s) - cleaning up", job.ID, job.Status)
		workers, _ := o.DB.ListWorkersByJob(job.ID)
		o.cleanup(ctx, workers)
		_ = o.DB.FailJob(job.ID, "server restarted mid-run")
	}
}

// CancelJob marks a job as cancelled and immediately cleans up its VMs.
func (o *Orchestrator) CancelJob(ctx context.Context, jobID string) error {
	if err := o.DB.FinishJob(jobID, "cancelled"); err != nil {
		return err
	}
	workers, _ := o.DB.ListWorkersByJob(jobID)
	o.cleanup(ctx, workers)
	o.emit(jobID, ws.EventJobStatus, ws.JobStatusPayload{Status: "cancelled"})
	return nil
}

// sshDiagnose runs ssh -vvv against a worker with the master's keypair and
// returns the captured stderr (where ssh writes its verbose output). Used
// when ansible fails so the failure message tells us exactly what SSH saw -
// which key it offered, which the server accepted/rejected, etc.
func sshDiagnose(keyPath, ip string) string {
	cmd := exec.Command("ssh",
		"-vvv",
		"-i", keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "PreferredAuthentications=publickey",
		fmt.Sprintf("root@%s", ip),
		"echo", "benchere-ssh-ok",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	return out.String()
}
