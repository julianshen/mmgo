package parser

import "strings"

// NormalizeCSS converts Mermaid's comma-separated CSS-declaration syntax
// (`fill:#fff,stroke:#000`) into the semicolon-separated form CSS
// actually accepts. Without this conversion, browsers and canvas
// parsers see a malformed value at the first comma and silently fall
// back to the default fill (typically black), producing the "all
// nodes black" regression on `style` and `classDef` rules.
//
// Commas inside parens (e.g., `rgb(0, 0, 0)`) and inside string
// literals are preserved as-is.
func NormalizeCSS(css string) string {
	if strings.IndexByte(css, ',') < 0 {
		return css
	}
	var sb strings.Builder
	sb.Grow(len(css))
	depth := 0
	inQuote := false
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case c == '"' || c == '\'':
			inQuote = !inQuote
		case c == '(' && !inQuote:
			depth++
		case c == ')' && !inQuote:
			if depth > 0 {
				depth--
			}
		}
		if c == ',' && depth == 0 && !inQuote {
			sb.WriteByte(';')
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}
