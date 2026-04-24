package flowchart

import (
	"math"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout"
)

// branchLabelPos must put the label on the OUTWARD side of the node
// center — i.e. farther from (cx, cy) than the midpoint of port→stem.
func TestBranchLabelPos_OffsetIsOutward(t *testing.T) {
	// Diamond center at (100, 50) with left/right/bottom exit ports.
	cx, cy := 100.0, 50.0
	const rankStep = 40.0
	const fontSize = 14.0

	cases := []struct {
		name     string
		port     layout.Point
		stem     layout.Point
		wantSide string // "left", "right", "center"
	}{
		{"left vertex TB", layout.Point{X: 70, Y: 50}, layout.Point{X: 70, Y: 50 + rankStep}, "left"},
		{"right vertex TB", layout.Point{X: 130, Y: 50}, layout.Point{X: 130, Y: 50 + rankStep}, "right"},
		{"left vertex LR", layout.Point{X: 100, Y: 20}, layout.Point{X: 100 + rankStep, Y: 20}, "above"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lx, ly := branchLabelPos(tc.port, tc.stem, cx, cy, fontSize)
			// Midpoint of port→stem.
			mx := (tc.port.X + tc.stem.X) * 0.5
			my := (tc.port.Y + tc.stem.Y) * 0.5
			// Distance from center.
			dMid := math.Hypot(mx-cx, my-cy)
			dLabel := math.Hypot(lx-cx, ly-cy)
			if dLabel <= dMid {
				t.Errorf("label at (%.2f, %.2f) dist=%.2f should be > midpoint dist=%.2f (outward invariant broken)",
					lx, ly, dLabel, dMid)
			}
		})
	}
}

// When port == stem (degenerate zero-length segment), branchLabelPos
// must not divide by zero — it returns the sample point unchanged.
func TestBranchLabelPos_ZeroLengthSegment(t *testing.T) {
	port := layout.Point{X: 10, Y: 20}
	lx, ly := branchLabelPos(port, port, 0, 0, 14)
	if lx != port.X || ly != port.Y {
		t.Errorf("zero-length segment: got (%v, %v), want (%v, %v)", lx, ly, port.X, port.Y)
	}
}

// The offset magnitude from the first-segment sample point is
// exactly fontSize/2 + 4 — the documented breathing room.
func TestBranchLabelPos_OffsetMagnitude(t *testing.T) {
	const fontSize = 16.0
	port := layout.Point{X: 0, Y: 0}
	stem := layout.Point{X: 0, Y: 10}
	// Node center left of the segment → label goes right (+x).
	lx, ly := branchLabelPos(port, stem, -100, 5, fontSize)

	// Sample is at 40% along port→stem.
	const frac = 0.4
	sx := port.X + frac*(stem.X-port.X)
	sy := port.Y + frac*(stem.Y-port.Y)
	got := math.Hypot(lx-sx, ly-sy)
	want := fontSize/2 + 4
	if math.Abs(got-want) > 0.001 {
		t.Errorf("offset magnitude = %.3f, want %.3f", got, want)
	}
}
