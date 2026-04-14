package flowchart

import (
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func markerID(ah diagram.ArrowHead, ls diagram.LineStyle) string {
	return fmt.Sprintf("arrow-%s-%s", ah, ls)
}

func buildMarkers(d *diagram.FlowchartDiagram, th Theme) []Marker {
	needed := map[string]diagram.ArrowHead{}
	for _, e := range d.Edges {
		if e.ArrowHead == diagram.ArrowHeadNone || e.ArrowHead == diagram.ArrowHeadUnknown {
			continue
		}
		id := markerID(e.ArrowHead, e.LineStyle)
		needed[id] = e.ArrowHead
	}

	var markers []Marker
	for id, ah := range needed {
		markers = append(markers, buildMarker(id, ah, th))
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
				Style:  fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	case diagram.ArrowHeadCross:
		m.Children = []any{
			&Polyline{
				Points: "0,0 10,5 0,10 10,5",
				Style:  fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	case diagram.ArrowHeadCircle:
		m.RefX = 5
		m.Children = []any{
			&Circle{
				CX: 5, CY: 5, R: 4,
				Style: fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke),
			},
		}
	}
	return m
}

func renderEdges(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	type indexed struct {
		diagram.Edge
		idx int
	}
	fromTo := map[string][]indexed{}
	for i, e := range d.Edges {
		key := e.From + "->" + e.To
		fromTo[key] = append(fromTo[key], indexed{Edge: e, idx: i})
	}

	var elems []any
	for eid, elayout := range l.Edges {
		key := eid.From + "->" + eid.To
		candidates := fromTo[key]
		if len(candidates) == 0 {
			continue
		}
		ae := candidates[0]
		fromTo[key] = candidates[1:]

		elems = append(elems, renderEdge(ae.Edge, elayout, pad, th, fontSize)...)
	}
	return elems
}

func renderEdge(e diagram.Edge, el layout.EdgeLayout, pad float64, th Theme, fontSize float64) []any {
	pts := el.Points
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

		elems = append(elems, &Rect{
			X: lx - 20, Y: ly - 10,
			Width: 40, Height: 20,
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
	return fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none;%s", th.EdgeStroke, extra)
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
