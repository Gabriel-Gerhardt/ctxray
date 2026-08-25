package analyze

import "strings"

// buildReport aggregates finished turns into the headline stats. It has
// exactly one job — sum things up — and knows nothing about how turns
// were built or how blocks got attributed.
func buildReport(sessionID string, models []string, wip []wipTurn, toolCallCounts map[string]int) Report {
	stats := Stats{ToolCallCounts: toolCallCounts}
	turns := make([]Turn, len(wip))

	for i, w := range wip {
		turns[i] = w.turn
		accumulateStats(&stats, w.turn, i == 0)
	}

	if len(wip) > 0 {
		stats.FinalContextTokens = wip[len(wip)-1].turn.ContextTotal
	}
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	// Against ToolTokens, not TotalContextEntered. Overhead can never be
	// flagged dead — it has no text to check — so including it in the
	// denominator caps the percentage at "how much of your window wasn't
	// system prompt", which varies wildly with how many tool schemas are
	// loaded and says nothing about whether output went to waste.
	if stats.ToolTokens > 0 {
		stats.DeadTokenPct = float64(stats.DeadTokens) / float64(stats.ToolTokens)
	}

	return Report{
		SessionID: sessionID,
		Models:    models,
		Turns:     turns,
		Stats:     stats,
	}
}

func accumulateStats(stats *Stats, t Turn, isFirst bool) {
	if isFirst {
		stats.StartTime = t.Timestamp
	}
	stats.EndTime = t.Timestamp
	stats.TurnCount++
	if t.ContextTotal > stats.PeakContextTokens {
		stats.PeakContextTokens = t.ContextTotal
	}

	for _, b := range t.NewBlocks {
		stats.TotalContextEntered += b.Tokens
		switch {
		case b.Source == "context:overhead":
			stats.OverheadBaseline += b.Tokens
		case b.Source == "context:overhead-growth":
			stats.OverheadGrowth += b.Tokens
		case strings.HasPrefix(b.Source, "tool:"):
			stats.ToolTokens += b.Tokens
		}
		if b.Dead {
			stats.DeadTokens += b.Tokens
			stats.DeadTokenBlocks++
		}
	}

	stats.TotalInputTokens += t.InputTokens
	stats.TotalOutputTokens += t.OutputTokens
	stats.TotalThinkingTokens += t.ThinkingTokens
	stats.TotalCacheCreation += t.CacheCreationTokens
	stats.TotalCacheRead += t.CacheReadTokens
}
