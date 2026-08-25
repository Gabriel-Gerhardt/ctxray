# ctxray

**Flamegraph for your agent's context window.**

[![ci](https://github.com/Gabriel-Gerhardt/ctxray/actions/workflows/ci.yml/badge.svg)](https://github.com/Gabriel-Gerhardt/ctxray/actions/workflows/ci.yml)
[![golangci-lint](https://img.shields.io/badge/lint-golangci--lint-informational?logo=go&logoColor=white)](https://golangci-lint.run/)
[![go version](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![license](https://img.shields.io/github/license/Gabriel-Gerhardt/ctxray)](LICENSE)

Your last agent run burned 160k tokens. Which ones actually did anything?

Point `ctxray` at a Claude Code session transcript and it answers that in one self-contained HTML file: where every token in the context window came from, and how much of it was never mentioned again.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/screenshot-dark.png">
  <img src="docs/screenshot.png" alt="full ctxray report: the dead-token headline, the five biggest dead blocks, and a bar per source showing where the context window went">
</picture>

Every agent CLI tells you *how many* tokens you spent. None of them tell you *on what*. `ctxray` reconstructs it from the transcript already sitting on your disk — no telemetry, no API key, no dashboard to sign up for, no dependencies.

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

Claude Code writes one JSON object per line to `~/.claude/projects/<project>/<session>.jsonl` as a session runs — every message, every tool call, every tool result, and the exact token usage Anthropic billed for each assistant turn.

`ctxray` reads that file once and reconstructs three things:

1. **What entered the context window.** The delta between two consecutive turns' billed context size is attributed to whatever landed in the conversation since the last turn — a tool result, a stretch of user text, or, on turn one, the system prompt and tool schemas.
2. **What the assistant produced in exchange.** Output tokens split across the reply text, extended thinking, and any tool calls.
3. **What never got used again.** Every tool result over a size threshold is checked against every assistant turn from that point on, its own reply included. If none of its distinctive content shows up anywhere later, it's flagged dead.

The report opens on the total dead-token count, lists the five biggest dead blocks, then draws one bar per source — Bash, Read, Grep, your own messages, the system prompt — showing what each put into the window across the whole session, with the never-referenced share hatched inside its own bar.

## Ceiling

Token counts under roughly 1,000 are *estimated* from character length (~4 chars/token) and scaled to match what Anthropic actually billed for that turn. The totals are exact; the split between blocks within a turn is attribution, not a tokenizer count.

"Dead" is a heuristic, not a proof: a block is flagged when none of its distinctive words show up in the assistant's reply that turn or any later one. A tool result can matter without being quoted back — a `Read` that just confirms a hunch, a `Grep` with zero matches that rules something out — and those get hatched too. Treat the dead-token percentage as a lead worth chasing, not a verdict on the session.

Exact per-source numbers live in the hover tooltip, and tooltips are hover-only — keyboard focus doesn't surface the same text yet.

## Contact

- LinkedIn: https://www.linkedin.com/in/gabriel-gerhardt27/
- Email: gabrielgerhardt27@gmail.com
- GitHub: https://github.com/Gabriel-Gerhardt
