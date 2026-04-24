package flowchart

import (
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

const (
	polygonSkew        = 0.15
	defaultStrokeWidth = 1.5
	doubleCircleGap    = 3.0
	subroutineBand     = 0.1
)

func renderNode(n diagram.Node, nl layout.NodeLayout, pad float64, th Theme, fontSize float64) []any {
	shapeStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.NodeFill, th.NodeStroke, defaultStrokeWidth)
	textStyle := fmt.Sprintf("fill:%s;font-size:%.2fpx", th.NodeText, fontSize)

	cx := nl.X + pad
	cy := nl.Y + pad
	w := nl.Width
	h := nl.Height

	var elems []any

	switch n.Shape {
	case diagram.NodeShapeRectangle:
		// RX/RY 2 matches mermaid-cli's default rectangle (a slight
		// rounding, distinct from NodeShapeRoundedRectangle below).
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), RX: 2, RY: 2, Style: shapeStyle,
		})
	case diagram.NodeShapeRoundedRectangle:
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), RX: 5, RY: 5, Style: shapeStyle,
		})
	case diagram.NodeShapeStadium:
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), RX: svgFloat(h / 2), RY: svgFloat(h / 2), Style: shapeStyle,
		})
	case diagram.NodeShapeCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r), Style: shapeStyle})
	case diagram.NodeShapeDoubleCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r), Style: shapeStyle})
		elems = append(elems, &Circle{
			CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r + doubleCircleGap),
			Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth),
		})
	case diagram.NodeShapeDiamond:
		elems = append(elems, &Polygon{Points: diamondPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeHexagon:
		elems = append(elems, &Polygon{Points: hexagonPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeParallelogram:
		elems = append(elems, &Polygon{Points: parallelogramPoints(cx, cy, w, h, polygonSkew, false), Style: shapeStyle})
	case diagram.NodeShapeParallelogramAlt:
		elems = append(elems, &Polygon{Points: parallelogramPoints(cx, cy, w, h, polygonSkew, true), Style: shapeStyle})
	case diagram.NodeShapeTrapezoid:
		elems = append(elems, &Polygon{Points: trapezoidPoints(cx, cy, w, h, polygonSkew, false), Style: shapeStyle})
	case diagram.NodeShapeTrapezoidAlt:
		elems = append(elems, &Polygon{Points: trapezoidPoints(cx, cy, w, h, polygonSkew, true), Style: shapeStyle})
	case diagram.NodeShapeAsymmetric:
		elems = append(elems, &Polygon{Points: asymmetricPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeCylinder:
		elems = append(elems, &Path{D: svgutil.CylinderPath(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeSubroutine:
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), Style: shapeStyle,
		})
		bandX1 := cx - w/2 + w*subroutineBand
		bandX2 := cx + w/2 - w*subroutineBand
		lineStyle := fmt.Sprintf("stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Line{X1: svgFloat(bandX1), Y1: svgFloat(cy - h/2), X2: svgFloat(bandX1), Y2: svgFloat(cy + h/2), Style: lineStyle})
		elems = append(elems, &Line{X1: svgFloat(bandX2), Y1: svgFloat(cy - h/2), X2: svgFloat(bandX2), Y2: svgFloat(cy + h/2), Style: lineStyle})

	// --- Stage 2: simple polygons -------------------------------------
	case diagram.NodeShapeTriangle:
		elems = append(elems, &Polygon{Points: trianglePoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeFlippedTriangle:
		elems = append(elems, &Polygon{Points: flippedTrianglePoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeHourglass:
		elems = append(elems, &Polygon{Points: hourglassPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeNotchedPentagon:
		elems = append(elems, &Polygon{Points: notchedPentagonPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeOdd:
		elems = append(elems, &Polygon{Points: oddPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeFlag:
		elems = append(elems, &Polygon{Points: flagPoints(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeSlopedRect:
		elems = append(elems, &Polygon{Points: slopedRectPoints(cx, cy, w, h), Style: shapeStyle})

	// --- Stage 2: circle variants -------------------------------------
	case diagram.NodeShapeSmallCircle:
		// `start` glyph. Fills the (small) layout box so edges
		// terminate at the visible circle edge.
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r), Style: shapeStyle})
	case diagram.NodeShapeFilledCircle:
		// Junction dot: filled with the stroke color so it reads as
		// a solid bullet.
		r := math.Min(w, h) / 2
		filled := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.NodeStroke, th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r), Style: filled})
	case diagram.NodeShapeFramedCircle:
		// `stop` glyph: outer ring with a smaller filled inner dot —
		// same layering as the state renderer's end marker.
		outerR := math.Min(w, h) / 2
		innerR := outerR * 0.4
		ring := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.NodeFill, th.NodeStroke, defaultStrokeWidth)
		dot := fmt.Sprintf("fill:%s;stroke:none", th.NodeStroke)
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(outerR), Style: ring})
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(innerR), Style: dot})
	case diagram.NodeShapeCrossCircle:
		// Summary glyph: circle with an X drawn from corner to corner.
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r), Style: shapeStyle})
		d := r / math.Sqrt2 // project r onto the 45° diagonal
		crossStyle := fmt.Sprintf("stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Line{X1: svgFloat(cx - d), Y1: svgFloat(cy - d), X2: svgFloat(cx + d), Y2: svgFloat(cy + d), Style: crossStyle})
		elems = append(elems, &Line{X1: svgFloat(cx - d), Y1: svgFloat(cy + d), X2: svgFloat(cx + d), Y2: svgFloat(cy - d), Style: crossStyle})

	// --- Stage 2: modified rectangles ---------------------------------
	case diagram.NodeShapeDividedRect:
		// Rect with a horizontal divider splitting the top and bottom
		// halves — the `divided-process` glyph.
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), Style: shapeStyle,
		})
		div := fmt.Sprintf("stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Line{X1: svgFloat(cx - w/2), Y1: svgFloat(cy), X2: svgFloat(cx + w/2), Y2: svgFloat(cy), Style: div})
	case diagram.NodeShapeWindowPane:
		// Rect split into four panes by a cross — `internal-storage`.
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), Style: shapeStyle,
		})
		div := fmt.Sprintf("stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Line{X1: svgFloat(cx - w/2), Y1: svgFloat(cy), X2: svgFloat(cx + w/2), Y2: svgFloat(cy), Style: div})
		elems = append(elems, &Line{X1: svgFloat(cx), Y1: svgFloat(cy - h/2), X2: svgFloat(cx), Y2: svgFloat(cy + h/2), Style: div})
	case diagram.NodeShapeLinedRect:
		// Rect with a vertical sidebar on the left — `lined-process`.
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), Style: shapeStyle,
		})
		bandX := cx - w/2 + w*subroutineBand
		elems = append(elems, &Line{
			X1: svgFloat(bandX), Y1: svgFloat(cy - h/2), X2: svgFloat(bandX), Y2: svgFloat(cy + h/2),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%.2f", th.NodeStroke, defaultStrokeWidth),
		})
	case diagram.NodeShapeForkJoin:
		// Activity-diagram fork/join bar: thin filled slab that
		// fills the layout box (extendedShapeSize allocates a
		// narrow-height box so the bar matches its bounding rect,
		// and edges terminate flush with the bar top/bottom).
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2),
			Width: svgFloat(w), Height: svgFloat(h),
			Style: fmt.Sprintf("fill:%s;stroke:none", th.NodeStroke),
		})
	case diagram.NodeShapeNotchedRect:
		elems = append(elems, &Polygon{Points: notchedRectPoints(cx, cy, w, h), Style: shapeStyle})

	default:
		elems = append(elems, &Rect{
			X: svgFloat(cx - w/2), Y: svgFloat(cy - h/2), Width: svgFloat(w), Height: svgFloat(h), Style: shapeStyle,
		})
	}

	lines := strings.Split(n.Label, "\n")
	lineHeight := fontSize * 1.2
	startY := cy - float64(len(lines)-1)*lineHeight/2
	for i, line := range lines {
		elems = append(elems, &Text{
			X: svgFloat(cx), Y: svgFloat(startY + float64(i)*lineHeight),
			Anchor: "middle", Dominant: "central", FontSize: svgFloat(fontSize),
			Style: textStyle, Content: line,
		})
	}

	return elems
}

func diamondPoints(cx, cy, w, h float64) string {
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx, cy-h/2, cx+w/2, cy, cx, cy+h/2, cx-w/2, cy)
}

func hexagonPoints(cx, cy, w, h float64) string {
	d := w * polygonSkew
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2+d, cy-h/2, cx+w/2-d, cy-h/2, cx+w/2, cy,
		cx+w/2-d, cy+h/2, cx-w/2+d, cy+h/2, cx-w/2, cy)
}

func parallelogramPoints(cx, cy, w, h float64, skew float64, reverse bool) string {
	s := w * skew
	if reverse {
		return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cx-w/2+s, cy-h/2, cx+w/2+s, cy-h/2,
			cx+w/2-s, cy+h/2, cx-w/2-s, cy+h/2)
	}
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2+s, cy-h/2, cx+w/2-s, cy-h/2,
		cx+w/2+s, cy+h/2, cx-w/2-s, cy+h/2)
}

func trapezoidPoints(cx, cy, w, h float64, indent float64, alt bool) string {
	d := w * indent
	if alt {
		return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cx-w/2+d, cy-h/2, cx+w/2-d, cy-h/2,
			cx+w/2, cy+h/2, cx-w/2, cy+h/2)
	}
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2, cx+w/2, cy-h/2,
		cx+w/2-d, cy+h/2, cx-w/2+d, cy+h/2)
}

func asymmetricPoints(cx, cy, w, h float64) string {
	s := w * polygonSkew
	return fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
		cx-w/2, cy-h/2, cx+w/2-s, cy-h/2,
		cx+w/2, cy+h/2, cx-w/2, cy+h/2)
}

