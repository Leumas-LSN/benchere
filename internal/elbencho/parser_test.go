package elbencho_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Leumas-LSN/benchere/internal/elbencho"
)

// sampleCSV mirrors the real elbencho 3.x --livecsv schema:
// ISO Date,Label,Phase,RuntimeMS,Rank,MixType,Done%,DoneBytes,MiB/s,IOPS,
// Entries,Entries/s,Lat Ent us,Lat IO us,Active,CPU,Service
//
// Only "Total" rows carry aggregated metrics; the parser must skip per-rank
// rows. There is no p99 column in --livecsv.
const sampleCSV = `ISO Date,Label,Phase,RuntimeMS,Rank,MixType,Done%,DoneBytes,MiB/s,IOPS,Entries,Entries/s,Lat Ent us,Lat IO us,Active,CPU,Service
2026-04-25T10:00:01,4k_100read_100random,READ,1000,Total,Read,1,1024,165.3,42355,0,0,0,94,1,12,svc1
2026-04-25T10:00:02,4k_100read_100random,READ,2000,Total,Read,2,2048,171.2,43827,0,0,0,91,1,12,svc1
`

func TestTailCSV(t *testing.T) {
	f, _ := os.CreateTemp("", "elbencho_test_*.csv")
	f.WriteString(sampleCSV)
	f.Close()
	defer os.Remove(f.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out := make(chan elbencho.Metric, 10)
	go elbencho.TailCSV(ctx, f.Name(), out)

	var metrics []elbencho.Metric
	for m := range out {
		metrics = append(metrics, m)
	}

	if len(metrics) < 2 {
		t.Fatalf("want >= 2 metrics, got %d", len(metrics))
	}
	if metrics[0].IOPSRead != 42355 {
		t.Errorf("want IOPSRead 42355, got %f", metrics[0].IOPSRead)
	}
	if metrics[0].ThroughputReadMBps != 165.3 {
		t.Errorf("want throughput 165.3, got %f", metrics[0].ThroughputReadMBps)
	}
	// 94 us = 0.094 ms
	if metrics[0].LatencyAvgMs != 0.094 {
		t.Errorf("want latency 0.094ms, got %f", metrics[0].LatencyAvgMs)
	}
}
