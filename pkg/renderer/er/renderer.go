package er

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	entityPadX      = 15.0
	entityPadY      = 10.0
	headerH         = 30.0
	attrRowH        = 20.0
	minEntityW      = 120.0
)

type Options struct {
	FontSize float64
}

func Render(d *diagram.ERDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("er render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("er render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	for _, e := range d.Entities {
		w, h := entitySize(e, ruler, fontSize)
		g.SetNode(e.Name, graph.NodeAttrs{Label: e.Name, Width: w, Height: h})
	}
	for _, r := range d.Relationships {
		g.SetEdge(r.From, r.To, graph.EdgeAttrs{Label: r.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
	pad := defaultPadding
	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:#fff;stroke:none",
	})
	if markers := buildERMarkers(d); len(markers) > 0 {
		children = append(children, &defs{Markers: markers})
	}
	children = append(children, renderEdges(d, l, pad, fontSize)...)
	children = append(children, renderEntities(d, l, pad, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("er render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func polylineD(pts []layout.Point) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "M%.2f,%.2f", pts[0].X, pts[0].Y)
	for _, p := range pts[1:] {
		fmt.Fprintf(&sb, " L%.2f,%.2f", p.X, p.Y)
	}
	return sb.String()
}

// See erStartGeoms in markers.go for why start markers are inlined
// rather than referenced via SVG marker-start.
func startMarkerGroup(c diagram.ERCardinality, start, next layout.Point) *group {
	children, refX, refY, ok := startMarkerGeom(c)
	if !ok {
		return nil
	}
	return svgutil.InlineMarkerAt(start.X, start.Y, next.X, next.Y, refX, refY, children)
}

func entitySize(e diagram.EREntity, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, _ := ruler.Measure(e.Name, fontSize)
	w = tw + 2*entityPadX
	h = headerH
	if len(e.Attributes) > 0 {
		h += entityPadY + float64(len(e.Attributes))*attrRowH
		for _, a := range e.Attributes {
			aw, _ := ruler.Measure(attrText(a), fontSize-1)
			if aw+2*entityPadX > w {
				w = aw + 2*entityPadX
			}
		}
	}
	if w < minEntityW {
		w = minEntityW
	}
	h += entityPadY
	return w, h
}

func attrText(a diagram.ERAttribute) string {
	s := a.Type + " " + a.Name
	if a.Key != diagram.ERKeyNone {
		s += " " + a.Key.String()
	}
	return s
}

func renderEntities(d *diagram.ERDiagram, l *layout.Result, pad, fontSize float64) []any {
	var elems []any
	for _, e := range d.Entities {
		nl, ok := l.Nodes[e.Name]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(y),
			Width: svgFloat(w), Height: svgFloat(h),
			Style: "fill:#ECECFF;stroke:#9370DB;stroke-width:1.5",
		})
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(y + headerH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize),
			Content: e.Name,
		})

		if len(e.Attributes) > 0 {
			sepY := y + headerH
			elems = append(elems, &line{
				X1: svgFloat(x), Y1: svgFloat(sepY),
				X2: svgFloat(x + w), Y2: svgFloat(sepY),
				Style: "stroke:#9370DB;stroke-width:1",
			})
			for i, a := range e.Attributes {
				ay := sepY + entityPadY/2 + float64(i)*attrRowH + attrRowH/2
				elems = append(elems, &text{
					X: svgFloat(x + entityPadX), Y: svgFloat(ay),
					Anchor: "start", Dominant: "central",
					Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
					Content: attrText(a),
				})
			}
		}
	}
	return elems
}

func renderEdges(d *diagram.ERDiagram, l *layout.Result, pad, fontSize float64) []any {
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

	relQueue := make(map[string][]diagram.ERRelationship)
	for _, r := range d.Relationships {
		key := r.From + "->" + r.To
		relQueue[key] = append(relQueue[key], r)
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		if len(el.Points) < 2 {
			continue
		}
		key := eid.From + "->" + eid.To
		candidates := relQueue[key]
		if len(candidates) == 0 {
			continue
		}
		rel := candidates[0]
		relQueue[key] = candidates[1:]

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}
		// pts[1] and pts[len-2] alias for 2-point edges; cache before
		// mutating either endpoint, or the dst clip reads the already-
		// clipped src as its direction reference.
		srcDir := pts[1]
		dstDir := pts[len(pts)-2]
		if src, ok := l.Nodes[eid.From]; ok {
			x, y := svgutil.ClipToRectEdge(src.X+pad, src.Y+pad, src.Width, src.Height, srcDir.X, srcDir.Y)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			last := len(pts) - 1
			x, y := svgutil.ClipToRectEdge(dst.X+pad, dst.Y+pad, dst.Width, dst.Height, dstDir.X, dstDir.Y)
			pts[last] = layout.Point{X: x, Y: y}
		}

		style := "stroke:#333;stroke-width:1.5;fill:none"
		endRef := markerRef(markerEndID(rel.ToCard))
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style:     style,
				MarkerEnd: endRef,
			})
		} else {
			elems = append(elems, &path{
				D:         polylineD(pts),
				Style:     style,
				MarkerEnd: endRef,
			})
		}
		if g := startMarkerGroup(rel.FromCard, pts[0], srcDir); g != nil {
			elems = append(elems, g)
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
				Content: rel.Label,
			})
		}
	}
	return elems
}
