// Package svgutil provides shared SVG XML types and helpers used
// across all diagram renderers.
package svgutil

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// Perpendicular returns the right-hand unit normal to the segment
// from src to dst (i.e. the segment direction rotated 90° clockwise),
// along with the segment length. When the two points coincide it
// returns (0, 0, 0) — callers must treat that as the degenerate case
// instead of dividing by it.
func Perpendicular(src, dst layout.Point) (nx, ny, length float64) {
	dx := dst.X - src.X
	dy := dst.Y - src.Y
	length = math.Hypot(dx, dy)
	if length == 0 {
		return 0, 0, 0
	}
	return -dy / length, dx / length, length
}

// FormatNumber renders v as the shortest readable numeric string:
// integer form when v rounds to an integer within maxDecimals tolerance,
// otherwise the shortest fixed-point form up to maxDecimals digits with
// trailing zeros stripped. NaN/Inf collapse to "0".
func FormatNumber(v float64, maxDecimals int) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	scale := math.Pow(10, float64(maxDecimals))
	rounded := math.Round(v*scale) / scale
	if rounded == math.Trunc(rounded) {
		return strconv.FormatFloat(rounded, 'f', 0, 64)
	}
	s := strconv.FormatFloat(rounded, 'f', maxDecimals, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

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

// ClipToDiamondEdge returns the point on the axis-aligned rhombus
// with vertices at (cx±w/2, cy) and (cx, cy±h/2) along the ray from
// the center toward (ox, oy). The rhombus is |dx|/(w/2) + |dy|/(h/2)
// = 1 on its boundary; we scale the direction vector by the inverse
// sum so the endpoint sits exactly on the nearest diamond edge. If
// (ox, oy) coincides with the center the center is returned.
func ClipToDiamondEdge(cx, cy, w, h, ox, oy float64) (x, y float64) {
	dx, dy := ox-cx, oy-cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	halfW, halfH := w/2, h/2
	denom := math.Abs(dx)/halfW + math.Abs(dy)/halfH
	if denom == 0 {
		return cx, cy
	}
	t := 1 / denom
	// Clamp so a reference already inside the rhombus doesn't pull
	// the endpoint past it (same safety as ClipToRectEdge).
	if t > 1 {
		t = 1
	}
	return cx + dx*t, cy + dy*t
}

// ClipToHexagonEdge returns the point on the stretched hexagon
// boundary (center (cx, cy), bounding size (w, h), diagonal inset
// d = w*skew) along the ray toward (ox, oy). The hexagon has flat
// top/bottom caps spanning x ∈ [cx-w/2+d, cx+w/2-d] and single
// vertices at the left/right midpoints (cx±w/2, cy), matching the
// flowchart hexagonPoints geometry.
//
// By full symmetry we work in a single quadrant with |dx|, |dy|:
// rays that exit through a cap hit y = h/2 with x ≤ w/2-d; rays
// steeper-in-x than that hit the adjacent diagonal, whose line is
// (h/2)|x| + d·|y| = h·w/4.
func ClipToHexagonEdge(cx, cy, w, h, skew, ox, oy float64) (x, y float64) {
	dx, dy := ox-cx, oy-cy
	if dx == 0 && dy == 0 {
		return cx, cy
	}
	halfW, halfH := w/2, h/2
	d := w * skew
	ax, ay := math.Abs(dx), math.Abs(dy)

	var t float64
	if ay > 0 {
		tCap := halfH / ay
		if tCap*ax <= halfW-d {
			t = tCap
		} else {
			t = halfH * halfW / (halfH*ax + d*ay)
		}
	} else {
		t = halfW / ax
	}
	// Clamp so a reference already inside the hex doesn't pull the
	// endpoint past it (same safety as ClipToRectEdge).
	if t > 1 {
		t = 1
	}
	return cx + dx*t, cy + dy*t
}

// ClipToPolygonEdge returns the point on the polygon boundary along
// the ray from (cx, cy) toward (ox, oy). The polygon is given as
// absolute-coordinate vertices in winding order; the closing edge
// from poly[n-1] back to poly[0] is implicit. Works for any planar
// polygon — the algorithm intersects the ray with each edge segment
// and returns the smallest positive parameter t.
//
// For convex polygons with the center inside, exactly one edge gives
// a valid hit. For non-convex shapes whose center sits outside the
// interior or on a self-intersection (e.g. an hourglass bowtie), the
// result can be degenerate; callers should keep such shapes on the
// rect fallback.
//
// If no edge is crossed (ray parallel to all of them — impossible for
// closed polygons in general but defensible against degenerate input),
// the center is returned. Inside-clamp safety mirrors ClipToRectEdge:
// a reference already inside the polygon doesn't pull the endpoint
// past it.
func ClipToPolygonEdge(cx, cy float64, poly []layout.Point, ox, oy float64) (x, y float64) {
	dx, dy := ox-cx, oy-cy
	if dx == 0 && dy == 0 || len(poly) < 3 {
		return cx, cy
	}
	bestT := math.Inf(1)
	n := len(poly)
	for i := 0; i < n; i++ {
		p1 := poly[i]
		p2 := poly[(i+1)%n]
		ex, ey := p2.X-p1.X, p2.Y-p1.Y
		fx, fy := p1.X-cx, p1.Y-cy
		det := dy*ex - dx*ey
		if math.Abs(det) < 1e-12 {
			continue
		}
		t := (ex*fy - ey*fx) / det
		u := (dx*fy - dy*fx) / det
		const eps = 1e-9
		if t > eps && u >= -eps && u <= 1+eps && t < bestT {
			bestT = t
		}
	}
	if math.IsInf(bestT, 1) {
		return cx, cy
	}
	if bestT > 1 {
		bestT = 1
	}
	return cx + dx*bestT, cy + dy*bestT
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
	XMLName             xml.Name `xml:"svg"`
	XMLNS               string   `xml:"xmlns,attr"`
	ViewBox             string   `xml:"viewBox,attr"`
	Role                string   `xml:"role,attr,omitempty"`
	AriaRoleDescription string   `xml:"aria-roledescription,attr,omitempty"`
	Children            []any    `xml:",any"`
}

// Title is the SVG <title> element. Screen readers announce it as the
// document's accessible name.
type Title struct {
	XMLName xml.Name `xml:"title"`
	Content string   `xml:",chardata"`
}

// Desc is the SVG <desc> element. Screen readers announce it as the
// document's longer description.
type Desc struct {
	XMLName xml.Name `xml:"desc"`
	Content string   `xml:",chardata"`
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

// GroupBoxPadX / GroupBoxPadY / GroupBoxLabelH are the shared
// metrics for "labelled bounding rectangle around a group of nodes"
// — used by class-diagram namespaces and state-diagram composite
// states. Per-renderer corner radius stays a local choice.
const (
	GroupBoxPadX   = 12.0
	GroupBoxPadY   = 10.0
	GroupBoxLabelH = 22.0
)

// BBoxOver returns the bounding box of the named layout nodes,
// translated by `pad`. Missing IDs are skipped. Returns
// `Empty()==true` when no IDs resolved.
func BBoxOver(ids []string, nodes map[string]layout.NodeLayout, pad float64) BBox {
	bb := NewInfiniteBBox()
	for _, id := range ids {
		n, ok := nodes[id]
		if !ok {
			continue
		}
		bb.Expand(n.X+pad, n.Y+pad, n.Width, n.Height)
	}
	return bb
}

// Note rendering metrics shared across renderers that draw
// sticky-note rectangles (class, state, and any future diagram type
// with notes). Per-renderer code applies these uniformly so notes
// sized in one diagram type read consistently in another.
const (
	NotePadX  = 10.0
	NotePadY  = 8.0
	NoteGap   = 16.0
	NoteLineH = 18.0
)

// IndexByID builds a `map[id]item` over `items`, keyed by whatever
// the `key` callback returns. Last-seen wins on duplicate keys.
// Returns nil for an empty input so callers can range over a nil
// map idempotently.
func IndexByID[T any](items []T, key func(T) string) map[string]T {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string]T, len(items))
	for _, it := range items {
		out[key(it)] = it
	}
	return out
}

// GroupByID is the slice-valued analogue of IndexByID — returns
// `map[id][]item` with all entries sharing a key kept together in
// source order. Used for inline style stacks where multiple
// `style ID …` lines accumulate per node.
func GroupByID[T any](items []T, key func(T) string) map[string][]T {
	if len(items) == 0 {
		return nil
	}
	out := make(map[string][]T, len(items))
	for _, it := range items {
		k := key(it)
		out[k] = append(out[k], it)
	}
	return out
}

// RankDirFor maps a diagram.Direction to the layout package's
// RankDir. DirectionUnknown (and any unrecognised value) defaults
// to top-to-bottom — the convention every diagram type follows.
func RankDirFor(d diagram.Direction) layout.RankDir {
	switch d {
	case diagram.DirectionBT:
		return layout.RankDirBT
	case diagram.DirectionLR:
		return layout.RankDirLR
	case diagram.DirectionRL:
		return layout.RankDirRL
	}
	return layout.RankDirTB
}

// BBox accumulates an axis-aligned bounding box of arbitrary
// rectangles. Initialise with NewInfiniteBBox so the first Expand
// call seeds the extents instead of clamping against zero.
type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

// NewInfiniteBBox returns a BBox seeded with ±Inf so any Expand call
// replaces the extents. Used by callers that want to know whether
// any rectangles contributed (Empty()==true means none did).
func NewInfiniteBBox() BBox {
	return BBox{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
}

// Expand grows the bbox to include the centred rectangle (cx, cy,
// w, h). Centred form (rather than top-left + size) is what the
// layout engine produces.
func (b *BBox) Expand(cx, cy, w, h float64) {
	left, right := cx-w/2, cx+w/2
	top, bottom := cy-h/2, cy+h/2
	if left < b.MinX {
		b.MinX = left
	}
	if right > b.MaxX {
		b.MaxX = right
	}
	if top < b.MinY {
		b.MinY = top
	}
	if bottom > b.MaxY {
		b.MaxY = bottom
	}
}

// Empty reports whether the bbox is still at its ±Inf seed values
// — i.e. no Expand call has happened. Renderers use this to skip
// emitting a degenerate rectangle when the source set was empty.
func (b BBox) Empty() bool {
	return math.IsInf(b.MinX, 1) || math.IsInf(b.MaxX, -1)
}

// Anchor renders an SVG `<a>` hyperlink. Children inside (rects,
// paths, text) become clickable. Uses the SVG 2 unprefixed `href`
// attribute, which all modern renderers accept; older xlink:href
// is omitted to avoid pulling an extra namespace declaration onto
// every Doc.
type Anchor struct {
	XMLName  xml.Name `xml:"a"`
	Href     string   `xml:"href,attr,omitempty"`
	Target   string   `xml:"target,attr,omitempty"`
	Children []any    `xml:",any"`
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

// MergeStr writes src into *dst when src is non-empty. Centralises
// the "override on explicit set, inherit on zero" pattern shared by
// every renderer's resolveTheme.
func MergeStr(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// MergeFloat writes src into *dst when src is positive. Mirrors
// MergeStr for numeric Config fields where the float zero value
// signals "inherit default" (so a caller can't explicitly request
// 0; that's the documented contract everywhere it's used).
func MergeFloat(dst *float64, src float64) {
	if src > 0 {
		*dst = src
	}
}

// MergeBoolPtr writes src into *dst when src is non-nil. Used for
// tri-state Show* config flags where nil = "inherit default" and
// &false vs &true distinguishes "explicitly off" from "explicitly on".
func MergeBoolPtr(dst **bool, src *bool) {
	if src != nil {
		*dst = src
	}
}
