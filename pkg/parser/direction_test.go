package parser

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseDirectionValid(t *testing.T) {
	cases := []struct {
		in   string
		want diagram.Direction
	}{
		{"", diagram.DirectionTB},
		{"TB", diagram.DirectionTB},
		{"TD", diagram.DirectionTB}, // alias
		{"BT", diagram.DirectionBT},
		{"LR", diagram.DirectionLR},
		{"RL", diagram.DirectionRL},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseDirection(tc.in)
			if err != nil {
				t.Fatalf("ParseDirection(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParseDirection(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseDirectionInvalid(t *testing.T) {
	for _, in := range []string{
		"WAT",          // unknown
		"LR x",         // extra tokens
		"TB extra",     // extra tokens after valid prefix
		"\tLR",         // leading whitespace counts as extra token
		"lower",        // lowercase isn't an alias
	} {
		t.Run(in, func(t *testing.T) {
			_, err := ParseDirection(in)
			if err == nil {
				t.Errorf("ParseDirection(%q) returned no error", in)
			}
		})
	}
}
