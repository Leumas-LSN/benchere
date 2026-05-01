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
// In v2.0.4 the prefill jobfile uses size=25% with offset_increment=25% so
// that 4 numjobs split the device into 4 equal stripes and write each
// stripe once concurrently. fio resolves the percentage locally on every
// worker against the actual device size at /dev/disk/by-id/..., which means
// the prefill adapts to whatever data_disk_gb the user picked in NewJob
// without needing to be passed an explicit byte count.
//
// Net effect vs the v1.x and v2.0.x prefill (which had every numjob
// restart at offset 0 and write sizeGB each, producing 4x the necessary
// write IO): a single full pass over the device, parallelized 4-way,
// roughly 4x faster on bandwidth-limited backends. The total IO still
// covers the whole surface so thin allocation, dedup and zero detection
// are all defeated in one go.
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
// section per target.
//
// v2.0.4 layout: 4 numjobs split the device into 4 equal stripes via
// offset_increment=25 percent and size=25 percent. fio resolves the
// percentage locally on each worker against the actual device size
// (/dev/disk/by-id/scsi-...), so the prefill adapts transparently to
// whatever data_disk_gb the operator picks in NewJob (5 GB, 30 GB,
// 500 GB, 1 TB, anything works).
//
// sizeGB is retained for caller-side logging and for legacy callers; the
// jobfile itself does not embed the byte count anymore.
func buildPrefillJobfile(targets []string, sizeGB int) string {
	_ = sizeGB
	var b strings.Builder
	b.WriteString("[global]\n")
	b.WriteString("ioengine=libaio\n")
	b.WriteString("direct=1\n")
	b.WriteString("rw=write\n")
	b.WriteString("bs=1M\n")
	b.WriteString("iodepth=32\n")
	b.WriteString("numjobs=4\n")
	b.WriteString("offset_increment=25%\n")
	b.WriteString("size=25%\n")
	b.WriteString("group_reporting=1\n")
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
