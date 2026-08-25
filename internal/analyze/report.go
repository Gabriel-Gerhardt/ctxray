package analyze

// buildReport aggregates finished turns into the headline stats. It has
// exactly one job — sum things up — and knows nothing about how turns
// were built or how blocks got attributed.
func buildReport(sessionID string, models []string, wip []wipTurn, toolCallCounts map[string]int) Report {
	stats := Stats{ToolCallCounts: toolCallCounts}
	turns := make([]Turn, len(wip))
	timeline := make([]TimelinePoint, len(wip))

	for i, w := range wip {
		turns[i] = w.turn
		timeline[i] = TimelinePoint{TurnIndex: w.turn.Index, Timestamp: w.turn.Timestamp, ContextTokens: w.turn.ContextTotal}
		accumulateStats(&stats, w.turn, i == 0)
	}

	if len(wip) > 0 {
		stats.FinalContextTokens = wip[len(wip)-1].turn.ContextTotal
	}
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	if stats.TotalContextEntered > 0 {
		stats.DeadTokenPct = float64(stats.DeadTokens) / float64(stats.TotalContextEntered)
	}

	return Report{
		SessionID: sessionID,
		Models:    models,
		Turns:     turns,
		Timeline:  timeline,
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
