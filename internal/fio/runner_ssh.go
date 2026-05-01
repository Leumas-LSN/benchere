package fio

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// PerWorkerRunConfig is the input for one fio profile run executed via
// SSH on each worker. It replaces the historical fio --client=hostsfile
// path which buffered every per-interval JSON document until end of run
// and so produced no live signal on the dashboard during the multi-minute
// profile execution.
//
// In this mode we SSH into each worker, run fio locally there with
// --status-interval=N, and stream JSON+ snapshots back to the master.
// An aggregator merges per-worker snapshots by sequence number into a
// single fio.Metric stream that the orchestrator consumes exactly the
// same way it did with fio.Run before.
type PerWorkerRunConfig struct {
	Hosts             []string
	Jobfile           string // local path on master, scp'd to /tmp on every worker
	Label             string // profile name, copied onto every emitted Metric
	OutputDir         string // when non-empty, per-host stdout is captured here
	SSHKey            string // path to private key on master
	StatusIntervalSec int    // defaults to 2
}

// RunPerWorker is the v2.1+ replacement for fio.Run. It SCPs the jobfile
// to every worker, spawns N parallel SSH+fio processes, and streams the
// merged per-interval Metric to out. The channel is closed when every
// worker process has terminated (or one failed and the others were
// cancelled by the errgroup).
func RunPerWorker(ctx context.Context, cfg PerWorkerRunConfig, out chan<- Metric) error {
	defer close(out)

	if len(cfg.Hosts) == 0 {
		return fmt.Errorf("RunPerWorker: no hosts")
	}
	if cfg.Jobfile == "" {
		return fmt.Errorf("RunPerWorker: no jobfile")
	}
	interval := cfg.StatusIntervalSec
	if interval <= 0 {
		interval = 2
	}
	sshKey := cfg.SSHKey
	if sshKey == "" {
		sshKey = "/opt/benchere/id_rsa"
	}

	if cfg.OutputDir != "" {
		_ = os.MkdirAll(cfg.OutputDir, 0o755)
	}

	remoteJobfile := fmt.Sprintf("/tmp/benchere-fio-%s-%d.fio",
		sanitizeLabel(cfg.Label), os.Getpid())

	if err := scpJobfileToAll(ctx, cfg.Hosts, sshKey, cfg.Jobfile, remoteJobfile); err != nil {
		return err
	}
	defer cleanupRemoteJobfile(cfg.Hosts, sshKey, remoteJobfile)

	// Echo the resolved per-host fio invocation to a single .cmd file so
	// the bundle has a copyable record of what ran.
	if cfg.OutputDir != "" {
		cmdLine := fmt.Sprintf("# v2.1 per-worker SSH execution, %d hosts, status_interval=%ds\n",
			len(cfg.Hosts), interval)
		for _, h := range cfg.Hosts {
			cmdLine += fmt.Sprintf("ssh -i %s root@%s fio --output-format=json+ --status-interval=%d %s\n",
				sshKey, h, interval, remoteJobfile)
		}
		_ = os.WriteFile(filepath.Join(cfg.OutputDir, sanitizeLabel(cfg.Label)+".cmd"),
			[]byte(cmdLine), 0o644)
		// Also stash the jobfile for the bundle.
		if data, err := os.ReadFile(cfg.Jobfile); err == nil {
			_ = os.WriteFile(filepath.Join(cfg.OutputDir, sanitizeLabel(cfg.Label)+".jobfile"),
				data, 0o644)
		}
	}

	snapCh := make(chan workerSnap, 256)

	g, gctx := errgroup.WithContext(ctx)
	for _, host := range cfg.Hosts {
		host := host
		g.Go(func() error {
			return runOneFio(gctx, host, sshKey, remoteJobfile, interval,
				cfg.OutputDir, cfg.Label, snapCh)
		})
	}

	aggDone := make(chan struct{})
	go func() {
		defer close(aggDone)
		aggregateAndEmit(gctx, len(cfg.Hosts), interval, cfg.Label, snapCh, out)
	}()

	err := g.Wait()
	close(snapCh)
	<-aggDone

	return err
}

// workerSnap tags a per-worker snapshot with its source host and the
// snapshot index emitted by that host's fio. Aggregation is by seq, not
// by timestamp, so worker clock skew does not misalign buckets.
type workerSnap struct {
	host string
	seq  int
	snap Snapshot
}

func scpJobfileToAll(ctx context.Context, hosts []string, sshKey, localPath, remotePath string) error {
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, h := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			cmd := exec.CommandContext(ctx,
				"scp",
				"-i", sshKey,
				"-o", "StrictHostKeyChecking=no",
				"-o", "BatchMode=yes",
				"-o", "ConnectTimeout=10",
				localPath,
				fmt.Sprintf("root@%s:%s", host, remotePath),
			)
			if out, err := cmd.CombinedOutput(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("scp jobfile to %s: %w (%s)",
						host, err, strings.TrimSpace(string(out)))
				}
				mu.Unlock()
			}
		}(h)
	}
	wg.Wait()
	return firstErr
}

func cleanupRemoteJobfile(hosts []string, sshKey, remotePath string) {
	for _, h := range hosts {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx,
			"ssh",
			"-i", sshKey,
			"-o", "StrictHostKeyChecking=no",
			"-o", "BatchMode=yes",
			"-o", "ConnectTimeout=5",
			fmt.Sprintf("root@%s", h),
			"rm", "-f", remotePath,
		)
		_ = cmd.Run()
		cancel()
	}
}

// runOneFio SSHes into one worker, runs fio there with --status-interval,
// parses each JSON+ document as it crosses a column-0 brace boundary,
// tags it with the worker host and a sequence number, and pushes onto
// out. The function returns when the SSH process exits.
func runOneFio(ctx context.Context, host, sshKey, remoteJobfile string, intervalSec int, outputDir, label string, out chan<- workerSnap) error {
	fioCmd := fmt.Sprintf("fio --output-format=json+ --status-interval=%d %s",
		intervalSec, remoteJobfile)
	args := []string{
		"-i", sshKey,
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "ServerAliveInterval=15",
		"-o", "ServerAliveCountMax=3",
		fmt.Sprintf("root@%s", host),
		fioCmd,
	}
	cmd := exec.CommandContext(ctx, "ssh", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ssh stdout pipe %s: %w", host, err)
	}

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	var logF *os.File
	if outputDir != "" {
		logPath := filepath.Join(outputDir,
			fmt.Sprintf("%s.%s.stdout", sanitizeLabel(label), sanitizeHost(host)))
		if f, err := os.Create(logPath); err == nil {
			logF = f
		}
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ssh start %s: %w", host, err)
	}

	parseErr := streamSnapshotsForHost(ctx, stdoutPipe, logF, host, out)
	waitErr := cmd.Wait()

	if logF != nil {
		logF.Close()
	}

	if waitErr != nil {
		return fmt.Errorf("fio on %s: %w (stderr: %s)",
			host, waitErr, strings.TrimSpace(stderrBuf.String()))
	}
	if parseErr != nil && parseErr != io.EOF {
		return fmt.Errorf("parse on %s: %w", host, parseErr)
	}
	return nil
}

// streamSnapshotsForHost reads SSH stdout line by line, accumulating
// each top-level JSON object (delimited by column-0 { and column-0 }),
// and pushes the parsed Snapshot tagged with the host and a 1-based seq.
// raw, when non-nil, also receives every byte fio emitted so the bundle
// preserves the exact stdout per host.
func streamSnapshotsForHost(ctx context.Context, src io.Reader, raw io.Writer, host string, out chan<- workerSnap) error {
	r := bufio.NewReader(src)
	var buf strings.Builder
	started := false
	seq := 0

	emit := func() {
		txt := buf.String()
		buf.Reset()
		started = false
		snap, err := ParseStatusSnapshot([]byte(txt))
		if err != nil {
			return
		}
		seq++
		select {
		case out <- workerSnap{host: host, seq: seq, snap: *snap}:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := r.ReadString('\n')
		if line != "" {
			if raw != nil {
				_, _ = raw.Write([]byte(line))
			}
			lineNoNL := strings.TrimRight(line, "\r\n")
			if !started {
				if lineNoNL == "{" {
					started = true
					buf.WriteString(line)
				}
			} else {
				buf.WriteString(line)
				if lineNoNL == "}" {
					emit()
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// aggregateAndEmit merges per-worker snapshots by seq into cluster-level
// Metrics and pushes them onto out in seq order. A bucket is flushed as
// soon as every expected host has reported, or after a 3x interval
// timeout if some host straggles. When the input channel closes, any
// remaining buckets are flushed in seq order so no data is dropped.
func aggregateAndEmit(ctx context.Context, expectedHosts, intervalSec int, label string, in <-chan workerSnap, out chan<- Metric) {
	type bucket struct {
		snaps     map[string]Snapshot
		firstSeen time.Time
	}
	buckets := make(map[int]*bucket)

	timeout := time.Duration(intervalSec*3) * time.Second
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	flush := func(b *bucket) {
		if len(b.snaps) == 0 {
			return
		}
		m := mergeSnapshots(label, b.snaps)
		select {
		case out <- m:
		case <-ctx.Done():
		}
	}

	for {
		select {
		case s, ok := <-in:
			if !ok {
				keys := make([]int, 0, len(buckets))
				for k := range buckets {
					keys = append(keys, k)
				}
				sort.Ints(keys)
				for _, k := range keys {
					flush(buckets[k])
				}
				return
			}
			b, ok := buckets[s.seq]
			if !ok {
				b = &bucket{snaps: make(map[string]Snapshot), firstSeen: time.Now()}
				buckets[s.seq] = b
			}
			b.snaps[s.host] = s.snap
			if len(b.snaps) >= expectedHosts {
				flush(b)
				delete(buckets, s.seq)
			}
		case now := <-ticker.C:
			stale := make([]int, 0)
			for k, b := range buckets {
				if now.Sub(b.firstSeen) > timeout {
					stale = append(stale, k)
				}
			}
			sort.Ints(stale)
			for _, k := range stale {
				flush(buckets[k])
				delete(buckets, k)
			}
		case <-ctx.Done():
			return
		}
	}
}

// mergeSnapshots folds N per-worker Snapshots into one cluster-level
// Metric. The math is:
//
//   - IOPS, throughput: SUM across workers (additive).
//   - Mean read/write latency: weighted average by IOPS.
//   - Tail latencies (p50, p95, p99, p99.9, write p99): MAX across
//     workers, because the right interpretation of "p99 latency in the
//     cluster" is the worst p99 any single worker observed (a slow
//     fraction at any one place is visible globally to the application).
//   - LatencyAvgMs: the legacy single-value field, prefers read mean,
//     falls back to write mean for write-only profiles.
//   - Timestamp: median of the per-worker timestamps in the bucket so
//     chart ordering tracks rough physical time.
//
// ProfileName is set from label.
func mergeSnapshots(label string, hostMap map[string]Snapshot) Metric {
	var m Metric
	m.ProfileName = label

	var totalIOPSr, totalIOPSw float64
	var totalBWr, totalBWw float64
	var weightedLatRead, weightedLatWrite float64
	var maxP50, maxP95, maxP99, maxP999, maxWriteP99 float64

	for _, s := range hostMap {
		totalIOPSr += s.IOPSRead
		totalIOPSw += s.IOPSWrite
		totalBWr += s.ThroughputReadMBps
		totalBWw += s.ThroughputWriteMBps
		if s.IOPSRead > 0 {
			weightedLatRead += s.LatencyReadMeanMs * s.IOPSRead
		}
		if s.IOPSWrite > 0 {
			weightedLatWrite += s.LatencyWriteMeanMs * s.IOPSWrite
		}
		if s.LatencyReadP50Ms > maxP50 {
			maxP50 = s.LatencyReadP50Ms
		}
		if s.LatencyReadP95Ms > maxP95 {
			maxP95 = s.LatencyReadP95Ms
		}
		if s.LatencyReadP99Ms > maxP99 {
			maxP99 = s.LatencyReadP99Ms
		}
		if s.LatencyReadP999Ms > maxP999 {
			maxP999 = s.LatencyReadP999Ms
		}
		if s.LatencyWriteP99Ms > maxWriteP99 {
			maxWriteP99 = s.LatencyWriteP99Ms
		}
	}

	m.IOPSRead = totalIOPSr
	m.IOPSWrite = totalIOPSw
	m.ThroughputReadMBps = totalBWr
	m.ThroughputWriteMBps = totalBWw

	if totalIOPSr > 0 {
		m.LatencyReadAvgMs = weightedLatRead / totalIOPSr
	}
	if totalIOPSw > 0 {
		m.LatencyWriteAvgMs = weightedLatWrite / totalIOPSw
	}
	m.LatencyAvgMs = m.LatencyReadAvgMs
	if m.LatencyAvgMs == 0 {
		m.LatencyAvgMs = m.LatencyWriteAvgMs
	}
	m.LatencyP50Ms = maxP50
	m.LatencyP95Ms = maxP95
	m.LatencyP99Ms = maxP99
	m.LatencyP999Ms = maxP999
	m.LatencyWriteP99Ms = maxWriteP99

	m.Timestamp = pickMedianTimestamp(hostMap)
	return m
}

func pickMedianTimestamp(snaps map[string]Snapshot) time.Time {
	ts := make([]time.Time, 0, len(snaps))
	for _, s := range snaps {
		if !s.Timestamp.IsZero() {
			ts = append(ts, s.Timestamp)
		}
	}
	if len(ts) == 0 {
		return time.Now()
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Before(ts[j]) })
	return ts[len(ts)/2]
}

func sanitizeHost(host string) string {
	r := strings.NewReplacer(".", "-", ":", "-", "/", "-")
	return r.Replace(host)
}
