package elbencho

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RunConfig struct {
	Hosts       []string
	ConfigFile  string
	Targets     []string // e.g. ["/dev/sdb", "/dev/sdc"]
	LiveCSVPath string
	CSVPath     string
	Label       string

	// OutputDir, when non-empty, is where the runner persists the literal
	// invocation, stdout, stderr and CSV resfile under {Label}.{cmd,stdout,stderr,csv}.
	// The directory is created with MkdirAll. Failures to write artifacts are
	// logged but never returned (best-effort capture for debug bundles).
	OutputDir string
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

	return runWithCapture(ctx, "elbencho", args, cfg.OutputDir, cfg.Label, cfg.CSVPath)
}

// Prefill writes sequentially across each target on every host so that
// thin-provisioned backends (Ceph RBD, ZFS sparse zvols, ...) allocate every
// extent before any read profile runs. Without this, read benchmarks against
// freshly-provisioned worker disks measure the backend's fast path for
// unwritten extents at network bandwidth instead of real storage performance.
//
// Sequential 1 MiB writes with O_DIRECT, 4 threads per service, no random
// pattern. The backend allocates physical blocks as the writes land.
//
// outputDir, when non-empty, receives prefill.cmd, prefill.stdout, prefill.stderr.
func Prefill(ctx context.Context, hosts []string, targets []string, sizeGB int, outputDir string) error {
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

	return runWithCapture(ctx, "elbencho", args, outputDir, "prefill", "")
}

// runWithCapture invokes elbencho with the given args and, when outputDir is
// non-empty, persists the full command line, stdout, stderr and (if csvPath
// is provided and exists at the end of the run) the CSV resfile under
// outputDir/{label}.{cmd,stdout,stderr,csv}. Any persistence error is swallowed
// after a stderr log so the benchmark itself is never blocked on disk issues.
func runWithCapture(ctx context.Context, name string, args []string, outputDir, label, csvPath string) error {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdoutW io.Writer = os.Stdout
	var stderrW io.Writer = os.Stderr
	var stdoutFile, stderrFile *os.File

	if outputDir != "" && label != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[elbencho] mkdir %s: %v\n", outputDir, err)
		} else {
			cmdLine := name + " " + strings.Join(quoteAll(args), " ") + "\n"
			if err := os.WriteFile(filepath.Join(outputDir, label+".cmd"), []byte(cmdLine), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "[elbencho] write cmd: %v\n", err)
			}
			if f, err := os.Create(filepath.Join(outputDir, label+".stdout")); err != nil {
				fmt.Fprintf(os.Stderr, "[elbencho] create stdout file: %v\n", err)
			} else {
				stdoutFile = f
				stdoutW = io.MultiWriter(os.Stdout, f)
			}
			if f, err := os.Create(filepath.Join(outputDir, label+".stderr")); err != nil {
				fmt.Fprintf(os.Stderr, "[elbencho] create stderr file: %v\n", err)
			} else {
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

	if outputDir != "" && label != "" && csvPath != "" {
		if data, err := os.ReadFile(csvPath); err == nil {
			_ = os.WriteFile(filepath.Join(outputDir, label+".csv"), data, 0o644)
		}
	}

	if runErr != nil {
		return fmt.Errorf("elbencho run %s: %w", label, runErr)
	}
	return nil
}

// CaptureVersion writes `elbencho --version` output to outputDir/elbencho-version.txt.
// Best-effort: any error is swallowed so a missing or broken elbencho does not
// poison the bundle pipeline.
func CaptureVersion(ctx context.Context, outputDir string) {
	if outputDir == "" {
		return
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return
	}
	cmd := exec.CommandContext(ctx, "elbencho", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, []byte(fmt.Sprintf("\n# elbencho --version failed: %v\n", err))...)
	}
	_ = os.WriteFile(filepath.Join(outputDir, "elbencho-version.txt"), out, 0o644)
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
