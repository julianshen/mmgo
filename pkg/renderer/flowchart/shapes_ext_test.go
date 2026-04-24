package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// renderShape helps keep the per-shape assertions compact.
func renderShape(t *testing.T, shape diagram.NodeShape) []any {
	t.Helper()
	n := diagram.Node{ID: "A", Label: "Test", Shape: shape}
	nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
	return renderNode(n, nl, 10, DefaultTheme(), 16)
}

// countType returns how many elems are of the given concrete type.
func countType[T any](elems []any) int {
	n := 0
	for _, e := range elems {
		if _, ok := e.(T); ok {
			n++
		}
	}
	return n
}

// Simple polygons: one *Polygon + one *Text = 2 elements total.
func TestRenderPolygonShapes(t *testing.T) {
	cases := []diagram.NodeShape{
		diagram.NodeShapeTriangle,
		diagram.NodeShapeFlippedTriangle,
		diagram.NodeShapeHourglass,
		diagram.NodeShapeNotchedPentagon,
		diagram.NodeShapeOdd,
		diagram.NodeShapeFlag,
		diagram.NodeShapeSlopedRect,
		diagram.NodeShapeNotchedRect, // also a polygon (rect with a corner cut)
	}
	for _, shape := range cases {
		t.Run(shape.String(), func(t *testing.T) {
			elems := renderShape(t, shape)
			if got := countType[*Polygon](elems); got != 1 {
				t.Errorf("shape %s: want 1 *Polygon, got %d (%v)", shape, got, elems)
			}
			// No *Rect should sneak in for a polygon shape.
			if got := countType[*Rect](elems); got != 0 {
				t.Errorf("shape %s: want 0 *Rect, got %d", shape, got)
			}
		})
	}
}

// SmallCircle / FilledCircle emit one *Circle each.
func TestRenderSmallAndFilledCircle(t *testing.T) {
	for _, shape := range []diagram.NodeShape{
		diagram.NodeShapeSmallCircle,
		diagram.NodeShapeFilledCircle,
	} {
		t.Run(shape.String(), func(t *testing.T) {
			elems := renderShape(t, shape)
			if got := countType[*Circle](elems); got != 1 {
				t.Errorf("shape %s: want 1 *Circle, got %d", shape, got)
			}
		})
	}
}

// FramedCircle emits two *Circle (outer ring + inner dot).
func TestRenderFramedCircle(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeFramedCircle)
	if got := countType[*Circle](elems); got != 2 {
		t.Errorf("framed-circle: want 2 *Circle (ring + dot), got %d", got)
	}
}

// CrossCircle emits one *Circle + two *Line (the X).
func TestRenderCrossCircle(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeCrossCircle)
	if got := countType[*Circle](elems); got != 1 {
		t.Errorf("cross-circle: want 1 *Circle, got %d", got)
	}
	if got := countType[*Line](elems); got != 2 {
		t.Errorf("cross-circle: want 2 *Line (the cross), got %d", got)
	}
}

// DividedRect = *Rect + 1 *Line (horizontal divider).
func TestRenderDividedRect(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeDividedRect)
	if got := countType[*Rect](elems); got != 1 {
		t.Errorf("divided-rect: want 1 *Rect, got %d", got)
	}
	if got := countType[*Line](elems); got != 1 {
		t.Errorf("divided-rect: want 1 *Line, got %d", got)
	}
}

// WindowPane = *Rect + 2 *Line (cross).
func TestRenderWindowPane(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeWindowPane)
	if got := countType[*Rect](elems); got != 1 {
		t.Errorf("window-pane: want 1 *Rect, got %d", got)
	}
	if got := countType[*Line](elems); got != 2 {
		t.Errorf("window-pane: want 2 *Line (cross), got %d", got)
	}
}

// LinedRect = *Rect + 1 *Line (sidebar).
func TestRenderLinedRect(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeLinedRect)
	if got := countType[*Rect](elems); got != 1 {
		t.Errorf("lined-rect: want 1 *Rect, got %d", got)
	}
	if got := countType[*Line](elems); got != 1 {
		t.Errorf("lined-rect: want 1 *Line, got %d", got)
	}
}

// ForkJoin = a single filled *Rect bar (no stroke, no lines).
func TestRenderForkJoin(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeForkJoin)
	if got := countType[*Rect](elems); got != 1 {
		t.Errorf("fork-join: want 1 *Rect, got %d", got)
	}
	if got := countType[*Line](elems); got != 0 {
		t.Errorf("fork-join: want 0 *Line (it's a solid bar), got %d", got)
	}
}
