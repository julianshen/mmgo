package parser

import "strings"

// SplitClickArgs splits a click-action's tail (everything after the
// keyword and node id) into up to `max` whitespace-separated parts,
// respecting double-quoted runs so a tooltip like `"Open the docs"`
// is captured as a single part.
//
// Used by the `click`, `link`, and `callback` keyword parsers across
// flowchart, class, and any future diagram type that needs the same
// argument shape.
func SplitClickArgs(s string, max int) []string {
	var parts []string
	i := 0
	for i < len(s) && len(parts) < max {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' {
			i++
			end := strings.IndexByte(s[i:], '"')
			if end < 0 {
				parts = append(parts, s[i:])
				break
			}
			parts = append(parts, s[i:i+end])
			i = i + end + 1
		} else {
			end := strings.IndexAny(s[i:], " \t")
			if end < 0 {
				parts = append(parts, s[i:])
				break
			}
			parts = append(parts, s[i:i+end])
			i = i + end
		}
	}
	return parts
}
