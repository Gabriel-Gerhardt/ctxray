package render

import "html/template"

// ViewModel is everything template.html.tmpl reads. It exists so the
// template only ever does formatting-free field access — every number is
// already a string, every color already resolved.
type ViewModel struct {
	SessionID   string
	GeneratedAt string
	Models      string
	Stats       StatsVM
	TopDead     []TopDeadVM
	Turns       []TurnVM
	TimelineSVG template.HTML
	Legend      []LegendItem
	ToolCalls   []ToolCallVM
}

// TopDeadVM is one row of the plain-text "biggest dead weight" ranking —
// no chart, no legend to learn, just a sorted list of what to go look at
// first.
type TopDeadVM struct {
	Rank   int
	Label  string
	Tokens string
	Slot   string
}

type StatsVM struct {
	TurnCount           int
	Duration            string
	StartClock          string
	EndClock            string
	PeakContextTokens   string
	TotalOutputTokens   string
	TotalThinkingTokens string
	DeadTokens          string
	DeadTokensExact     string
	DeadTokenPct        string
	DeadTokenBlocks     int
}

type TurnVM struct {
	RowTitle    string // "turn #14 · 00:36:37" — the index/clock live here, in the hover, instead of as two more columns competing with the bar for attention
	DeltaLabel  string
	RowWidthPct float64 // this turn's delta relative to the session's biggest turn — a real flamegraph encodes magnitude in length, not just in a text label
	Blocks      []BlockVM
}

type BlockVM struct {
	WidthPct float64
	Slot     string
	Dead     bool
	Title    string
}

type LegendItem struct {
	Slot  string
	Label string
}

type ToolCallVM struct {
	Name  string
	Count int
	Slot  string
}
