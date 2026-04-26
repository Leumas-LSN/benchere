package elbencho_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Leumas-LSN/benchere/internal/elbencho"
)

const sampleCSV = `ISO date,UNIX time,label,num workers,read (MiB/s),write (MiB/s),read (IOPS),write (IOPS),read lat (us) avg,write lat (us) avg,read lat (us) 99.0%,write lat (us) 99.0%
2026-04-25T10:00:01,1745575201,4k_100read_100random,4,165.3,0,42355,0,94,0,980,0
2026-04-25T10:00:02,1745575202,4k_100read_100random,4,171.2,0,43827,0,91,0,950,0
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
