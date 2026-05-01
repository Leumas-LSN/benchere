package benchmark

import (
	"math"
	"testing"
	"time"
)

func TestPhaseAggregator_BasicStats(t *testing.T) {
	a := NewPhaseAggregator("oltp-4k-70-30")
	now := time.Now()
	a.Push(storageMetric{
		Timestamp: now, IOPSRead: 80000, IOPSWrite: 30000,
		ThroughputReadMBps: 312.5, LatencyP50Ms: 0.5, LatencyP95Ms: 1.2, LatencyP99Ms: 2.0, LatencyAvgMs: 0.6,
	})
	a.Push(storageMetric{
		Timestamp: now.Add(2 * time.Second), IOPSRead: 84000, IOPSWrite: 32000,
		ThroughputReadMBps: 328.1, LatencyP50Ms: 0.55, LatencyP95Ms: 1.3, LatencyP99Ms: 2.4, LatencyAvgMs: 0.65,
	})

	s := a.Snapshot(now.Add(2 * time.Second))
	if s.SamplesCount != 2 {
		t.Errorf("SamplesCount=%d want 2", s.SamplesCount)
	}
	if s.IOPSReadAvg < 81000 || s.IOPSReadAvg > 83000 {
		t.Errorf("IOPSReadAvg=%v want ~82000", s.IOPSReadAvg)
	}
	if s.IOPSReadMax != 84000 {
		t.Errorf("IOPSReadMax=%v want 84000", s.IOPSReadMax)
	}
	if s.LatP99Ms < 2.0 || s.LatP99Ms > 2.5 {
		t.Errorf("LatP99Ms=%v want a value between samples", s.LatP99Ms)
	}
	// CV% = sample_stddev/mean * 100. With 80000, 84000:
	// mean=82000, sample stddev (n-1=1) = sqrt(8000000) / 1 = 2828.4
	// CV = 2828.4/82000 * 100 = ~3.45%
	if math.Abs(s.IOPSCVPct-3.45) > 0.5 {
		t.Errorf("IOPSCVPct=%v want ~3.45 (sample stddev)", s.IOPSCVPct)
	}
}

func TestPhaseAggregator_EmptyIsSafe(t *testing.T) {
	a := NewPhaseAggregator("none")
	s := a.Snapshot(time.Now())
	if s.SamplesCount != 0 {
		t.Errorf("expected zero samples on empty aggregator, got %d", s.SamplesCount)
	}
}

// TestPhaseAggregator_WriteOnlyCV covers v2.1.1 fix: a write-only profile
// (rwmixread=0) feeds 0 into iops_read for every sample, so the CV must
// be computed from iops_write or it falsely shows 0%.
func TestPhaseAggregator_WriteOnlyCV(t *testing.T) {
	a := NewPhaseAggregator("peak-write-iops")
	now := time.Now()
	a.Push(storageMetric{
		Timestamp: now, IOPSRead: 0, IOPSWrite: 25000,
	})
	a.Push(storageMetric{
		Timestamp: now.Add(2 * time.Second), IOPSRead: 0, IOPSWrite: 27000,
	})
	a.Push(storageMetric{
		Timestamp: now.Add(4 * time.Second), IOPSRead: 0, IOPSWrite: 29000,
	})
	s := a.Snapshot(now.Add(4 * time.Second))
	if s.IOPSReadAvg != 0 {
		t.Errorf("IOPSReadAvg=%v want 0", s.IOPSReadAvg)
	}
	if s.IOPSWriteAvg != 27000 {
		t.Errorf("IOPSWriteAvg=%v want 27000", s.IOPSWriteAvg)
	}
	// Sample stddev of [25000, 27000, 29000] = 2000, mean = 27000, CV = ~7.4%
	if s.IOPSCVPct < 6.0 || s.IOPSCVPct > 9.0 {
		t.Errorf("IOPSCVPct=%v want ~7.4 (computed from iops_write because read leg is empty)", s.IOPSCVPct)
	}
}

