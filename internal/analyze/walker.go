package analyze

import (
	"strings"
	"time"

	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

// pendingBlock is content that landed in the conversation but hasn't been
// attributed to a token count yet — it sits here from the moment it's
// seen until the next assistant usage snapshot says how many tokens
// actually entered the window.
type pendingBlock struct {
	source   string
	label    string
	rawChars int
	text     string
}

// wipTurn is Turn plus the raw text ctxray needs for the dead-block pass,
// which only runs once every turn exists. newText is parallel to
// turn.NewBlocks (empty string for non-tool blocks); outText is this
// turn's assistant text, checked against earlier turns' tool blocks.
type wipTurn struct {
	turn    Turn
	newText []string
	outText string
}

// walker is the one piece of code that knows how to read entries in
// order. Its only job is turning a stream of user/assistant messages into
// turns; token attribution and dead-block detection are separate,
// stateless passes over its output — see attribute.go and dead.go.
type walker struct {
	toolNameByID   map[string]string
	toolCallCounts map[string]int
	modelSeen      map[string]bool
	models         []string
	pending        []pendingBlock
	lastContext    int
	wip            []wipTurn
}

func newWalker() *walker {
	return &walker{
		toolNameByID:   map[string]string{},
		toolCallCounts: map[string]int{},
		modelSeen:      map[string]bool{},
	}
}

// Build walks a parsed transcript once, reconstructing the sequence of
// assistant turns and attributing every token Anthropic billed to the
// content block that most plausibly caused it.
func Build(entries []transcript.Entry, sessionID string) Report {
	w := newWalker()
	for _, e := range entries {
		if e.Message == nil {
			continue // attachment / queue-operation / summary / system lines carry no usage
		}
		switch e.Message.Role {
		case "user":
			w.handleUser(e.Message)
		case "assistant":
			ts, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
			w.handleAssistant(e.Message, ts)
		}
	}
	markDeadBlocks(w.wip)
	return buildReport(sessionID, w.models, w.wip, w.toolCallCounts)
}

func (w *walker) handleUser(msg *transcript.Message) {
	for _, blk := range msg.Content.Blocks() {
		switch blk.Type {
		case "tool_result":
			name := w.toolNameByID[blk.ToolUseID]
			if name == "" {
				name = "unknown"
			}
			text := blk.Content.Text()
			w.pending = append(w.pending, pendingBlock{
				source:   "tool:" + name,
				label:    "→ " + name + ": " + firstLine(text, 90),
				rawChars: blk.Content.RawLen(),
				text:     text,
			})
		case "text":
			if strings.TrimSpace(blk.Text) == "" {
				continue
			}
			w.pending = append(w.pending, pendingBlock{
				source:   "user",
				label:    firstLine(blk.Text, 90),
				rawChars: len(blk.Text),
				text:     blk.Text,
			})
		}
	}
}

func (w *walker) handleAssistant(msg *transcript.Message, ts time.Time) {
	w.registerToolUses(msg)
	w.registerModel(msg.Model)

	if msg.Usage == nil {
		return // no usage snapshot on this line (e.g. an interrupted turn) — nothing to attribute
	}

	ctxTotal := msg.Usage.ContextTokens()
	delta := ctxTotal - w.lastContext
	if delta < 0 {
		delta = 0 // context shrank (compaction) — never attribute negative tokens
	}
	w.lastContext = ctxTotal

	newBlocks, newText := attributeDelta(w.pending, delta, len(w.wip) == 0)
	w.pending = nil

	thinkingTokens := msg.Usage.OutputTokensDetails.ThinkingTokens
	outBlocks, outText := attributeOutput(msg.Content.Blocks(), msg.Usage.OutputTokens, thinkingTokens)

	w.wip = append(w.wip, wipTurn{
		turn: Turn{
			Index:               len(w.wip),
			Timestamp:           ts,
			Model:               msg.Model,
			InputTokens:         msg.Usage.InputTokens,
			OutputTokens:        msg.Usage.OutputTokens,
			ThinkingTokens:      thinkingTokens,
			CacheCreationTokens: msg.Usage.CacheCreationInputTokens,
			CacheReadTokens:     msg.Usage.CacheReadInputTokens,
			ContextTotal:        ctxTotal,
			ContextDelta:        delta,
			NewBlocks:           newBlocks,
			OutBlocks:           outBlocks,
		},
		newText: newText,
		outText: outText,
	})
}

func (w *walker) registerToolUses(msg *transcript.Message) {
	for _, blk := range msg.Content.Blocks() {
		if blk.Type != "tool_use" {
			continue
		}
		w.toolNameByID[blk.ID] = blk.Name
		w.toolCallCounts[blk.Name]++
	}
}

func (w *walker) registerModel(model string) {
	if model == "" || w.modelSeen[model] {
		return
	}
	w.modelSeen[model] = true
	w.models = append(w.models, model)
}
