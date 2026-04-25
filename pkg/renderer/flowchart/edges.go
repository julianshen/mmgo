package flowchart

import (
	"fmt"
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func isVisibleArrow(ah diagram.ArrowHead) bool {
	return ah != diagram.ArrowHeadNone && ah != diagram.ArrowHeadUnknown
}

func markerID(ah diagram.ArrowHead, ls diagram.LineStyle) string {
	return fmt.Sprintf("arrow-%s-%s", ah, ls)
}

// buildMarkers returns one Marker per distinct (arrowhead, line-style)
// pair used by any edge in d, in deterministic alphabetic order.
// Sorting is mandatory: Go map iteration is randomized, so without it
// multi-arrow diagrams produce byte-different SVG across runs.
func buildMarkers(d *diagram.FlowchartDiagram, th Theme) []Marker {
	needed := map[string]diagram.ArrowHead{}
	for _, e := range d.AllEdges() {
		if !isVisibleArrow(e.ArrowHead) {
			continue
		}
		needed[markerID(e.ArrowHead, e.LineStyle)] = e.ArrowHead
	}
	ids := make([]string, 0, len(needed))
	for id := range needed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	markers := make([]Marker, 0, len(ids))
	for _, id := range ids {
		markers = append(markers, buildMarker(id, needed[id], th))
	}
	return markers
}

func buildMarker(id string, ah diagram.ArrowHead, th Theme) Marker {
	m := Marker{
		ID: id, ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 8, Height: 8,
		Orient: "auto",
	}

	switch ah {
	case diagram.ArrowHeadArrow:
		m.Children = []any{
			&Polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", th.EdgeStroke)},
		}
	case diagram.ArrowHeadOpen:
		m.RefX = 10
		m.Children = []any{
			&Polyline{
				Points: "0,1 10,5 0,9",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", th.EdgeStroke, defaultStrokeWidth),
			},
		}
	case diagram.ArrowHeadCross:
		// Two crossing strokes form a real "X". A single polyline
		// can't lift the pen, so use a Path with two M…L segments.
		m.Children = []any{
			&Path{
				D:     "M0,0 L10,10 M0,10 L10,0",
				Style: fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", th.EdgeStroke, defaultStrokeWidth),
			},
		}
	case diagram.ArrowHeadCircle:
		m.RefX = 5
		// mmdc renders the `o` terminator as a small filled disc, not
		// a hollow ring — fill with the edge stroke colour for parity.
		m.Children = []any{
			&Circle{
				CX: 5, CY: 5, R: 4,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.EdgeStroke, th.EdgeStroke, defaultStrokeWidth),
			},
		}
	}
	return m
}

// renderEdges joins the AST edges (`d.Edges` plus every subgraph's
// Edges) with their layout geometry. Parallel edges between the same
// (from, to) pair are matched by stable order: layout EdgeIDs are
// sorted by (From, To, ID), and AST edges are popped FIFO from a
// per-key queue. The ID tiebreaker prevents the previous bug where
// sort.Slice's undefined ordering for ties could swap labels and
// arrowheads between parallel edges.
func renderEdges(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64, ruler *textmeasure.Ruler) []any {
	fromTo := map[string][]diagram.Edge{}
	for _, e := range d.AllEdges() {
		key := e.From + "->" + e.To
		fromTo[key] = append(fromTo[key], e)
	}
	// Shape lookup drives endpoint clipping: diamonds use rhombus
	// clip, circle-family nodes use radial clip, everything else
	// clips to the bounding rect. Built once so per-edge clipping
	// stays O(1).
	shapeByID := map[string]diagram.NodeShape{}
	for _, n := range d.AllNodes() {
		shapeByID[n.ID] = n.Shape
	}

	edgeKeys := make([]graph.EdgeID, 0, len(l.Edges))
	for eid := range l.Edges {
		edgeKeys = append(edgeKeys, eid)
	}
	sort.Slice(edgeKeys, func(i, j int) bool {
		if edgeKeys[i].From != edgeKeys[j].From {
			return edgeKeys[i].From < edgeKeys[j].From
		}
		if edgeKeys[i].To != edgeKeys[j].To {
			return edgeKeys[i].To < edgeKeys[j].To
		}
		return edgeKeys[i].ID < edgeKeys[j].ID
	})

	var elems []any
	for _, eid := range edgeKeys {
		elayout := l.Edges[eid]
		key := eid.From + "->" + eid.To
		candidates := fromTo[key]
		if len(candidates) == 0 {
			continue
		}
		ae := candidates[0]
		fromTo[key] = candidates[1:]

		elems = append(elems, renderEdge(ae, elayout, pad, th, fontSize, ruler, l, eid, shapeByID)...)
	}
	return elems
}

func renderEdge(e diagram.Edge, el layout.EdgeLayout, pad float64, th Theme, fontSize float64, ruler *textmeasure.Ruler, l *layout.Result, eid graph.EdgeID, shapeByID map[string]diagram.NodeShape) []any {
	pts := make([]layout.Point, len(el.Points))
	copy(pts, el.Points)
	if len(pts) == 0 {
		return nil
	}

	for i := range pts {
		pts[i].X += pad
		pts[i].Y += pad
	}
	// Clip endpoints to source/target node boundaries so marker-end
	// arrowheads don't land inside (and get covered by) the node rect.
	// Cache direction references before mutating either endpoint —
	// pts[1] and pts[len-2] alias for 2-point edges.
	//
	// The l != nil guard exists because several unit tests construct
	// an EdgeLayout directly and pass nil (they don't exercise the
	// clip path); production paths always go through renderEdges
	// which has a non-nil *layout.Result.
	// Self-loops carry pre-synthesised bezier control points (from
	// layout.selfLoopPoints); back-edges get a renderer-synthesised
	// quadratic bow via backEdgeBow. Either way the standard center-
	// to-boundary clip would mangle the curve, so skip it.
	isSelfLoop := e.From == e.To && e.From != ""
	if l != nil && len(pts) >= 2 && !isSelfLoop && !el.BackEdge {
		srcDir := pts[1]
		dstDir := pts[len(pts)-2]
		if src, ok := l.Nodes[eid.From]; ok {
			x, y := clipToShape(shapeByID[eid.From], src, pad, srcDir.X, srcDir.Y)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			last := len(pts) - 1
			x, y := clipToShape(shapeByID[eid.To], dst, pad, dstDir.X, dstDir.Y)
			pts[last] = layout.Point{X: x, Y: y}
		}
		// SVG marker-start renders the marker centred on the line's
		// first vertex, so half of it falls behind the source node and
		// gets covered. Shift pts[0] inward (away from source) by the
		// marker length so the start marker has a visible gap to live in.
		if isVisibleArrow(e.ArrowTail) {
			pts[0] = shiftInward(pts[0], pts[1], startMarkerRefX(e.ArrowTail))
		}
	}

	style := edgeStyle(th, e.LineStyle)
	var elems []any

	switch {
	case isSelfLoop:
		if len(pts) == 4 {
			d := fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
				pts[0].X, pts[0].Y, pts[1].X, pts[1].Y,
				pts[2].X, pts[2].Y, pts[3].X, pts[3].Y)
			p := &Path{D: d, Style: style}
			if isVisibleArrow(e.ArrowHead) {
				p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
			}
			if isVisibleArrow(e.ArrowTail) {
				// Self-loops are never rasterised by canvas (SVG-only
				// bezier), so marker-start is safe here.
				p.MarkerStart = fmt.Sprintf("url(#%s)", markerID(e.ArrowTail, e.LineStyle))
			}
			elems = append(elems, p)
		}
	case el.BackEdge && len(pts) >= 2:
		bow := backEdgeBow(pts[0], pts[len(pts)-1])
		d := fmt.Sprintf("M%.2f,%.2f Q%.2f,%.2f %.2f,%.2f",
			pts[0].X, pts[0].Y, bow.X, bow.Y,
			pts[len(pts)-1].X, pts[len(pts)-1].Y)
		beStyle := style
		if e.LineStyle != diagram.LineStyleDotted {
			beStyle += backEdgeDash
		}
		p := &Path{D: d, Style: beStyle}
		if isVisibleArrow(e.ArrowHead) {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		if isVisibleArrow(e.ArrowTail) {
			// Back-edges use curved beziers that canvas handles
			// differently; keeping marker-start for now.
			p.MarkerStart = fmt.Sprintf("url(#%s)", markerID(e.ArrowTail, e.LineStyle))
		}
		elems = append(elems, p)
	case len(pts) == 2:
		line := &Line{
			X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
			X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
			Style: style,
		}
		if isVisibleArrow(e.ArrowHead) {
			line.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, line)
		elems = append(elems, startMarkerElems(e.ArrowTail, pts[0], pts[1], th)...)
	case len(pts) >= 3:
		p := &Path{D: svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension), Style: style}
		if isVisibleArrow(e.ArrowHead) {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, p)
		elems = append(elems, startMarkerElems(e.ArrowTail, pts[0], pts[1], th)...)
	}

	if e.Label != "" {
		lx := el.LabelPos.X + pad
		ly := el.LabelPos.Y + pad
		// Branch edges (source is a multi-outlet node) get their label
		// positioned 40% along the first segment from the exit port,
		// offset perpendicular outward so readers can tell which
		// branch each label belongs to.
		if l != nil && len(pts) >= 2 {
			if srcNL, ok := l.Nodes[eid.From]; ok && len(srcNL.ExitPorts) > 0 {
				cx := srcNL.X + pad
				cy := srcNL.Y + pad
				lx, ly = branchLabelPos(pts[0], pts[1], cx, cy, fontSize)
			}
		}
		textStyle := fmt.Sprintf("fill:%s;font-size:%.2fpx", th.EdgeText, fontSize)

		labelW, labelH := measureLabel(ruler, e.Label, fontSize)
		const labelPad = 4.0
		// Backdrop fill follows the theme background so labels stay
		// readable on dark themes (a hardcoded white would punch holes).
		elems = append(elems, &Rect{
			X: svgFloat(lx - labelW/2 - labelPad), Y: svgFloat(ly - labelH/2 - labelPad),
			Width: svgFloat(labelW + 2*labelPad), Height: svgFloat(labelH + 2*labelPad),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
		})
		elems = append(elems, &Text{
			X: svgFloat(lx), Y: svgFloat(ly),
			Anchor: "middle", Dominant: "central",
			FontSize: svgFloat(fontSize), Style: textStyle, Content: e.Label,
		})
	}

	return elems
}

// startMarkerElems renders the tail-side arrowhead inline rather than
// via SVG `marker-start`, because tdewolff/canvas's SVG-to-PNG parser
// drops `marker-start` references silently — bidirectional edges
// (`<-->`, `o--o`, `x--x`) otherwise show only one terminator in the
// rasterised output. The inline shape is positioned at the line start
// and oriented to face outward (back toward the source node).
//
// `head` is the marker shape; `start` is the line's first vertex
// (already shifted inward by markerVisibleLen so the symbol sits in
// the visible gap); `next` is the next polyline vertex used to compute
// the line direction.
func startMarkerElems(head diagram.ArrowHead, start, next layout.Point, th Theme) []any {
	if !isVisibleArrow(head) {
		return nil
	}
	if math.Hypot(next.X-start.X, next.Y-start.Y) == 0 {
		return nil
	}
	m := buildMarker("_tail", head, th)
	// buildMarker uses viewBox "0 0 10 10" with refX/refY as the
	// anchor. InlineMarkerAt places children so that (refX, refY)
	// coincides with (startX, startY), rotated to face from start
	// toward next — i.e. outward, back toward the source node.
	// refY=5 centres vertically; refX is per-shape (9 for arrow, 5
	// for circle, 9 for cross).
	g := svgutil.InlineMarkerAt(start.X, start.Y, next.X, next.Y, float64(m.RefX), float64(m.RefY), m.Children)
	// Wrap the svgutil.Group (which uses svgutil.Float) in a
	// flowchart Group so the renderer's XML marshaler picks it up.
	fg := &Group{Transform: g.Transform}
	fg.Children = g.Children
	return []any{fg}
}

const (
	backEdgeBowRatio = 0.2
	backEdgeBowMin   = 30.0
	backEdgeDash     = ";stroke-dasharray:6,3"
)

func shiftInward(p, q layout.Point, dist float64) layout.Point {
	dx := q.X - p.X
	dy := q.Y - p.Y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return p
	}
	return layout.Point{X: p.X + dx/length*dist, Y: p.Y + dy/length*dist}
}

func startMarkerRefX(ah diagram.ArrowHead) float64 {
	switch ah {
	case diagram.ArrowHeadCircle:
		return 5
	default:
		return 9
	}
}

// backEdgeBow returns the quadratic-bezier control point for a back-
// edge: the midpoint of src→dst pushed perpendicular to that segment.
func backEdgeBow(src, dst layout.Point) layout.Point {
	mx := (src.X + dst.X) / 2
	my := (src.Y + dst.Y) / 2
	nx, ny, length := svgutil.Perpendicular(src, dst)
	if length == 0 {
		return layout.Point{X: mx, Y: my}
	}
	mag := length * backEdgeBowRatio
	if mag < backEdgeBowMin {
		mag = backEdgeBowMin
	}
	return layout.Point{X: mx + nx*mag, Y: my + ny*mag}
}

// branchLabelPos returns the label anchor for a branch edge: placed at
// 40% along the first segment (port → stem) and offset perpendicular
// to that segment, away from the source node's center. The offset uses
// fontSize/2+4 as breathing room — enough that the label clears the
// edge stroke but close enough that it reads as attached to the
// branch it labels.
func branchLabelPos(port, stem layout.Point, cx, cy, fontSize float64) (x, y float64) {
	const t = 0.4
	sx := port.X + t*(stem.X-port.X)
	sy := port.Y + t*(stem.Y-port.Y)
	nx, ny, length := svgutil.Perpendicular(port, stem)
	if length == 0 {
		return sx, sy
	}
	// Flip the normal if it points toward the node center.
	if nx*(sx-cx)+ny*(sy-cy) < 0 {
		nx, ny = -nx, -ny
	}
	off := fontSize/2 + 4
	return sx + nx*off, sy + ny*off
}

// measureLabel returns the rendered (width, height) of label at the
// given font size. ruler is the production text measurer; when nil
// (test helpers that don't initialize a real ruler) it returns a small
// fallback box. Render() always passes a non-nil ruler — this guard
// exists purely so renderEdge cannot panic on a nil passed in by
// internal test code.
func measureLabel(ruler *textmeasure.Ruler, label string, fontSize float64) (w, h float64) {
	if ruler == nil {
		return 40, 20
	}
	return ruler.Measure(label, fontSize)
}

func edgeStyle(th Theme, ls diagram.LineStyle) string {
	if ls == diagram.LineStyleInvisible {
		return "stroke:none;fill:none"
	}
	base := fmt.Sprintf("stroke:%s;stroke-width:%v;fill:none", th.EdgeStroke, defaultStrokeWidth)
	switch ls {
	case diagram.LineStyleDotted:
		return base + ";stroke-dasharray:2,2"
	case diagram.LineStyleThick:
		return fmt.Sprintf("stroke:%s;stroke-width:3;fill:none", th.EdgeStroke)
	default:
		return base
	}
}

// clipToShape picks the right endpoint-clip geometry for the given
// node shape. Circle-family nodes use radial clipping; Diamond and
// Hexagon use their O(1) specialised clippers; the rest of the
// polygon family (triangles, trapezoids, parallelograms, notched
// rects, etc.) goes through the generic ClipToPolygonEdge using
// vertices from polygonVerts. Everything else (rect-based glyphs,
// path-based shapes like cylinder/cloud where the bounding rect is
// a reasonable approximation) falls back to ClipToRectEdge.
//
// Hourglass is a self-intersecting bowtie whose self-cross sits at
// the layout center, making ray-from-center clipping degenerate, so
// it stays on the rect fallback by design (polygonVerts returns nil
// for it).
func clipToShape(shape diagram.NodeShape, n layout.NodeLayout, pad, ox, oy float64) (x, y float64) {
	cx, cy := n.X+pad, n.Y+pad
	switch shape {
	case diagram.NodeShapeCircle,
		diagram.NodeShapeDoubleCircle,
		diagram.NodeShapeSmallCircle,
		diagram.NodeShapeFilledCircle,
		diagram.NodeShapeFramedCircle,
		diagram.NodeShapeCrossCircle:
		r := math.Min(n.Width, n.Height) / 2
		return svgutil.ClipToCircleEdge(cx, cy, r, ox, oy)
	case diagram.NodeShapeDiamond:
		return svgutil.ClipToDiamondEdge(cx, cy, n.Width, n.Height, ox, oy)
	case diagram.NodeShapeHexagon:
		return svgutil.ClipToHexagonEdge(cx, cy, n.Width, n.Height, polygonSkew, ox, oy)
	}
	if verts := polygonVerts(shape, cx, cy, n.Width, n.Height); verts != nil {
		return svgutil.ClipToPolygonEdge(cx, cy, verts, ox, oy)
	}
	return svgutil.ClipToRectEdge(cx, cy, n.Width, n.Height, ox, oy)
}
