// Package analyze turns a parsed transcript into the numbers and shapes
// the report cares about: how the context window grew turn by turn, what
// put tokens into it, and how much of that never got used again.
package analyze

import "time"

// Report is everything the renderer needs, already computed.
type Report struct {
	SessionID string
	Models    []string
	Turns     []Turn
	Stats     Stats
	Warnings  []string
}

// Stats are the headline numbers — the ones meant to end up in a
// screenshot.
type Stats struct {
	TurnCount           int
	StartTime           time.Time
	EndTime             time.Time
	Duration            time.Duration
	TotalInputTokens    int // sum of Usage.input_tokens across every turn — the sliver that was neither cached nor newly written to cache
	TotalOutputTokens   int
	TotalThinkingTokens int
	TotalCacheCreation  int
	TotalCacheRead      int
	TotalContextEntered int // sum of every token that ever newly entered the window, across the whole session
	FinalContextTokens  int // window size at the last turn
	PeakContextTokens   int
	ToolCallCounts      map[string]int

	// ToolTokens is everything that entered from a tool result — the only
	// content the dead-block pass can actually judge. Overhead is inferred
	// from a billing delta rather than read from the transcript, so it has
	// no text to check and can never be flagged either way; counting it in
	// the denominator would make the percentage a fact about how large the
	// system prompt and tool schemas are rather than about wasted output.
	ToolTokens int

	// OverheadBaseline is the unattributable growth on the first turn: the
	// system prompt and tool schemas entering the window once. Unavoidable.
	// OverheadGrowth is the same kind of unattributable growth on every
	// later turn — schemas re-cached, reminders injected — which is not
	// unavoidable, and is worth separating for exactly that reason.
	OverheadBaseline int
	OverheadGrowth   int

	DeadTokens      int
	DeadTokenBlocks int
	DeadTokenPct    float64 // DeadTokens / ToolTokens, 0–1
}

// Turn is one assistant reply: what entered the context window to produce
// it (NewBlocks), and what the assistant produced in exchange (OutBlocks).
type Turn struct {
	Index               int
	Timestamp           time.Time
	Model               string
	InputTokens         int
	OutputTokens        int
	ThinkingTokens      int
	CacheCreationTokens int
	CacheReadTokens     int
	ContextTotal        int // InputTokens + CacheCreationTokens + CacheReadTokens: the billed window size for this request
	ContextDelta        int // ContextTotal minus the previous turn's ContextTotal, floored at 0
	NewBlocks           []Block
	OutBlocks           []Block
}

// Block is one attributed slice of tokens: a tool result, a stretch of
// user text, an assistant reply, a thinking block, or a tool call.
type Block struct {
	Source   string // "user" | "tool:<name>" | "context:overhead" | "context:overhead-growth" | "assistant:text" | "assistant:thinking" | "assistant:tool_use:<name>"
	Label    string
	Tokens   int
	RawChars int
	Dead     bool // only ever set on Source == "tool:*" blocks inside NewBlocks
}
