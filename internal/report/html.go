package report

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"math"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
)

//go:embed templates/report.html
var reportTmplSrc string

type ProfileSummary struct {
	ProfileName       string
	MaxIOPSRead       float64
	MaxIOPSWrite      float64
	MaxThroughputRead float64
	AvgLatencyMs      float64
	P99LatencyMs      float64
	Verdict           string // "pass", "fail", or "" if no thresholds set
}

type NodeSummaryRow struct {
	NodeName string
	MaxCPU   float64
	AvgCPU   float64
	AvgRAM   float64
}

type IOPSChart struct {
	Profile string
	SVG     template.HTML
}

type reportData struct {
	Job          db.Job
	Summary      []ProfileSummary
	IOPSCharts   []IOPSChart
	NodeChartSVG template.HTML
	NodeSummary  []NodeSummaryRow
}

type Generator struct {
	db *db.DB
}

func NewGenerator(database *db.DB) *Generator { return &Generator{db: database} }

func (g *Generator) RenderHTML(job db.Job, results []db.Result, snaps []db.ProxmoxSnapshot) ([]byte, error) {
	tmpl, err := template.New("report.html").Parse(reportTmplSrc)
	if err != nil {
		return nil, err
	}

	data := g.buildReportData(job, results, snaps)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) buildReportData(job db.Job, results []db.Result, snaps []db.ProxmoxSnapshot) reportData {
	data := reportData{Job: job}

	const (
		brandOrange = "#f97316"
		clusterBlue = "#0ea5e9"
	)

	// Summary per profile
	byProfile := make(map[string][]db.Result)
	for _, r := range results {
		byProfile[r.ProfileName] = append(byProfile[r.ProfileName], r)
	}
	for profile, rs := range byProfile {
		s := ProfileSummary{ProfileName: profile}
		var sumLat, sumP99 float64
		for _, r := range rs {
			if r.IOPSRead > s.MaxIOPSRead {
				s.MaxIOPSRead = r.IOPSRead
			}
			if r.IOPSWrite > s.MaxIOPSWrite {
				s.MaxIOPSWrite = r.IOPSWrite
			}
			if r.ThroughputReadMBps > s.MaxThroughputRead {
				s.MaxThroughputRead = r.ThroughputReadMBps
			}
			sumLat += r.LatencyAvgMs
			sumP99 += r.LatencyP99Ms
		}
		if len(rs) > 0 {
			s.AvgLatencyMs = sumLat / float64(len(rs))
			s.P99LatencyMs = sumP99 / float64(len(rs))
		}

		// Verdict computation
		if g.db != nil {
			prof, err := g.db.GetProfileByName(profile)
			if err == nil && prof.ThresholdsJSON != "" {
				var t db.ProfileThresholds
				if json.Unmarshal([]byte(prof.ThresholdsJSON), &t) == nil {
					pass := true
					if t.MinIOPSRead > 0 && s.MaxIOPSRead < t.MinIOPSRead {
						pass = false
					}
					if t.MinIOPSWrite > 0 && s.MaxIOPSWrite < t.MinIOPSWrite {
						pass = false
					}
					if t.MaxLatencyMs > 0 && s.AvgLatencyMs > t.MaxLatencyMs {
						pass = false
					}
					if pass {
						s.Verdict = "pass"
					} else {
						s.Verdict = "fail"
					}
				}
			}
		}

		data.Summary = append(data.Summary, s)

		// IOPS chart
		var pts []Point
		for i, r := range rs {
			pts = append(pts, Point{X: float64(i), Y: r.IOPSRead})
		}
		data.IOPSCharts = append(data.IOPSCharts, IOPSChart{
			Profile: profile,
			SVG:     template.HTML(LineChart("", pts, 760, 240, brandOrange)),
		})
	}

	// Node chart and summary
	byNode := make(map[string][]db.ProxmoxSnapshot)
	for _, s := range snaps {
		byNode[s.NodeName] = append(byNode[s.NodeName], s)
	}
	var nodePts []Point
	t0 := time.Time{}
	for _, s := range snaps {
		if t0.IsZero() {
			t0 = s.Timestamp
		}
		nodePts = append(nodePts, Point{X: s.Timestamp.Sub(t0).Seconds(), Y: s.CPUPct})
	}
	data.NodeChartSVG = template.HTML(LineChart("", nodePts, 760, 240, clusterBlue))

	for node, ns := range byNode {
		row := NodeSummaryRow{NodeName: node}
		var sumCPU, sumRAM float64
		for _, n := range ns {
			if n.CPUPct > row.MaxCPU {
				row.MaxCPU = n.CPUPct
			}
			sumCPU += n.CPUPct
			sumRAM += n.RAMPct
		}
		row.AvgCPU = math.Round(sumCPU/float64(len(ns))*10) / 10
		row.AvgRAM = math.Round(sumRAM/float64(len(ns))*10) / 10
		data.NodeSummary = append(data.NodeSummary, row)
	}

	return data
}
