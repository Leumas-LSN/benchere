package benchmark

import (
	"math"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/ws"
	"github.com/google/uuid"
)

// PhaseAggregator collects storageMetric samples for one profile run and
// produces a PhaseSummary at the end. It runs entirely in the persist
// goroutine so it does not need synchronization.
type PhaseAggregator struct {
	profileName string

	count int

	iopsRead       []float64
	iopsWrite      []float64
	throughputRead  []float64
	throughputWrite []float64

	// Latencies are tracked as the LAST observed value (steady-state),
	// not averaged across samples. fio reports cumulative percentiles per
	// snapshot which already incorporate history; averaging would smear.
	lastLatP50Ms      float64
	lastLatP95Ms      float64
	lastLatP99Ms      float64
	lastLatP999Ms     float64
	lastLatWriteP99Ms float64
}

func NewPhaseAggregator(profileName string) *PhaseAggregator {
	return &PhaseAggregator{profileName: profileName}
}

func (a *PhaseAggregator) Push(m storageMetric) {
	a.count++
	a.iopsRead = append(a.iopsRead, m.IOPSRead)
	a.iopsWrite = append(a.iopsWrite, m.IOPSWrite)
	a.throughputRead = append(a.throughputRead, m.ThroughputReadMBps)
	a.throughputWrite = append(a.throughputWrite, m.ThroughputWriteMBps)

	if m.LatencyP50Ms > 0 {
		a.lastLatP50Ms = m.LatencyP50Ms
	}
	if m.LatencyP95Ms > 0 {
		a.lastLatP95Ms = m.LatencyP95Ms
	}
	if m.LatencyP99Ms > 0 {
		a.lastLatP99Ms = m.LatencyP99Ms
	}
	if m.LatencyP999Ms > 0 {
		a.lastLatP999Ms = m.LatencyP999Ms
	}
	if m.LatencyWriteP99Ms > 0 {
		a.lastLatWriteP99Ms = m.LatencyWriteP99Ms
	}
}

// PhaseSnapshot is the in-memory view that gets persisted as
// db.PhaseSummary and broadcast as ws.PhaseSummaryPayload.
type PhaseSnapshot struct {
	ProfileName            string
	SamplesCount           int
	IOPSReadAvg            float64
	IOPSReadMin            float64
	IOPSReadMax            float64
	IOPSWriteAvg           float64
	IOPSWriteMin           float64
	IOPSWriteMax           float64
	ThroughputReadMBpsAvg  float64
	ThroughputReadMBpsMax  float64
	ThroughputWriteMBpsAvg float64
	ThroughputWriteMBpsMax float64
	LatP50Ms               float64
	LatP95Ms               float64
	LatP99Ms               float64
	LatP999Ms              float64
	LatWriteP99Ms          float64
	IOPSCVPct              float64
	FinishedAt             time.Time
}

func (a *PhaseAggregator) Snapshot(at time.Time) PhaseSnapshot {
	rAvg, rMin, rMax := stats(a.iopsRead)
	wAvg, wMin, wMax := stats(a.iopsWrite)
	rTAvg, _, rTMax := stats(a.throughputRead)
	wTAvg, _, wTMax := stats(a.throughputWrite)

	cv := 0.0
	if rAvg > 0 {
		cv = stddev(a.iopsRead, rAvg) / rAvg * 100
	}

	return PhaseSnapshot{
		ProfileName:            a.profileName,
		SamplesCount:           a.count,
		IOPSReadAvg:            rAvg,
		IOPSReadMin:            rMin,
		IOPSReadMax:            rMax,
		IOPSWriteAvg:           wAvg,
		IOPSWriteMin:           wMin,
		IOPSWriteMax:           wMax,
		ThroughputReadMBpsAvg:  rTAvg,
		ThroughputReadMBpsMax:  rTMax,
		ThroughputWriteMBpsAvg: wTAvg,
		ThroughputWriteMBpsMax: wTMax,
		LatP50Ms:               a.lastLatP50Ms,
		LatP95Ms:               a.lastLatP95Ms,
		LatP99Ms:               a.lastLatP99Ms,
		LatP999Ms:              a.lastLatP999Ms,
		LatWriteP99Ms:          a.lastLatWriteP99Ms,
		IOPSCVPct:              cv,
		FinishedAt:             at,
	}
}

func (s PhaseSnapshot) ToDBRecord(id, jobID string) db.PhaseSummary {
	return db.PhaseSummary{
		ID: id, JobID: jobID, ProfileName: s.ProfileName, SamplesCount: s.SamplesCount,
		IOPSReadAvg: s.IOPSReadAvg, IOPSReadMin: s.IOPSReadMin, IOPSReadMax: s.IOPSReadMax,
		IOPSWriteAvg: s.IOPSWriteAvg, IOPSWriteMin: s.IOPSWriteMin, IOPSWriteMax: s.IOPSWriteMax,
		ThroughputReadMBpsAvg:  s.ThroughputReadMBpsAvg, ThroughputReadMBpsMax: s.ThroughputReadMBpsMax,
		ThroughputWriteMBpsAvg: s.ThroughputWriteMBpsAvg, ThroughputWriteMBpsMax: s.ThroughputWriteMBpsMax,
		LatP50Ms: s.LatP50Ms, LatP95Ms: s.LatP95Ms, LatP99Ms: s.LatP99Ms, LatP999Ms: s.LatP999Ms,
		LatWriteP99Ms: s.LatWriteP99Ms, IOPSCVPct: s.IOPSCVPct, FinishedAt: s.FinishedAt,
	}
}

func (s PhaseSnapshot) ToWSPayload() ws.PhaseSummaryPayload {
	return ws.PhaseSummaryPayload{
		ProfileName: s.ProfileName, SamplesCount: s.SamplesCount,
		IOPSReadAvg: s.IOPSReadAvg, IOPSReadMin: s.IOPSReadMin, IOPSReadMax: s.IOPSReadMax,
		IOPSWriteAvg: s.IOPSWriteAvg, IOPSWriteMin: s.IOPSWriteMin, IOPSWriteMax: s.IOPSWriteMax,
		ThroughputReadMBpsAvg:  s.ThroughputReadMBpsAvg, ThroughputReadMBpsMax: s.ThroughputReadMBpsMax,
		ThroughputWriteMBpsAvg: s.ThroughputWriteMBpsAvg, ThroughputWriteMBpsMax: s.ThroughputWriteMBpsMax,
		LatP50Ms: s.LatP50Ms, LatP95Ms: s.LatP95Ms, LatP99Ms: s.LatP99Ms, LatP999Ms: s.LatP999Ms,
		LatWriteP99Ms: s.LatWriteP99Ms, IOPSCVPct: s.IOPSCVPct,
		FinishedAt: s.FinishedAt.UTC().Format(time.RFC3339),
	}
}

// stats returns avg, min, max. Returns 0, 0, 0 on empty slice.
func stats(xs []float64) (avg, min, max float64) {
	if len(xs) == 0 {
		return 0, 0, 0
	}
	min = xs[0]
	max = xs[0]
	sum := 0.0
	for _, x := range xs {
		sum += x
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	return sum / float64(len(xs)), min, max
}

func stddev(xs []float64, mean float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	var sumSq float64
	for _, x := range xs {
		d := x - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(xs)-1))
}

// Reference uuid so the import is used by callers. Not strictly needed.
var _ = uuid.NewString
