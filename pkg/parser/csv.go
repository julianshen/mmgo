package parser

import "strings"

// SplitUnquotedCommas splits s on commas that are outside single- or
// double-quoted spans. Whitespace around each item is preserved;
// callers typically TrimSpace before use. An empty input returns an
// empty slice.
//
// The quote chars `'` and `"` are supported because Mermaid grammars
// use both — kanban metadata uses single quotes, CSV-like bracket
// lists tend to use double.
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
