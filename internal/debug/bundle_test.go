package debug_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/debug"
)

func TestScrubSettingsMap(t *testing.T) {
	in := map[string]string{
		"proxmox_url":      "https://10.90.0.1:8006",
		"proxmox_token":    "user@pam!benchere=secret-uuid-XYZ",
		"proxmox_password": "supersecret",
		"network_bridge":   "vmbr0",
		"ssh_key_path":     "/opt/benchere/id_rsa",
	}
	out := debug.ScrubSettingsMap(in)
	if out["proxmox_url"] != "https://10.90.0.1:8006" {
		t.Errorf("non-secret got mutated: %q", out["proxmox_url"])
	}
	if out["network_bridge"] != "vmbr0" {
		t.Errorf("non-secret got mutated: %q", out["network_bridge"])
	}
	for _, k := range []string{"proxmox_token", "proxmox_password", "ssh_key_path"} {
		if !strings.Contains(out[k], "SCRUBBED") {
			t.Errorf("expected %s to be scrubbed, got %q", k, out[k])
		}
	}
}

func TestScrubStorageCfgText(t *testing.T) {
	in := `dir: local
  path /var/lib/vz
  content vztmpl,iso

rbd: ceph-pool
  monhost 10.0.0.1 10.0.0.2
  password aaa-bbb-ccc-ddd
  pool rbd
`
	out := debug.ScrubStorageCfgText(in)
	if strings.Contains(out, "aaa-bbb-ccc-ddd") {
		t.Fatalf("password leaked into output:\n%s", out)
	}
	if !strings.Contains(out, "***SCRUBBED***") {
		t.Fatalf("missing scrub marker:\n%s", out)
	}
	if !strings.Contains(out, "monhost 10.0.0.1") {
		t.Fatalf("non-secret line got mutated:\n%s", out)
	}
}

func TestBuildManifest_Stable(t *testing.T) {
	created, _ := time.Parse(time.RFC3339, "2026-04-27T10:00:00Z")
	finished, _ := time.Parse(time.RFC3339, "2026-04-27T10:30:00Z")
	gen, _ := time.Parse(time.RFC3339, "2026-04-27T11:00:00Z")
	entries := []debug.ManifestEntry{
		{Path: "z.txt", Size: 10},
		{Path: "a/b.txt", Size: 200},
		{Path: "MANIFEST.txt", Size: 0},
	}
	m1 := debug.BuildManifest("v1.10.0", "j-1", created, finished, gen, entries)
	m2 := debug.BuildManifest("v1.10.0", "j-1", created, finished, gen, entries)
	if m1 != m2 {
		t.Fatal("manifest output is not deterministic")
	}
	if !strings.Contains(m1, "Benchere debug bundle") {
		t.Errorf("missing header: %q", m1)
	}
	// Sorted check: a/b.txt should appear before z.txt
	a := strings.Index(m1, "a/b.txt")
	z := strings.Index(m1, "z.txt")
	if a == -1 || z == -1 || a > z {
		t.Errorf("entries not sorted alphabetically:\n%s", m1)
	}
}

// TestBuildBundle_Structure verifies the assembler produces a tar.gz
// containing the expected core files when given a minimal DB and a job dir
// with a few artifacts. External collectors (journalctl, proxmox, ceph) are
// disabled so the test is hermetic.
func TestBuildBundle_Structure(t *testing.T) {
	// Set up a temporary DB.
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.sqlite")
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	// Insert a job and a result.
	job := db.Job{
		ID:         "test-job-id",
		Name:       "smoke",
		ClientName: "client",
		Status:     "done",
		Mode:       "storage",
		CreatedAt:  time.Now().Add(-1 * time.Hour),
	}
	if err := d.CreateJob(job); err != nil {
		t.Fatal(err)
	}
	if err := d.FinishJob(job.ID, "done"); err != nil {
		t.Fatal(err)
	}
	// Insert a fake setting to ensure scrubbing works through the bundle.
	if err := d.SetSetting("proxmox_password", "REVEALED-IF-NOT-SCRUBBED"); err != nil {
		t.Fatal(err)
	}
	if err := d.SetSetting("network_bridge", "vmbr0"); err != nil {
		t.Fatal(err)
	}

	// Set up a fake jobs dir with one elbencho stdout file.
	jobsDir := filepath.Join(tmpDir, "jobs")
	jobArtifactDir := filepath.Join(jobsDir, job.ID, "elbencho")
	if err := os.MkdirAll(jobArtifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobArtifactDir, "prefill.stdout"),
		[]byte("hello prefill\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	src := debug.BundleSources{
		DB:             d,
		Proxmox:        nil,
		JobsDir:        jobsDir,
		Version:        "v1.10.0-test",
		JobID:          job.ID,
		IncludeJournal: false,
		IncludeProxmox: false,
		IncludeCeph:    false,
		IncludeDBCopy:  true,
	}
	if err := debug.Build(context.Background(), &buf, src); err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Validate tar.gz structure.
	gz, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		files[hdr.Name] = data
	}

	required := []string{
		"MANIFEST.txt",
		"README.txt",
		"benchere/version.txt",
		"benchere/settings.json",
		"benchere/db.sqlite",
		"job/elbencho/prefill.stdout",
		"job/results.csv",
		"job/metrics.csv",
	}
	for _, name := range required {
		if _, ok := files[name]; !ok {
			t.Errorf("missing entry %q in bundle (got: %v)", name, mapKeys(files))
		}
	}
	if !strings.Contains(string(files["benchere/settings.json"]), "***SCRUBBED***") {
		t.Errorf("scrubbing did not apply, settings.json=%s", files["benchere/settings.json"])
	}
	if strings.Contains(string(files["benchere/settings.json"]), "REVEALED-IF-NOT-SCRUBBED") {
		t.Errorf("password leaked into bundle, settings.json=%s",
			files["benchere/settings.json"])
	}
	if string(files["job/elbencho/prefill.stdout"]) != "hello prefill\n" {
		t.Errorf("artifact mismatch: %q", string(files["job/elbencho/prefill.stdout"]))
	}
	// db.sqlite must look like a SQLite file (magic header).
	if !bytes.HasPrefix(files["benchere/db.sqlite"], []byte("SQLite format 3")) {
		t.Errorf("db.sqlite does not look like a SQLite file: first bytes %q",
			files["benchere/db.sqlite"][:min(20, len(files["benchere/db.sqlite"]))])
	}
}

func TestCleanOldJobDirs(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old-job")
	young := filepath.Join(root, "young-job")
	for _, p := range []string{old, young} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour)
	if err := os.Chtimes(old, twoWeeksAgo, twoWeeksAgo); err != nil {
		t.Fatal(err)
	}
	n := debug.CleanOldJobDirs(root, 7*24*time.Hour)
	if n != 1 {
		t.Errorf("expected 1 dir cleaned, got %d", n)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("old dir should have been removed: %v", err)
	}
	if _, err := os.Stat(young); err != nil {
		t.Errorf("young dir should still exist: %v", err)
	}
}

func mapKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
