package render

// ViewModel is everything template.html.tmpl reads. It exists so the
// template only ever does formatting-free field access — every number is
// already a string, every color already resolved.
type ViewModel struct {
	SessionID   string
	GeneratedAt string
	Models      string
	Stats       StatsVM
	TopDead     []TopDeadVM
	Sources     []SourceVM
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

// SourceVM is one bar of the "where the tokens came from" chart: everything
// a single source (Bash, Read, the user, the system prompt) put into the
// context window across the whole session. Grouping by source rather than
// by turn is what makes the chart answer "which tool is eating my window" —
// per-turn rows scatter one tool's cost across dozens of bars.
type SourceVM struct {
	Label        string // "Bash", "user", "system / tool schemas"
	Slot         string
	Tokens       string
	RowWidthPct  float64 // this source's total against the biggest source's
	LiveWidthPct float64 // share of this bar that did get referenced again
	DeadPct      float64 // share of this bar that was never referenced again
	Title        string
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
