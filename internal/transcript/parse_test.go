package transcript

import (
	"strings"
	"testing"
)

func TestParse_StringContent(t *testing.T) {
	line := `{"type":"user","message":{"role":"user","content":"hello there"}}`
	entries, errs := Parse(strings.NewReader(line))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	blocks := entries[0].Message.Content.Blocks()
	if len(blocks) != 1 || blocks[0].Text != "hello there" {
		t.Errorf("Blocks() = %+v, want a single text block", blocks)
	}
}

func TestParse_BlockContent(t *testing.T) {
	line := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"},{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"ls"}}]}}`
	entries, errs := Parse(strings.NewReader(line))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	blocks := entries[0].Message.Content.Blocks()
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(blocks))
	}
	if blocks[1].Name != "Bash" {
		t.Errorf("blocks[1].Name = %q, want Bash", blocks[1].Name)
	}
}

func TestParse_SkipsMalformedLinesWithoutFailingTheRest(t *testing.T) {
	input := "not json\n" + `{"type":"user","message":{"role":"user","content":"ok"}}`
	entries, errs := Parse(strings.NewReader(input))
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the malformed line should be skipped, not fatal)", len(entries))
	}
}

func TestParse_LinesWithoutMessageParseWithNilMessage(t *testing.T) {
	input := `{"type":"attachment","uuid":"x"}` + "\n" + `{"type":"user","message":{"role":"user","content":"hi"}}`
	entries, errs := Parse(strings.NewReader(input))
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (both lines parse; only the second has a Message)", len(entries))
	}
	if entries[0].Message != nil {
		t.Errorf("entries[0].Message = %+v, want nil", entries[0].Message)
	}
}
