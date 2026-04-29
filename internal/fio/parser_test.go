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
	if err := streamSnapshots(context.Background(), combined, rawSink, "test-label", out); err != nil && err != io.EOF {
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
	if !strings.Contains(got, "size=50G") {
		t.Errorf("missing size: %s", got)
	}
	if !strings.Contains(got, "filename=/dev/sda") || !strings.Contains(got, "filename=/dev/sdb") {
		t.Errorf("missing target: %s", got)
	}
	if !strings.Contains(got, "[prefill_0]") || !strings.Contains(got, "[prefill_1]") {
		t.Errorf("missing per-target sections: %s", got)
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
