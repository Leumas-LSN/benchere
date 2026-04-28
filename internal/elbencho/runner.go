package elbencho

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RunConfig struct {
	Hosts       []string
	ConfigFile  string
	Targets     []string // e.g. ["/dev/sdb", "/dev/sdc"]
	LiveCSVPath string
	CSVPath     string
	Label       string
}

func Run(ctx context.Context, cfg RunConfig) error {
	args := []string{
		"--hosts", strings.Join(cfg.Hosts, ","),
		"--configfile", cfg.ConfigFile,
		"--livecsv", cfg.LiveCSVPath,
		"--csvfile", cfg.CSVPath,
		"--label", cfg.Label,
	}
	args = append(args, cfg.Targets...)

	cmd := exec.CommandContext(ctx, "elbencho", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("elbencho run %s: %w", cfg.Label, err)
	}
	return nil
}

// Prefill writes sequentially across each target on every host so that
// thin-provisioned backends (Ceph RBD, ZFS sparse zvols, ...) allocate every
// extent before any read profile runs. Without this, read benchmarks against
// freshly-provisioned worker disks measure the backend's zero-block fast path
// (~RAM speed at network bandwidth) instead of real storage performance.
//
// Sequential 1 MiB writes with O_DIRECT, 4 threads per service, no random
// pattern. The backend allocates physical blocks as the writes land. The
// CSV outputs are not used downstream and end up in the OS tmpdir.
func Prefill(ctx context.Context, hosts []string, targets []string, sizeGB int) error {
	if len(hosts) == 0 || len(targets) == 0 || sizeGB <= 0 {
		return fmt.Errorf("prefill: hosts/targets/size required (got hosts=%d targets=%d sizeGB=%d)",
			len(hosts), len(targets), sizeGB)
	}
	args := []string{
		"--hosts", strings.Join(hosts, ","),
		"--write",
		"--block", "1M",
		"--size", fmt.Sprintf("%dG", sizeGB),
		"--threads", "4",
		"--iodepth", "4",
		"--direct",
		"--label", "prefill",
	}
	args = append(args, targets...)

	cmd := exec.CommandContext(ctx, "elbencho", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("elbencho prefill: %w", err)
	}
	return nil
}
