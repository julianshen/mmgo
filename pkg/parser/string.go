package parser

import "strings"

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
