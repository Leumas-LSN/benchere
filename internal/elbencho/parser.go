package elbencho

import (
	"bufio"
	"context"
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Metric struct {
	Timestamp           time.Time
	ProfileName         string
	IOPSRead            float64
	IOPSWrite           float64
	ThroughputReadMBps  float64
	ThroughputWriteMBps float64
	LatencyAvgMs        float64
}

// TailCSV reads existing lines then blocks waiting for new ones. Closes out when ctx is done.
// It waits up to 30s for the file to appear (elbencho creates it at first write).
func TailCSV(ctx context.Context, path string, out chan<- Metric) {
	defer close(out)
	var f *os.File
	for {
		var err error
		f, err = os.Open(path)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(500 * time.Millisecond):
		}
	}
	defer f.Close()

	r := csv.NewReader(bufio.NewReader(f))
	r.LazyQuotes = true

	header, err := r.Read()
	if err != nil {
		return
	}
	idx := buildIndex(header)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		record, err := r.Read()
		if err == io.EOF {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return
		}
		m, ok := parseRecord(record, idx)
		if !ok {
			continue
		}
		select {
		case out <- m:
		case <-ctx.Done():
			return
		}
	}
}

func buildIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(h)] = i
	}
	return m
}

// parseRecord parses one live CSV row.
//
// Live CSV columns: ISO Date, Label, Phase, RuntimeMS, Rank, MixType, Done%, DoneBytes, MiB/s, IOPS, ...
// Only "Total" Rank rows carry aggregated metrics.
// Routing logic:
//   - MixType="Read"  → read fields
//   - MixType="Write" → write fields
//   - MixType empty, Phase="READ" or Phase starts with "RWMIX" with read% > 0 → read
//   - MixType empty, Phase="WRITE" or Phase="RWMIX0" → write
func parseRecord(rec []string, idx map[string]int) (Metric, bool) {
	getString := func(key string) string {
		i, ok := idx[key]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}
	getFloat := func(key string) float64 {
		i, ok := idx[key]
		if !ok || i >= len(rec) {
			return 0
		}
		v, _ := strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
		return v
	}

	if getString("Rank") != "Total" {
		return Metric{}, false
	}

	phase   := getString("Phase")
	mixType := getString("MixType")
	iops       := getFloat("IOPS")
	throughput := getFloat("MiB/s")
	latency    := getFloat("Lat IO us") / 1000

	m := Metric{
		Timestamp:    time.Now(),
		ProfileName:  getString("Label"),
		LatencyAvgMs: latency,
	}

	isRead := false
	switch {
	case mixType == "Read":
		isRead = true
	case mixType == "Write":
		isRead = false
	case phase == "READ":
		isRead = true
	case phase == "WRITE":
		isRead = false
	case phase == "RWMIX0":
		// 0% read = 100% write
		isRead = false
	default:
		// RWMIX100 or unknown single-row phase → treat as read
		isRead = true
	}

	if isRead {
		m.IOPSRead = iops
		m.ThroughputReadMBps = throughput
	} else {
		m.IOPSWrite = iops
		m.ThroughputWriteMBps = throughput
	}

	return m, true
}

func safeGet(rec []string, i int) string {
	if i >= 0 && i < len(rec) {
		return strings.TrimSpace(rec[i])
	}
	return ""
}
