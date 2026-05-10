package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// suppressedGlyphShapes is the canonical set of small-glyph shapes
// that mermaid renders without a text label (terminator dots, fork
// bars, etc). suppressLabel must agree exactly — adding a shape here
// without updating the helper, or vice versa, breaks visual parity.
var suppressedGlyphShapes = map[diagram.NodeShape]bool{
	diagram.NodeShapeSmallCircle:  true,
	diagram.NodeShapeFilledCircle: true,
	diagram.NodeShapeFramedCircle: true,
	diagram.NodeShapeCrossCircle:  true,
	diagram.NodeShapeForkJoin:     true,
}

// Independent oracle for suppressLabel — the shape-iteration test
// above branches on `suppressLabel(shape)` itself, so a wrongly-true
// helper would silently pass. This explicit map catches that.
func TestSuppressLabelMatchesGlyphSet(t *testing.T) {
	for _, shape := range allFlowchartShapes() {
		got := suppressLabel(shape)
		want := suppressedGlyphShapes[shape]
		if got != want {
			t.Errorf("suppressLabel(%s) = %v, want %v", shape, got, want)
		}
	}
}

// allFlowchartShapes returns the same list TestRenderAllShapes
// iterates. Hoisted into a helper so the oracle test stays in sync.
func allFlowchartShapes() []diagram.NodeShape {
	return []diagram.NodeShape{
		diagram.NodeShapeRectangle,
		diagram.NodeShapeRoundedRectangle,
		diagram.NodeShapeStadium,
		diagram.NodeShapeDiamond,
		diagram.NodeShapeHexagon,
		diagram.NodeShapeCircle,
		diagram.NodeShapeDoubleCircle,
		diagram.NodeShapeParallelogram,
		diagram.NodeShapeParallelogramAlt,
		diagram.NodeShapeTrapezoid,
		diagram.NodeShapeTrapezoidAlt,
		diagram.NodeShapeCylinder,
		diagram.NodeShapeSubroutine,
		diagram.NodeShapeAsymmetric,
		diagram.NodeShapeUnknown,
		diagram.NodeShapeTriangle,
		diagram.NodeShapeFlippedTriangle,
		diagram.NodeShapeHourglass,
		diagram.NodeShapeNotchedPentagon,
		diagram.NodeShapeOdd,
		diagram.NodeShapeFlag,
		diagram.NodeShapeSlopedRect,
		diagram.NodeShapeSmallCircle,
		diagram.NodeShapeFilledCircle,
		diagram.NodeShapeFramedCircle,
		diagram.NodeShapeCrossCircle,
		diagram.NodeShapeDividedRect,
		diagram.NodeShapeWindowPane,
		diagram.NodeShapeLinedRect,
		diagram.NodeShapeForkJoin,
		diagram.NodeShapeNotchedRect,
		diagram.NodeShapeCloud,
		diagram.NodeShapeBang,
		diagram.NodeShapeBolt,
		diagram.NodeShapeDocument,
		diagram.NodeShapeLinedDocument,
		diagram.NodeShapeDelay,
		diagram.NodeShapeHorizontalCylinder,
		diagram.NodeShapeLinedCylinder,
		diagram.NodeShapeCurvedTrapezoid,
		diagram.NodeShapeBowTieRect,
		diagram.NodeShapeTaggedRect,
		diagram.NodeShapeTaggedDocument,
		diagram.NodeShapeStackedRect,
		diagram.NodeShapeStackedDocument,
		diagram.NodeShapeBrace,
		diagram.NodeShapeBraceR,
		diagram.NodeShapeBraces,
		diagram.NodeShapeDataStore,
		diagram.NodeShapeTextBlock,
	}
}

func TestRenderAllShapes(t *testing.T) {
	for _, shape := range allFlowchartShapes() {
		t.Run(shape.String(), func(t *testing.T) {
			n := diagram.Node{ID: "A", Label: "Test", Shape: shape}
			nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
			elems := renderNode(n, nl, 10, DefaultTheme(), 16, nil)
			if suppressLabel(shape) {
				if len(elems) < 1 {
					t.Fatalf("shape %s: expected at least 1 element, got %d", shape, len(elems))
				}
				return
			}
			minElems := 2
			if shape == diagram.NodeShapeTextBlock {
				minElems = 1
			}
			if len(elems) < minElems {
				t.Fatalf("shape %s: expected at least %d elements, got %d", shape, minElems, len(elems))
			}
			txt, ok := elems[len(elems)-1].(*Text)
			if !ok {
				t.Fatalf("shape %s: last element should be *Text, got %T", shape, elems[len(elems)-1])
			}
			if txt.Content != "Test" {
				t.Errorf("shape %s: text = %q, want %q", shape, txt.Content, "Test")
			}
		})
	}
}

func TestRenderRoundedRectHasRX(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "R", Shape: diagram.NodeShapeRoundedRectangle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16, nil)
	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("expected *Rect, got %T", elems[0])
	}
	if rect.RX != 5 {
		t.Errorf("RX = %f, want 5", rect.RX)
	}
}

func TestRenderStadiumHasFullRX(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "S", Shape: diagram.NodeShapeStadium},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16, nil)
	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("expected *Rect, got %T", elems[0])
	}
	if rect.RX != 20 {
		t.Errorf("RX = %f, want 20 (h/2)", rect.RX)
	}
}

func TestRenderDoubleCircleHasTwoCircles(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "D", Shape: diagram.NodeShapeDoubleCircle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16, nil)
	circles := 0
	for _, e := range elems {
		if _, ok := e.(*Circle); ok {
			circles++
		}
	}
	if circles != 2 {
		t.Errorf("expected 2 circles, got %d", circles)
	}
}

func TestRenderSubroutineHasLines(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "Sub", Shape: diagram.NodeShapeSubroutine},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16, nil)
	lines := 0
	for _, e := range elems {
		if _, ok := e.(*Line); ok {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("subroutine should have 2 vertical lines, got %d", lines)
	}
}

func TestRenderMultiLineLabel(t *testing.T) {
	elems := renderNode(
		diagram.Node{ID: "A", Label: "Line1\nLine2\nLine3", Shape: diagram.NodeShapeRectangle},
		layout.NodeLayout{X: 50, Y: 50, Width: 80, Height: 40}, 0, DefaultTheme(), 16, nil)
	texts := 0
	for _, e := range elems {
		if txt, ok := e.(*Text); ok {
			texts++
			if !strings.HasPrefix(txt.Content, "Line") {
				t.Errorf("unexpected text: %q", txt.Content)
			}
		}
	}
	if texts != 3 {
		t.Errorf("expected 3 text elements, got %d", texts)
	}
}
