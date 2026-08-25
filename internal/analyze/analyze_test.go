package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

// TestBuild_SampleTranscript is a regression test against the shipped demo
// transcript: a 30-turn, ~1h50m webhook-debugging session built so that
// specific tool results are referenced later (the source files the fix
// touches, the failing test log, the final diff) and specific others are
// not (a build-cache listing, two CI log dumps, a dependency tree, the
// green full-suite run, a coverage profile, two vendor greps).
// It pins both the aggregate dead-block count and each named block's
// individual verdict, so a change to the attribution or dead-block
// heuristic that silently flips one of them fails loudly here instead of
// only showing up as a different percentage in a screenshot.
func TestBuild_SampleTranscript(t *testing.T) {
	entries, errs := transcript.ParseFile("../../testdata/sample.jsonl")
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}

	report := Build(entries, "test-session")

	if report.SessionID != "test-session" {
		t.Errorf("SessionID = %q, want %q", report.SessionID, "test-session")
	}
	if got, want := report.Stats.TurnCount, 30; got != want {
		t.Errorf("TurnCount = %d, want %d", got, want)
	}
	if got, want := report.Stats.DeadTokenBlocks, 11; got != want {
		t.Errorf("DeadTokenBlocks = %d, want %d", got, want)
	}
	// The demo session is deliberately mixed, not a strawman: roughly half
	// its context earns its place. Pinning a band rather than a floor keeps
	// a heuristic change that flips the balance either way visible here.
	if pct := report.Stats.DeadTokenPct; pct < 0.4 || pct > 0.6 {
		t.Errorf("DeadTokenPct = %.3f, want roughly half (0.4–0.6) — the demo transcript is a realistic mix, not all dead weight", pct)
	}
	if report.Stats.DeadTokens < 50_000 {
		t.Errorf("DeadTokens = %d, want a five-figure count (the whole point of this transcript is a large absolute number)", report.Stats.DeadTokens)
	}
	if report.Stats.PeakContextTokens < 100_000 {
		t.Errorf("PeakContextTokens = %d, want a realistic six-figure session size", report.Stats.PeakContextTokens)
	}
	if d := report.Stats.Duration; d < time.Hour {
		t.Errorf("Duration = %s, want a multi-hour session", d)
	}

	wantDead := map[string]bool{
		".turbo-cache:":            true,  // build-cache listing — never mentioned again
		"[runner-":                 true,  // two CI log dumps, both abandoned
		"acme-payments@4.18.2":     true,  // dependency tree
		"ok  \tacme-payments/":     true,  // the green full-suite run nobody reads twice
		"mode: atomic":             true,  // coverage profile over the vendor tree
		"Deprecated: Do not use":   true,  // two vendor greps
		"→ Read: package webhook":  false, // the source files the fix actually touches
		"→ Read: package queue":    false, // consumer.go, read to confirm the dead-letter path
		"→ Bash: === RUN   TestWe": false, // the failing test log, quoted back in the diagnosis
		"→ Bash: package webhook":  false, // the final git diff
	}

	matched := map[string]bool{}
	for _, turn := range report.Turns {
		for _, b := range turn.NewBlocks {
			for substr, wantDeadVal := range wantDead {
				if !strings.Contains(b.Label, substr) {
					continue
				}
				matched[substr] = true
				if b.Dead != wantDeadVal {
					t.Errorf("block %q: Dead = %v, want %v (label %q)", substr, b.Dead, wantDeadVal, b.Label)
				}
			}
		}
	}
	for substr := range wantDead {
		if !matched[substr] {
			t.Errorf("expected a block labeled with %q in the sample transcript, found none", substr)
		}
	}
}
