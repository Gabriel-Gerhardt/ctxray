// Package transcript parses Claude Code session .jsonl transcripts: one
// JSON object per line, describing a user turn, an assistant turn, a tool
// result, or harness bookkeeping we don't care about.
package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Entry is one line of a session transcript. Only "user" and "assistant"
// message lines carry the fields ctxray needs; every other line type
// (attachments, queue operations, summaries, ...) parses fine but has a
// nil Message and is ignored by the analyzer.
type Entry struct {
	Type       string   `json:"type"`
	UUID       string   `json:"uuid"`
	ParentUUID *string  `json:"parentUuid"`
	Timestamp  string   `json:"timestamp"`
	SessionID  string   `json:"sessionId"`
	CWD        string   `json:"cwd"`
	GitBranch  string   `json:"gitBranch"`
	Message    *Message `json:"message"`
}

// Message is the Anthropic-shaped message embedded in a user or assistant
// entry.
type Message struct {
	Role    string  `json:"role"`
	Model   string  `json:"model"`
	Content Content `json:"content"`
	Usage   *Usage  `json:"usage"`
}

// Usage is the token accounting Anthropic reports on every assistant turn.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	OutputTokensDetails      struct {
		ThinkingTokens int `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
}

// ContextTokens is the total size of the context window billed for this
// request: everything served from cache, everything newly written to
// cache, and the handful of tokens that were neither. It is the number
// that actually describes "how big was the window at this point".
func (u *Usage) ContextTokens() int {
	if u == nil {
		return 0
	}
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

// Block is one content block: text, extended thinking, a tool call, or the
// result of one. Which fields are populated depends on Type.
type Block struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   Content         `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Content models Anthropic's polymorphic content field: either a bare
// string (plain user text, the common case for a simple prompt) or an
// array of typed blocks (everything else).
type Content struct {
	raw    string
	blocks []Block
	isText bool
}

func (c *Content) UnmarshalJSON(data []byte) error {
	data = trimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("content as string: %w", err)
		}
		c.raw = s
		c.isText = true
		return nil
	}
	var blocks []Block
	if err := json.Unmarshal(data, &blocks); err != nil {
		return fmt.Errorf("content as blocks: %w", err)
	}
	c.blocks = blocks
	return nil
}

// Blocks returns the content as a block list, synthesizing a single text
// block when the underlying JSON was a bare string.
func (c Content) Blocks() []Block {
	if c.isText {
		if c.raw == "" {
			return nil
		}
		return []Block{{Type: "text", Text: c.raw}}
	}
	return c.blocks
}

// Text concatenates every text-bearing part of the content. Used by the
// "was this ever mentioned again" dead-context heuristic.
func (c Content) Text() string {
	if c.isText {
		return c.raw
	}
	var b strings.Builder
	for _, blk := range c.blocks {
		switch blk.Type {
		case "text":
			b.WriteString(blk.Text)
		case "thinking":
			b.WriteString(blk.Thinking)
		case "tool_result":
			b.WriteString(blk.Content.Text())
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// RawLen is a cheap proxy for how many bytes of JSON this content
// serializes to. It is not a token count — it is the weight used to split
// an observed token delta across the several blocks that landed in
// context between two assistant turns.
func (c Content) RawLen() int {
	if c.isText {
		return len(c.raw)
	}
	n := 0
	for _, blk := range c.blocks {
		n += len(blk.Text) + len(blk.Thinking) + len(blk.Input) + blk.Content.RawLen() + 16
	}
	return n
}

func trimSpace(b []byte) []byte {
	i := 0
	for i < len(b) && (b[i] == ' ' || b[i] == '\t' || b[i] == '\n' || b[i] == '\r') {
		i++
	}
	j := len(b)
	for j > i && (b[j-1] == ' ' || b[j-1] == '\t' || b[j-1] == '\n' || b[j-1] == '\r') {
		j--
	}
	return b[i:j]
}
