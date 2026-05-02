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

// MatchKeywordValue returns (value, true) when line begins with kw
// followed by a `:`, whitespace, or end-of-line; otherwise (zero,
// false). Avoids matching prefixes — `accTitleFoo` does not match
// `accTitle` because the byte after the keyword is an identifier
// character. The returned value has any leading `:` and surrounding
// whitespace stripped.
//
// Use this for body-keyword statements like `title: …`, `accTitle …`,
// `accDescr: …` where the keyword takes a free-form value.
func MatchKeywordValue(line, kw string) (string, bool) {
	if !strings.HasPrefix(line, kw) {
		return "", false
	}
	rest := line[len(kw):]
	if rest != "" && rest[0] != ':' && rest[0] != ' ' && rest[0] != '\t' {
		return "", false
	}
	return TrimKeyword(line, kw), true
}
