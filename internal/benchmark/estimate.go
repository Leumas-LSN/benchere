package benchmark

import (
	"strconv"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/db"
)

// EstimateInput is the wizard payload accepted by POST /api/jobs/estimate.
// All fields are optional and missing values fall back to safe defaults so
// the wizard can call the endpoint after each step change without first
// being valid.
type EstimateInput struct {
	Mode             string   `json:"mode"`
	Workers          int      `json:"workers"`
	DataDisks        int      `json:"data_disks"`
	DataDiskGB       int      `json:"data_disk_gb"`
	Profiles         []string `json:"profiles"`
	RuntimeSeconds   int      `json:"runtime_sec"`
	RampSeconds      int      `json:"ramp_sec"`
	StressTimeoutSec int      `json:"stress_timeout_sec"`
}

// EstimateOutput holds the wallclock duration and prefill plus benchmark
// write estimate. Both numbers are rough but defensible: the wizard is
// expected to surface them as approximate budgets, not contractual SLAs.
type EstimateOutput struct {
	WallclockSec int   `json:"wallclock_sec"`
	BytesWritten int64 `json:"bytes_written"`
}

// prefillThroughputMBps is the assumed sustained sequential write rate per
// worker during the elbencho or fio prefill phase. A modern NVMe-backed
// storage easily exceeds this; underprovisioned ZFS or spinning rust may
// fall short. The estimator rounds the wallclock up so users are not
// surprised when a slow pool takes longer.
const prefillThroughputMBps = 500

// Estimate computes the wallclock and bytes-written budget for a wizard
// configuration. profiles is keyed by profile name so the estimator can
// peek at runtime/ramp overrides and the rw= field embedded in the
// fio config_json. A nil profiles map is acceptable; the function
// falls back to the input-provided runtime/ramp values for every
// profile in that case.
func Estimate(in EstimateInput, profiles map[string]*db.Profile) (EstimateOutput, error) {
	mode := in.Mode
	if mode == "" {
		mode = "storage"
	}

	workers := in.Workers
	if workers < 1 {
		workers = 1
	}

	runtime := in.RuntimeSeconds
	if runtime <= 0 {
		runtime = 120
	}
	ramp := in.RampSeconds
	if ramp < 0 {
		ramp = 0
	}

	stressTimeout := in.StressTimeoutSec
	if stressTimeout < 0 {
		stressTimeout = 0
	}

	// Prefill duration. Each worker prefills its own data disks
	// sequentially (one elbencho or fio job), but workers prefill in
	// parallel because the orchestrator launches them simultaneously.
	// Wall time = (per-worker prefill bytes) / (per-worker throughput).
	// Total prefill bytes per worker = data_disks * data_disk_gb GB.
	dataDisks := in.DataDisks
	if dataDisks < 0 {
		dataDisks = 0
	}
	dataDiskGB := in.DataDiskGB
	if dataDiskGB < 0 {
		dataDiskGB = 0
	}
	perWorkerPrefillMB := dataDisks * dataDiskGB * 1024
	prefillSec := 0
	if perWorkerPrefillMB > 0 {
		prefillSec = perWorkerPrefillMB / prefillThroughputMBps
		if prefillSec < 1 {
			prefillSec = 1
		}
	}

	// Storage and mixed: prefill plus the sum of (runtime + ramp) for
	// each selected profile. cpu mode skips both.
	storageWall := 0
	if mode == "storage" || mode == "mixed" {
		storageWall = prefillSec
		if len(in.Profiles) == 0 {
			// Wizard at step 3 with no profiles selected: still surface
			// the prefill budget so users see numbers move when they
			// pick a pool.
		} else {
			for _, name := range in.Profiles {
				pRuntime := runtime
				pRamp := ramp
				if profiles != nil {
					if p, ok := profiles[name]; ok && p != nil {
						pRuntime = profileRuntime(p, runtime)
						pRamp = profileRamp(p, ramp)
					}
				}
				storageWall += pRuntime + pRamp
			}
		}
	}

	// CPU stage runs sequentially after storage in mixed mode (V1).
	// Pure cpu mode is just the stress timeout.
	wall := storageWall
	switch mode {
	case "cpu":
		wall = stressTimeout
	case "mixed":
		wall += stressTimeout
	}

	// Bytes written. Prefill writes the full per-worker capacity once,
	// across every worker. When at least one selected profile writes
	// (rw set to randwrite, write or randrw with nonzero write share),
	// double the prefill total to cover the steady-state writes during
	// the run. This is intentionally an over-estimate, not a tight
	// bound, so users do not under-provision the pool.
	prefillBytes := int64(dataDisks) * int64(dataDiskGB) * int64(1024) * int64(1024) * int64(1024) * int64(workers)
	bytes := prefillBytes
	if mode == "storage" || mode == "mixed" {
		if anyProfileWrites(in.Profiles, profiles) {
			bytes += prefillBytes
		}
	}

	return EstimateOutput{
		WallclockSec: wall,
		BytesWritten: bytes,
	}, nil
}

// profileRuntime extracts the runtime= value from a profile config_json
// (fio INI). Falls back to the wizard runtime when the profile does not
// override.
func profileRuntime(p *db.Profile, fallback int) int {
	if v, ok := iniValue(p.ConfigJSON, "runtime"); ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// profileRamp extracts the ramp_time= value from a profile config_json.
func profileRamp(p *db.Profile, fallback int) int {
	if v, ok := iniValue(p.ConfigJSON, "ramp_time"); ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}

// iniValue reads the first occurrence of key= from a fio-style INI body.
// The match ignores leading whitespace so [global] keys behave the same
// as section keys.
func iniValue(body, key string) (string, bool) {
	prefix := key + "="
	for _, line := range strings.Split(body, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(l, prefix)), true
		}
	}
	return "", false
}

// anyProfileWrites returns true when at least one profile name is
// associated with a fio config that writes (randwrite, write, randrw).
// Unknown profile names are conservatively assumed to write so the
// budget never under-estimates.
func anyProfileWrites(names []string, profiles map[string]*db.Profile) bool {
	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		if profiles == nil {
			return true
		}
		p, ok := profiles[name]
		if !ok || p == nil {
			return true
		}
		rw, _ := iniValue(p.ConfigJSON, "rw")
		rw = strings.ToLower(strings.TrimSpace(rw))
		switch rw {
		case "write", "randwrite", "randrw":
			return true
		}
	}
	return false
}
