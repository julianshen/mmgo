package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// shiftInward must NOT overshoot q when the segment is shorter than
// the requested shift — otherwise the returned point lies past q and
// the direction vector used by startMarkerElems flips, pointing the
// tail marker into the line instead of outward toward the source.
func TestShiftInward_ShortSegmentReturnsPUnchanged(t *testing.T) {
	p := layout.Point{X: 10, Y: 10}
	q := layout.Point{X: 13, Y: 10} // 3px segment, shift dist = 9
	got := shiftInward(p, q, 9)
	if got != p {
		t.Errorf("shiftInward over short segment overshot: got %+v, want %+v (unchanged)", got, p)
	}
}

func TestShiftInward_ZeroLengthSegmentReturnsP(t *testing.T) {
	p := layout.Point{X: 5, Y: 5}
	got := shiftInward(p, p, 9)
	if got != p {
		t.Errorf("zero-length segment: got %+v, want %+v", got, p)
	}
}

func TestShiftInward_LongSegmentMovesByExactlyDist(t *testing.T) {
	p := layout.Point{X: 0, Y: 0}
	q := layout.Point{X: 100, Y: 0}
	got := shiftInward(p, q, 9)
	want := layout.Point{X: 9, Y: 0}
	if got != want {
		t.Errorf("long segment shift: got %+v, want %+v", got, want)
	}
}

// startMarkerElems must early-return for "no arrow" head values so
// edges with neither tail nor head don't accidentally emit invisible
// SVG nodes.
func TestStartMarkerElems_InvisibleArrowReturnsNil(t *testing.T) {
	for _, ah := range []diagram.ArrowHead{diagram.ArrowHeadNone, diagram.ArrowHeadUnknown} {
		got := startMarkerElems(ah, layout.Point{}, layout.Point{X: 10, Y: 0}, DefaultTheme())
		if got != nil {
			t.Errorf("head=%v should produce no elements, got %v", ah, got)
		}
	}
}

// startMarkerElems must early-return for a degenerate zero-length
// segment — there's no direction to orient the marker against.
func TestStartMarkerElems_ZeroLengthReturnsNil(t *testing.T) {
	p := layout.Point{X: 5, Y: 5}
	got := startMarkerElems(diagram.ArrowHeadArrow, p, p, DefaultTheme())
	if got != nil {
		t.Errorf("zero-length segment should produce no elements, got %v", got)
	}
}

// For each visible arrow shape, startMarkerElems emits exactly one
// element wrapping the same children buildMarker uses for marker-end —
// guaranteeing the head and tail of a bidirectional edge look identical.
func TestStartMarkerElems_ReusesBuildMarkerChildren(t *testing.T) {
	cases := []struct {
		name string
		head diagram.ArrowHead
	}{
		{"arrow", diagram.ArrowHeadArrow},
		{"circle", diagram.ArrowHeadCircle},
		{"cross", diagram.ArrowHeadCross},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			elems := startMarkerElems(tc.head, layout.Point{X: 50, Y: 50}, layout.Point{X: 100, Y: 50}, DefaultTheme())
			if len(elems) != 1 {
				t.Fatalf("expected 1 wrapper element, got %d", len(elems))
			}
			g, ok := elems[0].(*Group)
			if !ok {
				t.Fatalf("wrapper is %T, want *Group", elems[0])
			}
			if g.Transform == "" {
				t.Error("inline marker group must carry a transform that anchors it at the line start")
			}
			wantChildren := buildMarker("_test", tc.head, DefaultTheme()).Children
			if len(g.Children) != len(wantChildren) {
				t.Errorf("inline children count = %d, marker-end children = %d (must match for visual parity)",
					len(g.Children), len(wantChildren))
			}
		})
	}
}

