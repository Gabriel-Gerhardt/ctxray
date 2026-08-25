package render

import (
	"fmt"
	"strings"

	"github.com/Gabriel-Gerhardt/ctxray/internal/analyze"
)

const (
	timelineW = 1000.0
	timelineH = 160.0
)

// buildTimelineSVG renders context-window size over the session as a
// filled line chart, server-side. A dozen to a few hundred points don't
// need a charting library — just coordinate math and two SVG shapes.
func buildTimelineSVG(points []analyze.TimelinePoint) string {
	if len(points) == 0 {
		return ""
	}
	peak := 1
	for _, p := range points {
		if p.ContextTokens > peak {
			peak = p.ContextTokens
		}
	}

	coords := make([]string, len(points))
	for i, p := range points {
		x := timelineW * float64(i) / float64(maxInt(len(points)-1, 1))
		y := timelineH - timelineH*float64(p.ContextTokens)/float64(peak)
		coords[i] = fmt.Sprintf("%.2f,%.2f", x, y)
	}
	line := strings.Join(coords, " ")
	area := fmt.Sprintf("0,%.2f %s %.2f,%.2f", timelineH, line, timelineW, timelineH)

	var b strings.Builder
	fmt.Fprintf(&b, `<polygon points="%s" class="ctxray-timeline-area"/>`, area)
	fmt.Fprintf(&b, `<polyline points="%s" class="ctxray-timeline-line"/>`, line)
	return b.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
