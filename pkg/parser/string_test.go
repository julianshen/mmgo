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

func TestExtractBracketLabel(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantHead  string
		wantLabel string
		wantErr   bool
	}{
		{"no-brackets", "Foo", "Foo", "", false},
		{"double-quote", `Foo["Display Name"]`, "Foo", "Display Name", false},
		{"single-quote", `Foo['Display']`, "Foo", "Display", false},
		{"head-trimmed", `  Foo  ["X"]`, "Foo", "X", false},
		{"unclosed-bracket", `Foo["x`, "", "", true},
		{"closing-before-opening", `Foo]"x"[`, "", "", true},
		{"unquoted-contents", `Foo[bar]`, "", "", true},
		{"trailing-junk", `Foo["x"]junk`, "", "", true},
		{"trailing-whitespace-allowed", `Foo["x"]   `, "Foo", "x", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			head, label, err := ExtractBracketLabel(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got head=%q label=%q", head, label)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if head != tc.wantHead || label != tc.wantLabel {
				t.Errorf("got (%q, %q), want (%q, %q)", head, label, tc.wantHead, tc.wantLabel)
			}
		})
	}
}
