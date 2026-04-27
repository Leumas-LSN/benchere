package report

import "fmt"

// KPI represents one tile in the report header.
type KPI struct {
	Label string
	Value string
	Unit  string
	Sub   string
}

// computeKPIs returns 4 KPI tiles tailored to the job mode.
// "storage" -> IOPS R / IOPS W / Throughput / Latency p99
// "cpu"     -> CPU max / CPU avg / NodeChargeMax / StressDuration
// "mixed"   -> hybrid 4-tile mix
func computeKPIs(mode string, summary []ProfileSummary, nodes []NodeSummaryRow) []KPI {
	switch mode {
	case "cpu":
		return cpuKPIs(nodes)
	case "mixed":
		return mixedKPIs(summary, nodes)
	default:
		return storageKPIs(summary)
	}
}

func storageKPIs(summary []ProfileSummary) []KPI {
	maxR, maxW, maxThru, maxP99 := 0.0, 0.0, 0.0, 0.0
	whoR, whoW, whoT, whoP := "", "", "", ""
	for _, s := range summary {
		if s.MaxIOPSRead > maxR {
			maxR = s.MaxIOPSRead
			whoR = s.ProfileName
		}
		if s.MaxIOPSWrite > maxW {
			maxW = s.MaxIOPSWrite
			whoW = s.ProfileName
		}
		if s.MaxThroughputRead > maxThru {
			maxThru = s.MaxThroughputRead
			whoT = s.ProfileName
		}
		if s.P99LatencyMs > maxP99 {
			maxP99 = s.P99LatencyMs
			whoP = s.ProfileName
		}
	}
	return []KPI{
		{Label: "IOPS READ MAX", Value: formatInt(maxR), Unit: "IOPS", Sub: profileSub(whoR)},
		{Label: "IOPS WRITE MAX", Value: formatInt(maxW), Unit: "IOPS", Sub: profileSub(whoW)},
		{Label: "DEBIT MAX", Value: formatFloat(maxThru, 0), Unit: "MB/s", Sub: profileSub(whoT)},
		{Label: "LATENCE P99 MAX", Value: formatFloat(maxP99, 2), Unit: "ms", Sub: profileSub(whoP)},
	}
}

func cpuKPIs(nodes []NodeSummaryRow) []KPI {
	maxCPU, avgCPU := 0.0, 0.0
	whoMax := ""
	if len(nodes) > 0 {
		sum := 0.0
		for _, n := range nodes {
			if n.MaxCPU > maxCPU {
				maxCPU = n.MaxCPU
				whoMax = n.NodeName
			}
			sum += n.AvgCPU
		}
		avgCPU = sum / float64(len(nodes))
	}
	return []KPI{
		{Label: "CPU MAX", Value: formatFloat(maxCPU, 1), Unit: "%", Sub: profileSub(whoMax)},
		{Label: "CPU AVG", Value: formatFloat(avgCPU, 1), Unit: "%", Sub: ""},
		{Label: "NODES SOUS CHARGE", Value: fmt.Sprintf("%d", len(nodes)), Unit: "", Sub: ""},
		{Label: "DUREE STRESS", Value: "-", Unit: "s", Sub: ""},
	}
}

func mixedKPIs(summary []ProfileSummary, nodes []NodeSummaryRow) []KPI {
	stor := storageKPIs(summary)
	cpu := cpuKPIs(nodes)
	if len(stor) >= 2 && len(cpu) >= 2 {
		return []KPI{stor[0], stor[2], cpu[0], cpu[2]}
	}
	return stor
}

func formatInt(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

func formatFloat(v float64, decimals int) string {
	return fmt.Sprintf("%.*f", decimals, v)
}

func profileSub(name string) string {
	if name == "" {
		return ""
	}
	return "profil " + name
}
