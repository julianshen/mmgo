package parser

import (
	"slices"
	"testing"
)

func TestSplitUnquotedCommas(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{`"hello, world",b`, []string{`"hello, world"`, "b"}},
		{`'comma, inside',x`, []string{`'comma, inside'`, "x"}},
		{"a, b, c", []string{"a", " b", " c"}},
		{"a,", []string{"a", ""}},
		{",b", []string{"", "b"}},
		// Mixed quote types coexist.
		{`"a, 'b, c', d"`, []string{`"a, 'b, c', d"`}},
		// Double quote inside single-quoted span stays literal.
		{`'x " y',z`, []string{`'x " y'`, "z"}},
		// Unterminated quote consumes the rest as one token.
		{`a,"unterminated, b`, []string{"a", `"unterminated, b`}},
	}
	for _, c := range cases {
		got := SplitUnquotedCommas(c.in)
		if !slices.Equal(got, c.want) {
			t.Errorf("SplitUnquotedCommas(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
