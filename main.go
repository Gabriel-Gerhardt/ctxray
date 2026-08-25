// Command ctxray turns a Claude Code session transcript into a
// self-contained HTML report: where the tokens in your context window
// came from, turn by turn, and how much of it never got used again.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/Gabriel-Gerhardt/ctxray/internal/analyze"
	"github.com/Gabriel-Gerhardt/ctxray/internal/render"
	"github.com/Gabriel-Gerhardt/ctxray/internal/transcript"
)

func main() {
	out := flag.String("o", "ctxray-report.html", "output HTML file")
	open := flag.Bool("open", false, "open the report in the default browser when done")
	flag.Usage = printUsage
	flag.Parse()

	if flag.NArg() != 1 {
		printUsage()
		os.Exit(2)
	}

	report, err := analyzeSession(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctxray: %v\n", err)
		os.Exit(1)
	}

	if err := writeReport(report, *out); err != nil {
		fmt.Fprintf(os.Stderr, "ctxray: %v\n", err)
		os.Exit(1)
	}

	printSummary(report, *out)

	if *open {
		if err := openBrowser(*out); err != nil {
			fmt.Fprintf(os.Stderr, "ctxray: could not open browser: %v\n", err)
		}
	}
}

func analyzeSession(path string) (report analyze.Report, err error) {
	entries, parseErrs := transcript.ParseFile(path)
	if len(entries) == 0 {
		if len(parseErrs) > 0 {
			// A single error here means the file itself couldn't be
			// opened (see transcript.ParseFile) — surface it as-is
			// rather than the generic "nothing to parse" message below,
			// which would misreport a missing file as a format problem.
			return report, parseErrs[0]
		}
		return report, fmt.Errorf("found nothing to parse in %s — is this a Claude Code session .jsonl?", path)
	}
	if len(parseErrs) > 0 {
		fmt.Fprintf(os.Stderr, "ctxray: skipped %d malformed line(s)\n", len(parseErrs))
	}

	report = analyze.Build(entries, sessionIDOf(entries, path))
	if report.Stats.TurnCount == 0 {
		return report, fmt.Errorf("parsed the file but found zero assistant turns with usage data — nothing to report")
	}
	return report, nil
}

func sessionIDOf(entries []transcript.Entry, fallbackPath string) string {
	for _, e := range entries {
		if e.SessionID != "" {
			return e.SessionID
		}
	}
	return filepath.Base(fallbackPath)
}

func writeReport(report analyze.Report, outPath string) (err error) {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return render.Render(report, f)
}

func printSummary(report analyze.Report, outPath string) {
	s := report.Stats
	fmt.Printf("ctxray: %d turns · %s tokens entered · peak window %s\n",
		s.TurnCount, shortTokens(s.TotalContextEntered), shortTokens(s.PeakContextTokens))
	fmt.Printf("ctxray: %.1f%% dead context — %s tokens across %d tool result(s) never referenced again\n",
		s.DeadTokenPct*100, shortTokens(s.DeadTokens), s.DeadTokenBlocks)
	fmt.Printf("ctxray: report written to %s\n", outPath)
}

func shortTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func openBrowser(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", abs).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", abs).Start()
	default:
		return exec.Command("xdg-open", abs).Start()
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `ctxray — dead-code analysis for your agent's context window

Usage:
  ctxray [flags] <session.jsonl>

Flags:
  -o string   output HTML file (default "ctxray-report.html")
  -open       open the report in the default browser when done

Example:
  ctxray ~/.claude/projects/*/*.jsonl -open

ctxray reads a Claude Code session transcript and finds the tool output
that entered the context window and was never referenced again — and
which tool put it there.
`)
}
