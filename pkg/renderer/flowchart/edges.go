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
		m.Children = []any{
			&Polyline{
				Points: "0,0 10,5 0,10 10,5",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", th.EdgeStroke, defaultStrokeWidth),
			},
		}
	case diagram.ArrowHeadCircle:
		m.RefX = 5
		m.Children = []any{
			&Circle{
				CX: 5, CY: 5, R: 4,
				Style: fmt.Sprintf("stroke:%s;stroke-width:%.2f;fill:none", th.EdgeStroke, defaultStrokeWidth),
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
	if l != nil && len(pts) >= 2 {
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
	}

	style := edgeStyle(th, e.LineStyle)
	var elems []any

	if len(pts) == 2 {
		line := &Line{
			X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
			X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
			Style: style,
		}
		if isVisibleArrow(e.ArrowHead) {
			line.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		if isVisibleArrow(e.ArrowTail) {
			line.MarkerStart = fmt.Sprintf("url(#%s)", markerID(e.ArrowTail, e.LineStyle))
		}
		elems = append(elems, line)
	} else if len(pts) >= 3 {
		p := &Path{D: svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension), Style: style}
		if isVisibleArrow(e.ArrowHead) {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		if isVisibleArrow(e.ArrowTail) {
			p.MarkerStart = fmt.Sprintf("url(#%s)", markerID(e.ArrowTail, e.LineStyle))
		}
		elems = append(elems, p)
	}

	if e.Label != "" {
		lx := el.LabelPos.X + pad
		ly := el.LabelPos.Y + pad
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
// node shape. Circle-family nodes (Circle, DoubleCircle, SmallCircle,
// FilledCircle, FramedCircle, CrossCircle) use radial clipping;
// Diamond uses rhombus-edge intersection; everything else falls back
// to the axis-aligned bounding rect (which is correct for rect-based
// shapes and "close enough" for the polygon family where exact edge
// geometry would need per-shape intersection code).
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
	default:
		return svgutil.ClipToRectEdge(cx, cy, n.Width, n.Height, ox, oy)
	}
}
