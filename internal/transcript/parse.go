package transcript

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ParseFile reads a Claude Code session .jsonl transcript from disk.
func ParseFile(path string) (entries []Entry, errs []error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, []error{err}
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads every line of a transcript and returns the entries it could
// decode, in file order. A line that isn't valid JSON is skipped and
// reported, not fatal — a truncated or hand-edited log shouldn't crash the
// tool, it should just lose that one line.
func Parse(r io.Reader) (entries []Entry, errs []error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024*1024) // a single tool_result line can be huge

	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			errs = append(errs, fmt.Errorf("line %d: %w", lineNo, err))
			continue
		}
		entries = append(entries, e)
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return entries, errs
}
