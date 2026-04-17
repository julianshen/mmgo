package c4

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 13.0
	defaultPadding  = 20.0
	titleH          = 30.0
	boxPadX         = 15.0
	boxPadY         = 10.0
	minBoxW         = 160.0
	minBoxH         = 80.0
	kindLabelH      = 18.0
)

// Colors per C4 convention: people = blue, systems = dark blue,
// external = gray, containers = medium blue, components = light blue.
type palette struct {
	fill, stroke, text string
}

var palettes = map[diagram.C4ElementKind]palette{
	diagram.C4ElementPerson:       {"#08427B", "#073B6F", "white"},
	diagram.C4ElementPersonExt:    {"#686868", "#4D4D4D", "white"},
	diagram.C4ElementSystem:       {"#1168BD", "#0B4884", "white"},
	diagram.C4ElementSystemExt:    {"#999999", "#6B6B6B", "white"},
	diagram.C4ElementSystemDB:     {"#1168BD", "#0B4884", "white"},
	diagram.C4ElementContainer:    {"#438DD5", "#3C7FC0", "white"},
	diagram.C4ElementContainerDB: {"#438DD5", "#3C7FC0", "white"},
	diagram.C4ElementComponent:    {"#85BBF0", "#78A8D8", "#000"},
}

type Options struct {
	FontSize float64
}

func Render(d *diagram.C4Diagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("c4 render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("c4 render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	for _, e := range d.Elements {
		w, h := elementSize(e, ruler, fontSize)
		g.SetNode(e.ID, graph.NodeAttrs{Label: e.Label, Width: w, Height: h})
	}
	for _, r := range d.Relations {
		g.SetEdge(r.From, r.To, graph.EdgeAttrs{Label: r.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
	pad := defaultPadding

	titleOffset := 0.0
	if d.Title != "" {
		titleOffset = titleH
	}
	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad + titleOffset

	var children []any
	children = append(children, &defs{Markers: []marker{buildArrowMarker()}})
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:#fff;stroke:none",
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize+2),
			Content: d.Title,
		})
	}

	children = append(children, renderEdges(d, l, pad, titleOffset, fontSize)...)
	children = append(children, renderElements(d, l, pad, titleOffset, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("c4 render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func elementSize(e diagram.C4Element, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	lw, _ := ruler.Measure(e.Label, fontSize)
	w = lw + 2*boxPadX
	h = kindLabelH + fontSize + 2*boxPadY
	if e.Technology != "" {
		tw, _ := ruler.Measure("["+e.Technology+"]", fontSize-2)
		if tw+2*boxPadX > w {
			w = tw + 2*boxPadX
		}
		h += fontSize - 2
	}
	if e.Description != "" {
		dw, _ := ruler.Measure(e.Description, fontSize-2)
		if dw+2*boxPadX > w {
			w = dw + 2*boxPadX
		}
		h += fontSize - 2
	}
	if w < minBoxW {
		w = minBoxW
	}
	if h < minBoxH {
		h = minBoxH
	}
	return w, h
}

func renderElements(d *diagram.C4Diagram, l *layout.Result, pad, titleOff, fontSize float64) []any {
	var elems []any
	for _, e := range d.Elements {
		nl, ok := l.Nodes[e.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad + titleOff
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		p := palettes[e.Kind]
		if p.fill == "" {
			p = palette{"#1168BD", "#0B4884", "white"}
		}

		rx := svgFloat(0)
		if e.Kind == diagram.C4ElementPerson || e.Kind == diagram.C4ElementPersonExt {
			rx = svgFloat(h / 4)
		}

		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(y),
			Width: svgFloat(w), Height: svgFloat(h),
			RX: rx, RY: rx,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", p.fill, p.stroke),
		})

		curY := y + kindLabelH
		kindLabel := kindDisplayLabel(e.Kind)
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(y + kindLabelH/2 + 2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-style:italic;opacity:0.85", p.text, fontSize-3),
			Content: kindLabel,
		})
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(curY + fontSize/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", p.text, fontSize),
			Content: e.Label,
		})
		curY += fontSize + 4
		if e.Technology != "" {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", p.text, fontSize-2),
				Content: "[" + e.Technology + "]",
			})
			curY += fontSize - 2
		}
		if e.Description != "" {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;opacity:0.9", p.text, fontSize-2),
				Content: e.Description,
			})
		}
	}
	return elems
}

func kindDisplayLabel(k diagram.C4ElementKind) string {
	switch k {
	case diagram.C4ElementPerson:
		return "«Person»"
	case diagram.C4ElementPersonExt:
		return "«Person, External»"
	case diagram.C4ElementSystem:
		return "«System»"
	case diagram.C4ElementSystemExt:
		return "«System, External»"
	case diagram.C4ElementSystemDB:
		return "«Database»"
	case diagram.C4ElementContainer:
		return "«Container»"
	case diagram.C4ElementContainerDB:
		return "«Database»"
	case diagram.C4ElementComponent:
		return "«Component»"
	default:
		return ""
	}
}

func renderEdges(d *diagram.C4Diagram, l *layout.Result, pad, titleOff, fontSize float64) []any {
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

	relQueue := make(map[string][]diagram.C4Relation)
	for _, r := range d.Relations {
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
		var rel diagram.C4Relation
		if len(candidates) > 0 {
			rel = candidates[0]
			relQueue[key] = candidates[1:]
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad + titleOff}
		}

		style := "stroke:#333;stroke-width:1.5;fill:none"
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: "url(#c4-arrow)",
			})
		} else {
			pathD := fmt.Sprintf("M%.2f,%.2f", pts[0].X, pts[0].Y)
			for _, p := range pts[1:] {
				pathD += fmt.Sprintf(" L%.2f,%.2f", p.X, p.Y)
			}
			elems = append(elems, &path{D: pathD, Style: style, MarkerEnd: "url(#c4-arrow)"})
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad + titleOff
			label := rel.Label
			if rel.Technology != "" {
				label = rel.Label + " [" + rel.Technology + "]"
			}
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-2),
				Content: label,
			})
		}
	}
	return elems
}

func buildArrowMarker() marker {
	return marker{
		ID: "c4-arrow", ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: "fill:#333"}},
	}
}
