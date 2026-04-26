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

// subgraphTitleY returns the Y anchor for a subgraph's title centered
// inside its title band. tdewolff/canvas treats dominant-baseline
// ="central" closer to the alphabetic baseline than browsers do,
// pushing the visible text up so its top can clip the rect's top
// border. Anchoring at 60% of the band height (instead of 50%)
// shifts the visible glyphs ~3px lower, restoring a clean gap.
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

// renderedRect records the final rect position/dimensions of a rendered
// subgraph so that the parent can expand its own bbox to contain it.
type renderedRect struct{ X, Y, W, H float64 }

func (r renderedRect) left() float64   { return r.X }
func (r renderedRect) right() float64  { return r.X + r.W }
func (r renderedRect) top() float64    { return r.Y }
func (r renderedRect) bottom() float64 { return r.Y + r.H }

func renderSubgraphGroup(sg *diagram.Subgraph, l *layout.Result, pad float64, th Theme, fontSize float64) (*Group, renderedRect) {
	g := &Group{ID: sg.ID}

	var childRects []renderedRect
	for i := range sg.Children {
		cg, cr := renderSubgraphGroup(sg.Children[i], l, pad, th, fontSize)
		g.Children = append(g.Children, cg)
		childRects = append(childRects, cr)
	}

	bb, ok := subgraphBBox(sg.AllNodes(), l.Nodes)

	for _, cr := range childRects {
		if cr.W == 0 && cr.H == 0 {
			continue
		}
		cl := cr.left() - pad
		ct := cr.top() - pad
		cr_ := cr.right() - pad
		cb := cr.bottom() - pad
		if !ok {
			bb = bbox{MinX: cl, MinY: ct, MaxX: cr_, MaxY: cb}
			ok = true
			continue
		}
		if cl < bb.MinX {
			bb.MinX = cl
		}
		if ct < bb.MinY {
			bb.MinY = ct
		}
		if cr_ > bb.MaxX {
			bb.MaxX = cr_
		}
		if cb > bb.MaxY {
			bb.MaxY = cb
		}
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
