package block

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize    = 14.0
	defaultPadding     = 20.0
	defaultStrokeWidth = 1.5
	nodePadX           = 20.0
	nodePadY           = 12.0
	minNodeW           = 80.0
	minNodeH           = 40.0
)

type Options struct {
	FontSize float64
}

func Render(d *diagram.BlockDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("block render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("block render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	rankDir := layout.RankDirTB
	if d.Columns > 0 && len(d.Edges) == 0 {
		// Column layout with no edges: arrange left-to-right for visual flow.
		rankDir = layout.RankDirLR
	}

	g := graph.New()
	for _, n := range d.Nodes {
		w, h := nodeSize(n, ruler, fontSize)
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: w, Height: h})
	}
	for _, e := range d.Edges {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: rankDir})
	pad := defaultPadding

	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	children = append(children, &defs{Markers: []marker{buildArrowMarker()}})
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:#fff;stroke:none",
	})

	children = append(children, renderEdges(d, l, pad, fontSize)...)
	children = append(children, renderNodes(d, l, pad, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("block render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func nodeSize(n diagram.BlockNode, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, th := ruler.Measure(n.Label, fontSize)
	w = tw + 2*nodePadX
	h = th + 2*nodePadY
	if w < minNodeW {
		w = minNodeW
	}
	if h < minNodeH {
		h = minNodeH
	}
	if n.Shape == diagram.BlockShapeCircle {
		side := w
		if h > side {
			side = h
		}
		return side, side
	}
	return w, h
}

func renderNodes(d *diagram.BlockDiagram, l *layout.Result, pad, fontSize float64) []any {
	var elems []any
	for _, n := range d.Nodes {
		nl, ok := l.Nodes[n.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		style := "fill:#ECECFF;stroke:#9370DB;stroke-width:1.5"
		switch n.Shape {
		case diagram.BlockShapeDiamond:
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				cx, y, x+w, cy, cx, y+h, x, cy)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeCircle:
			r := w / 2
			if h/2 < r {
				r = h / 2
			}
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r),
				Style: style,
			})
		case diagram.BlockShapeRound:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: svgFloat(h / 2), RY: svgFloat(h / 2),
				Style: style,
			})
		case diagram.BlockShapeStadium:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: svgFloat(h / 2), RY: svgFloat(h / 2),
				Style: style,
			})
		default:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: 5, RY: 5,
				Style: style,
			})
		}

		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(cy),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize),
			Content: n.Label,
		})
	}
	return elems
}

func renderEdges(d *diagram.BlockDiagram, l *layout.Result, pad, fontSize float64) []any {
	edgeKeys := make([]graph.EdgeID, 0, len(l.Edges))
	for eid := range l.Edges {
		edgeKeys = append(edgeKeys, eid)
	}
	sort.Slice(edgeKeys, func(i, j int) bool {
		if edgeKeys[i].From != edgeKeys[j].From {
			return edgeKeys[i].From < edgeKeys[j].From
		}
		return edgeKeys[i].To < edgeKeys[j].To
	})

	edgeQueue := make(map[string][]diagram.BlockEdge)
	for _, e := range d.Edges {
		key := e.From + "->" + e.To
		edgeQueue[key] = append(edgeQueue[key], e)
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		if len(el.Points) < 2 {
			continue
		}
		key := eid.From + "->" + eid.To
		var edge diagram.BlockEdge
		if candidates := edgeQueue[key]; len(candidates) > 0 {
			edge = candidates[0]
			edgeQueue[key] = candidates[1:]
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}
		style := "stroke:#333;stroke-width:1.5;fill:none"
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: "url(#block-arrow)",
			})
		} else {
			var b strings.Builder
			fmt.Fprintf(&b, "M%.2f,%.2f", pts[0].X, pts[0].Y)
			for _, p := range pts[1:] {
				fmt.Fprintf(&b, " L%.2f,%.2f", p.X, p.Y)
			}
			elems = append(elems, &path{D: b.String(), Style: style, MarkerEnd: "url(#block-arrow)"})
		}

		if edge.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
				Content: edge.Label,
			})
		}
	}
	return elems
}

func buildArrowMarker() marker {
	return marker{
		ID: "block-arrow", ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: "fill:#333"}},
	}
}
