package report

import "testing"

func TestComputeKPIs_Storage(t *testing.T) {
	summary := []ProfileSummary{
		{ProfileName: "p1", MaxIOPSRead: 30000, MaxIOPSWrite: 15000, MaxThroughputRead: 200, P99LatencyMs: 1.2},
		{ProfileName: "p2", MaxIOPSRead: 50000, MaxIOPSWrite: 20000, MaxThroughputRead: 300, P99LatencyMs: 0.8},
	}
	kpis := computeKPIs("storage", summary, nil)
	if len(kpis) != 4 {
		t.Fatalf("expected 4 KPIs, got %d", len(kpis))
	}
	if kpis[0].Label == "" || kpis[0].Value == "" {
		t.Errorf("first KPI must have label and value, got %+v", kpis[0])
	}
}

func TestComputeKPIs_CPU(t *testing.T) {
	nodes := []NodeSummaryRow{
		{NodeName: "aqua", MaxCPU: 80, AvgCPU: 50},
	}
	kpis := computeKPIs("cpu", nil, nodes)
	if len(kpis) != 4 {
		t.Fatalf("expected 4 KPIs, got %d", len(kpis))
	}
}

func TestComputeKPIs_EmptyData(t *testing.T) {
	kpis := computeKPIs("storage", nil, nil)
	if len(kpis) != 4 {
		t.Fatalf("expected 4 KPIs even with empty input, got %d", len(kpis))
	}
}
