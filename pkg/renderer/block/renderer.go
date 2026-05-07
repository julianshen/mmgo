package block

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
	nodePadX           = 20.0
	nodePadY           = 12.0
	minNodeW           = 80.0
	minNodeH           = 40.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.BlockDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("block render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("block render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	rankDir := layout.RankDirTB
	if d.Columns > 0 && len(d.Edges) == 0 {
		// Column layout with no edges: arrange left-to-right for visual flow.
		rankDir = layout.RankDirLR
	}

	g := graph.New()
	for _, n := range d.Nodes {
		w, h := nodeSize(n, ruler, fontSize)
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: w, Height: h})
	}
	for _, e := range d.Edges {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: rankDir})
	pad := defaultPadding

	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	if d.AccTitle != "" {
		children = append(children, &svgutil.Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgutil.Desc{Content: d.AccDescr})
	}
	children = append(children, &defs{Markers: buildArrowMarkers(th)})
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		// Frontmatter `title:` renders as a centered caption above
		// the diagram body. Uses the existing NodeText surface so
		// the heading reads against any theme.
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(14),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:14px;font-weight:bold", th.NodeText),
			Content: d.Title,
		})
	}
	children = append(children, renderEdges(d, l, pad, fontSize, th)...)
	children = append(children, renderNodes(d, l, pad, fontSize, th)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("block render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func nodeSize(n diagram.BlockNode, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, th := ruler.Measure(n.Label, fontSize)
	w = tw + 2*nodePadX
	h = th + 2*nodePadY
	if w < minNodeW {
		w = minNodeW
	}
	if h < minNodeH {
		h = minNodeH
	}
	if n.Shape == diagram.BlockShapeCircle {
		side := w
		if h > side {
			side = h
		}
		return side, side
	}
	return w, h
}

func renderNodes(d *diagram.BlockDiagram, l *layout.Result, pad, fontSize float64, th Theme) []any {
	var elems []any
	for _, n := range d.Nodes {
		nl, ok := l.Nodes[n.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		style := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.NodeFill, th.NodeStroke)
		// classDef + per-node `style` overrides land last so author
		// declarations win over the theme defaults; later
		// declarations win over earlier ones.
		if extra := mergeNodeCSS(d, n); extra != "" {
			style += ";" + extra
		}
		switch n.Shape {
		case diagram.BlockShapeDiamond:
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				cx, y, x+w, cy, cx, y+h, x, cy)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeCircle:
			r := w / 2
			if h/2 < r {
				r = h / 2
			}
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r),
				Style: style,
			})
		case diagram.BlockShapeRound:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: svgFloat(h / 2), RY: svgFloat(h / 2),
				Style: style,
			})
		case diagram.BlockShapeStadium:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: svgFloat(h / 2), RY: svgFloat(h / 2),
				Style: style,
			})
		case diagram.BlockShapeHexagon:
			// Pointy-top hex with 1/4-width chamfers on each side.
			cut := h / 2
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x+cut, y, x+w-cut, y, x+w, cy, x+w-cut, y+h, x+cut, y+h, x, cy)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeDoubleCircle:
			// Two concentric circles share the inner fill; outer is
			// drawn as a stroked-only hairline so the label still
			// reads against the inner color.
			r := w / 2
			if h/2 < r {
				r = h / 2
			}
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r),
				Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:1.5", th.NodeStroke),
			})
			elems = append(elems, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(r - 4),
				Style: style,
			})
		case diagram.BlockShapeSubroutine:
			// Outer rect plus two inner vertical bars (the classic
			// flowchart "subroutine" frame).
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				Style: style,
			})
			elems = append(elems, &line{
				X1: svgFloat(x + 6), Y1: svgFloat(y),
				X2: svgFloat(x + 6), Y2: svgFloat(y + h),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", th.NodeStroke),
			})
			elems = append(elems, &line{
				X1: svgFloat(x + w - 6), Y1: svgFloat(y),
				X2: svgFloat(x + w - 6), Y2: svgFloat(y + h),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", th.NodeStroke),
			})
		case diagram.BlockShapeCylinder:
			// Body rect plus a top ellipse (the rim) and a half-arc
			// at the bottom for visual depth.
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y + 6),
				Width: svgFloat(w), Height: svgFloat(h - 12),
				Style: style,
			})
			elems = append(elems, &path{
				D:     fmt.Sprintf("M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f Z", x, y+6, w/2, 6.0, x+w, y+6, w/2, 6.0, x, y+6),
				Style: style,
			})
			elems = append(elems, &path{
				D:     fmt.Sprintf("M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f", x, y+h-6, w/2, 6.0, x+w, y+h-6),
				Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:1.5", th.NodeStroke),
			})
		case diagram.BlockShapeAsymmetric:
			// `>flag]` — chevron-tipped rect: pentagonal flag.
			notch := h / 3
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x, y, x+w-notch, y, x+w, cy, x+w-notch, y+h, x, y+h)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeParallelogram:
			// `[/.../]` slants the top-right and bottom-left corners.
			slant := h / 3
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x+slant, y, x+w, y, x+w-slant, y+h, x, y+h)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeParallelogramAlt:
			// `[\...\]` — mirrored slant.
			slant := h / 3
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x, y, x+w-slant, y, x+w, y+h, x+slant, y+h)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeTrapezoid:
			// `[/...\]` — wider at the bottom.
			slant := h / 3
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x+slant, y, x+w-slant, y, x+w, y+h, x, y+h)
			elems = append(elems, &polygon{Points: pts, Style: style})
		case diagram.BlockShapeTrapezoidAlt:
			// `[\.../]` — wider at the top.
			slant := h / 3
			pts := fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
				x, y, x+w, y, x+w-slant, y+h, x+slant, y+h)
			elems = append(elems, &polygon{Points: pts, Style: style})
		default:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: 5, RY: 5,
				Style: style,
			})
		}

		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(cy),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.NodeText, fontSize),
			Content: n.Label,
		})
	}
	return elems
}

func renderEdges(d *diagram.BlockDiagram, l *layout.Result, pad, fontSize float64, th Theme) []any {
	edgeKeys := make([]graph.EdgeID, 0, len(l.Edges))
	for eid := range l.Edges {
		edgeKeys = append(edgeKeys, eid)
	}
	sort.Slice(edgeKeys, func(i, j int) bool {
		if edgeKeys[i].From != edgeKeys[j].From {
			return edgeKeys[i].From < edgeKeys[j].From
		}
		if edgeKeys[i].To != edgeKeys[j].To {
			return edgeKeys[i].To < edgeKeys[j].To
		}
		return edgeKeys[i].ID < edgeKeys[j].ID
	})

	edgeQueue := make(map[string][]diagram.BlockEdge)
	for _, e := range d.Edges {
		key := e.From + "->" + e.To
		edgeQueue[key] = append(edgeQueue[key], e)
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		if len(el.Points) < 2 {
			continue
		}
		key := eid.From + "->" + eid.To
		var edge diagram.BlockEdge
		if candidates := edgeQueue[key]; len(candidates) > 0 {
			edge = candidates[0]
			edgeQueue[key] = candidates[1:]
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}
		style := edgeStyle(edge, th)
		if style == "" {
			// Invisible edge — skip rendering entirely.
			continue
		}
		markerEnd := edgeMarker(edge.ArrowHead)
		markerStart := edgeMarker(edge.ArrowTail)
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: markerEnd, MarkerStart: markerStart,
			})
		} else {
			var b strings.Builder
			fmt.Fprintf(&b, "M%.2f,%.2f", pts[0].X, pts[0].Y)
			for _, p := range pts[1:] {
				fmt.Fprintf(&b, " L%.2f,%.2f", p.X, p.Y)
			}
			elems = append(elems, &path{D: b.String(), Style: style, MarkerEnd: markerEnd, MarkerStart: markerStart})
		}

		if edge.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EdgeText, fontSize-1),
				Content: edge.Label,
			})
		}
	}
	return elems
}

// buildArrowMarkers returns the full set of head/tail markers used
// by the block edge lexicon. Each kind in BlockEdge.ArrowHead /
// ArrowTail picks one of these by ID; ArrowHeadNone uses no
// marker at all.
func buildArrowMarkers(th Theme) []marker {
	stroke := fmt.Sprintf("stroke:%s;fill:none", th.EdgeStroke)
	fill := fmt.Sprintf("fill:%s", th.EdgeStroke)
	return []marker{
		{
			ID: "block-arrow", ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: fill}},
		},
		{
			// `--x` cross marker — two diagonals through the centre.
			ID: "block-cross", ViewBox: "0 0 10 10",
			RefX: 5, RefY: 5, Width: 10, Height: 10, Orient: "auto",
			Children: []any{
				&line{X1: 0, Y1: 0, X2: 10, Y2: 10, Style: stroke},
				&line{X1: 10, Y1: 0, X2: 0, Y2: 10, Style: stroke},
			},
		},
		{
			// `--o` open-circle marker — hollow ring.
			ID: "block-circle", ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 10, Height: 10, Orient: "auto",
			Children: []any{
				&svgutil.Circle{CX: 5, CY: 5, R: 4, Style: fmt.Sprintf("fill:%s;stroke:%s", th.Background, th.EdgeStroke)},
			},
		},
	}
}

// mergeNodeCSS returns the CSS string a node inherits from its
// declared classes (looked up in d.CSSClasses) plus any per-node
// `style id ...` override (the latest entry in d.Styles wins).
// Result is "" when the node has no styling at all.
func mergeNodeCSS(d *diagram.BlockDiagram, n diagram.BlockNode) string {
	var parts []string
	for _, name := range n.CSSClasses {
		if css := d.CSSClasses[name]; css != "" {
			parts = append(parts, css)
		}
	}
	for _, s := range d.Styles {
		if s.NodeID == n.ID && s.CSS != "" {
			parts = append(parts, s.CSS)
		}
	}
	return strings.Join(parts, ";")
}

// edgeStyle assembles the stroke-pattern CSS for a block edge from
// its LineStyle. Thick / dotted / invisible map to width or
// dasharray modifications on top of the theme stroke colour.
// Returns "" when the edge should be skipped entirely (invisible).
func edgeStyle(e diagram.BlockEdge, th Theme) string {
	switch e.LineStyle {
	case diagram.LineStyleInvisible:
		return ""
	case diagram.LineStyleThick:
		return fmt.Sprintf("stroke:%s;stroke-width:3;fill:none", th.EdgeStroke)
	case diagram.LineStyleDotted:
		return fmt.Sprintf("stroke:%s;stroke-width:1.5;stroke-dasharray:4 4;fill:none", th.EdgeStroke)
	default:
		return fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke)
	}
}

func edgeMarker(h diagram.ArrowHead) string {
	switch h {
	case diagram.ArrowHeadArrow:
		return "url(#block-arrow)"
	case diagram.ArrowHeadCross:
		return "url(#block-cross)"
	case diagram.ArrowHeadCircle:
		return "url(#block-circle)"
	default:
		return ""
	}
}
