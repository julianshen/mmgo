package parser

import "strings"

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
