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

// subgraphTitleBand is the height reserved above a subgraph's contents
// for its centered title label. Bumping it further would push inner
// subgraph rects up into their parent's other content nodes, so the
// padding has to stay tight (~14px around the text); see
// subgraphTitleY for how we compensate for the rasteriser's
// dominant-baseline interpretation.
func subgraphTitleBand(fontSize float64) float64 { return fontSize + 14 }

// subgraphTitleY anchors the title at 60% of the band height rather
// than 50% because tdewolff/canvas treats dominant-baseline="central"
// closer to the alphabetic baseline than browsers do, pushing the
// visible text up into the rect's top border.
func subgraphTitleY(ry, titleBand float64) float64 {
	return ry + titleBand*0.6
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

// expandCorners expands the bbox to include the rectangle described
// by its corner coordinates.
func (b *bbox) expandCorners(x1, y1, x2, y2 float64) {
	if x1 < b.MinX {
		b.MinX = x1
	}
	if x2 > b.MaxX {
		b.MaxX = x2
	}
	if y1 < b.MinY {
		b.MinY = y1
	}
	if y2 > b.MaxY {
		b.MaxY = y2
	}
}

type renderedRect struct{ X, Y, W, H float64 }

func renderSubgraphGroup(sg *diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) (*Group, renderedRect) {
	g := &Group{ID: sg.ID}

	bb, ok := subgraphBBox(sg.AllNodes(), l.Nodes)

	for i := range sg.Children {
		cg, cr := renderSubgraphGroup(sg.Children[i], l, pad, th, fontSize)
		g.Children = append(g.Children, cg)
		if cr.W == 0 {
			continue
		}
		if !ok {
			bb = bbox{
				MinX: cr.X - pad, MinY: cr.Y - pad,
				MaxX: cr.X + cr.W - pad, MaxY: cr.Y + cr.H - pad,
			}
			ok = true
			continue
		}
		bb.expandCorners(cr.X-pad, cr.Y-pad, cr.X+cr.W-pad, cr.Y+cr.H-pad)
	}

	var rr renderedRect
	if ok {
		const sidePad = 8.0
		const nestingPad = 10.0
		titleBand := subgraphTitleBand(fontSize)
		childDepth := float64(maxSubgraphDepth(sg.Children))
		topOffset := (1 + childDepth) * titleBand

		const avgCharW = 0.6
		labelBoxW := float64(len(sg.Label))*fontSize*avgCharW + 2*sidePad + 8
		contentW := bb.MaxX - bb.MinX + 2*sidePad + 2*nestingPad
		rw := contentW
		if labelBoxW > rw {
			rw = labelBoxW
		}

		rx := bb.MinX - sidePad - nestingPad + pad
		if rw > contentW {
			rx -= (rw - contentW) / 2
		}
		ry := bb.MinY - topOffset + pad
		rh := bb.MaxY - bb.MinY + sidePad + nestingPad + topOffset
		rr = renderedRect{X: rx, Y: ry, W: rw, H: rh}

		rectElem := &Rect{
			X: svgFloat(rx), Y: svgFloat(ry), Width: svgFloat(rw), Height: svgFloat(rh),
			RX: 5, RY: 5,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.2f", th.SubgraphFill, th.SubgraphStroke, defaultStrokeWidth),
		}
		textElem := &Text{
			X: svgFloat(rx + rw/2), Y: svgFloat(subgraphTitleY(ry, titleBand)),
			Anchor: "middle", Dominant: "central",
			FontSize: svgFloat(fontSize),
			Style:    fmt.Sprintf("fill:%s;font-size:%.2fpx", th.SubgraphText, fontSize),
			Content:  sg.Label,
		}
		g.Children = append([]any{rectElem, textElem}, g.Children...)
	}
	return g, rr
}

func maxSubgraphDepth(sgs []*diagram.Subgraph) int {
	best := 0
	for _, sg := range sgs {
		if d := 1 + maxSubgraphDepth(sg.Children); d > best {
			best = d
		}
	}
	return best
}

func renderSubgraphs(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) ([]any, renderedRect) {
	var elems []any
	var overall renderedRect
	for _, sg := range d.Subgraphs {
		g, cr := renderSubgraphGroup(sg, l, pad, th, fontSize)
		elems = append(elems, g)
		if cr.W == 0 {
			continue
		}
		if overall.W == 0 {
			overall = cr
			continue
		}
		overall.X = math.Min(overall.X, cr.X)
		overall.Y = math.Min(overall.Y, cr.Y)
		right := math.Max(overall.X+overall.W, cr.X+cr.W)
		bottom := math.Max(overall.Y+overall.H, cr.Y+cr.H)
		overall.W = right - overall.X
		overall.H = bottom - overall.Y
	}
	return elems, overall
}
