package parser

import (
	"fmt"
	"strings"
)

// ParseClassDefLine parses `classDef NAME css-decls` and returns the
// name and the CSS with commas normalized to semicolons. Used by
// every diagram type that supports the classDef keyword. The caller
// supplies the rest of the line WITHOUT the `classDef ` prefix.
func ParseClassDefLine(rest string) (name, css string, err error) {
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("classDef requires a name and CSS")
	}
	return parts[0], NormalizeCSS(strings.TrimSpace(parts[1])), nil
}

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
