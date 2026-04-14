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

func subgraphBBox(nodes []diagram.Node, layoutNodes map[string]layout.NodeLayout) bbox {
	b := bbox{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
	for _, n := range nodes {
		nl, ok := layoutNodes[n.ID]
		if !ok {
			continue
		}
		left := nl.X - nl.Width/2
		right := nl.X + nl.Width/2
		top := nl.Y - nl.Height/2
		bottom := nl.Y + nl.Height/2
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
	return b
}

func allDescendantNodes(sg *diagram.Subgraph) []diagram.Node {
	var nodes []diagram.Node
	nodes = append(nodes, sg.Nodes...)
	for i := range sg.Children {
		nodes = append(nodes, allDescendantNodes(&sg.Children[i])...)
	}
	return nodes
}

func renderSubgraphGroup(sg diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) *Group {
	allNodes := allDescendantNodes(&sg)
	bb := subgraphBBox(allNodes, l.Nodes)

	const sgPad = 15.0
	rx := bb.MinX - sgPad + pad
	ry := bb.MinY - sgPad + pad
	rw := bb.MaxX - bb.MinX + 2*sgPad
	rh := bb.MaxY - bb.MinY + 2*sgPad

	g := &Group{
		ID: sg.ID,
		Children: []any{
			&Rect{
				X: rx, Y: ry, Width: rw, Height: rh,
				RX: 5, RY: 5,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.SubgraphFill, th.SubgraphStroke),
			},
			&Text{
				X: rx + 10, Y: ry + 18,
				FontSize: fontSize,
				Style:    fmt.Sprintf("fill:%s;font-size:%gpx", th.SubgraphText, fontSize),
				Content:  sg.Label,
			},
		},
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
