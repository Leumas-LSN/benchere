package report_test

import (
	"strings"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/report"
)

func TestLineChart_ContainsSVG(t *testing.T) {
	pts := []report.Point{{0, 100}, {1, 150}, {2, 130}, {3, 200}}
	svg := report.LineChart("Test Chart", pts, 800, 300, "#ff0000")
	if !strings.Contains(svg, "<svg") {
		t.Error("output is not SVG")
	}
	if !strings.Contains(svg, "#ff0000") {
		t.Error("color missing from SVG")
	}
	// Modern chart renders a smooth path + gradient area fill
	if !strings.Contains(svg, "<path") {
		t.Error("expected path element in SVG")
	}
	if !strings.Contains(svg, "linearGradient") {
		t.Error("expected gradient definition in SVG")
	}
}

func TestLineChart_EmptyPoints(t *testing.T) {
	// Empty input now renders a placeholder card (so the report does not
	// show a missing image when there are no data points to plot).
	svg := report.LineChart("Empty", nil, 800, 300, "#000")
	if !strings.Contains(svg, "<svg") {
		t.Error("expected SVG placeholder when points are empty")
	}
	if !strings.Contains(svg, "Aucune") {
		t.Error("expected 'Aucune donnée' placeholder text")
	}
}

func TestLineChart_SinglePoint(t *testing.T) {
	pts := []report.Point{{0, 42}}
	svg := report.LineChart("Single", pts, 400, 200, "#f97316")
	if !strings.Contains(svg, "<svg") {
		t.Error("output is not SVG")
	}
	if !strings.Contains(svg, "#f97316") {
		t.Error("color missing from SVG")
	}
}
