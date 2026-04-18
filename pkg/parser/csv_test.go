package parser

import (
	"strings"
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
	}
	for _, c := range cases {
		got := SplitUnquotedCommas(c.in)
		if !slicesEq(got, c.want) {
			t.Errorf("SplitUnquotedCommas(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func slicesEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Compare(a[i], b[i]) != 0 {
			return false
		}
	}
	return true
}
