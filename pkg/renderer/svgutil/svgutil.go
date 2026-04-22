// Package svgutil provides shared SVG XML types and helpers used
// across all diagram renderers.
package svgutil

import (
	"encoding/xml"
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/layout"
)

// CatmullRomTension is the default tension used when smoothing dagre's
// polyline waypoints into cubic splines. 0.5 produced noticeably
// exaggerated swoops at dummy-node waypoints; 0.3 yields a softer,
// more mmdc-like curve and is what the flowchart and class renderers
// settled on (see PR #73).
const CatmullRomTension = 0.3

// CatmullRomPath turns a polyline of 3+ waypoints into an SVG path "d"
// attribute drawn as a Catmull-Rom cubic spline at the given tension.
// For fewer than 3 points it returns the empty string — callers should
// emit a straight <line> for the 2-point case and skip empty edges.
func CatmullRomPath(pts []layout.Point, tension float64) string {
	if len(pts) < 3 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "M%.2f,%.2f", pts[0].X, pts[0].Y)
	for i := 0; i < len(pts)-1; i++ {
		p0 := pts[max(i-1, 0)]
		p1 := pts[i]
		p2 := pts[i+1]
		p3 := pts[min(i+2, len(pts)-1)]

		cp1x := p1.X + (p2.X-p0.X)*tension/3
		cp1y := p1.Y + (p2.Y-p0.Y)*tension/3
		cp2x := p2.X - (p3.X-p1.X)*tension/3
		cp2y := p2.Y - (p3.Y-p1.Y)*tension/3

		fmt.Fprintf(&sb, " C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cp1x, cp1y, cp2x, cp2y, p2.X, p2.Y)
	}
	return sb.String()
}

// CylinderEllipseRY is the cap-to-body height ratio for cylinder
// shapes. 0.1 matches mmdc's flowchart database glyph; the C4
// SystemDB/ContainerDB shapes reuse the same value for visual parity.
const CylinderEllipseRY = 0.1

// CylinderPath returns an SVG path "d" attribute for a vertical
// cylinder centered at (cx, cy) with overall size (w, h). The shape is
// rendered as two side lines, a bottom-cap arc, and a separate top-cap
// arc (drawn with a second moveto so the rim reads as an ellipse over
// an open top). Used by the flowchart cylinder node and the C4 DB
// element shapes.
func CylinderPath(cx, cy, w, h float64) string {
	ry := h * CylinderEllipseRY
	top := cy - h/2 + ry
	bot := cy + h/2 - ry
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		cx-w/2, top, cx-w/2, bot, w/2, ry, cx+w/2, bot,
		cx+w/2, top, w/2, ry, cx-w/2, top,
		cx-w/2, top, w/2, ry, cx+w/2, top)
}

// LabelChip returns a centered rounded-rect "chip" sized to wrap a
// label of (textW, textH) with the given padding on every side.
// The chip's center sits at (cx, cy) so callers can place it directly
// behind a centered <text> element. fill is applied as the rect's fill
// style with stroke disabled; cornerR controls the rx/ry rounding (use
// 0 for square corners).
//
// Used by class/ER/state edge labels. The flowchart renderer keeps its
// own rect construction because its local Rect type carries an extra
// Class attribute exercised by type assertions in tests.
func LabelChip(cx, cy, textW, textH, padding float64, fill string, cornerR float64) *Rect {
	return &Rect{
		X:      Float(cx - textW/2 - padding),
		Y:      Float(cy - textH/2 - padding),
		Width:  Float(textW + 2*padding),
		Height: Float(textH + 2*padding),
		RX:     Float(cornerR),
		RY:     Float(cornerR),
		Style:  fmt.Sprintf("fill:%s;stroke:none", fill),
	}
}

// ClipToRectEdge returns the point on the axis-aligned rectangle
// boundary (center (cx, cy), size (w, h)) where the ray toward
// (ox, oy) exits. If (ox, oy) already lies inside the rect the
// result is clamped to that point so the clip never overshoots its
// reference.
func ClipToRectEdge(cx, cy, w, h, ox, oy float64) (x, y float64) {
	dx, dy := ox-cx, oy-cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	halfW, halfH := w/2, h/2
	t := math.Inf(1)
	if dx != 0 {
		t = halfW / math.Abs(dx)
	}
	if dy != 0 {
		if ty := halfH / math.Abs(dy); ty < t {
			t = ty
		}
	}
	if t > 1 {
		t = 1
	}
	return cx + dx*t, cy + dy*t
}

// ClipToCircleEdge returns the point on the circle (center (cx, cy),
// radius r) along the ray toward (ox, oy). If (ox, oy) coincides
// with the center the center is returned unchanged.
func ClipToCircleEdge(cx, cy, r, ox, oy float64) (x, y float64) {
	dx, dy := ox-cx, oy-cy
	d := math.Sqrt(dx*dx + dy*dy)
	if d == 0 {
		return cx, cy
	}
	return cx + dx/d*r, cy + dy/d*r
}

// NegCoord formats -v for an SVG transform attribute, avoiding the
// "-0.00" output that a plain %.2f of -0 produces (ugly in
// golden-file diffs).
func NegCoord(v float64) string {
	if v == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", -v)
}

// InlineMarkerAt emits a transformed <g> that places children so
// their (refX, refY) anchor sits at (startX, startY) and their local
// +X axis points from (startX, startY) toward (nextX, nextY). Used
// to work around tdewolff/canvas mis-positioning marker-start when
// marker-end is also set on the same element; browsers render this
// inline group identically to a marker-start reference.
func InlineMarkerAt(startX, startY, nextX, nextY, refX, refY float64, children []any) *Group {
	angle := math.Atan2(nextY-startY, nextX-startX) * 180 / math.Pi
	return &Group{
		Transform: fmt.Sprintf("translate(%.2f,%.2f) rotate(%.2f) translate(%s,%s)",
			startX, startY, angle, NegCoord(refX), NegCoord(refY)),
		Children: children,
	}
}

type Float float64

func (v Float) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: fmt.Sprintf("%.2f", Round2(float64(v)))}, nil
}

func Round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

func Sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

const xmlDecl = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"

type Doc struct {
	XMLName  xml.Name `xml:"svg"`
	XMLNS    string   `xml:"xmlns,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	Role     string   `xml:"role,attr,omitempty"`
	AriaRole string   `xml:"aria-roledescription,attr,omitempty"`
	Children []any    `xml:",any"`
}

type Rect struct {
	XMLName xml.Name `xml:"rect"`
	X       Float    `xml:"x,attr"`
	Y       Float    `xml:"y,attr"`
	Width   Float    `xml:"width,attr"`
	Height  Float    `xml:"height,attr"`
	RX      Float    `xml:"rx,attr,omitempty"`
	RY      Float    `xml:"ry,attr,omitempty"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Line struct {
	XMLName     xml.Name `xml:"line"`
	X1          Float    `xml:"x1,attr"`
	Y1          Float    `xml:"y1,attr"`
	X2          Float    `xml:"x2,attr"`
	Y2          Float    `xml:"y2,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
}

type Text struct {
	XMLName   xml.Name `xml:"text"`
	X         Float    `xml:"x,attr"`
	Y         Float    `xml:"y,attr"`
	Anchor    string   `xml:"text-anchor,attr,omitempty"`
	Dominant  string   `xml:"dominant-baseline,attr,omitempty"`
	Style     string   `xml:"style,attr,omitempty"`
	Transform string   `xml:"transform,attr,omitempty"`
	Content   string   `xml:",chardata"`
}

type Circle struct {
	XMLName xml.Name `xml:"circle"`
	CX      Float    `xml:"cx,attr"`
	CY      Float    `xml:"cy,attr"`
	R       Float    `xml:"r,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Polygon struct {
	XMLName xml.Name `xml:"polygon"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Polyline struct {
	XMLName xml.Name `xml:"polyline"`
	Points  string   `xml:"points,attr"`
	Style   string   `xml:"style,attr,omitempty"`
}

type Path struct {
	XMLName     xml.Name `xml:"path"`
	D           string   `xml:"d,attr"`
	Style       string   `xml:"style,attr,omitempty"`
	MarkerStart string   `xml:"marker-start,attr,omitempty"`
	MarkerEnd   string   `xml:"marker-end,attr,omitempty"`
}

type Defs struct {
	XMLName xml.Name `xml:"defs"`
	Markers []Marker `xml:"marker,omitempty"`
}

type Group struct {
	XMLName   xml.Name `xml:"g"`
	Class     string   `xml:"class,attr,omitempty"`
	Transform string   `xml:"transform,attr,omitempty"`
	Children  []any    `xml:",any"`
}

type Marker struct {
	XMLName  xml.Name `xml:"marker"`
	ID       string   `xml:"id,attr"`
	ViewBox  string   `xml:"viewBox,attr"`
	RefX     Float    `xml:"refX,attr"`
	RefY     Float    `xml:"refY,attr"`
	Width    Float    `xml:"markerWidth,attr"`
	Height   Float    `xml:"markerHeight,attr"`
	Orient   string   `xml:"orient,attr"`
	Children []any    `xml:",any"`
}

func ViewBox(w, h float64) string {
	return fmt.Sprintf("0 0 %.2f %.2f", w, h)
}

func MarshalSVG(doc Doc) ([]byte, error) {
	raw, err := xml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return append([]byte(xmlDecl), raw...), nil
}
