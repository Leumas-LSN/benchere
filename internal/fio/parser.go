package fio

import (
	"encoding/json"
	"fmt"
	"time"
)

// Metric is the per-interval aggregate emitted to the orchestrator. The
// shape mirrors elbencho.Metric so persistMetrics can treat both engines
// the same way.
type Metric struct {
	Timestamp           time.Time
	ProfileName         string
	IOPSRead            float64
	IOPSWrite           float64
	ThroughputReadMBps  float64
	ThroughputWriteMBps float64

	// Legacy single-value avg latency, used by the dashboard tile.
	// Prefers read mean, falls back to write mean when read is zero.
	LatencyAvgMs float64

	// Read-leg percentile picture from fio clat_ns.percentile.
	LatencyReadAvgMs  float64
	LatencyWriteAvgMs float64
	LatencyP50Ms      float64
	LatencyP95Ms      float64
	LatencyP99Ms      float64
	LatencyP999Ms     float64
	LatencyWriteP99Ms float64
}

// rawSnapshot is the subset of fio JSON+ output we care about. Many fields
// are dropped; only those that drive a benchmark report are decoded.
//
// In single mode (fio invoked locally without --client) the document has
// "jobs" populated and a "timestamp_ms" field. In --client/--server mode
// fio buffers all status snapshots and emits a single document at the END
// of the run with "client_stats" populated (one entry per status interval,
// with "job_runtime" being elapsed milliseconds since the run started) and
// a top-level "timestamp" in seconds.
type rawSnapshot struct {
	FioVersion  string   `json:"fio version"`
	TimestampMs int64    `json:"timestamp_ms"`
	Timestamp   int64    `json:"timestamp"`
	Jobs        []rawJob `json:"jobs"`
	ClientStats []rawJob `json:"client_stats"`
}

type rawJob struct {
	JobName    string  `json:"jobname"`
	JobRuntime int64   `json:"job_runtime"`
	Read       rawIOSt `json:"read"`
	Write      rawIOSt `json:"write"`
}

type rawIOSt struct {
	IOPS    float64 `json:"iops"`
	BWKB    float64 `json:"bw"` // kilobytes per second
	BWBytes float64 `json:"bw_bytes"`
	ClatNS  rawClat `json:"clat_ns"`
	LatNS   rawLat  `json:"lat_ns"`
}

type rawClat struct {
	Mean       float64            `json:"mean"`
	Percentile map[string]float64 `json:"percentile"`
}

type rawLat struct {
	Mean float64 `json:"mean"`
}

// Snapshot is the typed view of one fio JSON+ status interval.
type Snapshot struct {
	FioVersion          string
	Timestamp           time.Time
	JobName             string
	IOPSRead            float64
	IOPSWrite           float64
	ThroughputReadMBps  float64
	ThroughputWriteMBps float64
	LatencyReadMeanMs   float64
	LatencyWriteMeanMs  float64
	LatencyReadP50Ms    float64
	LatencyReadP95Ms    float64
	LatencyReadP99Ms    float64
	LatencyReadP999Ms   float64
	LatencyWriteP99Ms   float64
}

// ParseStatusSnapshot decodes one fio JSON+ status object (single mode).
// The first job in the document is used (fio reports per-job, per-group;
// in single-host invocations we get one aggregated job entry).
func ParseStatusSnapshot(data []byte) (*Snapshot, error) {
	var raw rawSnapshot
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	if len(raw.Jobs) == 0 {
		return nil, fmt.Errorf("snapshot has no jobs")
	}
	j := raw.Jobs[0]
	s := snapshotFromJob(raw, j)
	if s.Timestamp.IsZero() {
		if raw.TimestampMs > 0 {
			s.Timestamp = time.Unix(raw.TimestampMs/1000, (raw.TimestampMs%1000)*1_000_000)
		} else if raw.Timestamp > 0 {
			s.Timestamp = time.Unix(raw.Timestamp, 0)
		} else {
			s.Timestamp = nowFn()
		}
	}
	return s, nil
}

// ParseFinalDocument decodes the JSON document fio writes once at the end
// of a run. It supports both shapes:
//
//   - single mode: a "jobs" array with one or more jobs. We return one
//     Snapshot, taking the first job (current behavior).
//   - --client/--server mode: a "client_stats" array with one entry per
//     status interval. We return one Snapshot per entry, with Timestamp
//     computed as run start (top-level "timestamp" in seconds) plus the
//     entry's "job_runtime" in milliseconds.
//
// This is the parser used after fio exits, so the live ordering of
// snapshots is preserved in the returned slice.
func ParseFinalDocument(data []byte) ([]Snapshot, error) {
	var raw rawSnapshot
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode final document: %w", err)
	}

	if len(raw.ClientStats) > 0 {
		runStart := time.Time{}
		if raw.Timestamp > 0 {
			runStart = time.Unix(raw.Timestamp, 0)
		} else if raw.TimestampMs > 0 {
			runStart = time.Unix(raw.TimestampMs/1000, (raw.TimestampMs%1000)*1_000_000)
		}
		out := make([]Snapshot, 0, len(raw.ClientStats))
		for _, j := range raw.ClientStats {
			s := snapshotFromJob(raw, j)
			if !runStart.IsZero() {
				s.Timestamp = runStart.Add(time.Duration(j.JobRuntime) * time.Millisecond)
			} else if s.Timestamp.IsZero() {
				s.Timestamp = nowFn()
			}
			out = append(out, *s)
		}
		return out, nil
	}

	if len(raw.Jobs) > 0 {
		j := raw.Jobs[0]
		s := snapshotFromJob(raw, j)
		if s.Timestamp.IsZero() {
			if raw.TimestampMs > 0 {
				s.Timestamp = time.Unix(raw.TimestampMs/1000, (raw.TimestampMs%1000)*1_000_000)
			} else if raw.Timestamp > 0 {
				s.Timestamp = time.Unix(raw.Timestamp, 0)
			} else {
				s.Timestamp = nowFn()
			}
		}
		return []Snapshot{*s}, nil
	}

	return nil, fmt.Errorf("final document has neither jobs nor client_stats")
}

// snapshotFromJob folds one rawJob entry into a Snapshot. Timestamp is
// left zero when it can not be derived from the job alone; the caller
// fills it in with run-level data.
func snapshotFromJob(raw rawSnapshot, j rawJob) *Snapshot {
	s := &Snapshot{
		FioVersion: raw.FioVersion,
		JobName:    j.JobName,
	}

	// IOPS
	s.IOPSRead = j.Read.IOPS
	s.IOPSWrite = j.Write.IOPS

	// Throughput. fio "bw" is KB/s. Convert to MB/s using SI 1 MB = 1000 KB
	// so the tile reads "MB/s" in the same units as elbencho's MiB/s
	// effectively (close enough; the live chart is for human reading).
	// bw_bytes is more precise when present, prefer it.
	if j.Read.BWBytes > 0 {
		s.ThroughputReadMBps = j.Read.BWBytes / (1024.0 * 1024.0)
	} else {
		s.ThroughputReadMBps = j.Read.BWKB / 1024.0
	}
	if j.Write.BWBytes > 0 {
		s.ThroughputWriteMBps = j.Write.BWBytes / (1024.0 * 1024.0)
	} else {
		s.ThroughputWriteMBps = j.Write.BWKB / 1024.0
	}

	// Latency. fio reports clat / lat in nanoseconds. Convert ns -> ms.
	s.LatencyReadMeanMs = j.Read.ClatNS.Mean / 1_000_000.0
	s.LatencyWriteMeanMs = j.Write.ClatNS.Mean / 1_000_000.0

	if v, ok := j.Read.ClatNS.Percentile["50.000000"]; ok {
		s.LatencyReadP50Ms = v / 1_000_000.0
	}
	if v, ok := j.Read.ClatNS.Percentile["95.000000"]; ok {
		s.LatencyReadP95Ms = v / 1_000_000.0
	}
	if v, ok := j.Read.ClatNS.Percentile["99.000000"]; ok {
		s.LatencyReadP99Ms = v / 1_000_000.0
	}
	if v, ok := j.Read.ClatNS.Percentile["99.900000"]; ok {
		s.LatencyReadP999Ms = v / 1_000_000.0
	}
	if v, ok := j.Write.ClatNS.Percentile["99.000000"]; ok {
		s.LatencyWriteP99Ms = v / 1_000_000.0
	}

	return s
}

// ToMetric folds a Snapshot into the engine-agnostic Metric type. Read and
// write throughput / latency are blended onto separate fields so a mixed
// rwmix profile shows both halves on the live charts.
func (s *Snapshot) ToMetric(profileName string) Metric {
	if profileName == "" {
		profileName = s.JobName
	}
	avg := s.LatencyReadMeanMs
	if avg == 0 {
		avg = s.LatencyWriteMeanMs
	}
	return Metric{
		Timestamp:           s.Timestamp,
		ProfileName:         profileName,
		IOPSRead:            s.IOPSRead,
		IOPSWrite:           s.IOPSWrite,
		ThroughputReadMBps:  s.ThroughputReadMBps,
		ThroughputWriteMBps: s.ThroughputWriteMBps,
		LatencyAvgMs:        avg,
		LatencyReadAvgMs:    s.LatencyReadMeanMs,
		LatencyWriteAvgMs:   s.LatencyWriteMeanMs,
		LatencyP50Ms:        s.LatencyReadP50Ms,
		LatencyP95Ms:        s.LatencyReadP95Ms,
		LatencyP99Ms:        s.LatencyReadP99Ms,
		LatencyP999Ms:       s.LatencyReadP999Ms,
		LatencyWriteP99Ms:   s.LatencyWriteP99Ms,
	}
}
