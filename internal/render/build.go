package render

import (
	"fmt"
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
		Sources:     buildSourceVMs(report.Turns, report.Stats.ToolCallCounts),
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
		ToolTokens:          formatTokens(s.ToolTokens),
		OverheadGrowth:      formatTokens(s.OverheadGrowth),
		HasOverheadGrowth:   s.OverheadGrowth > 0,
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

// minRowWidthPct keeps a nearly-empty source visible as a sliver instead of
// vanishing to a hairline — it should still read as "tiny", just not
// literally invisible.
const minRowWidthPct = 2.0

// sourceTotal accumulates one source's whole-session cost while the blocks
// are still being walked.
type sourceTotal struct {
	source string
	tokens int
	dead   int
}

// buildSourceVMs renders one bar per source — Bash, Read, Grep, the user's
// own messages, the system prompt — totalling everything that source put
// into the context window across the entire session, with the share that
// was never referenced again hatched inside its own bar.
//
// This is the question the chart is actually for: "which tool is eating my
// window". A row per turn cannot answer it — one tool's cost is smeared
// across dozens of bars, and the reader has to add them up by eye.
func buildSourceVMs(turns []analyze.Turn, callCounts map[string]int) []SourceVM {
	bySource := map[string]*sourceTotal{}
	for _, t := range turns {
		for _, b := range t.NewBlocks {
			s, ok := bySource[b.Source]
			if !ok {
				s = &sourceTotal{source: b.Source}
				bySource[b.Source] = s
			}
			s.tokens += b.Tokens
			if b.Dead {
				s.dead += b.Tokens
			}
		}
	}

	totals := make([]*sourceTotal, 0, len(bySource))
	maxTokens := 1
	for _, s := range bySource {
		totals = append(totals, s)
		if s.tokens > maxTokens {
			maxTokens = s.tokens
		}
	}
	// Biggest first: the chart's job is to put the worst offender at the top.
	sort.Slice(totals, func(i, j int) bool {
		if totals[i].tokens != totals[j].tokens {
			return totals[i].tokens > totals[j].tokens
		}
		return totals[i].source < totals[j].source
	})

	out := make([]SourceVM, 0, len(totals))
	for _, s := range totals {
		if s.tokens <= 0 {
			continue
		}
		rowWidth := 100 * float64(s.tokens) / float64(maxTokens)
		if rowWidth < minRowWidthPct {
			rowWidth = minRowWidthPct
		}
		deadPct := 100 * float64(s.dead) / float64(s.tokens)
		out = append(out, SourceVM{
			Label:        sourceLabel(s.source),
			Slot:         string(slotFor(s.source)),
			Tokens:       formatTokens(s.tokens),
			DeadTokens:   formatTokens(s.dead),
			HasDead:      s.dead > 0,
			RowWidthPct:  rowWidth,
			LiveWidthPct: 100 - deadPct,
			DeadPct:      deadPct,
			Title:        sourceTitle(s, callCounts),
		})
	}
	return out
}

// sourceLabel turns an internal source key into what the chart should call
// it. Tool sources keep their own name rather than collapsing into their
// color slot, so a session full of MCP tools still names each one.
func sourceLabel(source string) string {
	switch {
	case source == "user":
		return "user"
	case source == "context:overhead":
		return "system baseline"
	case source == "context:overhead-growth":
		return "system re-entered"
	default:
		if name, ok := strings.CutPrefix(source, "tool:"); ok {
			return name
		}
		return source
	}
}

func sourceTitle(s *sourceTotal, callCounts map[string]int) string {
	label := sourceLabel(s.source)
	calls := ""
	if name, ok := strings.CutPrefix(s.source, "tool:"); ok {
		if n := callCounts[name]; n > 0 {
			calls = fmt.Sprintf(" across %d call(s)", n)
		}
	}
	if s.dead == 0 {
		return fmt.Sprintf("%s — %s tokens%s, all referenced again later", label, formatTokens(s.tokens), calls)
	}
	return fmt.Sprintf("%s — %s tokens%s, of which %s never referenced again",
		label, formatTokens(s.tokens), calls, formatTokens(s.dead))
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
	{slotOverhead, "system: schemas + reminders (baseline and re-entered)"},
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
