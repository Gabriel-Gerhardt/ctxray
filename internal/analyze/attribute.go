package analyze

import (
	"encoding/json"
	"strings"

	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

// charsPerToken is the same rough English/code average every tokenizer
// estimator in the wild uses. It is a proxy for splitting an observed
// token delta across several blocks, not a claim about exact tokenization
// — see the Ceiling note in the README.
const charsPerToken = 4.0

// attributeDelta assigns an observed context-window delta to the content
// that plausibly caused it. When the delta exceeds what the pending
// blocks account for — always true on turn one, where the system prompt
// and every tool schema first enter the cache — the remainder goes to a
// synthetic "context:overhead" block instead of being silently folded
// into whichever block happened to be pending.
func attributeDelta(pending []pendingBlock, delta int, firstTurn bool) (blocks []Block, texts []string) {
	if delta <= 0 {
		return nil, nil
	}
	if len(pending) == 0 {
		return []Block{overheadBlock(delta, firstTurn)}, []string{""}
	}

	estimates := make([]int, len(pending))
	estSum := 0
	for i, p := range pending {
		estimates[i] = estimateTokens(p.rawChars)
		estSum += estimates[i]
	}
	if estSum == 0 {
		return splitEvenly(pending, delta)
	}
	if estSum <= delta {
		return attributeWithOverhead(pending, estimates, delta, estSum, firstTurn)
	}
	return attributeScaledDown(pending, estimates, delta, estSum)
}

// attributeWithOverhead is the common case: the heuristic token estimate
// for what we saw undershoots what was actually billed, so each block
// keeps its own estimate and the gap becomes an overhead block.
func attributeWithOverhead(pending []pendingBlock, estimates []int, delta, estSum int, firstTurn bool) (blocks []Block, texts []string) {
	blocks = make([]Block, 0, len(pending)+1)
	texts = make([]string, 0, len(pending)+1)
	for i, p := range pending {
		blocks = append(blocks, Block{Source: p.source, Label: p.label, Tokens: estimates[i], RawChars: p.rawChars})
		texts = append(texts, p.text)
	}
	if remainder := delta - estSum; remainder > 0 {
		blocks = append(blocks, overheadBlock(remainder, firstTurn))
		texts = append(texts, "")
	}
	return blocks, texts
}

// attributeScaledDown handles the rarer case where the estimate overshoots
// the real delta (the char/token ratio for this content wasn't 4:1) —
// every block is scaled down proportionally so the row still sums to
// exactly what was billed.
func attributeScaledDown(pending []pendingBlock, estimates []int, delta, estSum int) (blocks []Block, texts []string) {
	blocks = make([]Block, len(pending))
	texts = make([]string, len(pending))
	assigned := 0
	for i, p := range pending {
		t := estimates[i] * delta / estSum
		blocks[i] = Block{Source: p.source, Label: p.label, Tokens: t, RawChars: p.rawChars}
		texts[i] = p.text
		assigned += t
	}
	distributeRemainder(blocks, delta-assigned)
	return blocks, texts
}

func splitEvenly(pending []pendingBlock, delta int) (blocks []Block, texts []string) {
	n := len(pending)
	blocks = make([]Block, n)
	texts = make([]string, n)
	base := delta / n
	assigned := 0
	for i, p := range pending {
		blocks[i] = Block{Source: p.source, Label: p.label, Tokens: base, RawChars: p.rawChars}
		texts[i] = p.text
		assigned += base
	}
	distributeRemainder(blocks, delta-assigned)
	return blocks, texts
}

func distributeRemainder(blocks []Block, remainder int) {
	if remainder != 0 && len(blocks) > 0 {
		blocks[len(blocks)-1].Tokens += remainder
	}
}

// overheadBlock accounts for growth the transcript does not explain. On
// the first turn that is the system prompt and tool schemas arriving once,
// which no session can avoid. On any later turn it means something
// re-entered the window — schemas re-cached after a server reconnects,
// injected reminders — which a session very much can avoid, so the two are
// not the same finding and do not share a bar.
func overheadBlock(tokens int, firstTurn bool) Block {
	if firstTurn {
		return Block{
			Source: "context:overhead",
			Label:  "system prompt, tool schemas, cache bookkeeping",
			Tokens: tokens,
		}
	}
	return Block{
		Source: "context:overhead-growth",
		Label:  "re-entered context: schemas re-cached, reminders injected",
		Tokens: tokens,
	}
}

func estimateTokens(chars int) int {
	if chars <= 0 {
		return 0
	}
	t := int(float64(chars)/charsPerToken + 0.5)
	if t < 1 {
		t = 1
	}
	return t
}

// attributeOutput splits an assistant turn's output tokens across the
// blocks it actually produced: thinking gets the thinking-token count
// straight from usage, everything else (reply text, tool calls) splits
// the rest proportionally by size. Returns the attributed blocks plus the
// concatenated reply/thinking text, which the dead-block pass in dead.go
// checks later turns' output against.
func attributeOutput(blocks []transcript.Block, outputTokens, thinkingTokens int) (out []Block, text string) {
	var thinking, rest []transcript.Block
	for _, b := range blocks {
		if b.Type == "thinking" {
			thinking = append(thinking, b)
		} else if b.Type == "text" || b.Type == "tool_use" {
			rest = append(rest, b)
		}
	}

	var sb strings.Builder
	out = append(out, splitThinking(thinking, thinkingTokens, &sb)...)

	restBudget := outputTokens - thinkingTokens
	if restBudget < 0 {
		restBudget = 0
	}
	out = append(out, splitRest(rest, restBudget, &sb)...)
	return out, sb.String()
}

func splitThinking(blocks []transcript.Block, budget int, sb *strings.Builder) []Block {
	sum := 0
	for _, b := range blocks {
		sum += len(b.Thinking)
	}
	out := make([]Block, 0, len(blocks))
	assigned := 0
	for _, b := range blocks {
		t := 0
		if sum > 0 {
			t = budget * len(b.Thinking) / sum
		}
		assigned += t
		out = append(out, Block{Source: "assistant:thinking", Label: firstLine(b.Thinking, 90), Tokens: t, RawChars: len(b.Thinking)})
		sb.WriteString(b.Thinking)
		sb.WriteByte('\n')
	}
	distributeRemainder(out, budget-assigned)
	return out
}

func splitRest(blocks []transcript.Block, budget int, sb *strings.Builder) []Block {
	sizes := make([]int, len(blocks))
	sum := 0
	for i, b := range blocks {
		sizes[i] = restBlockSize(b)
		sum += sizes[i]
	}
	out := make([]Block, 0, len(blocks))
	assigned := 0
	for i, b := range blocks {
		t := 0
		if sum > 0 {
			t = budget * sizes[i] / sum
		}
		assigned += t
		out = append(out, restBlock(b, t, sizes[i]))
		if b.Type == "text" {
			sb.WriteString(b.Text)
			sb.WriteByte('\n')
		}
	}
	distributeRemainder(out, budget-assigned)
	return out
}

func restBlockSize(b transcript.Block) int {
	if b.Type == "text" {
		return len(b.Text)
	}
	return len(b.Input) + len(b.Name) // tool_use
}

func restBlock(b transcript.Block, tokens, rawChars int) Block {
	if b.Type == "text" {
		return Block{Source: "assistant:text", Label: firstLine(b.Text, 90), Tokens: tokens, RawChars: rawChars}
	}
	return Block{
		Source:   "assistant:tool_use:" + b.Name,
		Label:    "▶ " + b.Name + ": " + summarizeToolInput(b.Input),
		Tokens:   tokens,
		RawChars: rawChars,
	}
}

// summarizeToolInput pulls whichever field of a tool call's input is most
// likely to tell a human what happened at a glance, without hardcoding
// every tool ctxray will ever see.
func summarizeToolInput(input json.RawMessage) string {
	if len(input) == 0 {
		return "(no input)"
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(input, &m); err != nil {
		return firstLine(string(input), 70)
	}
	for _, key := range []string{"command", "file_path", "path", "pattern", "query", "description", "prompt", "url"} {
		if raw, ok := m[key]; ok {
			var s string
			if err := json.Unmarshal(raw, &s); err == nil && s != "" {
				return firstLine(s, 70)
			}
		}
	}
	return firstLine(string(input), 70)
}
