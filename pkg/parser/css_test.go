package parser

import "testing"

func TestParseClassDefLine(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantCSS  string
	}{
		{"foo fill:#f00", "foo", "fill:#f00"},
		{"important fill:#f96,stroke:#333", "important", "fill:#f96;stroke:#333"},
		// Embedded function call (rgb(...)) keeps its commas.
		{"box fill:rgb(255, 0, 0),stroke:#000", "box", "fill:rgb(255, 0, 0);stroke:#000"},
	}
	for _, tc := range cases {
		t.Run(tc.wantName, func(t *testing.T) {
			name, css, err := ParseClassDefLine(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName || css != tc.wantCSS {
				t.Errorf("got (%q, %q), want (%q, %q)", name, css, tc.wantName, tc.wantCSS)
			}
		})
	}
}

func TestParseClassDefLineMalformed(t *testing.T) {
	for _, in := range []string{
		"",
		"onlyname",
	} {
		t.Run(in, func(t *testing.T) {
			if _, _, err := ParseClassDefLine(in); err == nil {
				t.Errorf("expected error for %q", in)
			}
		})
	}
}

func TestNormalizeCSSPreservesParens(t *testing.T) {
	got := NormalizeCSS("fill:rgb(255, 0, 0),stroke:#000")
	want := "fill:rgb(255, 0, 0);stroke:#000"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNormalizeCSSNoComma(t *testing.T) {
	in := "fill:#abc"
	if got := NormalizeCSS(in); got != in {
		t.Errorf("expected pass-through, got %q", got)
	}
}

func TestExtractCSSClassShorthand(t *testing.T) {
	cases := []struct {
		in       string
		wantID   string
		wantCSS  string
		wantOK   bool
	}{
		{"Foo", "Foo", "", true},
		{"Foo:::hot", "Foo", "hot", true},
		// Chained `:::` is rejected; ID/CSS values returned in this
		// case are unspecified — callers only inspect ok.
		{"Foo:::a:::b", "Foo:::a:::b", "", false},
		{":::hot", "", "hot", true},        // empty id is the caller's problem
		{"", "", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			gotID, gotCSS, gotOK := ExtractCSSClassShorthand(tc.in)
			if gotID != tc.wantID || gotCSS != tc.wantCSS || gotOK != tc.wantOK {
				t.Errorf("got (%q, %q, %v), want (%q, %q, %v)",
					gotID, gotCSS, gotOK, tc.wantID, tc.wantCSS, tc.wantOK)
			}
		})
	}
}

func TestExpandLineBreaks(t *testing.T) {
	got := ExpandLineBreaks(`a\nb\nc`)
	want := "a\nb\nc"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
