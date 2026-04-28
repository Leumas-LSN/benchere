package debug

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ManifestEntry is one row in the bundle index. Size is the size in bytes of
// the entry as it lives in the archive (not the gzipped size).
type ManifestEntry struct {
	Path string
	Size int64
}

// BuildManifest renders a stable text index of the bundle, suitable for
// MANIFEST.txt. The header lines describe the bundle context and the
// remaining lines list every entry sorted by path.
func BuildManifest(version, jobID string, jobCreated, jobFinished, generated time.Time, entries []ManifestEntry) string {
	sorted := make([]ManifestEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	var b strings.Builder
	fmt.Fprintf(&b, "Benchere debug bundle\n")
	fmt.Fprintf(&b, "Version:     %s\n", version)
	fmt.Fprintf(&b, "Job ID:      %s\n", jobID)
	if !jobCreated.IsZero() {
		fmt.Fprintf(&b, "Job created: %s\n", jobCreated.UTC().Format(time.RFC3339))
	}
	if !jobFinished.IsZero() {
		fmt.Fprintf(&b, "Job ended:   %s\n", jobFinished.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "Generated:   %s\n", generated.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "Entries:     %d\n", len(sorted))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "PATH\tSIZE")
	for _, e := range sorted {
		fmt.Fprintf(&b, "%s\t%d\n", e.Path, e.Size)
	}
	return b.String()
}

// BuildReadme renders the README.txt body. It is a one-paragraph summary so
// the operator who downloads the bundle understands the contents and the
// scrubbing guarantees.
func BuildReadme(version string) string {
	return fmt.Sprintf(`This archive is a Benchere %s debug bundle.
It is intended for support diagnosis and contains:
  - a SQLite snapshot of the live database (benchere/db.sqlite)
  - the in-memory JobConfig used by the orchestrator (job/config.json)
  - the application settings, with passwords and keys scrubbed (benchere/settings.json)
  - relevant journalctl output captured around the job window
  - the raw elbencho stdout, stderr and CSV resfiles per phase
  - the ansible playbook stdout and stderr
  - per-worker sysinfo captured over SSH right before VM cleanup
  - the Proxmox cluster topology and node statuses
  - a best-effort dump of Ceph status, df, pools and OSDs
Secrets that match one of the known patterns (password, secret, token, key)
are replaced with %s before they enter the archive. The DB snapshot is the
unfiltered live state, so user-supplied client/job names are preserved as is.
`, version, scrubMarker)
}
