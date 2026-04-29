package fio

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Prefill writes sequentially across each target on every host so that
// thin-provisioned backends (Ceph RBD with object-map, ZFS sparse zvols,
// LVM-thin, NFS sparse) allocate every extent before any read profile runs.
// Without this, read benchmarks against freshly-provisioned worker disks
// measure the backend's zero-block fast path rather than real storage.
//
// IMPORTANT: fio's --size in --client/--server mode is per-job, per-filename.
// Setting size=50G in a job means each (client, job, filename) tuple gets
// 50G. There is NO need to multiply by host count or target count, unlike
// elbencho whose --size is total across all (host, target) pairs. This
// makes prefill sizing for fio simpler and more predictable.
//
// Sequential 1 MiB writes with O_DIRECT, 4 jobs, iodepth 16 per target.
// outputDir, when non-empty, receives prefill.cmd, prefill.stdout,
// prefill.stderr, prefill.jobfile, and prefill.hostsfile.
//
// Multi-host invocation uses --client=<hostsfile> (one host per line). See
// the package doc on runner.go for why the comma-separated form does not
// work.
func Prefill(ctx context.Context, hosts []string, targets []string, sizeGB int, outputDir string) error {
	if len(hosts) == 0 || len(targets) == 0 || sizeGB <= 0 {
		return fmt.Errorf("prefill: hosts/targets/size required (got hosts=%d targets=%d sizeGB=%d)",
			len(hosts), len(targets), sizeGB)
	}

	jobfile := buildPrefillJobfile(targets, sizeGB)

	tmp, err := os.CreateTemp("", "benchere-fio-prefill-*.fio")
	if err != nil {
		return fmt.Errorf("prefill: tmp jobfile: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(jobfile); err != nil {
		tmp.Close()
		return fmt.Errorf("prefill: write jobfile: %w", err)
	}
	tmp.Close()

	hostsfile, err := writeHostsFile("prefill", hosts)
	if err != nil {
		return fmt.Errorf("prefill: %w", err)
	}
	defer os.Remove(hostsfile)

	args := []string{
		"--client=" + hostsfile,
		tmp.Name(),
	}
	cmd := exec.CommandContext(ctx, "fio", args...)

	var stdoutW io.Writer = os.Stdout
	var stderrW io.Writer = os.Stderr
	var stdoutFile, stderrFile *os.File

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[fio] prefill mkdir %s: %v\n", outputDir, err)
		} else {
			cmdLine := "fio " + strings.Join(quoteAll(args), " ") + "\n"
			_ = os.WriteFile(filepath.Join(outputDir, "prefill.cmd"), []byte(cmdLine), 0o644)
			_ = os.WriteFile(filepath.Join(outputDir, "prefill.jobfile"), []byte(jobfile), 0o644)
			if data, err := os.ReadFile(hostsfile); err == nil {
				_ = os.WriteFile(filepath.Join(outputDir, "prefill.hostsfile"), data, 0o644)
			}
			if f, err := os.Create(filepath.Join(outputDir, "prefill.stdout")); err == nil {
				stdoutFile = f
				stdoutW = io.MultiWriter(os.Stdout, f)
			}
			if f, err := os.Create(filepath.Join(outputDir, "prefill.stderr")); err == nil {
				stderrFile = f
				stderrW = io.MultiWriter(os.Stderr, f)
			}
		}
	}
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW

	runErr := cmd.Run()

	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}

	if runErr != nil {
		return fmt.Errorf("fio prefill: %w", runErr)
	}
	return nil
}

// buildPrefillJobfile renders one [global] section plus one [prefill_<i>]
// section per target. Each per-target job writes sizeGB of data sequentially
// to its filename.
func buildPrefillJobfile(targets []string, sizeGB int) string {
	var b strings.Builder
	b.WriteString("[global]\n")
	b.WriteString("ioengine=libaio\n")
	b.WriteString("direct=1\n")
	b.WriteString("rw=write\n")
	b.WriteString("bs=1M\n")
	b.WriteString("iodepth=16\n")
	b.WriteString("numjobs=4\n")
	b.WriteString("group_reporting=1\n")
	b.WriteString(fmt.Sprintf("size=%dG\n", sizeGB))
	b.WriteString("refill_buffers=1\n")
	b.WriteString("buffer_compress_percentage=0\n")
	b.WriteString("\n")
	for i, t := range targets {
		b.WriteString(fmt.Sprintf("[prefill_%d]\n", i))
		b.WriteString("filename=" + t + "\n")
		b.WriteString("\n")
	}
	return b.String()
}
