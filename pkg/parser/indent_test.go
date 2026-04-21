package parser

import "testing"

func TestIndentWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"foo", 0},
		{"  foo", 2},
		{"\tfoo", TabWidth},
		{"\t\tfoo", 2 * TabWidth},
		{"  \tfoo", 2 + TabWidth},
		{"\t  foo", TabWidth + 2},
		// The equivalence the shared helper is built to provide:
		// one tab and TabWidth spaces produce the same depth.
		{"    foo", TabWidth},
		// Content after the indent doesn't count.
		{"  foo  bar", 2},
		// All-whitespace line counts the whitespace.
		{"   ", 3},
		// Anything other than space/tab terminates the count — CR
		// from CRLF-stripped input and NBSP must not sneak in.
		{"\rfoo", 0},
		{"\u00a0foo", 0},
	}
	for _, c := range cases {
		if got := IndentWidth(c.in); got != c.want {
			t.Errorf("IndentWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
