package analyze

import (
	"testing"

	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

// TestBuild_SampleTranscript is a regression test against the shipped demo
// transcript: it pins the two blocks that were hand-crafted to be
// unreferenced (a directory listing and a test-suite wall of output) and
// the one that was hand-crafted to be referenced later (a Read result
// whose function name shows up in the final reply).
func TestBuild_SampleTranscript(t *testing.T) {
	entries, errs := transcript.ParseFile("../../testdata/sample.jsonl")
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	report := Build(entries, "test-session")

	if report.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", report.SessionID, "test-session")
	}
	if got, want := report.Stats.TurnCount, 5; got != want {
		t.Errorf("TurnCount = %d, want %d", got, want)
	}
	if got, want := report.Stats.DeadTokenBlocks, 2; got != want {
		t.Errorf("DeadTokenBlocks = %d, want %d (the ls dump and the test-suite wall)", got, want)
	}
	if report.Stats.DeadTokens <= 0 {
		t.Errorf("DeadTokens = %d, want > 0", report.Stats.DeadTokens)
	}
	if report.Stats.TotalContextEntered <= 0 {
		t.Errorf("TotalContextEntered = %d, want > 0", report.Stats.TotalContextEntered)
	}

	found := false
	for _, turn := range report.Turns {
		for _, b := range turn.NewBlocks {
			if b.Source != "tool:Read" {
				continue
			}
			found = true
			if b.Dead {
				t.Errorf("the Read result (referenced later as computeChecksumForge) was flagged dead")
			}
		}
	}
	if !found {
		t.Fatal("expected at least one tool:Read block in the sample transcript")
	}
}
