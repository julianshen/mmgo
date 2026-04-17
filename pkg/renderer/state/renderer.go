package state

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
	defaultFontSize    = 14.0
	defaultPadding     = 20.0
	defaultStrokeWidth = 1.5
	minStateW          = 100.0
	minStateH          = 40.0
	statePadX          = 20.0
	statePadY          = 12.0
	startEndR          = 10.0
	forkBarW           = 60.0
	forkBarH           = 6.0
	choiceSize         = 30.0
)

type Options struct {
	FontSize float64
}

func Render(d *diagram.StateDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("state render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("state render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	startIdx := 0
	allStates := collectAllStates(d.States)
	for _, s := range allStates {
		w, h := stateNodeSize(s, ruler, fontSize)
		g.SetNode(s.ID, graph.NodeAttrs{Label: s.Label, Width: w, Height: h})
	}
	for _, t := range d.Transitions {
		from, to := t.From, t.To
		if from == "[*]" {
			startIdx++
			from = fmt.Sprintf("__start_%d__", startIdx)
			g.SetNode(from, graph.NodeAttrs{Width: startEndR * 2, Height: startEndR * 2})
		}
		if to == "[*]" {
			startIdx++
			to = fmt.Sprintf("__end_%d__", startIdx)
			g.SetNode(to, graph.NodeAttrs{Width: startEndR * 2, Height: startEndR * 2})
		}
		g.SetEdge(from, to, graph.EdgeAttrs{Label: t.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
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
	children = append(children, renderNodes(allStates, l, pad, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("state render: %w", err)
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func collectAllStates(states []diagram.StateDef) []diagram.StateDef {
	var all []diagram.StateDef
	for _, s := range states {
		all = append(all, s)
		if len(s.Children) > 0 {
			all = append(all, collectAllStates(s.Children)...)
		}
	}
	return all
}

func stateNodeSize(s diagram.StateDef, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	switch s.Kind {
	case diagram.StateKindFork, diagram.StateKindJoin:
		return forkBarW, forkBarH
	case diagram.StateKindChoice:
		return choiceSize, choiceSize
	}
	tw, th := ruler.Measure(s.Label, fontSize)
	w = tw + 2*statePadX
	h = th + 2*statePadY
	if w < minStateW {
		w = minStateW
	}
	if h < minStateH {
		h = minStateH
	}
	return w, h
}

func renderNodes(states []diagram.StateDef, l *layout.Result, pad, fontSize float64) []any {
	var elems []any
	for _, s := range states {
		nl, ok := l.Nodes[s.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad

		switch s.Kind {
		case diagram.StateKindFork, diagram.StateKindJoin:
			elems = append(elems, &rect{
				X: svgFloat(cx - forkBarW/2), Y: svgFloat(cy - forkBarH/2),
				Width: svgFloat(forkBarW), Height: svgFloat(forkBarH),
				Style: "fill:#333;stroke:none",
			})
		case diagram.StateKindChoice:
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				cx, cy-choiceSize/2,
				cx+choiceSize/2, cy,
				cx, cy+choiceSize/2,
				cx-choiceSize/2, cy)
			elems = append(elems, &polygon{Points: pts, Style: "fill:#ECECFF;stroke:#333;stroke-width:1.5"})
		default:
			w := nl.Width
			h := nl.Height
			x := cx - w/2
			y := cy - h/2
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: 8, RY: 8,
				Style: "fill:#ECECFF;stroke:#9370DB;stroke-width:1.5",
			})
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(cy),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize),
				Content: s.Label,
			})
		}
	}

	for id, nl := range l.Nodes {
		if len(id) > 8 && id[:8] == "__start_" {
			elems = append(elems, &circle{
				CX: svgFloat(nl.X + pad), CY: svgFloat(nl.Y + pad), R: svgFloat(startEndR),
				Style: "fill:#333;stroke:#333",
			})
		}
		if len(id) > 6 && id[:6] == "__end_" {
			elems = append(elems, &circle{
				CX: svgFloat(nl.X + pad), CY: svgFloat(nl.Y + pad), R: svgFloat(startEndR),
				Style: "fill:#fff;stroke:#333;stroke-width:2",
			})
			elems = append(elems, &circle{
				CX: svgFloat(nl.X + pad), CY: svgFloat(nl.Y + pad), R: svgFloat(startEndR - 3),
				Style: "fill:#333;stroke:none",
			})
		}
	}
	return elems
}

func renderEdges(d *diagram.StateDiagram, l *layout.Result, pad, fontSize float64) []any {
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

	transMap := make(map[string][]diagram.StateTransition)
	for _, t := range d.Transitions {
		from := t.From
		to := t.To
		key := from + "->" + to
		transMap[key] = append(transMap[key], t)
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		if len(el.Points) < 2 {
			continue
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}

		style := "stroke:#333;stroke-width:1.5;fill:none"
		ln := &line{
			X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
			X2: svgFloat(pts[len(pts)-1].X), Y2: svgFloat(pts[len(pts)-1].Y),
			Style: style,
		}
		ln.MarkerEnd = "url(#state-arrow)"
		elems = append(elems, ln)

		origFrom := eid.From
		origTo := eid.To
		for _, prefix := range []string{"__start_", "__end_"} {
			if len(origFrom) > len(prefix) && origFrom[:len(prefix)] == prefix {
				origFrom = "[*]"
			}
			if len(origTo) > len(prefix) && origTo[:len(prefix)] == prefix {
				origTo = "[*]"
			}
		}
		key := origFrom + "->" + origTo
		if candidates := transMap[key]; len(candidates) > 0 {
			t := candidates[0]
			transMap[key] = candidates[1:]
			if t.Label != "" {
				lx := el.LabelPos.X + pad
				ly := el.LabelPos.Y + pad
				elems = append(elems, &text{
					X: svgFloat(lx), Y: svgFloat(ly),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
					Content: t.Label,
				})
			}
		}
	}
	return elems
}

func buildArrowMarker() marker {
	return marker{
		ID: "state-arrow", ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: "fill:#333"}},
	}
}
