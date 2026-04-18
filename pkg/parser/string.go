package parser

import "strings"

// Unquote strips a single pair of matching surrounding double quotes,
// after first trimming whitespace. Strings without surrounding quotes
// are returned as-is (but still whitespace-trimmed).
func Unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
