package analyze

import (
	"strings"
	"testing"

	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

// TestBuild_SampleTranscript is a regression test against the shipped demo
// transcript: a 17-turn, ~18-minute session hand-crafted so that specific
// tool results are referenced later (main.go's computeChecksumForge,
// config.go's port default, the focused TestConfigDefaultPort run, the
// TODO grep, the git diff stat) and specific others are not (a directory
// listing, a full test wall, git log, an unrelated file read, a broad
// grep, go vet, a go.sum dump, go build, a noisy grep, a changelog read).
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
	if got, want := report.Stats.TurnCount, 17; got != want {
		t.Errorf("TurnCount = %d, want %d", got, want)
	}
	if got, want := report.Stats.DeadTokenBlocks, 10; got != want {
		t.Errorf("DeadTokenBlocks = %d, want %d", got, want)
	}
	if report.Stats.DeadTokenPct < 0.5 {
		t.Errorf("DeadTokenPct = %.3f, want a dramatically high fraction (this transcript is mostly dead weight on purpose)", report.Stats.DeadTokenPct)
	}
	if report.Stats.DeadTokens < 10_000 {
		t.Errorf("DeadTokens = %d, want a five-figure count (the whole point of this transcript is a large absolute number)", report.Stats.DeadTokens)
	}
	if report.Stats.PeakContextTokens < 10_000 {
		t.Errorf("PeakContextTokens = %d, want a realistic five-figure session size", report.Stats.PeakContextTokens)
	}

	wantDead := map[string]bool{
		"112":                                          true,  // find/ls -laR dump — never mentioned again
		"=== RUN   TestWombatService_000":              true,  // full test wall — only the focused rerun is mentioned
		"chore: touch up wombat_service":               true,  // git log
		"package service":                              true,  // unrelated file read, never discussed
		"func HandleWombatService000":                  true,  // broad grep across the repo
		"# example-repo/src/wombat_service000":         true,  // go vet output
		"github.com/example/wombat-service":            true,  // go.sum dump
		"compiling example-repo/src/wombat_service000": true,  // go build output
		"reporting metrics on the import path":         true,  // noisy grep
		"## v0.40.0":                                   true,  // changelog read
		"package main":                                 false, // referenced via computeChecksumForge
		"package config":                               false, // referenced via the port-default fix
		"src/wombat_service.go:12:// TODO":             false, // referenced via "retry backoff"
		"=== RUN   TestConfigDefaultPort":              false, // referenced by name in the final reply
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
