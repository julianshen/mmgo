// Extended shape geometry helpers for Mermaid's @{shape:...} syntax.
//
// Each helper returns SVG element(s) positioned at an absolute
// (cx, cy) center with overall bounding size (w, h). The box is the
// layout-assigned node rect; the shape draws INSIDE it, so text
// centered at (cx, cy) always lands inside the visible glyph.
package flowchart

import (
	"fmt"
)

// ---------- Simple polygons -----------------------------------------

// trianglePoints: apex at top, base at bottom.
func trianglePoints(cx, cy, w, h float64) string {
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx, cy-h/2,
		cx+w/2, cy+h/2,
		cx-w/2, cy+h/2)
}

// flippedTrianglePoints: base at top, apex at bottom.
func flippedTrianglePoints(cx, cy, w, h float64) string {
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2,
		cx+w/2, cy-h/2,
		cx, cy+h/2)
}

// hourglassPoints: bowtie of four corners. SVG polygon auto-closes
// in vertex order, so going top-left → top-right → bottom-left →
// bottom-right produces the bowtie (not a rectangle).
func hourglassPoints(cx, cy, w, h float64) string {
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2,
		cx+w/2, cy-h/2,
		cx-w/2, cy+h/2,
		cx+w/2, cy+h/2)
}

// notchedPentagonPoints: rectangle with the top-left corner cut at
// 45° into the body. The notch size is a fraction of the shorter
// dimension so the shape still reads as a pentagon at all sizes.
func notchedPentagonPoints(cx, cy, w, h float64) string {
	notch := min(w, h) * 0.25
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2+notch, cy-h/2,
		cx+w/2, cy-h/2,
		cx+w/2, cy+h/2,
		cx-w/2, cy+h/2,
		cx-w/2, cy-h/2+notch)
}

// notchedRectPoints: rectangle with the top-right corner cut off at
// 45° — the classic index-card / `card` glyph.
func notchedRectPoints(cx, cy, w, h float64) string {
	notch := min(w, h) * 0.25
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2,
		cx+w/2-notch, cy-h/2,
		cx+w/2, cy-h/2+notch,
		cx+w/2, cy+h/2,
		cx-w/2, cy+h/2)
}

// oddPoints: rectangle with a right-side notch that makes the shape
// irregular — visually distinct from asymmetric's left-notch pentagon
// while still fitting a 5-vertex polygon budget.
func oddPoints(cx, cy, w, h float64) string {
	notch := w * polygonSkew
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2,
		cx+w/2, cy-h/2,
		cx+w/2-notch, cy,
		cx+w/2, cy+h/2,
		cx-w/2, cy+h/2)
}

// flagPoints: pennant — rectangle with a triangular point cut on the
// right, like a flag on a pole. Used for `flag` / `paper-tape`.
func flagPoints(cx, cy, w, h float64) string {
	notch := w * polygonSkew
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2,
		cx+w/2-notch, cy-h/2,
		cx+w/2, cy,
		cx+w/2-notch, cy+h/2,
		cx-w/2, cy+h/2)
}

// slopedRectPoints: quadrilateral with the top-left corner lowered
// to produce a sloping top edge — Mermaid's `manual-input` glyph.
func slopedRectPoints(cx, cy, w, h float64) string {
	slope := h * 0.25
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2+slope,
		cx+w/2, cy-h/2,
		cx+w/2, cy+h/2,
		cx-w/2, cy+h/2)
}

