package parser

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// ParseDirection maps a Mermaid direction keyword to diagram.Direction.
// Empty input and "TD" are aliases for "TB", matching mermaid-cli.
// Trailing whitespace-separated tokens are rejected so a misspelled
// direction (e.g. "LR x") doesn't silently fall through.
func ParseDirection(s string) (diagram.Direction, error) {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return diagram.DirectionUnknown, fmt.Errorf("extra tokens after direction %q", s)
	}
	switch s {
	case "", "TB", "TD":
		return diagram.DirectionTB, nil
	case "BT":
		return diagram.DirectionBT, nil
	case "LR":
		return diagram.DirectionLR, nil
	case "RL":
		return diagram.DirectionRL, nil
	default:
		return diagram.DirectionUnknown, fmt.Errorf("unknown direction %q", s)
	}
}
