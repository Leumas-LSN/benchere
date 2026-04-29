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
//     full JSON document per interval to stdout in single mode. In
//     --client/--server mode fio buffers all snapshots and emits a single
//     final document with client_stats[] populated. Run handles both: it
//     keeps live boundary parsing for forward compat and re-parses the
//     captured stdout after fio exits using ParseFinalDocument.
//
//   - Latency in fio is reported in nanoseconds inside clat_ns / lat_ns.
//     Metric.LatencyAvgMs is converted to milliseconds for consistency
//     with elbencho metrics.
//
// IMPORTANT: fio's --client flag does NOT accept comma-separated hosts.
// fio --client=h1,h2 parses as host=h1, port=<first integer of h2> due
// to the host:port grammar. The correct multi-host invocation is
// --client=hostsfile, where hostsfile is a path to a text file with one
// host per line. Reproduced on fio 3.39:
//
//	$ fio --client=10.0.0.1,10.0.0.2 --version
//	fio: failed to connect to 10.0.0.1:10
//
// We write a temporary hostsfile per invocation and pass --client=<path>.
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
	//   {Label}.hostsfile    copy of the hostsfile passed to --client
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
//
// In --client/--server mode fio does not flush per-interval JSON to stdout;
// it accumulates client_stats[] internally and writes the whole document
// once at exit. Run therefore re-parses the captured stdout via
// ParseFinalDocument after cmd.Wait() and emits one Metric per entry, in
// time order, before closing out.
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

	hostsfile, err := writeHostsFile(cfg.Label, cfg.Hosts)
	if err != nil {
		return err
	}
	defer os.Remove(hostsfile)

	args := []string{
		"--client=" + hostsfile,
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
	stdoutPath := ""

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
			// copy the hostsfile so the bundle has the exact host list used
			if data, err := os.ReadFile(hostsfile); err == nil {
				_ = os.WriteFile(filepath.Join(cfg.OutputDir, cfg.Label+".hostsfile"), data, 0o644)
			}
			stdoutPath = filepath.Join(cfg.OutputDir, cfg.Label+".stdout")
			if f, err := os.Create(stdoutPath); err == nil {
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

	// Live boundary parsing: fio in single mode (no --client) emits one
	// pretty-printed JSON document per status interval, with the outer
	// "{" and "}" at column 0. We accumulate each one and emit a Metric
	// as soon as it closes. In --client mode this loop typically yields
	// nothing because fio buffers everything until exit; the post-Wait
	// final-document parser below covers that case.
	emitted := make(map[string]bool)
	parseErr := streamSnapshots(ctx, stdoutPipe, stdoutFile, cfg.Label, out, emitted)

	waitErr := cmd.Wait()

	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}

	// After fio exits, re-read the captured stdout file and extract any
	// snapshots we missed. In --client mode the whole document only lands
	// here; in single mode this is a defensive second pass that filters
	// against the live emitted set.
	if waitErr == nil && stdoutPath != "" {
		if data, err := os.ReadFile(stdoutPath); err == nil && len(data) > 0 {
			snaps, perr := ParseFinalDocument(data)
			if perr == nil {
				for _, snap := range snaps {
					key := snap.JobName + "|" + snap.Timestamp.Format(time.RFC3339Nano)
					if emitted[key] {
						continue
					}
					m := snap.ToMetric(cfg.Label)
					select {
					case out <- m:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		}
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
// non-nil, captures every byte fio emitted. emitted, when non-nil, is
// populated with a key per metric we sent so the post-Wait final-document
// pass can avoid duplicates.
//
// fio pretty-prints status JSON with the outer "{" on its own line at
// column 0 and the matching outer "}" also on its own line at column 0.
// All inner braces are indented. We therefore split on column 0 boundary
// lines exactly, not on trim-space matches which would also fire on
// indented inner closes.
func streamSnapshots(ctx context.Context, src io.Reader, raw io.Writer, label string, out chan<- Metric, emitted map[string]bool) error {
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
		if emitted != nil {
			key := snap.JobName + "|" + snap.Timestamp.Format(time.RFC3339Nano)
			emitted[key] = true
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

// writeHostsFile writes hosts (one per line) to a temp file and returns its
// path. Caller must os.Remove it. Returns an error if the temp file cannot
// be created or written.
//
// fio's --client flag interprets a comma-separated list as host:port, not
// as a host list. The supported multi-host syntax is --client=<file> where
// the file holds one host per line.
func writeHostsFile(label string, hosts []string) (string, error) {
	safe := sanitizeLabel(label)
	f, err := os.CreateTemp("", "benchere-fio-hosts-"+safe+"-*.txt")
	if err != nil {
		return "", fmt.Errorf("write hostsfile: %w", err)
	}
	for _, h := range hosts {
		if _, err := fmt.Fprintln(f, h); err != nil {
			f.Close()
			os.Remove(f.Name())
			return "", fmt.Errorf("write hostsfile: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("close hostsfile: %w", err)
	}
	return f.Name(), nil
}

// sanitizeLabel strips characters that os.CreateTemp's pattern parser would
// reject (path separators) so we can splice the label into the temp filename
// safely. Empty labels are tolerated.
func sanitizeLabel(label string) string {
	if label == "" {
		return "run"
	}
	r := strings.NewReplacer(
		string(os.PathSeparator), "_",
		"/", "_",
		"\\", "_",
		" ", "_",
	)
	return r.Replace(label)
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
