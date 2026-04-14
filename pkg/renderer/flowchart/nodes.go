package flowchart

import (
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

const (
	polygonSkew        = 0.15
	cylinderEllipseRY  = 0.1
	defaultStrokeWidth = 1.5
	doubleCircleGap    = 3.0
	subroutineBand     = 0.1
)

func renderNode(n diagram.Node, nl layout.NodeLayout, pad float64, th Theme, fontSize float64) []any {
	shapeStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", th.NodeFill, th.NodeStroke, defaultStrokeWidth)
	textStyle := fmt.Sprintf("fill:%s;font-size:%gpx", th.NodeText, fontSize)

	cx := nl.X + pad
	cy := nl.Y + pad
	w := nl.Width
	h := nl.Height

	var elems []any

	switch n.Shape {
	case diagram.NodeShapeRectangle:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
	case diagram.NodeShapeRoundedRectangle:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, RX: 5, RY: 5, Style: shapeStyle,
		})
	case diagram.NodeShapeStadium:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, RX: h / 2, RY: h / 2, Style: shapeStyle,
		})
	case diagram.NodeShapeCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: cx, CY: cy, R: r, Style: shapeStyle})
	case diagram.NodeShapeDoubleCircle:
		r := math.Min(w, h) / 2
		elems = append(elems, &Circle{CX: cx, CY: cy, R: r, Style: shapeStyle})
		elems = append(elems, &Circle{
			CX: cx, CY: cy, R: r + doubleCircleGap,
			Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:%g", th.NodeStroke, defaultStrokeWidth),
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
		elems = append(elems, &Path{D: cylinderPath(cx, cy, w, h), Style: shapeStyle})
	case diagram.NodeShapeSubroutine:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
		bandX1 := cx - w/2 + w*subroutineBand
		bandX2 := cx + w/2 - w*subroutineBand
		lineStyle := fmt.Sprintf("stroke:%s;stroke-width:%g", th.NodeStroke, defaultStrokeWidth)
		elems = append(elems, &Line{X1: bandX1, Y1: cy - h/2, X2: bandX1, Y2: cy + h/2, Style: lineStyle})
		elems = append(elems, &Line{X1: bandX2, Y1: cy - h/2, X2: bandX2, Y2: cy + h/2, Style: lineStyle})
	default:
		elems = append(elems, &Rect{
			X: cx - w/2, Y: cy - h/2, Width: w, Height: h, Style: shapeStyle,
		})
	}

	lines := strings.Split(n.Label, "\n")
	lineHeight := fontSize * 1.2
	startY := cy - float64(len(lines)-1)*lineHeight/2
	for i, line := range lines {
		elems = append(elems, &Text{
			X: cx, Y: startY + float64(i)*lineHeight,
			Anchor: "middle", Dominant: "central", FontSize: fontSize,
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

func cylinderPath(cx, cy, w, h float64) string {
	ry := h * cylinderEllipseRY
	top := cy - h/2 + ry
	bot := cy + h/2 - ry
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		cx-w/2, top, cx-w/2, bot, w/2, ry, cx+w/2, bot,
		cx+w/2, top, w/2, ry, cx-w/2, top,
		cx-w/2, top, w/2, ry, cx+w/2, top)
}
