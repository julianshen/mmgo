package mindmap

import (
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	nodePadX        = 20.0
	nodePadY        = 10.0
	minNodeW        = 80.0
	minNodeH        = 35.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.MindmapDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("mindmap render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("mindmap render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	ids := &idGen{m: make(map[*diagram.MindmapNode]string)}
	if d.Root != nil {
		addNodeToGraph(g, d.Root, ruler, fontSize, ids)
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
	pad := defaultPadding

	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Root != nil {
		children = append(children, renderEdges(l, pad, th)...)
		children = append(children, renderNodes(d.Root, l, pad, fontSize, 0, ids, th)...)
	}

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("mindmap render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if v != v || v < 0 {
		return 0
	}
	return v
}

type idGen struct {
	m       map[*diagram.MindmapNode]string
	counter int
}

func (g *idGen) id(n *diagram.MindmapNode) string {
	if id, ok := g.m[n]; ok {
		return id
	}
	g.counter++
	id := fmt.Sprintf("mm_%d", g.counter)
	g.m[n] = id
	return id
}

func addNodeToGraph(g *graph.Graph, node *diagram.MindmapNode, ruler *textmeasure.Ruler, fontSize float64, ids *idGen) {
	id := ids.id(node)
	w, h := nodeSize(node.Text, ruler, fontSize)
	g.SetNode(id, graph.NodeAttrs{Label: node.Text, Width: w, Height: h})
	for _, child := range node.Children {
		addNodeToGraph(g, child, ruler, fontSize, ids)
		g.SetEdge(id, ids.id(child), graph.EdgeAttrs{})
	}
}

func nodeSize(text string, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, th := ruler.Measure(text, fontSize)
	w = tw + 2*nodePadX
	h = th + 2*nodePadY
	if w < minNodeW {
		w = minNodeW
	}
	if h < minNodeH {
		h = minNodeH
	}
	return w, h
}

func renderNodes(node *diagram.MindmapNode, l *layout.Result, pad, fontSize float64, depth int, ids *idGen, th Theme) []any {
	var elems []any
	id := ids.id(node)
	nl, ok := l.Nodes[id]
	if !ok {
		return elems
	}

	cx := nl.X + pad
	cy := nl.Y + pad
	w := nl.Width
	h := nl.Height
	x := cx - w/2
	y := cy - h/2

	color := th.LevelColors[depth%len(th.LevelColors)]
	rx := svgFloat(0)
	switch node.Shape {
	case diagram.MindmapShapeRound, diagram.MindmapShapeCloud:
		rx = svgFloat(h / 2)
	case diagram.MindmapShapeSquare:
		rx = 0
	default:
		rx = 5
	}

	elems = append(elems, &rect{
		X: svgFloat(x), Y: svgFloat(y),
		Width: svgFloat(w), Height: svgFloat(h),
		RX: rx, RY: rx,
		Style: fmt.Sprintf("fill:%s;stroke:none", color),
	})
	elems = append(elems, &text{
		X: svgFloat(cx), Y: svgFloat(cy),
		Anchor: "middle", Dominant: "central",
		Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.NodeText, fontSize),
		Content: node.Text,
	})

	for _, child := range node.Children {
		elems = append(elems, renderNodes(child, l, pad, fontSize, depth+1, ids, th)...)
	}
	return elems
}

func renderEdges(l *layout.Result, pad float64, th Theme) []any {
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
	var elems []any
	style := fmt.Sprintf("stroke:%s;stroke-width:2;fill:none", th.EdgeStroke)
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		if len(el.Points) < 2 {
			continue
		}
		if len(el.Points) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(el.Points[0].X + pad), Y1: svgFloat(el.Points[0].Y + pad),
				X2: svgFloat(el.Points[1].X + pad), Y2: svgFloat(el.Points[1].Y + pad),
				Style: style,
			})
			continue
		}
		d := fmt.Sprintf("M%.2f,%.2f", el.Points[0].X+pad, el.Points[0].Y+pad)
		for _, p := range el.Points[1:] {
			d += fmt.Sprintf(" L%.2f,%.2f", p.X+pad, p.Y+pad)
		}
		elems = append(elems, &path{D: d, Style: style})
	}
	return elems
}
