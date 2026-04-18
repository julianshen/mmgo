package parser

import "testing"

func TestUnquote(t *testing.T) {
	cases := map[string]string{
		`"hello"`: "hello",
		`hello`:   "hello",
		`"`:       `"`,
		`""`:      ``,
		``:        ``,
		`"ab`:     `"ab`,
		`ab"`:     `ab"`,
	}
	for in, want := range cases {
		if got := Unquote(in); got != want {
			t.Errorf("Unquote(%q) = %q, want %q", in, got, want)
		}
	}
}
