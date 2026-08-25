package render

import "strings"

// slot is a fixed, semantic color identity — never generated, never
// hashed per tool name. Only the sources that actually tend to dominate a
// context window (user, Bash, Read, Grep) get their own identity; every
// other tool — Edit, Write, Glob, WebFetch, an MCP server's tools,
// whatever — folds into "other" instead of inventing a new hue. Past a
// handful of identities, more colors stop helping and start reading as
// noise; the fixed slot order (not per-tool hashing) is also what keeps
// every render of the same session colored identically.
type slot string

const (
	slotUser     slot = "user"
	slotBash     slot = "bash"
	slotRead     slot = "read"
	slotGrep     slot = "grep"
	slotOther    slot = "other"
	slotOverhead slot = "overhead"
)

// slotFor maps a block source to its fixed color slot.
func slotFor(source string) slot {
	switch {
	case source == "user":
		return slotUser
	case source == "context:overhead", source == "context:overhead-growth":
		return slotOverhead
	case source == "tool:Bash":
		return slotBash
	case source == "tool:Read":
		return slotRead
	case source == "tool:Grep":
		return slotGrep
	default:
		return slotOther
	}
}

// otherToolName extracts the tool name from a source when it belongs to
// the shared "other" slot, so the legend can still name what's in there
// instead of hiding it behind one generic label.
func otherToolName(source string) (name string, ok bool) {
	name, ok = strings.CutPrefix(source, "tool:")
	return name, ok && slotFor(source) == slotOther
}
