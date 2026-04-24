package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// polygonPointsAttr formats a vertex slice into the SVG `points`
// attribute string ("x1,y1 x2,y2 ..."). Centralised so every shape
// that exposes a vertex list renders identically.
func polygonPointsAttr(verts []layout.Point) string {
	var sb strings.Builder
	for i, p := range verts {
		if i > 0 {
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "%.2f,%.2f", p.X, p.Y)
	}
	return sb.String()
}

// --- Shape vertex builders ------------------------------------------
//
// Each builder returns the polygon's vertices in winding order. The
// existing *Points string functions are now thin wrappers, so the
// renderer keeps emitting the same SVG and the clipper sees the same
// geometry the renderer drew.

func diamondVerts(cx, cy, w, h float64) []layout.Point {
	return []layout.Point{
		{X: cx, Y: cy - h/2},
		{X: cx + w/2, Y: cy},
		{X: cx, Y: cy + h/2},
		{X: cx - w/2, Y: cy},
	}
}

func hexagonVerts(cx, cy, w, h float64) []layout.Point {
	d := w * polygonSkew
	return []layout.Point{
		{X: cx - w/2 + d, Y: cy - h/2},
		{X: cx + w/2 - d, Y: cy - h/2},
		{X: cx + w/2, Y: cy},
		{X: cx + w/2 - d, Y: cy + h/2},
		{X: cx - w/2 + d, Y: cy + h/2},
		{X: cx - w/2, Y: cy},
	}
}

func parallelogramVerts(cx, cy, w, h, skew float64, reverse bool) []layout.Point {
	s := w * skew
	if reverse {
		return []layout.Point{
			{X: cx - w/2 + s, Y: cy - h/2},
			{X: cx + w/2 + s, Y: cy - h/2},
			{X: cx + w/2 - s, Y: cy + h/2},
			{X: cx - w/2 - s, Y: cy + h/2},
		}
	}
	return []layout.Point{
		{X: cx - w/2 + s, Y: cy - h/2},
		{X: cx + w/2 - s, Y: cy - h/2},
		{X: cx + w/2 + s, Y: cy + h/2},
		{X: cx - w/2 - s, Y: cy + h/2},
	}
}

func trapezoidVerts(cx, cy, w, h, indent float64, alt bool) []layout.Point {
	d := w * indent
	if alt {
		return []layout.Point{
			{X: cx - w/2 + d, Y: cy - h/2},
			{X: cx + w/2 - d, Y: cy - h/2},
			{X: cx + w/2, Y: cy + h/2},
			{X: cx - w/2, Y: cy + h/2},
		}
	}
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx + w/2 - d, Y: cy + h/2},
		{X: cx - w/2 + d, Y: cy + h/2},
	}
}

func asymmetricVerts(cx, cy, w, h float64) []layout.Point {
	s := w * polygonSkew
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2 - s, Y: cy - h/2},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

func triangleVerts(cx, cy, w, h float64) []layout.Point {
	return []layout.Point{
		{X: cx, Y: cy - h/2},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

func flippedTriangleVerts(cx, cy, w, h float64) []layout.Point {
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx, Y: cy + h/2},
	}
}

// hourglassVerts produces a self-intersecting bowtie. The geometry is
// fine for rendering but degenerate for ray-from-center clipping
// (the self-intersection point IS the center), so callers should not
// route hourglass through ClipToPolygonEdge.
func hourglassVerts(cx, cy, w, h float64) []layout.Point {
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx - w/2, Y: cy + h/2},
		{X: cx + w/2, Y: cy + h/2},
	}
}

func notchedPentagonVerts(cx, cy, w, h float64) []layout.Point {
	notch := min(w, h) * 0.25
	return []layout.Point{
		{X: cx - w/2 + notch, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy - h/2 + notch},
	}
}

func notchedRectVerts(cx, cy, w, h float64) []layout.Point {
	notch := min(w, h) * 0.25
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2 - notch, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2 + notch},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

func oddVerts(cx, cy, w, h float64) []layout.Point {
	notch := w * polygonSkew
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx + w/2 - notch, Y: cy},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

func flagVerts(cx, cy, w, h float64) []layout.Point {
	notch := w * polygonSkew
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2},
		{X: cx + w/2 - notch, Y: cy - h/2},
		{X: cx + w/2, Y: cy},
		{X: cx + w/2 - notch, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

func slopedRectVerts(cx, cy, w, h float64) []layout.Point {
	slope := h * 0.25
	return []layout.Point{
		{X: cx - w/2, Y: cy - h/2 + slope},
		{X: cx + w/2, Y: cy - h/2},
		{X: cx + w/2, Y: cy + h/2},
		{X: cx - w/2, Y: cy + h/2},
	}
}

// polygonVerts returns the vertex list for any polygon shape, or nil
// if the shape isn't a polygon (or is one — like hourglass — whose
// center is degenerate for ray clipping).
func polygonVerts(shape diagram.NodeShape, cx, cy, w, h float64) []layout.Point {
	switch shape {
	case diagram.NodeShapeParallelogram:
		return parallelogramVerts(cx, cy, w, h, polygonSkew, false)
	case diagram.NodeShapeParallelogramAlt:
		return parallelogramVerts(cx, cy, w, h, polygonSkew, true)
	case diagram.NodeShapeTrapezoid:
		return trapezoidVerts(cx, cy, w, h, polygonSkew, false)
	case diagram.NodeShapeTrapezoidAlt:
		return trapezoidVerts(cx, cy, w, h, polygonSkew, true)
	case diagram.NodeShapeAsymmetric:
		return asymmetricVerts(cx, cy, w, h)
	case diagram.NodeShapeTriangle:
		return triangleVerts(cx, cy, w, h)
	case diagram.NodeShapeFlippedTriangle:
		return flippedTriangleVerts(cx, cy, w, h)
	case diagram.NodeShapeNotchedPentagon:
		return notchedPentagonVerts(cx, cy, w, h)
	case diagram.NodeShapeNotchedRect:
		return notchedRectVerts(cx, cy, w, h)
	case diagram.NodeShapeOdd:
		return oddVerts(cx, cy, w, h)
	case diagram.NodeShapeFlag:
		return flagVerts(cx, cy, w, h)
	case diagram.NodeShapeSlopedRect:
		return slopedRectVerts(cx, cy, w, h)
	default:
		return nil
	}
}
