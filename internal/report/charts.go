package report

import (
	"fmt"
	"math"
	"strings"
)

type Point struct {
	X, Y float64
}

// LineChart renders a polished SVG line chart suitable for embedding in the
// PDF/HTML report. It draws gridlines, axis labels, a gradient area fill and
// a smooth orange line. The `title` arg is ignored (handled by the template
// chrome) but kept for API stability.
func LineChart(title string, points []Point, width, height int, color string) string {
	_ = title

	if color == "" {
		color = "var(--chart-iops, #f97316)"
	}

	w := float64(width)
	h := float64(height)

	const (
		padTop    = 16.0
		padRight  = 14.0
		padBottom = 28.0
		padLeft   = 56.0
	)

	plotW := w - padLeft - padRight
	plotH := h - padTop - padBottom

	if len(points) == 0 {
		return emptyChart(width, height, "Aucune donnée")
	}

	if len(points) == 1 {
		x := padLeft + plotW/2
		y := padTop + plotH/2
		return fmt.Sprintf(
			`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" preserveAspectRatio="xMidYMid meet">`+
				`<rect x="0" y="0" width="%d" height="%d" rx="8" fill="var(--bg-surface, #fafafa)"/>`+
				`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" fill="var(--chart-axis, #737373)" font-family="-apple-system, sans-serif">Échantillon unique : %.0f</text>`+
				`<circle cx="%.1f" cy="%.1f" r="5" fill="%s"/>`+
				`</svg>`,
			width, height, width, height,
			x, y-14, points[0].Y, x, y, color,
		)
	}

	// Find max
	maxY := 0.0
	for _, p := range points {
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if maxY == 0 {
		maxY = 1
	}
	// Round max up to "nice" number for axis ticks
	niceMax := niceCeil(maxY)
	scaleX := plotW / float64(len(points)-1)
	scaleY := plotH / niceMax

	// Build line path (smooth via simple cubic interpolation)
	type pt struct{ x, y float64 }
	pts := make([]pt, len(points))
	for i, p := range points {
		pts[i] = pt{
			x: padLeft + float64(i)*scaleX,
			y: padTop + plotH - p.Y*scaleY,
		}
	}

	var line strings.Builder
	line.WriteString(fmt.Sprintf("M%.2f %.2f", pts[0].x, pts[0].y))
	for i := 1; i < len(pts); i++ {
		prev := pts[i-1]
		cur := pts[i]
		// Control points for smooth curve
		cx := (prev.x + cur.x) / 2
		line.WriteString(fmt.Sprintf(" C%.2f %.2f %.2f %.2f %.2f %.2f",
			cx, prev.y, cx, cur.y, cur.x, cur.y))
	}
	linePath := line.String()

	// Area fill path = line path + closing baseline
	areaPath := linePath + fmt.Sprintf(" L%.2f %.2f L%.2f %.2f Z",
		pts[len(pts)-1].x, padTop+plotH,
		pts[0].x, padTop+plotH,
	)

	// Gridlines + axis labels (4 horizontal lines including baseline)
	const ticks = 4
	var grid strings.Builder
	for i := 0; i <= ticks; i++ {
		v := niceMax * float64(i) / float64(ticks)
		y := padTop + plotH - v*scaleY
		grid.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--chart-grid, #eef0f3)" stroke-width="1"%s/>`,
			padLeft, y, w-padRight, y,
			func() string {
				if i == 0 {
					return ` stroke="var(--border-default, #d4d4d4)"`
				}
				return ""
			}(),
		))
		grid.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="9" fill="var(--chart-axis, #737373)" font-family="ui-monospace, SFMono-Regular, Menlo, monospace">%s</text>`,
			padLeft-8, y+3, formatTick(v),
		))
	}

	// X axis label (just first / last index)
	xLabels := fmt.Sprintf(
		`<text x="%.1f" y="%.1f" text-anchor="start" font-size="9" fill="var(--chart-axis, #a3a3a3)" font-family="ui-monospace, monospace">échantillon 1</text>`+
			`<text x="%.1f" y="%.1f" text-anchor="end" font-size="9" fill="var(--chart-axis, #a3a3a3)" font-family="ui-monospace, monospace">échantillon %d</text>`,
		padLeft, h-8,
		w-padRight, h-8, len(points),
	)

	gradientID := fmt.Sprintf("g_%x", hashColor(color))

	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" preserveAspectRatio="xMidYMid meet">`+
			`<defs>`+
			`<linearGradient id="%s" x1="0" y1="0" x2="0" y2="1">`+
			`<stop offset="0%%" stop-color="%s" stop-opacity="0.32"/>`+
			`<stop offset="100%%" stop-color="%s" stop-opacity="0.02"/>`+
			`</linearGradient>`+
			`</defs>`+
			`%s`+
			`<path d="%s" fill="url(#%s)" stroke="none"/>`+
			`<path d="%s" fill="none" stroke="%s" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>`+
			`%s`+
			`</svg>`,
		width, height,
		gradientID, color, color,
		grid.String(),
		areaPath, gradientID,
		linePath, color,
		xLabels,
	)
	return svg
}

func emptyChart(width, height int, msg string) string {
	return fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="100%%" preserveAspectRatio="xMidYMid meet">`+
			`<rect width="%d" height="%d" rx="8" fill="var(--bg-surface, #fafafa)"/>`+
			`<text x="%d" y="%d" text-anchor="middle" font-size="11" fill="var(--chart-axis, #a3a3a3)" font-family="-apple-system, sans-serif">%s</text>`+
			`</svg>`,
		width, height, width, height, width/2, height/2, msg,
	)
}

// niceCeil rounds up to a "nice" multiple (1, 2, 2.5, 5, 10, 20, 25, 50…).
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(v)))
	frac := v / exp
	switch {
	case frac <= 1.0:
		return 1.0 * exp
	case frac <= 2.0:
		return 2.0 * exp
	case frac <= 2.5:
		return 2.5 * exp
	case frac <= 5.0:
		return 5.0 * exp
	default:
		return 10 * exp
	}
}

// formatTick prints axis tick labels in a compact form.
func formatTick(v float64) string {
	switch {
	case v >= 1_000_000:
		return fmt.Sprintf("%.1fM", v/1_000_000)
	case v >= 10_000:
		return fmt.Sprintf("%.0fk", v/1_000)
	case v >= 1_000:
		return fmt.Sprintf("%.1fk", v/1_000)
	case v >= 10:
		return fmt.Sprintf("%.0f", v)
	case v >= 1:
		return fmt.Sprintf("%.1f", v)
	case v == 0:
		return "0"
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

// hashColor returns a tiny stable hash so multiple charts get unique gradient ids.
func hashColor(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
