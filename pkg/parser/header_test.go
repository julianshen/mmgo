package parser

import "testing"

func TestHasHeaderKeyword(t *testing.T) {
	cases := []struct {
		line string
		kw   string
		want bool
	}{
		{"graph", "graph", true},
		{"graph LR", "graph", true},
		{"graph\tTB", "graph", true},
		{"graph:", "graph", true},
		{"graph: LR", "graph", true},
		{"graphA", "graph", false},
		{"grapha LR", "graph", false},
		{"", "graph", false},
		{"graph", "flowchart", false},
		{"sankey-beta", "sankey-beta", true},
		{"sankey-beta: foo", "sankey-beta", true},
		// Edge cases:
		// - kw longer than line falls out of HasPrefix cleanly.
		// - A multi-byte rune following the keyword is not a valid
		//   boundary (its leading byte isn't space/tab/colon), so
		//   non-ASCII identifiers are rejected — matches the intent.
		// - A CR or LF right after the keyword is also not a boundary;
		//   Mermaid trims lines upstream so this only matters if the
		//   caller forgets to trim.
		{"gr", "graph", false},
		{"graph中", "graph", false},
		{"graph\n", "graph", false},
	}
	for _, c := range cases {
		if got := HasHeaderKeyword(c.line, c.kw); got != c.want {
			t.Errorf("HasHeaderKeyword(%q, %q) = %v, want %v", c.line, c.kw, got, c.want)
		}
	}
}
