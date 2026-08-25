package render

import (
	"fmt"
	"strconv"
	"time"
)

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// formatExact renders the full integer with thousands separators
// (27,834, not 27.8k) — for the one number on the page meant to be read
// digit by digit, not skimmed.
func formatExact(n int) string {
	s := strconv.Itoa(n)
	neg := ""
	if len(s) > 0 && s[0] == '-' {
		neg, s = "-", s[1:]
	}
	if len(s) <= 3 {
		return neg + s
	}
	var out []byte
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	out = append(out, s[:lead]...)
	for i := lead; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return neg + string(out)
}

func formatPct(f float64) string {
	return fmt.Sprintf("%.1f%%", f*100)
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

func formatClock(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("15:04:05")
}
