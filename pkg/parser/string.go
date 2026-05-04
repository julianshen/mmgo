package parser

import (
	"fmt"
	"strings"
)

// ExtractCSSClassShorthand extracts a trailing `:::name` shorthand
// from an identifier reference (`Foo:::hot` → ("Foo", "hot")). When
// no shorthand is present, returns the input unchanged with css="".
//
// Mermaid only allows a single shorthand per identifier; chained
// forms (`Foo:::a:::b`) are an error and the caller should surface
// it. This helper returns ok=false specifically for the chained
// case so callers can `if !ok { return error }`.
func ExtractCSSClassShorthand(s string) (id, css string, ok bool) {
	i := strings.Index(s, ":::")
	if i < 0 {
		return s, "", true
	}
	id = strings.TrimSpace(s[:i])
	css = strings.TrimSpace(s[i+3:])
	if strings.Contains(css, ":::") {
		return s, "", false
	}
	return id, css, true
}

// ExpandLineBreaks converts the literal two-character sequence `\n`
// (a backslash followed by `n`) into a real newline. Mermaid uses
// this convention to embed line breaks inside labels, note text, and
// other free-form fields without forcing the source to span actual
// lines.
func ExpandLineBreaks(s string) string {
	return strings.ReplaceAll(s, `\n`, "\n")
}

// Unquote strips a single pair of matching surrounding quotes — either
// `"..."` or `'...'` — after first trimming whitespace. Strings without
// surrounding quotes are returned as-is (but still whitespace-trimmed).
// Mermaid uses double quotes for most labels and single quotes for
// metadata values (e.g. kanban's `@{ priority: 'High' }`), so both
// styles must be accepted.
func Unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// ExtractBracketLabel splits an identifier reference of the form
// `Head["Display Label"]` into the head text and the unquoted label.
// Used by ER and class parsers to handle aliased entity / class names.
//
// Rules:
//   - No `[` present → returns (s, "", nil) unchanged.
//   - Unclosed `[` (no matching `]`, or `]` before `[`) → error.
//   - Bracket contents must be quoted (`"..."` or `'...'`); a bare
//     identifier inside brackets is rejected.
//   - Any non-whitespace content AFTER the closing `]` is rejected
//     so typos like `Foo["x"]junk` surface instead of silently
//     dropping the trailing text.
//
// The returned head is whitespace-trimmed.
func ExtractBracketLabel(s string) (head, label string, err error) {
	open := strings.IndexByte(s, '[')
	if open < 0 {
		return s, "", nil
	}
	closeIdx := strings.LastIndexByte(s, ']')
	if closeIdx <= open {
		return "", "", fmt.Errorf("bracketed label %q: unclosed `[`", s)
	}
	if trailing := strings.TrimSpace(s[closeIdx+1:]); trailing != "" {
		return "", "", fmt.Errorf("bracketed label %q: unexpected content after `]`", s)
	}
	inside := strings.TrimSpace(s[open+1 : closeIdx])
	unq := Unquote(inside)
	if unq == inside {
		return "", "", fmt.Errorf("bracketed label %q: contents must be quoted", s)
	}
	return strings.TrimSpace(s[:open]), unq, nil
}
