package report

import (
	"errors"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/db"
)

func TestComputeInfraLine_AggregatesNodes(t *testing.T) {
	job := db.Job{}
	workers := []db.Worker{
		{ProxmoxNode: "aqua"},
		{ProxmoxNode: "node-2"},
		{ProxmoxNode: "aqua"}, // dup
	}
	got := computeInfraLine(job, workers)
	if got.WorkerCount != 3 {
		t.Fatalf("expected 3 workers, got %d", got.WorkerCount)
	}
	if len(got.Nodes) != 2 || got.Nodes[0] != "aqua" || got.Nodes[1] != "node-2" {
		t.Fatalf("expected nodes [aqua node-2] sorted, got %v", got.Nodes)
	}
}

type fakeFetcher struct {
	profiles map[string]db.Profile
}

func (f *fakeFetcher) GetProfileByName(name string) (db.Profile, error) {
	if p, ok := f.profiles[name]; ok {
		return p, nil
	}
	return db.Profile{}, errors.New("not found")
}

func TestReadProfileConfigs_ParsesINIAndThresholds(t *testing.T) {
	cfg := `# elbencho profile
threads=8
block=4k
timelimit=300
rand=1
`
	thr := `{"min_iops_read":10000,"min_iops_write":5000,"max_latency_ms":2.5}`
	f := &fakeFetcher{profiles: map[string]db.Profile{
		"p1": {Name: "p1", ConfigJSON: cfg, ThresholdsJSON: thr},
	}}
	out := readProfileConfigs("", f, []string{"p1", "missing"})
	if len(out) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out))
	}
	if out[0].BlockSize != "4k" || out[0].RuntimeSec != 300 || out[0].Pattern != "random" {
		t.Errorf("p1 ini parse failed: %+v", out[0])
	}
	if out[0].ThresholdIOPSR != 10000 || out[0].ThresholdIOPSW != 5000 || out[0].ThresholdLatencyMs != 2.5 {
		t.Errorf("p1 thresholds parse failed: %+v", out[0])
	}
	if out[1].Name != "missing" || out[1].BlockSize != "" {
		t.Errorf("missing profile should be name-only, got %+v", out[1])
	}
}
