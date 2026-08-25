package analyze

import "strings"

// firstLine trims a block of text down to a single-line, human-scannable
// label for the flamegraph's hover tooltip.
func firstLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if len(s) > max {
		s = strings.TrimSpace(s[:max]) + "…"
	}
	if s == "" {
		return "(empty)"
	}
	return s
}
