package fio

import (
	"bytes"
	"context"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadFixture reads testdata/<name> as raw bytes.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return data
}

func TestParseStatusSnapshot_RandRW70(t *testing.T) {
	data := loadFixture(t, "snapshot_randrw_70r30w.json")

	snap, err := ParseStatusSnapshot(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// expected values from the captured fixture (fio --rw=randrw --rwmixread=70)
	const (
		wantIOPSRead  = 615626.760563
		wantIOPSWrite = 264252.347418
		wantBWReadB   = 2521270685.0 // bw_bytes
		wantBWWriteB  = 1082223774.0
		wantP99ReadNS = 1144.0
	)

	if !approxEqual(snap.IOPSRead, wantIOPSRead, 0.01) {
		t.Errorf("IOPSRead=%v want %v", snap.IOPSRead, wantIOPSRead)
	}
	if !approxEqual(snap.IOPSWrite, wantIOPSWrite, 0.01) {
		t.Errorf("IOPSWrite=%v want %v", snap.IOPSWrite, wantIOPSWrite)
	}

	wantBWReadMBps := wantBWReadB / (1024.0 * 1024.0)
	if !approxEqual(snap.ThroughputReadMBps, wantBWReadMBps, 0.01) {
		t.Errorf("ThroughputReadMBps=%v want %v", snap.ThroughputReadMBps, wantBWReadMBps)
	}
	wantBWWriteMBps := wantBWWriteB / (1024.0 * 1024.0)
	if !approxEqual(snap.ThroughputWriteMBps, wantBWWriteMBps, 0.01) {
		t.Errorf("ThroughputWriteMBps=%v want %v", snap.ThroughputWriteMBps, wantBWWriteMBps)
	}

	wantP99ReadMs := wantP99ReadNS / 1_000_000.0
	if !approxEqual(snap.LatencyReadP99Ms, wantP99ReadMs, 1e-9) {
		t.Errorf("LatencyReadP99Ms=%v want %v", snap.LatencyReadP99Ms, wantP99ReadMs)
	}
	if snap.JobName != "t" {
		t.Errorf("JobName=%q want %q", snap.JobName, "t")
	}
	if snap.Timestamp.IsZero() {
		t.Errorf("Timestamp is zero")
	}
}

func TestSnapshot_ToMetric_PreservesValues(t *testing.T) {
	data := loadFixture(t, "snapshot_randrw_70r30w.json")
	snap, err := ParseStatusSnapshot(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	m := snap.ToMetric("rand-4k-70r30w")
	if m.ProfileName != "rand-4k-70r30w" {
		t.Errorf("ProfileName=%q want rand-4k-70r30w", m.ProfileName)
	}
	if !approxEqual(m.IOPSRead, snap.IOPSRead, 0) {
		t.Errorf("IOPSRead carried wrong: %v vs %v", m.IOPSRead, snap.IOPSRead)
	}
	if !approxEqual(m.IOPSWrite, snap.IOPSWrite, 0) {
		t.Errorf("IOPSWrite carried wrong: %v vs %v", m.IOPSWrite, snap.IOPSWrite)
	}
	if !approxEqual(m.LatencyAvgMs, snap.LatencyReadMeanMs, 0) {
		t.Errorf("LatencyAvgMs should be read mean, got %v want %v", m.LatencyAvgMs, snap.LatencyReadMeanMs)
	}
}

func TestStreamSnapshots_BoundaryParsing(t *testing.T) {
	// Two snapshots concatenated, separated by a "Hostname banner" line that
	// fio sometimes emits in client mode.
	first := loadFixture(t, "snapshot_randrw_70r30w.json")
	combined := bytes.NewBuffer(nil)
	combined.Write(first)
	combined.WriteString("\n")
	combined.WriteString("hostname=worker-1\n") // ignorable banner
	combined.Write(first)
	combined.WriteString("\n")

	out := make(chan Metric, 4)
	rawSink := &bytes.Buffer{}
	emitted := make(map[string]bool)
	if err := streamSnapshots(context.Background(), combined, rawSink, "test-label", out, emitted); err != nil && err != io.EOF {
		t.Fatalf("streamSnapshots: %v", err)
	}
	close(out)

	var got []Metric
	for m := range out {
		got = append(got, m)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(got))
	}
	for i, m := range got {
		if m.IOPSRead == 0 {
			t.Errorf("metric %d: IOPSRead=0", i)
		}
		if m.ProfileName != "test-label" {
			t.Errorf("metric %d: ProfileName=%q want test-label", i, m.ProfileName)
		}
	}
	if !strings.Contains(rawSink.String(), "fio version") {
		t.Errorf("raw sink missing fio version marker")
	}
}

func TestParseStatusSnapshot_NoJobs(t *testing.T) {
	_, err := ParseStatusSnapshot([]byte(`{"fio version":"x","jobs":[]}`))
	if err == nil {
		t.Errorf("expected error on empty jobs")
	}
}

func TestBuildJobfile_ReplacesTarget(t *testing.T) {
	tmp, err := BuildJobfile("rand-4k-read",
		"[global]\nfilename=<TARGET>\n[a]\nrw=randread\n",
		[]string{"/dev/disk/by-id/foo"})
	if err != nil {
		t.Fatalf("BuildJobfile: %v", err)
	}
	defer os.Remove(tmp)
	data, _ := os.ReadFile(tmp)
	if !strings.Contains(string(data), "filename=/dev/disk/by-id/foo") {
		t.Errorf("placeholder not substituted: %s", string(data))
	}
}

func TestBuildPrefillJobfile_OnePerTarget(t *testing.T) {
	got := buildPrefillJobfile([]string{"/dev/sda", "/dev/sdb"}, 50)
	if !strings.Contains(got, "size=25%") {
		t.Errorf("missing size=25%% directive: %s", got)
	}
	if !strings.Contains(got, "filename=/dev/sda") || !strings.Contains(got, "filename=/dev/sdb") {
		t.Errorf("missing target: %s", got)
	}
	if !strings.Contains(got, "[prefill_0]") || !strings.Contains(got, "[prefill_1]") {
		t.Errorf("missing per-target sections: %s", got)
	}
}

// TestParseFinalDocumentClientStats covers the --client/--server output
// shape: a single document at the end of the run with client_stats[]
// holding one entry per status interval. Each entry has a job_runtime in
// milliseconds since the run started; ParseFinalDocument folds them into
// individual snapshots with strictly increasing timestamps.
func TestParseFinalDocumentClientStats(t *testing.T) {
	data := loadFixture(t, "client_stats_sample.json")

	snaps, err := ParseFinalDocument(data)
	if err != nil {
		t.Fatalf("ParseFinalDocument: %v", err)
	}
	if len(snaps) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snaps))
	}

	if snaps[0].IOPSRead <= 0 {
		t.Errorf("first snapshot IOPSRead=%v want > 0", snaps[0].IOPSRead)
	}
	if snaps[0].Timestamp.IsZero() {
		t.Errorf("first snapshot Timestamp is zero")
	}
	if snaps[0].JobName != "rand-4k-read" {
		t.Errorf("JobName=%q want rand-4k-read", snaps[0].JobName)
	}

	// Timestamps must be strictly increasing (entries are time-ordered).
	for i := 1; i < len(snaps); i++ {
		if !snaps[i].Timestamp.After(snaps[i-1].Timestamp) {
			t.Errorf("snapshot %d timestamp %v not after previous %v", i, snaps[i].Timestamp, snaps[i-1].Timestamp)
		}
	}

	// Every snapshot should carry a non-empty version string from the
	// run-level field, and at least the read leg should be populated.
	for i, s := range snaps {
		if s.FioVersion == "" {
			t.Errorf("snapshot %d missing fio version", i)
		}
		if s.IOPSRead <= 0 {
			t.Errorf("snapshot %d IOPSRead=%v want > 0", i, s.IOPSRead)
		}
	}
}

// TestParseFinalDocumentSingleMode confirms backward compatibility: when
// the document has the legacy jobs[] shape (no client_stats), we still
// return one snapshot just like ParseStatusSnapshot would.
func TestParseFinalDocumentSingleMode(t *testing.T) {
	data := loadFixture(t, "snapshot_randrw_70r30w.json")
	snaps, err := ParseFinalDocument(data)
	if err != nil {
		t.Fatalf("ParseFinalDocument: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].IOPSRead <= 0 {
		t.Errorf("IOPSRead=%v want > 0", snaps[0].IOPSRead)
	}
}

// TestParseFinalDocumentEmpty rejects documents that have neither
// jobs[] nor client_stats[].
func TestParseFinalDocumentEmpty(t *testing.T) {
	_, err := ParseFinalDocument([]byte(`{"fio version":"x"}`))
	if err == nil {
		t.Errorf("expected error on document with no jobs or client_stats")
	}
}

func approxEqual(a, b, tol float64) bool {
	if math.Abs(a-b) <= tol {
		return true
	}
	if b == 0 {
		return false
	}
	return math.Abs(a-b)/math.Abs(b) <= tol
}
