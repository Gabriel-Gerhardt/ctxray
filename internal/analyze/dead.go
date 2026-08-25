package analyze

import "strings"

const (
	deadBlockMinChars = 120 // trivial outputs ("OK", a single number) aren't worth judging
	maxSalientTokens  = 25
)

// markDeadBlocks flags every tool-result block whose distinctive content
// never shows up in the assistant's reply that turn or any later one —
// the closest cheap proxy for "this entered the window and was never used
// again". It is a
// heuristic, not a proof: a block can matter without being quoted (a Read
// that just confirms a hunch), and it can be flagged dead by coincidence
// on a very short or very generic result. See the Ceiling note in the
// README before trusting a single number from it.
func markDeadBlocks(wip []wipTurn) {
	if len(wip) == 0 {
		return
	}

	// suffixFrom[i] is the lowercase, concatenated assistant text from
	// turn i onward, built once from the tail so each block's "was this
	// mentioned later" check is one substring search instead of a
	// re-scan of the whole session.
	suffixFrom := make([]string, len(wip)+1)
	for i := len(wip) - 1; i >= 0; i-- {
		suffixFrom[i] = strings.ToLower(wip[i].outText) + suffixFrom[i+1]
	}

	for i := range wip {
		// suffixFrom[i], not [i+1]: a tool result counts as "used" if the
		// very turn that received it already talks about it — the common
		// case — not only if some later turn does.
		later := suffixFrom[i]
		for j := range wip[i].turn.NewBlocks {
			blk := &wip[i].turn.NewBlocks[j]
			if !strings.HasPrefix(blk.Source, "tool:") || blk.RawChars < deadBlockMinChars {
				continue
			}
			salient := salientTokens(wip[i].newText[j], maxSalientTokens)
			if len(salient) == 0 {
				continue // nothing distinctive enough to judge either way
			}
			blk.Dead = !anyContains(later, salient)
		}
	}
}

func anyContains(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

// stopWords are long-but-common words that would otherwise look
// "distinctive" and generate false negatives (a block wrongly counted as
// referenced just because the model happened to write "however" later).
// Past the everyday-English set, this also covers the generic vocabulary
// of code itself — "service", "handler", "config" — words long enough to
// clear the length floor but common enough in any codebase to turn up in
// an unrelated later sentence by coincidence rather than because anyone
// referenced this specific block.
var stopWords = map[string]bool{
	"because": true, "function": true, "package": true, "return": true,
	"should": true, "example": true, "default": true, "however": true,
	"another": true, "current": true, "already": true, "correctly": true,
	"following": true, "several": true, "without": true, "between": true,

	"service": true, "handler": true, "handle": true, "config": true,
	"client": true, "server": true, "request": true, "response": true,
	"context": true, "worker": true, "method": true, "result": true,
	"error": true, "errors": true, "string": true, "struct": true,
	"import": true, "module": true, "object": true, "process": true,
	"system": true, "public": true, "private": true, "static": true,
}

// salientTokens pulls the distinctive words out of a block of text — long
// enough to be meaningful, common words filtered out — and samples them
// evenly across the text rather than taking only the first few, so a
// 10,000-line grep dump isn't judged solely by its first screen.
func salientTokens(text string, max int) []string {
	var words []string
	seen := map[string]bool{}
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		w := strings.ToLower(cur.String())
		cur.Reset()
		if len(w) < 6 || stopWords[w] || seen[w] {
			return
		}
		seen[w] = true
		words = append(words, w)
	}
	for _, r := range text {
		if isWordRune(r) {
			cur.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	if len(words) <= max {
		return words
	}
	out := make([]string, max)
	step := float64(len(words)) / float64(max)
	for i := range out {
		out[i] = words[int(float64(i)*step)]
	}
	return out
}

func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
		r == '_' || r == '.' || r == '-' || r == '/'
}
