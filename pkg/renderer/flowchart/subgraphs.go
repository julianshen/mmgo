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

func renderSubgraphGroup(sg *diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) *Group {
	g := &Group{ID: sg.ID}

	bb, ok := subgraphBBox(sg.AllNodes(), l.Nodes)
	if ok {
		// Title sits in its own band at the top of the rect. The side
		// and bottom paddings stay thin; the top padding grows to
		// fontSize + breathing room so the label never collides with
		// either the border or the first row of nodes.
		const sidePad = 8.0
		titleBand := fontSize + 14
		rx := bb.MinX - sidePad + pad
		ry := bb.MinY - titleBand + pad
		rw := bb.MaxX - bb.MinX + 2*sidePad
		rh := bb.MaxY - bb.MinY + sidePad + titleBand
		g.Children = append(g.Children,
			&Rect{
				X: svgFloat(rx), Y: svgFloat(ry), Width: svgFloat(rw), Height: svgFloat(rh),
				RX: 5, RY: 5,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.SubgraphFill, th.SubgraphStroke, defaultStrokeWidth),
			},
			&Text{
				X: svgFloat(rx + rw/2), Y: svgFloat(ry + titleBand/2),
				Anchor: "middle", Dominant: "central",
				FontSize: svgFloat(fontSize),
				Style:    fmt.Sprintf("fill:%s;font-size:%.2fpx", th.SubgraphText, fontSize),
				Content:  sg.Label,
			},
		)
	}

	for i := range sg.Children {
		g.Children = append(g.Children, renderSubgraphGroup(sg.Children[i], l, pad, th, fontSize))
	}
	return g
}

// maxSubgraphDepth returns the deepest nesting level across the given
// top-level subgraphs. A flat list of subgraphs has depth 1; a subgraph
// containing one nested subgraph has depth 2, and so on. Returns 0 when
// there are no subgraphs.
func maxSubgraphDepth(sgs []*diagram.Subgraph) int {
	best := 0
	for _, sg := range sgs {
		if d := 1 + maxSubgraphDepth(sg.Children); d > best {
			best = d
		}
	}
	return best
}

func renderSubgraphs(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	var elems []any
	for _, sg := range d.Subgraphs {
		elems = append(elems, renderSubgraphGroup(sg, l, pad, th, fontSize))
	}
	return elems
}
