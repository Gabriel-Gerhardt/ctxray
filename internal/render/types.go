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
	Turns       []TurnVM
	TimelineSVG template.HTML
	Legend      []LegendItem
	ToolCalls   []ToolCallVM
}

type StatsVM struct {
	TurnCount           int
	Duration            string
	StartClock          string
	EndClock            string
	TotalContextEntered string
	FinalContextTokens  string
	PeakContextTokens   string
	TotalOutputTokens   string
	TotalThinkingTokens string
	DeadTokens          string
	DeadTokenPct        string
	DeadTokenBlocks     int
}

type TurnVM struct {
	Index      int
	Clock      string
	DeltaLabel string
	Blocks     []BlockVM
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
