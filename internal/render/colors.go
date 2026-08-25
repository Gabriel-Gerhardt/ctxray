package render

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// basePalette gives the common categories and tool names a hand-picked,
// consistent color across every report.
var basePalette = map[string]string{
	"user":               "#3b82f6",
	"context:overhead":   "#94a3b8",
	"assistant:text":     "#8b5cf6",
	"assistant:thinking": "#a5b4fc",
	"tool:Bash":          "#f97316",
	"tool:Read":          "#22c55e",
	"tool:Edit":          "#14b8a6",
	"tool:Write":         "#ec4899",
	"tool:Grep":          "#eab308",
	"tool:Glob":          "#f59e0b",
	"tool:WebFetch":      "#06b6d4",
	"tool:WebSearch":     "#0ea5e9",
	"tool:Task":          "#d946ef",
	"tool:NotebookEdit":  "#84cc16",
	"tool:TodoWrite":     "#f43f5e",
}

// colorFor returns a stable color for a block source. Anything outside
// the hand-picked palette — an MCP tool with an unpredictable name —
// gets a color hashed from its own name instead of a generic gray, so it
// stays visually distinct and identical across re-renders of the same
// session without needing a map entry for every tool that exists.
func colorFor(source string) string {
	if c, ok := basePalette[source]; ok {
		return c
	}
	if name, ok := strings.CutPrefix(source, "assistant:tool_use:"); ok {
		if c, ok := basePalette["tool:"+name]; ok {
			return c
		}
		return hashColor(name)
	}
	if name, ok := strings.CutPrefix(source, "tool:"); ok {
		return hashColor(name)
	}
	return hashColor(source)
}

func hashColor(seed string) string {
	h := fnv.New32a()
	h.Write([]byte(seed))
	hue := int(h.Sum32() % 360)
	return fmt.Sprintf("hsl(%d, 65%%, 50%%)", hue)
}
