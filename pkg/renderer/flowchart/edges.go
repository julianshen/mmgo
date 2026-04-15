package flowchart

import (
	"fmt"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func markerID(ah diagram.ArrowHead, ls diagram.LineStyle) string {
	return fmt.Sprintf("arrow-%s-%s", ah, ls)
}

// buildMarkers walks every edge in d (including subgraph-scoped edges)
// and returns one Marker per distinct (arrowhead, line-style) pair, in
// deterministic order. Markers must be sorted because Go map iteration
// is randomized — without sorting, multi-arrow diagrams produce
// byte-different SVG output across runs and break golden tests.
func buildMarkers(d *diagram.FlowchartDiagram, th Theme) []Marker {
	needed := map[string]diagram.ArrowHead{}
	collect := func(edges []diagram.Edge) {
		for _, e := range edges {
			if e.ArrowHead == diagram.ArrowHeadNone || e.ArrowHead == diagram.ArrowHeadUnknown {
				continue
			}
			needed[markerID(e.ArrowHead, e.LineStyle)] = e.ArrowHead
		}
	}
	collect(d.Edges)
	var walk func(sgs []diagram.Subgraph)
	walk = func(sgs []diagram.Subgraph) {
		for i := range sgs {
			collect(sgs[i].Edges)
			walk(sgs[i].Children)
		}
	}
	walk(d.Subgraphs)

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
				Style:  fmt.Sprintf("stroke:%s;stroke-width:%g;fill:none", th.EdgeStroke, defaultStrokeWidth),
			},
		}
	case diagram.ArrowHeadCross:
		m.Children = []any{
			&Polyline{
				Points: "0,0 10,5 0,10 10,5",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:%g;fill:none", th.EdgeStroke, defaultStrokeWidth),
			},
		}
	case diagram.ArrowHeadCircle:
		m.RefX = 5
		m.Children = []any{
			&Circle{
				CX: 5, CY: 5, R: 4,
				Style: fmt.Sprintf("stroke:%s;stroke-width:%g;fill:none", th.EdgeStroke, defaultStrokeWidth),
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
	for _, e := range allEdges(d) {
		key := e.From + "->" + e.To
		fromTo[key] = append(fromTo[key], e)
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

		elems = append(elems, renderEdge(ae, elayout, pad, th, fontSize, ruler)...)
	}
	return elems
}

func renderEdge(e diagram.Edge, el layout.EdgeLayout, pad float64, th Theme, fontSize float64, ruler *textmeasure.Ruler) []any {
	pts := make([]layout.Point, len(el.Points))
	copy(pts, el.Points)
	if len(pts) == 0 {
		return nil
	}

	for i := range pts {
		pts[i].X += pad
		pts[i].Y += pad
	}

	style := edgeStyle(th, e.LineStyle)
	var elems []any

	if len(pts) == 2 {
		line := &Line{
			X1: pts[0].X, Y1: pts[0].Y,
			X2: pts[1].X, Y2: pts[1].Y,
			Style: style,
		}
		if e.ArrowHead != diagram.ArrowHeadNone && e.ArrowHead != diagram.ArrowHeadUnknown {
			line.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, line)
	} else if len(pts) >= 3 {
		p := &Path{D: buildCurvePath(pts), Style: style}
		if e.ArrowHead != diagram.ArrowHeadNone && e.ArrowHead != diagram.ArrowHeadUnknown {
			p.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(e.ArrowHead, e.LineStyle))
		}
		elems = append(elems, p)
	}

	if e.Label != "" {
		lx := el.LabelPos.X + pad
		ly := el.LabelPos.Y + pad
		textStyle := fmt.Sprintf("fill:%s;font-size:%gpx", th.EdgeText, fontSize)

		labelW, labelH := ruler.Measure(e.Label, fontSize)
		const labelPad = 4.0
		elems = append(elems, &Rect{
			X: lx - labelW/2 - labelPad, Y: ly - labelH/2 - labelPad,
			Width: labelW + 2*labelPad, Height: labelH + 2*labelPad,
			Style: "fill:white;stroke:none",
		})
		elems = append(elems, &Text{
			X: lx, Y: ly,
			Anchor: "middle", Dominant: "central",
			FontSize: fontSize, Style: textStyle, Content: e.Label,
		})
	}

	return elems
}

func edgeStyle(th Theme, ls diagram.LineStyle) string {
	extra := ""
	switch ls {
	case diagram.LineStyleDotted:
		extra = "stroke-dasharray:2,2;"
	case diagram.LineStyleThick:
		extra = "stroke-width:3;"
	}
	return fmt.Sprintf("stroke:%s;stroke-width:%g;fill:none;%s", th.EdgeStroke, defaultStrokeWidth, extra)
}

func buildCurvePath(pts []layout.Point) string {
	if len(pts) < 3 {
		return ""
	}
	d := fmt.Sprintf("M%.2f,%.2f", pts[0].X, pts[0].Y)

	tension := 0.5
	for i := 0; i < len(pts)-1; i++ {
		p0 := pts[max(i-1, 0)]
		p1 := pts[i]
		p2 := pts[i+1]
		p3 := pts[min(i+2, len(pts)-1)]

		cp1x := p1.X + (p2.X-p0.X)*tension/3
		cp1y := p1.Y + (p2.Y-p0.Y)*tension/3
		cp2x := p2.X - (p3.X-p1.X)*tension/3
		cp2y := p2.Y - (p3.Y-p1.Y)*tension/3

		d += fmt.Sprintf(" C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			cp1x, cp1y, cp2x, cp2y, p2.X, p2.Y)
	}
	return d
}
