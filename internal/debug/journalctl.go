package debug

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// CaptureJournal returns journalctl output for the benchere unit, scoped to
// the job's window. since defaults to job.created_at - 1h, until defaults
// to job.finished_at + 5m (or now if the job is still running).
//
// Best-effort: if journalctl is missing or fails the returned bytes carry
// the error message instead of empty so the bundle is never silently empty.
func CaptureJournal(ctx context.Context, jobCreated, jobFinished time.Time) []byte {
	since := jobCreated.Add(-1 * time.Hour).Format("2006-01-02 15:04:05")
	var until string
	if !jobFinished.IsZero() {
		until = jobFinished.Add(5 * time.Minute).Format("2006-01-02 15:04:05")
	} else {
		until = time.Now().Format("2006-01-02 15:04:05")
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "journalctl",
		"-u", "benchere",
		"--since", since,
		"--until", until,
		"--no-pager",
		"-o", "short-iso",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		out = append(out, []byte(fmt.Sprintf("\n# journalctl failed: %v\n", err))...)
	}
	return out
}
