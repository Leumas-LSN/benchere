package fio

import (
	"math"
	"testing"
	"time"
)

func TestMergeSnapshots_SumsIOPSAndThroughput(t *testing.T) {
	host := func(iopsR, iopsW, bwR, bwW float64) Snapshot {
		return Snapshot{
			IOPSRead:            iopsR,
			IOPSWrite:           iopsW,
			ThroughputReadMBps:  bwR,
			ThroughputWriteMBps: bwW,
			LatencyReadMeanMs:   0.5,
			LatencyWriteMeanMs:  0.7,
		}
	}
	in := map[string]Snapshot{
		"w1": host(10000, 4000, 40, 16),
		"w2": host(12000, 5000, 48, 20),
		"w3": host(11000, 4500, 44, 18),
	}
	m := mergeSnapshots("oltp-4k-70-30", in)

	if m.IOPSRead != 33000 {
		t.Errorf("IOPSRead=%v want 33000", m.IOPSRead)
	}
	if m.IOPSWrite != 13500 {
		t.Errorf("IOPSWrite=%v want 13500", m.IOPSWrite)
	}
	if m.ThroughputReadMBps != 132 {
		t.Errorf("ThroughputReadMBps=%v want 132", m.ThroughputReadMBps)
	}
	if m.ThroughputWriteMBps != 54 {
		t.Errorf("ThroughputWriteMBps=%v want 54", m.ThroughputWriteMBps)
	}
	if m.ProfileName != "oltp-4k-70-30" {
		t.Errorf("ProfileName=%q want oltp-4k-70-30", m.ProfileName)
	}
}

func TestMergeSnapshots_WeightedAvgLatency(t *testing.T) {
	in := map[string]Snapshot{
		"slow": {
			IOPSRead:            1000,
			LatencyReadMeanMs:   5.0, // 1000 IOPS at 5ms
			IOPSWrite:           500,
			LatencyWriteMeanMs:  10.0,
		},
		"fast": {
			IOPSRead:            9000,
			LatencyReadMeanMs:   0.5, // 9000 IOPS at 0.5ms
			IOPSWrite:           4500,
			LatencyWriteMeanMs:  1.0,
		},
	}
	m := mergeSnapshots("test", in)

	// weighted read avg = (1000 * 5 + 9000 * 0.5) / 10000 = (5000 + 4500) / 10000 = 0.95
	if math.Abs(m.LatencyReadAvgMs-0.95) > 0.01 {
		t.Errorf("LatencyReadAvgMs=%v want ~0.95", m.LatencyReadAvgMs)
	}
	// weighted write avg = (500 * 10 + 4500 * 1) / 5000 = 9500 / 5000 = 1.9
	if math.Abs(m.LatencyWriteAvgMs-1.9) > 0.01 {
		t.Errorf("LatencyWriteAvgMs=%v want ~1.9", m.LatencyWriteAvgMs)
	}
}

func TestMergeSnapshots_TailLatencyTakesMax(t *testing.T) {
	in := map[string]Snapshot{
		"good": {
			IOPSRead:           5000,
			LatencyReadP50Ms:   0.3,
			LatencyReadP95Ms:   0.8,
			LatencyReadP99Ms:   1.2,
			LatencyReadP999Ms:  2.0,
			LatencyWriteP99Ms:  0.4,
		},
		"bad": {
			IOPSRead:           5000,
			LatencyReadP50Ms:   0.4,  // slightly worse p50
			LatencyReadP95Ms:   2.5,  // much worse p95
			LatencyReadP99Ms:   8.0,  // outlier p99
			LatencyReadP999Ms:  20.0, // outlier p99.9
			LatencyWriteP99Ms:  3.0,  // worse write p99
		},
	}
	m := mergeSnapshots("test", in)

	if m.LatencyP50Ms != 0.4 {
		t.Errorf("LatencyP50Ms=%v want 0.4 (max)", m.LatencyP50Ms)
	}
	if m.LatencyP95Ms != 2.5 {
		t.Errorf("LatencyP95Ms=%v want 2.5 (max)", m.LatencyP95Ms)
	}
	if m.LatencyP99Ms != 8.0 {
		t.Errorf("LatencyP99Ms=%v want 8.0 (max - worst worker visible globally)", m.LatencyP99Ms)
	}
	if m.LatencyP999Ms != 20.0 {
		t.Errorf("LatencyP999Ms=%v want 20.0 (max)", m.LatencyP999Ms)
	}
	if m.LatencyWriteP99Ms != 3.0 {
		t.Errorf("LatencyWriteP99Ms=%v want 3.0 (max)", m.LatencyWriteP99Ms)
	}
}

func TestMergeSnapshots_WriteOnly_LatencyAvgFallback(t *testing.T) {
	// All workers running a write-only profile: IOPSRead=0 everywhere.
	// LatencyAvgMs (legacy single-value field) should fall back to the
	// weighted write average.
	in := map[string]Snapshot{
		"w1": {IOPSRead: 0, IOPSWrite: 5000, LatencyWriteMeanMs: 2.0},
		"w2": {IOPSRead: 0, IOPSWrite: 5000, LatencyWriteMeanMs: 4.0},
	}
	m := mergeSnapshots("test", in)

	if m.LatencyReadAvgMs != 0 {
		t.Errorf("LatencyReadAvgMs=%v want 0 for write-only profile", m.LatencyReadAvgMs)
	}
	// weighted write avg = (5000 * 2 + 5000 * 4) / 10000 = 3.0
	if math.Abs(m.LatencyWriteAvgMs-3.0) > 0.01 {
		t.Errorf("LatencyWriteAvgMs=%v want 3.0", m.LatencyWriteAvgMs)
	}
	if m.LatencyAvgMs != m.LatencyWriteAvgMs {
		t.Errorf("LatencyAvgMs=%v want LatencyWriteAvgMs=%v (read fallback)",
			m.LatencyAvgMs, m.LatencyWriteAvgMs)
	}
}

func TestMergeSnapshots_PickMedianTimestamp(t *testing.T) {
	t1 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(2 * time.Second)
	t3 := t1.Add(4 * time.Second)
	in := map[string]Snapshot{
		"w1": {IOPSRead: 1, Timestamp: t1},
		"w2": {IOPSRead: 1, Timestamp: t2},
		"w3": {IOPSRead: 1, Timestamp: t3},
	}
	m := mergeSnapshots("test", in)

	if !m.Timestamp.Equal(t2) {
		t.Errorf("Timestamp=%v want %v (median)", m.Timestamp, t2)
	}
}

func TestMergeSnapshots_EmptyInputDoesNotPanic(t *testing.T) {
	m := mergeSnapshots("test", map[string]Snapshot{})
	if m.IOPSRead != 0 || m.IOPSWrite != 0 {
		t.Errorf("expected zero metric on empty input, got %+v", m)
	}
	if m.ProfileName != "test" {
		t.Errorf("ProfileName=%q want test", m.ProfileName)
	}
}

func TestSanitizeHost(t *testing.T) {
	cases := map[string]string{
		"10.91.0.69":  "10-91-0-69",
		"worker:8765": "worker-8765",
		"a/b":         "a-b",
	}
	for in, want := range cases {
		if got := sanitizeHost(in); got != want {
			t.Errorf("sanitizeHost(%q)=%q want %q", in, got, want)
		}
	}
}
