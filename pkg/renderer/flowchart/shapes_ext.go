// Extended shape geometry helpers for Mermaid's @{shape:...} syntax.
//
// Each helper returns SVG element(s) positioned at an absolute
// (cx, cy) center with overall bounding size (w, h). The box is the
// layout-assigned node rect; the shape draws INSIDE it, so text
// centered at (cx, cy) always lands inside the visible glyph.
package flowchart

// ---------- Simple polygons -----------------------------------------
//
// Each helper is a thin wrapper around its corresponding *Verts
// builder in polygon.go so the renderer and the edge clipper see the
// same vertex list. See polygon.go for vertex orderings and the
// hourglass-is-a-bowtie note.

func trianglePoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(triangleVerts(cx, cy, w, h))
}

func flippedTrianglePoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(flippedTriangleVerts(cx, cy, w, h))
}

func hourglassPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(hourglassVerts(cx, cy, w, h))
}

func notchedPentagonPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(notchedPentagonVerts(cx, cy, w, h))
}

func notchedRectPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(notchedRectVerts(cx, cy, w, h))
}

func oddPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(oddVerts(cx, cy, w, h))
}

func flagPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(flagVerts(cx, cy, w, h))
}

func slopedRectPoints(cx, cy, w, h float64) string {
	return polygonPointsAttr(slopedRectVerts(cx, cy, w, h))
}

