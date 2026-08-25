package render

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/Gabriel-Gerhardt/ctxray/internal/analyze"
)

func buildViewModel(report analyze.Report) ViewModel {
	return ViewModel{
		SessionID:   report.SessionID,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Models:      joinModels(report.Models),
		Stats:       buildStatsVM(report.Stats),
		TopDead:     buildTopDead(report.Turns),
		Turns:       buildTurnVMs(report.Turns),
		TimelineSVG: template.HTML(buildTimelineSVG(report.Timeline)), //nolint:gosec // server-generated SVG, no user input reaches raw markup
		Legend:      buildLegend(report.Turns),
		ToolCalls:   buildToolCallVMs(report.Stats.ToolCallCounts),
	}
}

func joinModels(models []string) string {
	if len(models) == 0 {
		return "unknown model"
	}
	return strings.Join(models, ", ")
}

func buildStatsVM(s analyze.Stats) StatsVM {
	return StatsVM{
		TurnCount:           s.TurnCount,
		Duration:            formatDuration(s.Duration),
		StartClock:          formatClock(s.StartTime),
		EndClock:            formatClock(s.EndTime),
		PeakContextTokens:   formatTokens(s.PeakContextTokens),
		TotalOutputTokens:   formatTokens(s.TotalOutputTokens),
		TotalThinkingTokens: formatTokens(s.TotalThinkingTokens),
		DeadTokens:          formatTokens(s.DeadTokens),
		DeadTokensExact:     formatExact(s.DeadTokens),
		DeadTokenPct:        formatPct(s.DeadTokenPct),
		DeadTokenBlocks:     s.DeadTokenBlocks,
	}
}

// topDeadCount caps the plain-text ranking — enough to answer "what's the
// worst of it" at a glance, not a second copy of the whole flamegraph.
const topDeadCount = 5

// buildTopDead lists the single biggest dead blocks across the whole
// session, sorted by token count. It exists because a sorted list needs
// no legend and no learning curve — it's the fastest path to "what
// should I go look at first", ahead of the chart that shows everything.
func buildTopDead(turns []analyze.Turn) []TopDeadVM {
	var dead []analyze.Block
	for _, t := range turns {
		for _, b := range t.NewBlocks {
			if b.Dead {
				dead = append(dead, b)
			}
		}
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].Tokens > dead[j].Tokens })
	if len(dead) > topDeadCount {
		dead = dead[:topDeadCount]
	}

	out := make([]TopDeadVM, len(dead))
	for i, b := range dead {
		out[i] = TopDeadVM{
			Rank:   i + 1,
			Label:  b.Label,
			Tokens: formatTokens(b.Tokens),
			Slot:   string(slotFor(b.Source)),
		}
	}
	return out
}

// minRowWidthPct keeps a nearly-empty turn visible as a sliver instead of
// vanishing to a hairline — it should still read as "tiny", just not
// literally invisible.
const minRowWidthPct = 2.0

// buildTurnVMs renders one flamegraph row per turn that actually added
// tokens to the context window. A turn with a zero delta — everything it
// needed was already cached — has nothing to show in an "inflow" chart:
// on a long session those can be most of the turns, and drawing an empty
// row for each just buries the ones that matter under scrolling.
func buildTurnVMs(turns []analyze.Turn) []TurnVM {
	maxDelta := 1
	for _, t := range turns {
		if t.ContextDelta > maxDelta {
			maxDelta = t.ContextDelta
		}
	}

	out := make([]TurnVM, 0, len(turns))
	for _, t := range turns {
		if t.ContextDelta <= 0 {
			continue
		}
		rowWidth := 100 * float64(t.ContextDelta) / float64(maxDelta)
		if rowWidth < minRowWidthPct {
			rowWidth = minRowWidthPct
		}
		out = append(out, TurnVM{
			RowTitle:    fmt.Sprintf("turn #%d · %s", t.Index, formatClock(t.Timestamp)),
			DeltaLabel:  "+" + formatTokens(t.ContextDelta),
			RowWidthPct: rowWidth,
			Blocks:      buildBlockVMs(t.NewBlocks, t.ContextDelta),
		})
	}
	return out
}

func buildBlockVMs(blocks []analyze.Block, total int) []BlockVM {
	if total <= 0 {
		return nil
	}
	out := make([]BlockVM, len(blocks))
	for i, b := range blocks {
		out[i] = BlockVM{
			WidthPct: 100 * float64(b.Tokens) / float64(total),
			Slot:     string(slotFor(b.Source)),
			Dead:     b.Dead,
			Title:    blockTitle(b),
		}
	}
	return out
}

func blockTitle(b analyze.Block) string {
	if b.Dead {
		return fmt.Sprintf("%s — %s tokens — never referenced again", b.Label, formatTokens(b.Tokens))
	}
	return fmt.Sprintf("%s — %s tokens", b.Label, formatTokens(b.Tokens))
}

// legendOrder is the fixed slot order the legend walks, matching the
// dataviz color-formula: identities in a fixed sequence, never reordered
// by frequency or rank. Only slots that actually appear in the session
// make it into the rendered legend.
var legendOrder = []struct {
	slot  slot
	label string
}{
	{slotUser, "user"},
	{slotOverhead, "system / tool schemas"},
	{slotBash, "Bash"},
	{slotRead, "Read"},
	{slotGrep, "Grep"},
}

// buildLegend walks the fixed slot order and keeps only what showed up.
// The shared "other" slot names every tool folded into it, so grouping
// rare tools under one color never hides which tools they were.
func buildLegend(turns []analyze.Turn) []LegendItem {
	present := map[slot]bool{}
	otherNames := map[string]bool{}
	for _, t := range turns {
		for _, b := range t.NewBlocks {
			present[slotFor(b.Source)] = true
			if name, ok := otherToolName(b.Source); ok {
				otherNames[name] = true
			}
		}
	}

	out := make([]LegendItem, 0, len(legendOrder)+1)
	for _, o := range legendOrder {
		if present[o.slot] {
			out = append(out, LegendItem{Slot: string(o.slot), Label: o.label})
		}
	}
	if len(otherNames) > 0 {
		names := make([]string, 0, len(otherNames))
		for n := range otherNames {
			names = append(names, n)
		}
		sort.Strings(names)
		out = append(out, LegendItem{Slot: string(slotOther), Label: "other (" + strings.Join(names, ", ") + ")"})
	}
	return out
}

func buildToolCallVMs(counts map[string]int) []ToolCallVM {
	out := make([]ToolCallVM, 0, len(counts))
	for name, n := range counts {
		out = append(out, ToolCallVM{Name: name, Count: n, Slot: string(slotFor("tool:" + name))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}
