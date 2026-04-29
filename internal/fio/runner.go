// Package fio drives fio in distributed --client/--server mode and streams
// per-status-interval JSON+ snapshots back as Metric values. It mirrors the
// shape of internal/elbencho so the orchestrator can treat both engines
// uniformly when persisting samples.
//
// Key differences vs elbencho:
//
//   - fio's --size in --client/--server mode is per-job, per-filename. Setting
//     size=50G in a job means each (client, job, filename) tuple writes 50G.
//     No multiplication by host count or target count is needed - unlike
//     elbencho where --size is the total across all (host, target) pairs.
//
//   - JSON+ output (--output-format=json+ --status-interval=N) emits one
//     full JSON document per interval to stdout. The Run loop parses each
//     boundary-aligned object and emits a Metric on the channel.
//
//   - Latency in fio is reported in nanoseconds inside clat_ns / lat_ns.
//     Metric.LatencyAvgMs is converted to milliseconds for consistency
//     with elbencho metrics.
package fio

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunConfig is the input for one fio profile run in distributed mode.
type RunConfig struct {
	// Hosts are the worker IPs reachable by fio --client. Each host must
	// be running fio --server.
	Hosts []string

	// Jobfile is the on-disk path to a fio jobfile containing both the
	// global section and the per-job sections. fio reads it locally and
	// distributes the same workload to every client.
	Jobfile string

	// Label is a short identifier used both as the fio "name" override
	// (when needed) and as the artifact filename prefix.
	Label string

	// OutputDir, when non-empty, receives:
	//   {Label}.cmd          literal invocation
	//   {Label}.stdout       streamed JSON+ snapshots (raw)
	//   {Label}.stderr       fio messages
	//   {Label}.jobfile      copy of the jobfile actually used
	OutputDir string

	// StatusIntervalSec is the interval in seconds between JSON status
	// snapshots. Defaults to 2 when zero.
	StatusIntervalSec int
}

// Run invokes fio in --client mode against the given hosts and parses each
// boundary-aligned JSON+ status snapshot. Each snapshot is converted to a
// Metric and sent on out. The channel is closed when fio exits.
//
// out is the only side-effect channel for live metrics; raw stdout is also
// persisted to OutputDir/{Label}.stdout when configured, so the bundle has
// the exact bytes fio emitted.
func Run(ctx context.Context, cfg RunConfig, out chan<- Metric) error {
	defer close(out)

	if len(cfg.Hosts) == 0 {
		return fmt.Errorf("fio.Run: no hosts")
	}
	if cfg.Jobfile == "" {
		return fmt.Errorf("fio.Run: no jobfile")
	}
	interval := cfg.StatusIntervalSec
	if interval <= 0 {
		interval = 2
	}

	args := []string{
		"--client=" + strings.Join(cfg.Hosts, ","),
		cfg.Jobfile,
		"--output-format=json+",
		fmt.Sprintf("--status-interval=%d", interval),
	}

	cmd := exec.CommandContext(ctx, "fio", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("fio.Run: stdout pipe: %w", err)
	}

	var stderrFile, stdoutFile, jobfileFile *os.File
	var stderrW io.Writer = os.Stderr

	if cfg.OutputDir != "" && cfg.Label != "" {
		if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[fio] mkdir %s: %v\n", cfg.OutputDir, err)
		} else {
			cmdLine := "fio " + strings.Join(quoteAll(args), " ") + "\n"
			if err := os.WriteFile(filepath.Join(cfg.OutputDir, cfg.Label+".cmd"), []byte(cmdLine), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "[fio] write cmd: %v\n", err)
			}
			// copy the jobfile so the bundle has the exact content used
			if data, err := os.ReadFile(cfg.Jobfile); err == nil {
				_ = os.WriteFile(filepath.Join(cfg.OutputDir, cfg.Label+".jobfile"), data, 0o644)
			}
			if f, err := os.Create(filepath.Join(cfg.OutputDir, cfg.Label+".stdout")); err == nil {
				stdoutFile = f
			}
			if f, err := os.Create(filepath.Join(cfg.OutputDir, cfg.Label+".stderr")); err == nil {
				stderrFile = f
				stderrW = io.MultiWriter(os.Stderr, f)
			}
		}
	}
	_ = jobfileFile
	cmd.Stderr = stderrW

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("fio.Run: start: %w", err)
	}

	// Read stdout, splitting at top-level JSON boundaries: each snapshot
	// starts with "{" at column 0 and ends with "}" on its own line at
	// column 0. JSON objects emitted by fio are pretty-printed with a
	// closing "}" at column 0, so we accumulate lines until that boundary.
	parseErr := streamSnapshots(ctx, stdoutPipe, stdoutFile, cfg.Label, out)

	waitErr := cmd.Wait()

	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}

	if waitErr != nil {
		return fmt.Errorf("fio.Run %s: %w", cfg.Label, waitErr)
	}
	if parseErr != nil && parseErr != io.EOF {
		return fmt.Errorf("fio.Run %s: parse: %w", cfg.Label, parseErr)
	}
	return nil
}

// streamSnapshots reads stdout line by line, accumulating each top-level
// JSON object, and emits one Metric per completed snapshot. raw, when
// non-nil, captures every byte fio emitted.
//
// fio pretty-prints status JSON with the outer "{" on its own line at
// column 0 and the matching outer "}" also on its own line at column 0.
// All inner braces are indented. We therefore split on column 0 boundary
// lines exactly, not on trim-space matches which would also fire on
// indented inner closes.
func streamSnapshots(ctx context.Context, src io.Reader, raw io.Writer, label string, out chan<- Metric) error {
	r := bufio.NewReader(src)
	var buf strings.Builder
	started := false

	emitFromBuf := func() {
		txt := buf.String()
		buf.Reset()
		started = false
		snap, err := ParseStatusSnapshot([]byte(txt))
		if err != nil {
			return
		}
		m := snap.ToMetric(label)
		select {
		case out <- m:
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
			// strip the trailing newline only for the column-0 detection
			lineNoNL := strings.TrimRight(line, "\r\n")
			if !started {
				if lineNoNL == "{" {
					started = true
					buf.WriteString(line)
				}
				// ignore anything before the first { (banners, blank lines)
			} else {
				buf.WriteString(line)
				if lineNoNL == "}" {
					emitFromBuf()
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

// CaptureVersion writes `fio --version` output to outputDir/fio-version.txt.
// Best-effort: any error is swallowed so a missing or broken fio does not
// poison the bundle pipeline.
func CaptureVersion(ctx context.Context, outputDir string) {
	if outputDir == "" {
		return
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return
	}
	cmd := exec.CommandContext(ctx, "fio", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, []byte(fmt.Sprintf("\n# fio --version failed: %v\n", err))...)
	}
	_ = os.WriteFile(filepath.Join(outputDir, "fio-version.txt"), out, 0o644)
}

// quoteAll wraps any argument that contains whitespace in double quotes so the
// captured .cmd file is safe to copy/paste into a shell.
func quoteAll(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		} else {
			out[i] = a
		}
	}
	return out
}

// elapsed is exposed for tests that need a deterministic timestamp.
var nowFn = func() time.Time { return time.Now() }
