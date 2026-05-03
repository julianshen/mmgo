package state

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
	defaultFontSize    = 14.0
	defaultPadding     = 20.0
	defaultStrokeWidth = 1.5
	minStateW          = 100.0
	minStateH          = 40.0
	statePadX          = 20.0
	statePadY          = 12.0
	// pseudoNodeR sizes the layout box dagre reserves for each pseudo
	// (start/end) node — kept slightly larger than the visual radii so
	// edges clip at the box and the glyph sits comfortably inside.
	pseudoNodeR       = 10.0
	startDotR         = 7.0
	endRingR          = 9.0
	endDotR           = 4.0
	forkBarW          = 60.0
	forkBarH          = 6.0
	choiceSize        = 30.0
	pseudoStartPrefix = "__start_"
	pseudoEndPrefix   = "__end_"
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.StateDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("state render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

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
			from = fmt.Sprintf("%s%d__", pseudoStartPrefix, startIdx)
			g.SetNode(from, graph.NodeAttrs{Width: pseudoNodeR * 2, Height: pseudoNodeR * 2})
		}
		if to == "[*]" {
			startIdx++
			to = fmt.Sprintf("%s%d__", pseudoEndPrefix, startIdx)
			g.SetNode(to, graph.NodeAttrs{Width: pseudoNodeR * 2, Height: pseudoNodeR * 2})
		}
		g.SetEdge(from, to, graph.EdgeAttrs{Label: t.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: svgutil.RankDirFor(d.Direction)})
	pad := defaultPadding

	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	children = append(children, &defs{Markers: []marker{buildArrowMarker(th)}})
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	children = append(children, renderEdges(d, l, pad, fontSize, ruler, th)...)
	children = append(children, renderNodes(allStates, l, pad, fontSize, th)...)

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

// titleBandHeight is the vertical band reserved for the state's
// title row inside its rounded rect. Shared between sizing and
// rendering so the description divider lands on the same y as
// stateNodeSize accounted for.
func titleBandHeight(fontSize float64) float64 {
	return fontSize + 2*statePadY
}

// descLineHeight is the per-line height of the description
// compartment. Shared between sizing and rendering for the same
// reason as titleBandHeight.
func descLineHeight(fontSize float64) float64 {
	return fontSize + 2
}

func stateNodeSize(s diagram.StateDef, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	switch s.Kind {
	case diagram.StateKindFork, diagram.StateKindJoin:
		return forkBarW, forkBarH
	case diagram.StateKindChoice:
		return choiceSize, choiceSize
	}
	tw, _ := ruler.Measure(s.Label, fontSize)
	w = tw + 2*statePadX
	h = titleBandHeight(fontSize)
	if s.Description != "" {
		descLines := strings.Split(s.Description, "\n")
		for _, line := range descLines {
			lw, _ := ruler.Measure(line, fontSize-1)
			if lw+2*statePadX > w {
				w = lw + 2*statePadX
			}
		}
		h += statePadY + float64(len(descLines))*descLineHeight(fontSize)
	}
	if w < minStateW {
		w = minStateW
	}
	if h < minStateH {
		h = minStateH
	}
	return w, h
}

func renderNodes(states []diagram.StateDef, l *layout.Result, pad, fontSize float64, th Theme) []any {
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
				Style: fmt.Sprintf("fill:%s;stroke:none", th.PseudoMark),
			})
		case diagram.StateKindChoice:
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				cx, cy-choiceSize/2,
				cx+choiceSize/2, cy,
				cx, cy+choiceSize/2,
				cx-choiceSize/2, cy)
			elems = append(elems, &polygon{
				Points: pts,
				Style:  fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.StateFill, th.ChoiceFill),
			})
		default:
			w := nl.Width
			h := nl.Height
			x := cx - w/2
			y := cy - h/2
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: 8, RY: 8,
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.StateFill, th.StateStroke),
			})
			if s.Description != "" {
				titleH := titleBandHeight(fontSize)
				elems = append(elems, &text{
					X: svgFloat(cx), Y: svgFloat(y + titleH/2),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.StateText, fontSize),
					Content: s.Label,
				})
				elems = append(elems, &line{
					X1: svgFloat(x), Y1: svgFloat(y + titleH),
					X2: svgFloat(x + w), Y2: svgFloat(y + titleH),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.StateStroke),
				})
				descLines := strings.Split(s.Description, "\n")
				lineH := descLineHeight(fontSize)
				for i, ln := range descLines {
					ly := y + titleH + statePadY/2 + float64(i)*lineH + lineH/2
					elems = append(elems, &text{
						X: svgFloat(cx), Y: svgFloat(ly),
						Anchor: "middle", Dominant: "central",
						Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.StateText, fontSize-1),
						Content: ln,
					})
				}
			} else {
				elems = append(elems, &text{
					X: svgFloat(cx), Y: svgFloat(cy),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.StateText, fontSize),
					Content: s.Label,
				})
			}
		}
	}

	pseudoIDs := make([]string, 0)
	for id := range l.Nodes {
		if isPseudoNode(id) {
			pseudoIDs = append(pseudoIDs, id)
		}
	}
	sort.Strings(pseudoIDs)
	for _, id := range pseudoIDs {
		nl := l.Nodes[id]
		cx := nl.X + pad
		cy := nl.Y + pad
		if isStartNode(id) {
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(startDotR),
				Style: fmt.Sprintf("fill:%s;stroke:none", th.PseudoMark),
			})
		} else {
			// End glyph: outer outlined ring with a smaller filled dot
			// inside. Without the wider gap (endRingR vs endDotR), the
			// ring reads as a slightly thicker dot and loses its
			// "stop"/end semantics against a white background.
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(endRingR),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.Background, th.PseudoMark),
			})
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(endDotR),
				Style: fmt.Sprintf("fill:%s;stroke:none", th.PseudoMark),
			})
		}
	}
	return elems
}

func renderEdges(d *diagram.StateDiagram, l *layout.Result, pad, fontSize float64, ruler *textmeasure.Ruler, th Theme) []any {
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
		// Clip endpoints to source/target node boundaries so the
		// marker-end arrowhead lands on the edge of the destination
		// shape, not buried inside it. Cache direction references
		// before mutating either endpoint — pts[1]/pts[len-2] alias
		// for 2-point edges.
		srcDir := pts[1]
		dstDir := pts[len(pts)-2]
		if src, ok := l.Nodes[eid.From]; ok {
			x, y := clipNodeEdge(eid.From, src, pad, srcDir)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			x, y := clipNodeEdge(eid.To, dst, pad, dstDir)
			pts[len(pts)-1] = layout.Point{X: x, Y: y}
		}

		style := fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke)
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: "url(#state-arrow)",
			})
		} else {
			elems = append(elems, &path{
				D:         svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension),
				Style:     style,
				MarkerEnd: "url(#state-arrow)",
			})
		}

		origFrom := eid.From
		origTo := eid.To
		if isPseudoNode(origFrom) {
			origFrom = "[*]"
		}
		if isPseudoNode(origTo) {
			origTo = "[*]"
		}
		key := origFrom + "->" + origTo
		if candidates := transMap[key]; len(candidates) > 0 {
			t := candidates[0]
			transMap[key] = candidates[1:]
			if t.Label != "" {
				base := layout.Point{X: el.LabelPos.X + pad, Y: el.LabelPos.Y + pad}
				p := labelPosition(pts, base)
				lines := strings.Split(t.Label, "\n")
				// Chip width is the widest line; height grows per line.
				lineH := fontSize + 2
				maxW := 0.0
				for _, ln := range lines {
					lw, _ := ruler.Measure(ln, fontSize-1)
					if lw > maxW {
						maxW = lw
					}
				}
				totalH := float64(len(lines)) * lineH
				elems = append(elems, svgutil.LabelChip(p.X, p.Y, maxW, totalH, 3, th.LabelBackdrop, 0))
				// Vertically centre the multi-line block on p.Y.
				topY := p.Y - totalH/2 + lineH/2
				for i, ln := range lines {
					elems = append(elems, &text{
						X: svgFloat(p.X), Y: svgFloat(topY + float64(i)*lineH),
						Anchor: "middle", Dominant: "central",
						Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EdgeText, fontSize-1),
						Content: ln,
					})
				}
			}
		}
	}
	return elems
}

// labelPosition nudges the layout-emitted label point to the side of
// the edge so labels on nearby edges don't pile on the same midpoint.
// The offset is always on the same side relative to the edge tangent
// (clockwise 90° in SVG's Y-down coordinates), so anti-parallel edges
// land on opposite sides and naturally separate — the cyclic-cluster
// case this targets. Co-directional parallel edges still collide and
// would need edge-index alternation to fully resolve.
func labelPosition(pts []layout.Point, base layout.Point) layout.Point {
	if len(pts) < 2 {
		return base
	}
	mid := len(pts) / 2 // guaranteed ≥ 1 since len(pts) ≥ 2
	dx := pts[mid].X - pts[mid-1].X
	dy := pts[mid].Y - pts[mid-1].Y
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		return base
	}
	const perpOffset = 10.0
	return layout.Point{
		X: base.X - dy/length*perpOffset,
		Y: base.Y + dx/length*perpOffset,
	}
}

func isPseudoNode(id string) bool {
	return isStartNode(id) || isEndNode(id)
}

// clipNodeEdge picks the right boundary clip for a state node. Regular
// states are rounded rects so a rect clip suffices; pseudo (start/end)
// nodes are circles, and clipping to their visible radius keeps the
// arrowhead tucked against the glyph instead of floating in the
// 20×20 layout box reserved around it.
func clipNodeEdge(id string, n layout.NodeLayout, pad float64, dir layout.Point) (float64, float64) {
	cx := n.X + pad
	cy := n.Y + pad
	if isStartNode(id) {
		return svgutil.ClipToCircleEdge(cx, cy, startDotR, dir.X, dir.Y)
	}
	if isEndNode(id) {
		return svgutil.ClipToCircleEdge(cx, cy, endRingR, dir.X, dir.Y)
	}
	return svgutil.ClipToRectEdge(cx, cy, n.Width, n.Height, dir.X, dir.Y)
}

func isStartNode(id string) bool {
	return len(id) > len(pseudoStartPrefix) && id[:len(pseudoStartPrefix)] == pseudoStartPrefix
}

func isEndNode(id string) bool {
	return len(id) > len(pseudoEndPrefix) && id[:len(pseudoEndPrefix)] == pseudoEndPrefix
}

// Width/height 12 was chosen empirically: 8 was barely visible at the
// default font size against the 1.5px stroke; mmdc's arrows render
// around 10–12px wide.
func buildArrowMarker(th Theme) marker {
	return marker{
		ID: "state-arrow", ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", th.EdgeStroke)}},
	}
}
