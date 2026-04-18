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
	}
	for _, c := range cases {
		if got := TrimKeyword(c.line, c.kw); got != c.want {
			t.Errorf("TrimKeyword(%q, %q) = %q, want %q", c.line, c.kw, got, c.want)
		}
	}
}
