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
			Color:    colorFor(b.Source),
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

// buildLegend lists every source that actually showed up, "user" and
// "system / tool schemas" first since they're on nearly every report,
// then every tool alphabetically.
func buildLegend(turns []analyze.Turn) []LegendItem {
	seen := map[string]bool{}
	head := []LegendItem{
		{Color: colorFor("user"), Label: "user"},
		{Color: colorFor("context:overhead"), Label: "system / tool schemas"},
	}
	seen["user"] = true
	seen["context:overhead"] = true

	var tools []LegendItem
	for _, t := range turns {
		for _, b := range t.NewBlocks {
			name, ok := strings.CutPrefix(b.Source, "tool:")
			if !ok || seen[b.Source] {
				continue
			}
			seen[b.Source] = true
			tools = append(tools, LegendItem{Color: colorFor(b.Source), Label: name})
		}
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].Label < tools[j].Label })
	return append(head, tools...)
}

func buildToolCallVMs(counts map[string]int) []ToolCallVM {
	out := make([]ToolCallVM, 0, len(counts))
	for name, n := range counts {
		out = append(out, ToolCallVM{Name: name, Count: n, Color: colorFor("tool:" + name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}
