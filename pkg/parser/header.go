package parser

import "strings"

// HasHeaderKeyword reports whether line begins with kw followed by a
// word boundary: end-of-string, whitespace, or `:` (Mermaid tolerates
// a trailing colon on most diagram headers). `grapha` is not
// mis-matched as `graph`.
func HasHeaderKeyword(line, kw string) bool {
	if !strings.HasPrefix(line, kw) {
		return false
	}
	if len(line) == len(kw) {
		return true
	}
	c := line[len(kw)]
	return c == ' ' || c == '\t' || c == ':'
}
