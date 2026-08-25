<p align="center">
  <img src="docs/brand.png" alt="ctxray" width="620">
</p>

<p align="center"><strong>Dead-code analysis for your agent's context window.</strong></p>

<p align="center">
  <a href="https://github.com/Gabriel-Gerhardt/ctxray/actions/workflows/ci.yml"><img src="https://github.com/Gabriel-Gerhardt/ctxray/actions/workflows/ci.yml/badge.svg" alt="ci"></a>
  <a href="https://golangci-lint.run/"><img src="https://img.shields.io/badge/lint-golangci--lint-informational?logo=go&logoColor=white" alt="golangci-lint"></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go&logoColor=white" alt="go version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/Gabriel-Gerhardt/ctxray" alt="license"></a>
</p>

Your last agent run burned 160k tokens. Which ones actually did anything?

Point `ctxray` at a coding-agent session transcript and it finds the tool output that entered the context window and was never referenced again — then shows you which tool put it there.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/screenshot-dark.png">
  <img src="docs/screenshot.png" alt="full ctxray report: the dead-token headline, the five biggest dead blocks, and a bar per source showing where the context window went">
</picture>

## About

Nothing ever leaves a context window on its own, and every request re-sends the whole thing. The 20k-token directory listing that landed on turn 3 gets paid for again on turn 4, and again on turn 40 — and it keeps crowding out what the model actually needs until the session compacts and starts throwing things away.

`ctxray` asks the question the token totals can't: not how much you spent, but **how much of it your agent ever read.**

Point it at a session log and it hands back one HTML file — how much each tool put into the context window, and how much of that was never referenced again. One Go binary. No telemetry, no API key, nothing to sign up for.

## Supported formats

Claude Code today (`~/.claude/projects/*/*.jsonl`). Only `internal/transcript` knows the format — everything downstream works on messages, tool results and per-turn usage. If your agent records those, it's a parser away.

## Quickstart

```bash
go install github.com/Gabriel-Gerhardt/ctxray@latest
ctxray ~/.claude/projects/*/*.jsonl -open
```

That's it — one binary, zero dependencies, and the report opens in your browser.

No Go toolchain? Grab a prebuilt binary for your platform from [the latest release](https://github.com/Gabriel-Gerhardt/ctxray/releases/latest).

Want to see a report before hunting down a real session file? The repo ships a synthetic demo transcript (no real conversation content):

```bash
git clone https://github.com/Gabriel-Gerhardt/ctxray && cd ctxray
go run . testdata/sample.jsonl -open
```

```
Usage: ctxray [flags] <session.jsonl>

  -o string   output HTML file (default "ctxray-report.html")
  -open       open the report in the default browser when done
```

## How it works

A session log is one JSON object per line: every message, every tool call, every tool result, and the exact token usage the API billed for each assistant turn.

`ctxray` reads that file once and reconstructs three things:

1. **What entered the context window.** The delta between two consecutive turns' billed context size is attributed to whatever landed in the conversation since the last turn — a tool result, a stretch of user text, or, on turn one, the system prompt and tool schemas.
2. **What the assistant produced in exchange.** Output tokens split across the reply text, extended thinking, and any tool calls.
3. **What never got used again.** Every tool result over a size threshold is checked against every assistant turn from that point on, its own reply included. If none of its distinctive content shows up anywhere later, it's flagged dead.

The report opens on the total dead-token count, lists the five biggest dead blocks, then draws one bar per source — Bash, Read, Grep, your own messages, the system prompt — showing what each put into the window across the whole session, with the never-referenced share hatched inside its own bar.

## What the number isn't

"Dead" is a heuristic, not a proof. A block gets flagged when the assistant never reproduces enough of its distinctive text afterward — roughly one long identifier's worth. So a `Read` that quietly confirmed a hunch, or a `Grep` with zero matches that ruled something out, gets hatched like waste. Treat the percentage as a lead worth chasing, not a verdict.

Per-block counts are estimated from character length (~4 chars/token) and scaled to what the API actually billed that turn. Session totals are exact; the split between blocks inside a turn is attribution.

The percentage is measured against tool output, not against the whole window. The system prompt and tool schemas enter the window as a billing delta with no text attached, so they can never be judged either way — counting them in the denominator would turn the number into a fact about how many tool schemas you have loaded rather than about wasted output.

## Contact

- LinkedIn: https://www.linkedin.com/in/gabriel-gerhardt27/
- Email: gabrielgerhardt27@gmail.com
- GitHub: https://github.com/Gabriel-Gerhardt
