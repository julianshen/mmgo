package flowchart

import (
	"fmt"
	"math"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

type bbox struct {
	MinX, MinY, MaxX, MaxY float64
}

// subgraphBBox returns the bounding box of the given nodes' layout
// rects. Returns ok=false when no nodes contributed (empty subgraph or
// every node missing from the layout) so callers can skip the box
// entirely instead of formatting `±Inf` / `NaN` into SVG attributes.
func subgraphBBox(nodes []diagram.Node, layoutNodes map[string]layout.NodeLayout) (b bbox, ok bool) {
	b = bbox{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
	for _, n := range nodes {
		nl, exists := layoutNodes[n.ID]
		if !exists {
			continue
		}
		b.expand(nl.X, nl.Y, nl.Width, nl.Height)
		ok = true
	}
	return b, ok
}

func (b *bbox) expand(cx, cy, w, h float64) {
	left := cx - w/2
	right := cx + w/2
	top := cy - h/2
	bottom := cy + h/2
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

func allDescendantNodes(sg *diagram.Subgraph) []diagram.Node {
	var nodes []diagram.Node
	nodes = append(nodes, sg.Nodes...)
	for i := range sg.Children {
		nodes = append(nodes, allDescendantNodes(&sg.Children[i])...)
	}
	return nodes
}

// allNodes walks the diagram and returns every node — top-level plus
// every node nested in a subgraph. Per the AST contract
// (pkg/diagram/flowchart.go), a node inside a subgraph is stored
// ONLY in that subgraph's Nodes slice, never duplicated at the top.
func allNodes(d *diagram.FlowchartDiagram) []diagram.Node {
	nodes := append([]diagram.Node(nil), d.Nodes...)
	for i := range d.Subgraphs {
		nodes = append(nodes, allDescendantNodes(&d.Subgraphs[i])...)
	}
	return nodes
}

// allEdges walks the diagram and returns every edge. Subgraph-scoped
// edges (declared inside a `subgraph ... end` block) live in the
// containing Subgraph.Edges and would otherwise be silently dropped by
// the renderer.
func allEdges(d *diagram.FlowchartDiagram) []diagram.Edge {
	edges := append([]diagram.Edge(nil), d.Edges...)
	var walk func(sgs []diagram.Subgraph)
	walk = func(sgs []diagram.Subgraph) {
		for i := range sgs {
			edges = append(edges, sgs[i].Edges...)
			walk(sgs[i].Children)
		}
	}
	walk(d.Subgraphs)
	return edges
}

func renderSubgraphGroup(sg diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) *Group {
	g := &Group{ID: sg.ID}

	allNodes := allDescendantNodes(&sg)
	bb, ok := subgraphBBox(allNodes, l.Nodes)
	if ok {
		const sgPad = 15.0
		rx := bb.MinX - sgPad + pad
		ry := bb.MinY - sgPad + pad
		rw := bb.MaxX - bb.MinX + 2*sgPad
		rh := bb.MaxY - bb.MinY + 2*sgPad
		g.Children = append(g.Children,
			&Rect{
				X: rx, Y: ry, Width: rw, Height: rh,
				RX: 5, RY: 5,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", th.SubgraphFill, th.SubgraphStroke, defaultStrokeWidth),
			},
			&Text{
				X: rx + 10, Y: ry + 18,
				FontSize: fontSize,
				Style:    fmt.Sprintf("fill:%s;font-size:%gpx", th.SubgraphText, fontSize),
				Content:  sg.Label,
			},
		)
	}

	for i := range sg.Children {
		g.Children = append(g.Children, renderSubgraphGroup(sg.Children[i], l, pad, th, fontSize))
	}
	return g
}

func renderSubgraphs(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	var elems []any
	for _, sg := range d.Subgraphs {
		elems = append(elems, renderSubgraphGroup(sg, l, pad, th, fontSize))
	}
	return elems
}
