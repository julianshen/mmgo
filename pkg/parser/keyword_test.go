package parser

import "testing"

func TestTrimKeyword(t *testing.T) {
	cases := []struct {
		line, kw, want string
	}{
		{"title My Chart", "title", "My Chart"},
		{"title:My Chart", "title", "My Chart"},
		{"title: My Chart", "title", "My Chart"},
		{"title:\tMy Chart", "title", "My Chart"},
		{"x-axis Low --> High", "x-axis", "Low --> High"},
		{"x-axis:Low --> High", "x-axis", "Low --> High"},
		{"quadrant-1:Expand", "quadrant-1", "Expand"},
		{"bar", "bar", ""},
		{"bar:", "bar", ""},
		// Precondition violations: kw not a prefix of line. The
		// helper assumes HasHeaderKeyword was called first; if the
		// caller skipped that step, the returned string is line
		// unchanged (or TrimSpace'd).
		{"foo", "bar", "foo"},
		{"ti", "title", "ti"},
	}
	for _, c := range cases {
		if got := TrimKeyword(c.line, c.kw); got != c.want {
			t.Errorf("TrimKeyword(%q, %q) = %q, want %q", c.line, c.kw, got, c.want)
		}
	}
}
