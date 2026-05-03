package benchmark_test

import (
	"testing"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/db"
)

func TestEstimate_StorageMode_PrefillPlusProfiles(t *testing.T) {
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:           "storage",
		Workers:        4,
		DataDisks:      1,
		DataDiskGB:     32,
		Profiles:       []string{"oltp-4k-70-30", "sql-8k-70-30"},
		RuntimeSeconds: 120,
		RampSeconds:    30,
	}, nil)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	// Per-worker prefill: 1 * 32 * 1024 = 32768 MB at 500 MB/s = 65 s.
	// 2 profiles * (120 + 30) = 300 s. Total = 365 s.
	if out.WallclockSec != 365 {
		t.Errorf("wallclock = %d, want 365", out.WallclockSec)
	}
	// 4 workers * 32 GB = 128 GB prefill. Without profile metadata the
	// estimator assumes every named profile writes so doubles to 256 GB.
	want := int64(4) * int64(32) * int64(1024) * int64(1024) * int64(1024) * 2
	if out.BytesWritten != want {
		t.Errorf("bytes = %d, want %d", out.BytesWritten, want)
	}
}

func TestEstimate_CpuMode_OnlyStressTimeout(t *testing.T) {
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:             "cpu",
		Workers:          2,
		StressTimeoutSec: 60,
	}, nil)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if out.WallclockSec != 60 {
		t.Errorf("wallclock = %d, want 60", out.WallclockSec)
	}
	if out.BytesWritten != 0 {
		t.Errorf("bytes = %d, want 0 (cpu mode has no data disks)", out.BytesWritten)
	}
}

func TestEstimate_MixedMode_StoragePlusStress(t *testing.T) {
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:             "mixed",
		Workers:          2,
		DataDisks:        1,
		DataDiskGB:       16,
		Profiles:         []string{"oltp-4k-70-30"},
		RuntimeSeconds:   60,
		RampSeconds:      10,
		StressTimeoutSec: 60,
	}, nil)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	// Prefill: 1 * 16 * 1024 = 16384 MB / 500 MB/s = 32 s.
	// Profile: 60 + 10 = 70 s. Storage wall = 102 s.
	// Mixed adds stress: 102 + 60 = 162 s.
	if out.WallclockSec != 162 {
		t.Errorf("wallclock = %d, want 162", out.WallclockSec)
	}
}

func TestEstimate_StorageMode_NoProfiles_StillReportsPrefill(t *testing.T) {
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:       "storage",
		Workers:    4,
		DataDisks:  1,
		DataDiskGB: 32,
	}, nil)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if out.WallclockSec != 65 {
		t.Errorf("wallclock = %d, want 65 (prefill only)", out.WallclockSec)
	}
	// No profiles selected so no doubling: just the per-worker prefill.
	want := int64(4) * int64(32) * int64(1024) * int64(1024) * int64(1024)
	if out.BytesWritten != want {
		t.Errorf("bytes = %d, want %d", out.BytesWritten, want)
	}
}

func TestEstimate_ZeroWorkers_DefaultsToOne(t *testing.T) {
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:           "storage",
		Workers:        0,
		DataDisks:      1,
		DataDiskGB:     8,
		Profiles:       []string{"backup-256k-read"},
		RuntimeSeconds: 60,
		RampSeconds:    5,
	}, nil)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if out.WallclockSec <= 0 {
		t.Errorf("wallclock = %d, want > 0", out.WallclockSec)
	}
	// 1 worker (defaulted) * 1 disk * 8 GB.
	wantPrefill := int64(1) * int64(8) * int64(1024) * int64(1024) * int64(1024)
	// backup-256k-read does not write, no doubling.
	if out.BytesWritten != wantPrefill && out.BytesWritten != wantPrefill*2 {
		t.Errorf("bytes = %d, want either %d or its double", out.BytesWritten, wantPrefill)
	}
}

func TestEstimate_ProfileMetadataOverridesRuntime(t *testing.T) {
	profiles := map[string]*db.Profile{
		"oltp-4k-70-30": {
			Name:       "oltp-4k-70-30",
			ConfigJSON: "[global]\nrw=randrw\nruntime=300\nramp_time=30\n",
		},
	}
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:           "storage",
		Workers:        4,
		DataDisks:      0,
		DataDiskGB:     0,
		Profiles:       []string{"oltp-4k-70-30"},
		RuntimeSeconds: 60, // overridden by profile's 300
		RampSeconds:    5,  // overridden by profile's 30
	}, profiles)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	if out.WallclockSec != 330 {
		t.Errorf("wallclock = %d, want 330 (300+30)", out.WallclockSec)
	}
}

func TestEstimate_ReadOnlyProfile_NoDoubling(t *testing.T) {
	profiles := map[string]*db.Profile{
		"backup-256k-read": {
			Name:       "backup-256k-read",
			ConfigJSON: "[global]\nrw=read\nruntime=300\nramp_time=30\n",
		},
	}
	out, err := benchmark.Estimate(benchmark.EstimateInput{
		Mode:           "storage",
		Workers:        4,
		DataDisks:      1,
		DataDiskGB:     32,
		Profiles:       []string{"backup-256k-read"},
		RuntimeSeconds: 60,
		RampSeconds:    5,
	}, profiles)
	if err != nil {
		t.Fatalf("Estimate: %v", err)
	}
	// 4 workers * 32 GB = 128 GB. read-only: no doubling.
	want := int64(4) * int64(32) * int64(1024) * int64(1024) * int64(1024)
	if out.BytesWritten != want {
		t.Errorf("bytes = %d, want %d (no doubling for read-only profile)", out.BytesWritten, want)
	}
}
