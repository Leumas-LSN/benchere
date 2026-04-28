package benchmark

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
)

// CollectWorkerArtifacts gathers per-worker sysinfo and per-VM Proxmox config
// before workers get destroyed at end of run. Output goes to
// {jobDir}/workers/worker-{i}/{...}.txt where i is 1-indexed. All errors are
// best-effort and logged to {jobDir}/errors.log so a single missing worker
// or unreachable VM does not poison the rest of the bundle.
//
// This MUST be called before cleanup() runs, since cleanup destroys the VMs
// and SSH/qm config queries become impossible afterwards.
func (o *Orchestrator) CollectWorkerArtifacts(ctx context.Context, jobID string, workers []db.Worker) {
	jd := o.jobDir(jobID)
	if jd == "" {
		return
	}
	root := filepath.Join(jd, "workers")
	if err := os.MkdirAll(root, 0o755); err != nil {
		o.logArtifactErr(jd, "workers root: "+err.Error())
		return
	}

	for i, w := range workers {
		dir := filepath.Join(root, fmt.Sprintf("worker-%d", i+1))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			o.logArtifactErr(jd, fmt.Sprintf("mkdir worker-%d: %v", i+1, err))
			continue
		}
		o.collectOneWorker(ctx, jd, dir, i+1, w)
	}
}

func (o *Orchestrator) collectOneWorker(ctx context.Context, jobDir, dir string, idx int, w db.Worker) {
	// info.txt - one line summary, written first so we always have something.
	info := fmt.Sprintf("idx=%d vmid=%d node=%s ip=%s status=%s\n",
		idx, w.VMID, w.ProxmoxNode, w.IP, w.Status)
	_ = os.WriteFile(filepath.Join(dir, "info.txt"), []byte(info), 0o644)

	// qm config via Proxmox API. Survives even if the VM is unreachable
	// over SSH because Proxmox still has the config until DeleteVM lands.
	if w.ProxmoxNode != "" && w.VMID > 0 && o.Proxmox != nil {
		if cfg, err := o.Proxmox.GetVMConfig(ctx, w.ProxmoxNode, w.VMID); err == nil {
			_ = os.WriteFile(filepath.Join(dir, "qm-config.txt"), []byte(cfg), 0o644)
		} else {
			writeUnavailable(filepath.Join(dir, "qm-config.txt"), err)
			o.logArtifactErr(jobDir, fmt.Sprintf("worker-%d qm config: %v", idx, err))
		}
	} else {
		writeUnavailable(filepath.Join(dir, "qm-config.txt"),
			fmt.Errorf("missing node/vmid"))
	}

	if w.IP == "" {
		// Without an IP, no SSH-based collection is possible. Mark every
		// expected file as unavailable so the bundle structure stays uniform.
		for _, name := range sshFiles() {
			writeUnavailable(filepath.Join(dir, name), fmt.Errorf("worker has no IP"))
		}
		o.logArtifactErr(jobDir, fmt.Sprintf("worker-%d skipped ssh: no ip", idx))
		return
	}

	type cmd struct {
		file    string
		shell   string
		timeout time.Duration
	}
	cmds := []cmd{
		{"lsblk.txt", "lsblk -O 2>&1", 10 * time.Second},
		{"mount.txt", "mount", 5 * time.Second},
		{"meminfo.txt", "cat /proc/meminfo", 5 * time.Second},
		{"cpuinfo.txt", "head -30 /proc/cpuinfo", 5 * time.Second},
		{"sys-block.txt", `for d in /sys/block/sd? /sys/block/vd? /sys/block/nvme?n?; do [ -d "$d" ] || continue; echo "=== $d ==="; for f in queue/scheduler queue/rotational queue/nr_requests queue/read_ahead_kb queue/max_sectors_kb queue/nr_zones device/queue_depth; do [ -e "$d/$f" ] || continue; printf '%s = %s\n' "$f" "$(cat $d/$f 2>/dev/null)"; done; echo; done`, 10 * time.Second},
		{"dmesg-tail.txt", "dmesg 2>&1 | tail -200", 10 * time.Second},
		{"ip-a.txt", "ip -4 a", 5 * time.Second},
		{"free.txt", "free -m", 5 * time.Second},
		{"uname.txt", "uname -a", 5 * time.Second},
		{"elbencho-version.txt", "elbencho --version 2>&1 || echo 'elbencho not installed'", 5 * time.Second},
	}

	for _, c := range cmds {
		path := filepath.Join(dir, c.file)
		out, err := o.runSSH(ctx, w.IP, c.shell, c.timeout)
		if err != nil {
			writeUnavailable(path, err)
			o.logArtifactErr(jobDir, fmt.Sprintf("worker-%d %s: %v", idx, c.file, err))
			continue
		}
		_ = os.WriteFile(path, out, 0o644)
	}
}

func sshFiles() []string {
	return []string{
		"lsblk.txt",
		"mount.txt",
		"meminfo.txt",
		"cpuinfo.txt",
		"sys-block.txt",
		"dmesg-tail.txt",
		"ip-a.txt",
		"free.txt",
		"uname.txt",
		"elbencho-version.txt",
	}
}

// runSSH executes a shell command on a worker over SSH using the orchestrator's
// SSH key. timeout is enforced via context; the parent ctx is also honored.
func (o *Orchestrator) runSSH(ctx context.Context, ip, shellCmd string, timeout time.Duration) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cctx, "ssh",
		"-i", o.SSHKey,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		fmt.Sprintf("root@%s", ip),
		shellCmd,
	)
	return cmd.CombinedOutput()
}

// writeUnavailable creates the named file with a "# unavailable: <reason>"
// stub so the bundle structure is uniform whether or not the collector
// succeeded. The file is created best-effort; any IO error is silently
// ignored since we are already in a failure path.
func writeUnavailable(path string, reason error) {
	msg := "# unavailable: " + reason.Error() + "\n"
	_ = os.WriteFile(path, []byte(msg), 0o644)
}

// logArtifactErr appends a single line to jobDir/errors.log. Used by every
// artifact collector so the operator can audit what was/was not captured.
func (o *Orchestrator) logArtifactErr(jobDir, msg string) {
	if jobDir == "" {
		return
	}
	line := fmt.Sprintf("%s %s\n", time.Now().UTC().Format(time.RFC3339), msg)
	f, err := os.OpenFile(filepath.Join(jobDir, "errors.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(strings.TrimSpace(line) + "\n")
}
