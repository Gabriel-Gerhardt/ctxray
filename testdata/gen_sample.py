"""Generate a synthetic but realistic Claude Code session transcript.

Nothing here comes from a real conversation. The point is a demo file whose
*shape* matches what a long agent run actually produces: ~2 hours, a
six-figure context window, and a mix of tool results that got used and tool
results that just sat there.
"""

import json
import math
import random
import uuid
from datetime import datetime, timedelta

random.seed(20260820)

SESSION_ID = "demo-4c81b7-example-session"
CWD = "/home/dev/acme-payments"
BRANCH = "fix/webhook-retry-backoff"
MODEL = "claude-sonnet-5"
START = datetime(2026, 8, 20, 13, 42, 9)

lines = []
clock = START
pending_chars = []          # rawChars of blocks awaiting the next usage snapshot
context_total = 0
last_uuid = None
_tool_ids = []


def uid():
    return str(uuid.UUID(int=random.getrandbits(128), version=4))


def est_tokens(chars):
    if chars <= 0:
        return 0
    return max(1, math.floor(chars / 4.0 + 0.5))


def advance(seconds):
    global clock
    clock += timedelta(seconds=seconds)


def ts():
    return clock.strftime("%Y-%m-%dT%H:%M:%S.000Z")


def emit(obj):
    lines.append(json.dumps(obj))


def user_text(text, gap=6):
    """A human turn."""
    global last_uuid
    advance(gap)
    u = uid()
    emit({
        "parentUuid": last_uuid, "isSidechain": False, "type": "user",
        "message": {"role": "user", "content": text},
        "uuid": u, "timestamp": ts(), "sessionId": SESSION_ID,
        "cwd": CWD, "gitBranch": BRANCH,
    })
    last_uuid = u
    pending_chars.append(len(text))


def tool_result(content, gap=3):
    """The harness handing a tool's output back to the model."""
    global last_uuid
    advance(gap)
    u = uid()
    tid = _tool_ids.pop(0)
    emit({
        "parentUuid": last_uuid, "isSidechain": False, "type": "user",
        "message": {"role": "user", "content": [
            {"type": "tool_result", "tool_use_id": tid, "content": content}
        ]},
        "uuid": u, "timestamp": ts(), "sessionId": SESSION_ID,
        "cwd": CWD, "gitBranch": BRANCH,
    })
    last_uuid = u
    pending_chars.append(len(content))


def assistant(text=None, thinking=None, tools=(), gap=8, overhead=0):
    """An assistant turn, with usage consistent with what entered the window."""
    global last_uuid, context_total
    advance(gap)

    content = []
    if thinking:
        content.append({"type": "thinking", "thinking": thinking, "signature": "sig_demo"})
    if text:
        content.append({"type": "text", "text": text})
    for name, tool_input in tools:
        tid = "toolu_" + uid().replace("-", "")[:21]
        _tool_ids.append(tid)
        content.append({"type": "tool_use", "id": tid, "name": name, "input": tool_input})

    delta = sum(est_tokens(c) for c in pending_chars) + overhead
    pending_chars.clear()

    prev_total = context_total
    input_tokens = 4
    cache_read = max(0, prev_total - input_tokens)
    cache_creation = delta
    context_total = input_tokens + cache_read + cache_creation

    out_text = len(text or "")
    out_think = len(thinking or "")
    out_tools = sum(len(json.dumps(i)) for _, i in tools)
    thinking_tokens = est_tokens(out_think)
    output_tokens = thinking_tokens + est_tokens(out_text + out_tools) + 12

    u = uid()
    emit({
        "parentUuid": last_uuid, "isSidechain": False,
        "message": {
            "model": MODEL, "id": "msg_" + uid()[:24], "type": "message",
            "role": "assistant", "content": content,
            "stop_reason": "tool_use" if tools else "end_turn",
            "usage": {
                "input_tokens": input_tokens,
                "cache_creation_input_tokens": cache_creation,
                "cache_read_input_tokens": cache_read,
                "output_tokens": output_tokens,
                "output_tokens_details": {"thinking_tokens": thinking_tokens},
            },
        },
        "type": "assistant", "uuid": u, "timestamp": ts(),
        "sessionId": SESSION_ID, "cwd": CWD, "gitBranch": BRANCH,
    })
    last_uuid = u


# ---------------------------------------------------------------------------
# Content generators.
#
# "Noise" output uses hashed / vendored / timestamped identifiers, the way
# real build and CI output does. Nothing in it recurs in later reasoning,
# which is exactly why ctxray ends up flagging it dead.
# ---------------------------------------------------------------------------

HEX = "0123456789abcdef"


def h(n):
    return "".join(random.choice(HEX) for _ in range(n))


def build_cache_listing(entries):
    out = [".turbo-cache:", "total " + str(entries * 37)]
    kinds = ["pack.gz", "chunk.js.map", "module.wasm", "bundle.css", "meta.bin"]
    for _ in range(entries):
        size = random.randint(1204, 998877)
        mins = random.randint(10, 59)
        out.append(
            "-rw-r--r--  1 dev  staff  %7d Aug 20 13:%02d "
            ".turbo-cache/%s/%s-%s.%s" % (
                size, mins, h(6), h(10), h(8), random.choice(kinds))
        )
    return "\n".join(out)


def dep_tree(entries):
    out = ["acme-payments@4.18.2 /home/dev/acme-payments"]
    for _ in range(entries):
        out.append("├─┬ @%s/%s@%d.%d.%d" % (
            h(5), h(7), random.randint(0, 9), random.randint(0, 40), random.randint(0, 30)))
        for _ in range(random.randint(1, 3)):
            out.append("│ ├── %s-%s@%d.%d.%d deduped" % (
                h(6), h(4), random.randint(0, 6), random.randint(0, 30), random.randint(0, 20)))
    return "\n".join(out)


def ci_log(entries):
    out = []
    for i in range(entries):
        t = (datetime(2026, 8, 20, 12, 0, 0) + timedelta(seconds=i * 3)).strftime("%H:%M:%S.%f")[:-3]
        out.append("2026-08-20T%sZ [runner-%s] step=%s status=ok elapsed=%dms artifact=%s" % (
            t, h(8), h(12), random.randint(4, 9000), h(16)))
    return "\n".join(out)


def vendor_grep(entries):
    out = []
    for _ in range(entries):
        out.append("vendor/%s/%s/%s.pb.go:%d:\t// Deprecated: Do not use. Marker %s." % (
            h(6), h(8), h(7), random.randint(20, 4000), h(10)))
    return "\n".join(out)


def coverage_dump(entries):
    out = ["mode: atomic"]
    for _ in range(entries):
        out.append("acme-payments/vendor/%s/%s.go:%d.%d,%d.%d %d %d" % (
            h(7), h(9), random.randint(1, 900), random.randint(1, 90),
            random.randint(1, 900), random.randint(1, 90),
            random.randint(1, 12), random.randint(0, 400)))
    return "\n".join(out)


# A real test log or source file repeats the same domain vocabulary on
# nearly every line. That matters here: the dead-context heuristic samples
# distinctive words across a block, so content made of nothing but unique
# hashes reads as dead even when the model clearly used it. Keeping the
# domain words dense is both more realistic and more honest to the tool.
TEST_NAMES = [
    "TestWebhookRetry_BackoffDoublesPerAttempt",
    "TestWebhookRetry_BackoffCapsAtMaxDelay",
    "TestWebhookRetry_RespectsRetryAttemptCap",
    "TestDispatcher_DeliverOnceSignsPayload",
    "TestDispatcher_DeliverOnceRetriesOn5xx",
    "TestDispatcher_SurfacesRetriesExhausted",
    "TestSignature_RoundTripsHMAC",
    "TestSignature_RejectsStaleTimestamp",
]

DOMAIN = [
    "backoffFor", "maxRetryDelay", "retryAttemptCap", "deliverOnce",
    "ErrRetriesExhausted", "dispatcher", "webhook", "attempt", "delivery",
    "payload", "endpoint", "backoff",
]


ENDPOINTS = ["billing.acme.test", "orders.acme.test", "ledger.acme.test"]


def full_suite_log(entries):
    """`go test ./...` across the whole tree: package paths and timings, and
    nothing a later turn ever refers back to."""
    out = []
    for _ in range(entries):
        out.append("ok  \tacme-payments/internal/%s/%s\t%d.%03ds" % (
            h(6), h(8), random.randint(0, 14), random.randint(0, 999)))
    out.append("ok  \tacme-payments/internal/webhook\t4.102s")
    return "\n".join(out)


def test_log(entries, failing=True):
    """A `go test -v` log. Long, repetitive, and mostly passing."""
    out = []
    for i in range(entries):
        name = TEST_NAMES[i % len(TEST_NAMES)]
        out.append("=== RUN   %s" % name)
        out.append("    webhook_retry_test.go:%d: %s attempt=%d backoff=%ds endpoint=%s" % (
            random.randint(40, 320), random.choice(DOMAIN),
            random.randint(1, 6), random.randint(1, 30), random.choice(ENDPOINTS)))
        out.append("--- PASS: %s (0.%02ds)" % (name, random.randint(1, 99)))
    if failing:
        out.append("=== RUN   TestWebhookRetry_BackoffCapsAtMaxDelay")
        out.append("    webhook_retry_test.go:214: backoffFor(6) exceeded the cap")
        out.append("        want backoff capped at maxRetryDelay = 30s")
        out.append("        got  backoff 960s (attempt 6 doubled past retryAttemptCap)")
        out.append("--- FAIL: TestWebhookRetry_BackoffCapsAtMaxDelay (0.03s)")
        out.append("FAIL\tacme-payments/internal/webhook\t4.219s")
    else:
        out.append("PASS")
        out.append("ok  \tacme-payments/internal/webhook\t4.102s")
    return "\n".join(out)


def go_source(pkg, symbols, funcs):
    """Plausible Go source, dense with the identifiers the model reasons about."""
    out = ["package %s" % pkg, "", "import (", '\t"errors"', '\t"time"', ")", ""]
    for s in symbols:
        out.append("// %s participates in the webhook retry path." % s)
        out.append("var %s = 30 * time.Second" % s)
        out.append("")
    for i in range(funcs):
        a, b, c = (random.choice(DOMAIN) for _ in range(3))
        out.append("// %s%d derives the %s used for this %s." % (a, i, b, c))
        out.append("func %s%d(attempt int, %s time.Duration) time.Duration {" % (a, i, b))
        out.append("\tif attempt >= retryAttemptCap {")
        out.append("\t\treturn maxRetryDelay // %s is exhausted for this delivery" % c)
        out.append("\t}")
        out.append("\tbackoff := baseRetryDelay << uint(attempt)")
        out.append("\tif backoff > maxRetryDelay {")
        out.append("\t\tbackoff = maxRetryDelay")
        out.append("\t}")
        out.append("\treturn backoff // dispatcher applies this before deliverOnce")
        out.append("}")
        out.append("")
    return "\n".join(out)


# ---------------------------------------------------------------------------
# The session.
# ---------------------------------------------------------------------------

emit({
    "type": "attachment", "timestamp": ts(), "sessionId": SESSION_ID,
    "attachment": {"type": "auto_mode"}, "uuid": uid(),
})

user_text(
    "Webhook deliveries are retrying forever in prod instead of capping at the "
    "30s max delay, and the on-call graph shows attempt counts climbing past 6. "
    "Find out why the backoff cap isn't holding, fix it, and make sure the test "
    "suite actually covers the cap. Take your time and check the CI history too.",
    gap=2,
)

# --- Phase 1: orientation. The big listing here never comes back up. --------
assistant(
    thinking="Big repo and I don't know the layout yet. Map it before touching anything.",
    tools=[("Bash", {"command": "ls -laR .turbo-cache/ && find . -type f -name '*.go' | wc -l"})],
    gap=9, overhead=13950,
)
tool_result(build_cache_listing(380) + "\n\n1487", gap=14)

assistant(
    thinking="That was the build cache, not source. Useless. Go straight at the webhook package.",
    text="Skipping the build cache — let me find the retry code itself.",
    tools=[("Glob", {"pattern": "internal/webhook/**/*.go"})],
    gap=11,
)
tool_result("\n".join([
    "internal/webhook/dispatcher.go",
    "internal/webhook/retry.go",
    "internal/webhook/retry_test.go",
    "internal/webhook/webhook_retry_test.go",
    "internal/webhook/signature.go",
]), gap=4)

assistant(
    text="`retry.go` is where the backoff lives. Reading it.",
    tools=[("Read", {"file_path": "internal/webhook/retry.go"})],
    gap=7,
)
tool_result(go_source("webhook", ["maxRetryDelay", "baseRetryDelay", "retryAttemptCap"], 140), gap=7)

assistant(
    thinking="maxRetryDelay exists, so the constant is defined. Question is whether anything clamps to it.",
    text="`retry.go` defines `maxRetryDelay` and `retryAttemptCap`, but I need to see where "
         "`maxRetryDelay` is actually applied to the computed delay.",
    tools=[("Grep", {"pattern": "maxRetryDelay|retryAttemptCap", "output_mode": "content"})],
    gap=14,
)
tool_result("\n".join([
    "internal/webhook/retry.go:12:var maxRetryDelay = 30 * time.Second",
    "internal/webhook/retry.go:13:var retryAttemptCap = 6",
    "internal/webhook/retry.go:88:\t// TODO: clamp to maxRetryDelay",
    "internal/webhook/dispatcher.go:141:\tdelay := backoffFor(attempt)",
]), gap=4)

assistant(
    thinking="There it is — line 88 is a TODO, not an implementation. The clamp was never written.",
    text="Found it. `maxRetryDelay` is declared but never applied — line 88 of `retry.go` is "
         "still a `TODO: clamp to maxRetryDelay`, and `dispatcher.go:141` takes `backoffFor(attempt)` raw. "
         "Before I patch it, let me confirm the test suite reproduces this.",
    tools=[("Bash", {"command": "go test ./internal/webhook/... -run TestWebhookRetry -v"})],
    gap=16,
)
tool_result(test_log(300), gap=560)

assistant(
    thinking="Confirmed. TestWebhookRetry_BackoffCapsAtMaxDelay fails: got 960s, want 30s cap.",
    text="Reproduced: `TestWebhookRetry_BackoffCapsAtMaxDelay` fails with backoff 960s where "
         "`maxRetryDelay` should have capped it at 30s — attempt 6 doubled straight past the cap.",
    tools=[("Read", {"file_path": "internal/webhook/dispatcher.go"})],
    gap=13,
)
tool_result(go_source("webhook", ["backoffFor", "deliverOnce"], 110), gap=8)

# --- Phase 2: the CI-history detour the user asked for. Mostly dead. -------
assistant(
    thinking="User explicitly asked about CI history. Pull the recent runs.",
    text="Checking CI history for when this regressed, since you asked.",
    tools=[("Bash", {"command": "gh run list --workflow=ci.yml --limit 40 --json databaseId,conclusion | jq -r '.[].databaseId' | head -3 | xargs -I{} gh run view {} --log"})],
    gap=12,
)
tool_result(ci_log(700), gap=150)

assistant(
    thinking="That log is all green infra noise — no failing step, nothing about the backoff. Dead end.",
    text="CI history is not going to tell us much — those runs are all green; the cap was never "
         "covered by a passing assertion in the first place.",
    tools=[("Bash", {"command": "npm ls --all"})],
    gap=15,
)
tool_result(dep_tree(220), gap=31)

assistant(
    thinking="Dependency tree isn't relevant either. Stop wandering and look at the actual test file.",
    tools=[("Read", {"file_path": "internal/webhook/webhook_retry_test.go"})],
    gap=10,
)
tool_result(go_source("webhook", ["assertBackoffCapped", "fixtureAttempts"], 95), gap=7)

assistant(
    thinking="assertBackoffCapped exists but is only wired into one case.",
    text="The test file has an `assertBackoffCapped` helper and a `fixtureAttempts` table, but "
         "only one case exercises the cap. I'll widen that after the fix.",
    tools=[("Grep", {"pattern": "backoffFor", "output_mode": "content", "-n": True})],
    gap=13,
)
tool_result("\n".join([
    "internal/webhook/retry.go:80:func backoffFor(attempt int) time.Duration {",
    "internal/webhook/retry.go:88:\t// TODO: clamp to maxRetryDelay",
    "internal/webhook/dispatcher.go:141:\tdelay := backoffFor(attempt)",
    "internal/webhook/webhook_retry_test.go:97:\t\tgot := backoffFor(tc.attempt)",
]), gap=4)

# --- Phase 3: the fix. -----------------------------------------------------
assistant(
    thinking="Clean fix: clamp inside backoffFor so every caller gets the cap, not just dispatcher.",
    text="Patching `backoffFor` itself so the clamp applies to every caller rather than just the "
         "dispatcher path.",
    tools=[("Edit", {
        "file_path": "internal/webhook/retry.go",
        "old_string": "\t// TODO: clamp to maxRetryDelay\n\treturn d",
        "new_string": "\tif d > maxRetryDelay {\n\t\td = maxRetryDelay\n\t}\n\treturn d",
    })],
    gap=22,
)
tool_result("The file internal/webhook/retry.go has been updated successfully.", gap=3)

assistant(
    thinking="Also cap the attempt count so we stop retrying past retryAttemptCap.",
    tools=[("Edit", {
        "file_path": "internal/webhook/dispatcher.go",
        "old_string": "\tdelay := backoffFor(attempt)",
        "new_string": "\tif attempt >= retryAttemptCap {\n\t\treturn errRetriesExhausted\n\t}\n\tdelay := backoffFor(attempt)",
    })],
    gap=18,
)
tool_result("The file internal/webhook/dispatcher.go has been updated successfully.", gap=3)

assistant(
    text="Both edits in. Widening the table test so the cap is actually asserted past attempt 6.",
    tools=[("Edit", {
        "file_path": "internal/webhook/webhook_retry_test.go",
        "old_string": "\t\t{attempt: 6, want: 960 * time.Second},",
        "new_string": "\t\t{attempt: 6, want: maxRetryDelay},\n\t\t{attempt: 9, want: maxRetryDelay},\n\t\t{attempt: 14, want: maxRetryDelay},",
    })],
    gap=26,
)
tool_result("The file internal/webhook/webhook_retry_test.go has been updated successfully.", gap=3)

assistant(
    text="Running the webhook suite again.",
    tools=[("Bash", {"command": "go test ./internal/webhook/... -run TestWebhookRetry -v"})],
    gap=9,
)
tool_result("\n".join([
    "=== RUN   TestWebhookRetry_BackoffCapsAtMaxDelay",
    "=== RUN   TestWebhookRetry_BackoffCapsAtMaxDelay/attempt_6",
    "=== RUN   TestWebhookRetry_BackoffCapsAtMaxDelay/attempt_9",
    "=== RUN   TestWebhookRetry_BackoffCapsAtMaxDelay/attempt_14",
    "--- PASS: TestWebhookRetry_BackoffCapsAtMaxDelay (0.02s)",
    "PASS",
    "ok  \tacme-payments/internal/webhook\t0.418s",
]), gap=181)

assistant(
    thinking="Green. Now make sure I didn't break anything else in the repo.",
    text="`TestWebhookRetry_BackoffCapsAtMaxDelay` passes at attempts 6, 9 and 14. Running the "
         "full suite to check for fallout.",
    tools=[("Bash", {"command": "go test ./... -count=1"})],
    gap=11,
)
tool_result(full_suite_log(520), gap=880)

user_text(
    "Sorry, stepped away. Before you go further — does the cap still hold when the endpoint "
    "keeps returning 503 instead of timing out? That's what prod is actually doing.",
    gap=1120,
)

assistant(
    thinking="503 goes down the retryable-status path. Check that it routes through backoffFor like the timeout path does.",
    text="Checking that the 5xx path goes through the same `backoffFor` call rather than its own delay.",
    tools=[("Grep", {"pattern": "StatusServiceUnavailable|retryableStatus", "output_mode": "content"})],
    gap=17,
)
tool_result("\n".join([
    "internal/webhook/dispatcher.go:118:func retryableStatus(code int) bool {",
    "internal/webhook/dispatcher.go:121:\tcase http.StatusServiceUnavailable, http.StatusBadGateway:",
    "internal/webhook/dispatcher.go:139:\tif retryableStatus(resp.StatusCode) {",
    "internal/webhook/dispatcher.go:141:\t\tdelay := backoffFor(attempt)",
]), gap=5)

assistant(
    thinking="Same code path — retryableStatus falls into the identical backoffFor branch I already capped.",
    text="Yes. `retryableStatus` covers 503 and 502, and `dispatcher.go:139` funnels both into the "
         "same `backoffFor(attempt)` call I just capped — so a 503 storm caps at `maxRetryDelay` "
         "and stops at `retryAttemptCap` exactly like a timeout does.",
    gap=21,
)

# --- Phase 4: more noise the model pulls and abandons. ---------------------
assistant(
    thinking="That's answered. The user mentioned prod graphs — let me see if coverage says the retry path is exercised.",
    text="Full suite is green. Checking whether the retry path is actually covered.",
    tools=[("Bash", {"command": "go test ./... -coverprofile=cover.out && go tool cover -func=cover.out"})],
    gap=14,
)
tool_result(coverage_dump(600), gap=560)

assistant(
    thinking="That dumped the whole vendor tree. Not useful — I want the one package.",
    tools=[("Bash", {"command": "go tool cover -func=cover.out | grep webhook"})],
    gap=13,
)
tool_result("\n".join([
    "acme-payments/internal/webhook/retry.go:80:\tbackoffFor\t\t100.0%",
    "acme-payments/internal/webhook/dispatcher.go:141:\tdeliverOnce\t\t92.3%",
    "total:\t\t\t\t\t(statements)\t\t88.1%",
]), gap=6)

assistant(
    thinking="backoffFor is at 100%. Good enough.",
    text="`backoffFor` is now at 100% statement coverage and `deliverOnce` at 92.3%.",
    tools=[("Grep", {"pattern": "Deprecated", "path": "vendor/", "output_mode": "content"})],
    gap=17,
)
tool_result(vendor_grep(400), gap=39)

assistant(
    thinking="Vendor deprecation warnings are unrelated to this change. Dropping that thread.",
    text="Those vendor deprecations are pre-existing and unrelated — not touching them in this change.",
    tools=[("Bash", {"command": "git diff --stat"})],
    gap=19,
)
tool_result("\n".join([
    " internal/webhook/dispatcher.go          |  3 +++",
    " internal/webhook/retry.go               |  4 +++-",
    " internal/webhook/webhook_retry_test.go  |  4 +++-",
    " 3 files changed, 9 insertions(+), 2 deletions(-)",
]), gap=4)

user_text("Nice. Does the dispatcher surface the exhausted case anywhere, or does it just swallow it?", gap=1760)

assistant(
    thinking="I returned errRetriesExhausted — need to check whether callers log or drop it.",
    tools=[("Grep", {"pattern": "errRetriesExhausted", "output_mode": "content"})],
    gap=12,
)
tool_result("\n".join([
    "internal/webhook/dispatcher.go:98:var errRetriesExhausted = errors.New(\"webhook: retries exhausted\")",
    "internal/webhook/dispatcher.go:142:\t\treturn errRetriesExhausted",
    "internal/queue/consumer.go:203:\t\tcase errors.Is(err, webhook.ErrRetriesExhausted):",
]), gap=5)

assistant(
    thinking="consumer.go references an exported ErrRetriesExhausted but dispatcher declares it unexported. That won't compile against the consumer's expectation.",
    text="Worth flagging: `queue/consumer.go:203` switches on an **exported** "
         "`webhook.ErrRetriesExhausted`, but `dispatcher.go:98` declares it unexported as "
         "`errRetriesExhausted`. Let me look at the consumer.",
    tools=[("Read", {"file_path": "internal/queue/consumer.go"})],
    gap=15,
)
tool_result(go_source("queue", ["ErrRetriesExhausted", "drainOnce"], 100), gap=8)

assistant(
    thinking="Consumer expects the exported name. Export it so the dead-letter path actually triggers.",
    text="Confirmed — `drainOnce` routes to the dead-letter queue on `ErrRetriesExhausted`, so the "
         "error has to be exported for that branch to ever fire. Exporting it.",
    tools=[("Edit", {
        "file_path": "internal/webhook/dispatcher.go",
        "old_string": "var errRetriesExhausted = errors.New(\"webhook: retries exhausted\")",
        "new_string": "// ErrRetriesExhausted is returned once retryAttemptCap is reached, so the\n// queue consumer can route the delivery to the dead-letter queue.\nvar ErrRetriesExhausted = errors.New(\"webhook: retries exhausted\")",
    })],
    gap=24,
)
tool_result("The file internal/webhook/dispatcher.go has been updated successfully.", gap=3)

assistant(
    tools=[("Bash", {"command": "sed -i 's/errRetriesExhausted/ErrRetriesExhausted/g' internal/webhook/dispatcher.go && go build ./... && go test ./internal/... -count=1"})],
    gap=8,
)
tool_result("\n".join([
    "ok  \tacme-payments/internal/webhook\t0.402s",
    "ok  \tacme-payments/internal/queue\t1.118s",
    "ok  \tacme-payments/internal/billing\t2.740s",
]), gap=505)

assistant(
    thinking="All green with the exported error wired through.",
    text="Green across `webhook`, `queue` and `billing` with `ErrRetriesExhausted` exported and "
         "the dead-letter branch reachable.",
    tools=[("Bash", {"command": "git diff"})],
    gap=13,
)
tool_result(go_source("webhook", ["ErrRetriesExhausted", "maxRetryDelay"], 80), gap=9)

# --- Phase 5: one more abandoned detour, then the wrap-up. ----------------
assistant(
    thinking="Let me double-check nothing else in the tree hardcodes a delay that skips the cap.",
    tools=[("Grep", {"pattern": "time.Second \\* [0-9]+|[0-9]+ \\* time.Second", "output_mode": "content"})],
    gap=16,
)
tool_result(vendor_grep(330), gap=36)

assistant(
    thinking="All vendor hits again. The repo's own code is clean.",
    text="Nothing in our own packages hardcodes a competing delay — the remaining hits are all vendored.",
    tools=[("Bash", {"command": "gh api repos/acme/payments/actions/runs --paginate | jq '.workflow_runs[] | select(.name==\"nightly\")'"})],
    gap=18,
)
tool_result(ci_log(500), gap=130)

assistant(
    thinking="Nightly history is more of the same green noise. Nothing to learn. Wrap up.",
    text="Nightly history has nothing on the cap either. Wrapping up.",
    tools=[("Bash", {"command": "gofmt -l . && go vet ./..."})],
    gap=14,
)
tool_result("", gap=47)

assistant(
    text=(
        "Done. Three things were wrong, not one:\n\n"
        "1. **The cap was never implemented.** `retry.go:88` was still a `TODO: clamp to "
        "maxRetryDelay` — `backoffFor` doubled without bound, which is why attempt 6 produced "
        "960s instead of 30s. Clamped inside `backoffFor` so every caller inherits it.\n\n"
        "2. **Nothing enforced `retryAttemptCap`.** The dispatcher looped past 6 attempts "
        "entirely. It now returns `ErrRetriesExhausted` at the cap.\n\n"
        "3. **The dead-letter path was unreachable.** `queue/consumer.go` switches on an exported "
        "`webhook.ErrRetriesExhausted`, but the dispatcher declared it unexported — so `drainOnce` "
        "could never route an exhausted delivery. Exported it.\n\n"
        "`TestWebhookRetry_BackoffCapsAtMaxDelay` now asserts the cap at attempts 6, 9 and 14 "
        "instead of the old hardcoded 960s expectation. `backoffFor` is at 100% coverage; "
        "`webhook`, `queue` and `billing` all pass.\n\n"
        "The CI history you asked about was a dead end — every run is green because no assertion "
        "ever covered the cap, so there was no regression point to find."
    ),
    gap=34,
)

with open("/home/user/ctxray/testdata/sample.jsonl", "w") as f:
    f.write("\n".join(lines) + "\n")

print("lines:", len(lines))
print("peak context:", context_total)
print("duration:", clock - START)
