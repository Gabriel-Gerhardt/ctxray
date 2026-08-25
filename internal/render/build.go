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
		TotalContextEntered: formatTokens(s.TotalContextEntered),
		FinalContextTokens:  formatTokens(s.FinalContextTokens),
		PeakContextTokens:   formatTokens(s.PeakContextTokens),
		TotalOutputTokens:   formatTokens(s.TotalOutputTokens),
		TotalThinkingTokens: formatTokens(s.TotalThinkingTokens),
		DeadTokens:          formatTokens(s.DeadTokens),
		DeadTokensExact:     formatExact(s.DeadTokens),
		DeadTokenPct:        formatPct(s.DeadTokenPct),
		DeadTokenBlocks:     s.DeadTokenBlocks,
	}
}

func buildTurnVMs(turns []analyze.Turn) []TurnVM {
	out := make([]TurnVM, len(turns))
	for i, t := range turns {
		out[i] = TurnVM{
			Index:      t.Index,
			Clock:      formatClock(t.Timestamp),
			DeltaLabel: "+" + formatTokens(t.ContextDelta),
			Blocks:     buildBlockVMs(t.NewBlocks, t.ContextDelta),
		}
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
