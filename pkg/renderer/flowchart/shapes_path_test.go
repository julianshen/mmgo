package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// Stage 3 shapes that render as a single *Path. Cover each family's
// distinguishing feature: cloud/bang/bolt have many curve or line
// commands; delay/curved-trapezoid use arcs; etc.
func TestRenderSinglePathShapes(t *testing.T) {
	cases := []diagram.NodeShape{
		diagram.NodeShapeCloud,
		diagram.NodeShapeBang,
		diagram.NodeShapeBolt,
		diagram.NodeShapeDocument,
		diagram.NodeShapeDelay,
		diagram.NodeShapeHorizontalCylinder,
		diagram.NodeShapeCurvedTrapezoid,
		diagram.NodeShapeBowTieRect,
		diagram.NodeShapeTaggedRect,
		diagram.NodeShapeDataStore,
		diagram.NodeShapeBrace,
		diagram.NodeShapeBraceR,
		diagram.NodeShapeBraces,
	}
	for _, shape := range cases {
		t.Run(shape.String(), func(t *testing.T) {
			elems := renderShape(t, shape)
			if got := countType[*Path](elems); got != 1 {
				t.Errorf("shape %s: want 1 *Path, got %d", shape, got)
			}
		})
	}
}

// Braces (both `{}`) produces one path with two subpaths joined in a
// single `d` attribute — there should still be exactly one *Path
// element, not two.
func TestRenderBracesIsSingleElement(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeBraces)
	if got := countType[*Path](elems); got != 1 {
		t.Errorf("braces: want 1 *Path holding both braces, got %d", got)
	}
}

// Shapes that are a path plus one adornment (lined document = path +
// line; lined cylinder = path + line; tagged document = 2 paths;
// stacked = 2 paths/rects).
func TestRenderLinedDocumentHasRule(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeLinedDocument)
	if got := countType[*Path](elems); got != 1 {
		t.Errorf("lined-document: want 1 *Path, got %d", got)
	}
	if got := countType[*Line](elems); got != 1 {
		t.Errorf("lined-document: want 1 *Line (the rule), got %d", got)
	}
}

func TestRenderLinedCylinderHasStripe(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeLinedCylinder)
	if got := countType[*Path](elems); got != 1 {
		t.Errorf("lined-cylinder: want 1 *Path, got %d", got)
	}
	if got := countType[*Line](elems); got != 1 {
		t.Errorf("lined-cylinder: want 1 *Line (stripe), got %d", got)
	}
}

func TestRenderTaggedDocumentHasTwoPaths(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeTaggedDocument)
	if got := countType[*Path](elems); got != 2 {
		t.Errorf("tagged-document: want 2 *Path (doc + tag), got %d", got)
	}
}

func TestRenderStackedRectHasTwoRects(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeStackedRect)
	if got := countType[*Rect](elems); got != 2 {
		t.Errorf("stacked-rect: want 2 *Rect (offset stack), got %d", got)
	}
}

func TestRenderStackedDocumentHasTwoPaths(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeStackedDocument)
	if got := countType[*Path](elems); got != 2 {
		t.Errorf("stacked-document: want 2 *Path (offset stack), got %d", got)
	}
}

// TextBlock has no border or fill — just the label text.
func TestRenderTextBlockHasNoBorder(t *testing.T) {
	elems := renderShape(t, diagram.NodeShapeTextBlock)
	if got := countType[*Rect](elems); got != 0 {
		t.Errorf("text-block: want 0 *Rect, got %d", got)
	}
	if got := countType[*Path](elems); got != 0 {
		t.Errorf("text-block: want 0 *Path, got %d", got)
	}
	if got := countType[*Circle](elems); got != 0 {
		t.Errorf("text-block: want 0 *Circle, got %d", got)
	}
	// Still emits the text label.
	if got := countType[*Text](elems); got < 1 {
		t.Errorf("text-block: want ≥1 *Text, got %d", got)
	}
}

// Brace paths use fill:none (they're strokes, not filled glyphs).
func TestRenderBracesAreStroked(t *testing.T) {
	for _, shape := range []diagram.NodeShape{
		diagram.NodeShapeBrace,
		diagram.NodeShapeBraceR,
		diagram.NodeShapeBraces,
	} {
		t.Run(shape.String(), func(t *testing.T) {
			elems := renderShape(t, shape)
			for _, e := range elems {
				p, ok := e.(*Path)
				if !ok {
					continue
				}
				if !strings.Contains(p.Style, "fill:none") {
					t.Errorf("shape %s: brace path should have fill:none, got %q", shape, p.Style)
				}
			}
		})
	}
}
