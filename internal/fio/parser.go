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
	LatencyAvgMs        float64
	LatencyP50Ms        float64
	LatencyP95Ms        float64
	LatencyP99Ms        float64
}

// rawSnapshot is the subset of fio JSON+ output we care about. Many fields
// are dropped; only those that drive a benchmark report are decoded.
type rawSnapshot struct {
	FioVersion  string   `json:"fio version"`
	TimestampMs int64    `json:"timestamp_ms"`
	Jobs        []rawJob `json:"jobs"`
}

type rawJob struct {
	JobName string  `json:"jobname"`
	Read    rawIOSt `json:"read"`
	Write   rawIOSt `json:"write"`
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
	LatencyWriteP99Ms   float64
}

// ParseStatusSnapshot decodes one fio JSON+ status object. The first job
// in the document is used (fio reports per-job, per-group; in our
// distributed setup with --client we get one aggregated job entry).
func ParseStatusSnapshot(data []byte) (*Snapshot, error) {
	var raw rawSnapshot
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}
	if len(raw.Jobs) == 0 {
		return nil, fmt.Errorf("snapshot has no jobs")
	}
	j := raw.Jobs[0]

	s := &Snapshot{
		FioVersion: raw.FioVersion,
		JobName:    j.JobName,
	}
	if raw.TimestampMs > 0 {
		s.Timestamp = time.Unix(raw.TimestampMs/1000, (raw.TimestampMs%1000)*1_000_000)
	} else {
		s.Timestamp = nowFn()
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
	if v, ok := j.Write.ClatNS.Percentile["99.000000"]; ok {
		s.LatencyWriteP99Ms = v / 1_000_000.0
	}

	return s, nil
}

// ToMetric folds a Snapshot into the engine-agnostic Metric type. Read and
// write throughput / latency are blended onto separate fields so a mixed
// rwmix profile shows both halves on the live charts.
func (s *Snapshot) ToMetric(profileName string) Metric {
	if profileName == "" {
		profileName = s.JobName
	}
	// LatencyAvgMs follows the elbencho convention: prefer the read mean
	// when present, fall back to write mean. For mixed workloads this is
	// the predominant signal the user reads on the dashboard tile.
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
		LatencyP50Ms:        s.LatencyReadP50Ms,
		LatencyP95Ms:        s.LatencyReadP95Ms,
		LatencyP99Ms:        s.LatencyReadP99Ms,
	}
}
