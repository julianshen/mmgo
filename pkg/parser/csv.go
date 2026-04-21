package parser

import "strings"

// SplitUnquotedCommas splits s on commas that are outside single- or
// double-quoted spans. Whitespace around each item is preserved;
// callers typically TrimSpace (and maybe Unquote) before use. An
// empty input returns nil.
//
// Both `'` and `"` are accepted because Mermaid grammars use both —
// kanban metadata uses single quotes, bracket lists and C4 argument
// lists tend to use double. Inside a quoted span a backslash escapes
// the next byte (so \" does not close the quote); the backslash and
// its escapee are preserved verbatim. Unterminated quotes are
// silently swallowed into the final token — validate separately if
// that must surface as an error.
func SplitUnquotedCommas(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0 && c == '\\' && i+1 < len(s):
			cur.WriteByte(c)
			cur.WriteByte(s[i+1])
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
			cur.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			cur.WriteByte(c)
		case c == ',':
			out = append(out, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 || len(out) > 0 {
		out = append(out, cur.String())
	}
	return out
}
