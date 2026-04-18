package parser

import "testing"

func TestUnquote(t *testing.T) {
	cases := map[string]string{
		`"hello"`:   "hello",
		`'hello'`:   "hello", // single quotes accepted (Mermaid metadata style)
		`hello`:     "hello",
		`"`:         `"`,
		`""`:        ``,
		`''`:        ``,
		``:          ``,
		`"ab`:       `"ab`,
		`ab"`:       `ab"`,
		`"mixed'`:   `"mixed'`, // mismatched quotes left alone
		`'mixed"`:   `'mixed"`,
	}
	for in, want := range cases {
		if got := Unquote(in); got != want {
			t.Errorf("Unquote(%q) = %q, want %q", in, got, want)
		}
	}
}
