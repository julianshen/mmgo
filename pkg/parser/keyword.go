package parser

import "strings"

// TrimKeyword strips the leading keyword kw from line plus any
// immediately following whitespace or colon. Use this after
// HasHeaderKeyword(line, kw) has confirmed the match.
//
// Motivation: HasHeaderKeyword accepts `:` as a word boundary so forms
// like `title:X`, `x-axis: Low --> High`, and `title X` all pass the
// match. A naive `strings.TrimSpace(strings.TrimPrefix(line, kw))`
// leaves a leading `:` in the colon forms; this helper strips it.
func TrimKeyword(line, kw string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(line, kw), ": \t"))
}
