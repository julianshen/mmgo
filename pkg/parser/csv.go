package parser

import "strings"

// SplitUnquotedCommas splits s on commas that are outside single- or
// double-quoted spans. Whitespace around each item is preserved;
// callers typically TrimSpace (and maybe Unquote) before use. An
// empty input returns nil.
//
// The quote chars `'` and `"` are both supported because Mermaid
// grammars use both — kanban metadata uses single quotes, bracket
// lists and C4 argument lists tend to use double. Inside a quoted
// span, a backslash escapes the next character so \" does not close
// the quote; the backslash and its escaped byte are preserved in
// the output. Unterminated quotes consume the rest of the input
// into a single token; callers that need to surface that as a
// syntax error must validate separately.
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
