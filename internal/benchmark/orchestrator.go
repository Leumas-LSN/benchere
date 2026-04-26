package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/ansible"
	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/elbencho"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/Leumas-LSN/benchere/internal/stress"
	"github.com/Leumas-LSN/benchere/internal/ws"
	"github.com/google/uuid"
)

type Orchestrator struct {
	DB          *db.DB
	Proxmox     *proxmox.Client
	Ansible     *ansible.Runner
	Stress      *stress.Runner
	Hub         *ws.Hub
	SSHKey      string
	ProfilesDir string
	OutputDir   string
}

func (o *Orchestrator) Run(ctx context.Context, cfg JobConfig) error {
	job := NewJob(cfg)
	if err := o.DB.CreateJob(job); err != nil {
		return err
	}
	return o.RunExisting(ctx, job, cfg)
}

func (o *Orchestrator) persistMetrics(ctx context.Context, jobID, profileName string, ch <-chan elbencho.Metric) {
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return
			}
			r := db.Result{
				ID: uuid.NewString(), JobID: jobID, ProfileName: profileName,
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
				LatencyP99Ms:        m.LatencyP99Ms,
			})
		case <-ctx.Done():
			return
		}
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
}

func (o *Orchestrator) fail(jobID string, err error) error {
	log.Printf("[job %s] FAILED: %v", jobID, err)
	_ = o.DB.FailJob(jobID, err.Error())
	o.emit(jobID, ws.EventJobStatus, ws.JobStatusPayload{Status: "failed"})
	return err
}

func (o *Orchestrator) cleanup(ctx context.Context, node string, workerDBIDs []string, workerVMIDs []int) {
	for _, vmid := range workerVMIDs {
		_ = o.Proxmox.StopVM(ctx, node, vmid)
		_ = o.Proxmox.DeleteVM(ctx, node, vmid)
	}
	for _, wid := range workerDBIDs {
		_ = o.DB.UpdateWorkerStatus(wid, "done")
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
	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "provisioning", Phase: "creating_vms"})
	if err := o.DB.UpdateJobStatus(job.ID, "provisioning"); err != nil {
		return o.fail(job.ID, err)
	}

	imageStorage := cfg.ImageStorage
	if imageStorage == "" {
		imageStorage = "local"
	}
	imageVol, err := o.Proxmox.EnsureCloudImage(ctx, cfg.ProxmoxNode, imageStorage, cfg.CloudImageURL)
	if err != nil {
		return o.fail(job.ID, fmt.Errorf("cloud image: %w", err))
	}
	networkBridge, _ := o.DB.GetSetting("network_bridge")

	// Static IP allocation: when worker_ip_pool / worker_cidr / worker_gateway
	// are set we assign each worker an address from the pool via cloud-init's
	// ipconfig0. This sidesteps the qemu-guest-agent IP discovery that's
	// brittle on cloud images that don't ship the agent.
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
		staticIPs, err = AllocateIPs(ctx, o.Proxmox, cfg.ProxmoxNode, pool, cfg.WorkerCount)
		if err != nil {
			return o.fail(job.ID, err)
		}
	}

	o.emitProvStep(job.ID, "vm_creating", "Création des machines virtuelles...", 0.05)
	var workerIPs []string
	var workerDBIDs []string
	var workerVMIDs []int
	defer func() { o.cleanup(context.Background(), cfg.ProxmoxNode, workerDBIDs, workerVMIDs) }()

	for i := 0; i < cfg.WorkerCount; i++ {
		vmid, err := o.Proxmox.NextVMID(ctx)
		if err != nil {
			return o.fail(job.ID, fmt.Errorf("allocate vmid: %w", err))
		}
		workerVMIDs = append(workerVMIDs, vmid)

		workerID := uuid.NewString()
		w := db.Worker{ID: workerID, JobID: job.ID, VMID: vmid, ProxmoxNode: cfg.ProxmoxNode, Status: "provisioning"}
		_ = o.DB.CreateWorker(w)

		ipConfig := ""
		var assignedIP string
		if len(staticIPs) > 0 {
			assignedIP = staticIPs[i].String()
			ipConfig = pool.IPConfig(staticIPs[i])
		}

		if _, err := o.Proxmox.CreateVM(ctx, proxmox.VMCreateParams{
			Node:        cfg.ProxmoxNode,
			VMID:        vmid,
			Name:        fmt.Sprintf("benchere-worker-%d", i+1),
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
			return o.fail(job.ID, fmt.Errorf("create vm %d: %w", vmid, err))
		}
		if err := o.Proxmox.StartVM(ctx, cfg.ProxmoxNode, vmid); err != nil {
			return o.fail(job.ID, err)
		}

		var ip string
		if assignedIP != "" {
			// Static IP path: we already know it. Just wait for sshd to come up.
			if err := o.Proxmox.WaitForSSH(ctx, assignedIP); err != nil {
				return o.fail(job.ID, err)
			}
			ip = assignedIP
		} else {
			// Legacy path: discover via qemu-guest-agent. Requires the agent
			// to be installed in the worker image (or via a cicustom snippet).
			discovered, err := o.Proxmox.WaitForIP(ctx, cfg.ProxmoxNode, vmid)
			if err != nil {
				return o.fail(job.ID, err)
			}
			if err := o.Proxmox.InjectSSHKey(ctx, cfg.ProxmoxNode, vmid, readFile(o.SSHKey+".pub")); err != nil {
				return o.fail(job.ID, fmt.Errorf("inject ssh key vm %d: %w", vmid, err))
			}
			ip = discovered
		}
		_ = o.DB.UpdateWorkerIP(workerID, ip)
		workerIPs = append(workerIPs, ip)
		workerDBIDs = append(workerDBIDs, workerID)
		o.emitProvStep(job.ID, "vm_creating",
			fmt.Sprintf("VM %d/%d créée", i+1, cfg.WorkerCount),
			0.05+0.35*float64(i+1)/float64(cfg.WorkerCount),
		)
	}

	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "provisioning", Phase: "ansible"})
	o.emitProvStep(job.ID, "ansible_start", "Déploiement Ansible (elbencho + stress-ng)...", 0.60)
	targets := make([]ansible.WorkerTarget, len(workerIPs))
	for i, ip := range workerIPs {
		targets[i] = ansible.WorkerTarget{IP: ip}
	}
	if err := o.Ansible.ProvisionWorkers(ctx, targets); err != nil {
		diag := ""
		if len(workerIPs) > 0 {
			diag = sshDiagnose(o.SSHKey, workerIPs[0])
		}
		return o.fail(job.ID, fmt.Errorf("ansible: %w\n\n--- ssh -vvv to %s ---\n%s", err, workerIPs[0], diag))
	}
	o.emitProvStep(job.ID, "ansible_done", "Ansible terminé, vérification des workers...", 0.90)
	for _, wid := range workerDBIDs {
		_ = o.DB.UpdateWorkerStatus(wid, "ready")
	}
	o.emitProvStep(job.ID, "workers_ready", "Tous les workers sont prêts. Démarrage du benchmark...", 1.0)

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
		for _, profileName := range cfg.Profiles {
			o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "running", Phase: profileName})

			profile, err := o.DB.GetProfileByName(profileName)
			if err != nil {
				return o.fail(job.ID, fmt.Errorf("profile %s: %w", profileName, err))
			}

			elbenchoCfg, err := elbencho.ProfileToConfig(profile.ConfigJSON)
			if err != nil {
				return o.fail(job.ID, fmt.Errorf("profile %s config: %w", profileName, err))
			}
			profileFile := filepath.Join(o.ProfilesDir, profileName+".elbencho")
			if err := os.WriteFile(profileFile, []byte(elbenchoCfg), 0644); err != nil {
				return o.fail(job.ID, err)
			}

			liveCSV := filepath.Join(o.OutputDir, fmt.Sprintf("live_%s_%s.csv", job.ID, profileName))
			finalCSV := filepath.Join(o.OutputDir, fmt.Sprintf("results_%s_%s.csv", job.ID, profileName))

			metricCtx, cancelMetricCtx := context.WithCancel(ctx)
			metricsCh := make(chan elbencho.Metric, 100)
			go elbencho.TailCSV(metricCtx, liveCSV, metricsCh)
			go o.persistMetrics(metricCtx, job.ID, profileName, metricsCh)

			targets := buildTargets(cfg.DataDisks)
			err = elbencho.Run(ctx, elbencho.RunConfig{
				Hosts:       workerIPs,
				ConfigFile:  profileFile,
				Targets:     targets,
				LiveCSVPath: liveCSV,
				CSVPath:     finalCSV,
				Label:       profileName,
			})
			cancelMetricCtx()
			if err != nil {
				return o.fail(job.ID, err)
			}
		}
	}

	if cfg.Mode == ModeCPU || cfg.Mode == ModeMixed {
		_ = o.Stress.StopOnWorkers(ctx, workerIPs)
	}

	_ = o.DB.FinishJob(job.ID, "done")
	o.emit(job.ID, ws.EventJobStatus, ws.JobStatusPayload{Status: "done"})
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
		log.Printf("[recovery] orphaned job %s (status=%s) — cleaning up", job.ID, job.Status)
		workers, _ := o.DB.ListWorkersByJob(job.ID)
		node, _ := o.DB.GetSetting("proxmox_node")
		node = strings.ToLower(node)
		for _, w := range workers {
			n := w.ProxmoxNode
			if n == "" {
				n = node
			}
			_ = o.Proxmox.StopVM(ctx, n, w.VMID)
			_ = o.Proxmox.DeleteVM(ctx, n, w.VMID)
			_ = o.DB.UpdateWorkerStatus(w.ID, "done")
		}
		_ = o.DB.FailJob(job.ID, "server restarted mid-run")
	}
}

// CancelJob marks a job as cancelled and immediately cleans up its VMs.
func (o *Orchestrator) CancelJob(ctx context.Context, jobID string) error {
	if err := o.DB.FinishJob(jobID, "cancelled"); err != nil {
		return err
	}
	workers, _ := o.DB.ListWorkersByJob(jobID)
	node, _ := o.DB.GetSetting("proxmox_node")
	node = strings.ToLower(node)
	for _, w := range workers {
		n := w.ProxmoxNode
		if n == "" {
			n = node
		}
		_ = o.Proxmox.StopVM(ctx, n, w.VMID)
		_ = o.Proxmox.DeleteVM(ctx, n, w.VMID)
		_ = o.DB.UpdateWorkerStatus(w.ID, "done")
	}
	o.emit(jobID, ws.EventJobStatus, ws.JobStatusPayload{Status: "cancelled"})
	return nil
}


// sshDiagnose runs ssh -vvv against a worker with the master's keypair and
// returns the captured stderr (where ssh writes its verbose output). Used
// when ansible fails so the failure message tells us exactly what SSH saw —
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
