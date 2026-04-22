package er

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	entityPadX      = 15.0
	entityPadY      = 10.0
	headerH         = 30.0
	attrRowH        = 24.0
	minEntityW      = 120.0
	// attrCellGap separates the type column from the name column inside
	// each attribute row. Matches the visual spacing mmdc uses.
	attrCellGap = 16.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.ERDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("er render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("er render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	for _, e := range d.Entities {
		w, h := entitySize(e, ruler, fontSize)
		g.SetNode(e.Name, graph.NodeAttrs{Label: e.Name, Width: w, Height: h})
	}
	for _, r := range d.Relationships {
		g.SetEdge(r.From, r.To, graph.EdgeAttrs{Label: r.Label})
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
	if markers := buildERMarkers(d); len(markers) > 0 {
		children = append(children, &defs{Markers: markers})
	}
	children = append(children, renderEdges(d, l, pad, fontSize, th, ruler)...)
	children = append(children, renderEntities(d, l, pad, fontSize, th, ruler)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("er render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

// See erStartGeoms in markers.go for why start markers are inlined
// rather than referenced via SVG marker-start.
func startMarkerGroup(c diagram.ERCardinality, start, next layout.Point) *group {
	children, refX, refY, ok := startMarkerGeom(c)
	if !ok {
		return nil
	}
	return svgutil.InlineMarkerAt(start.X, start.Y, next.X, next.Y, refX, refY, children)
}

func entitySize(e diagram.EREntity, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, _ := ruler.Measure(e.Name, fontSize)
	w = tw + 2*entityPadX
	h = headerH
	if len(e.Attributes) > 0 {
		typeColW, nameColW := attrColumnWidths(e.Attributes, ruler, fontSize-1)
		rowW := 2*entityPadX + typeColW + attrCellGap + nameColW
		if rowW > w {
			w = rowW
		}
		h += float64(len(e.Attributes)) * attrRowH
	}
	if w < minEntityW {
		w = minEntityW
	}
	return w, h
}

// attrColumnWidths returns the widest type-cell and the widest name-cell
// across all attributes. The Key suffix (PK/FK/UK) is appended to the
// name cell, so keys influence nameColW only.
func attrColumnWidths(attrs []diagram.ERAttribute, ruler *textmeasure.Ruler, fontSize float64) (typeColW, nameColW float64) {
	for _, a := range attrs {
		tw, _ := ruler.Measure(a.Type, fontSize)
		if tw > typeColW {
			typeColW = tw
		}
		nw, _ := ruler.Measure(nameCellText(a), fontSize)
		if nw > nameColW {
			nameColW = nw
		}
	}
	return typeColW, nameColW
}

// nameCellText is what shows in the name column. The key marker
// (PK/FK/UK) trails the name, mirroring mmdc's row layout.
func nameCellText(a diagram.ERAttribute) string {
	if a.Key != diagram.ERKeyNone {
		return a.Name + " " + a.Key.String()
	}
	return a.Name
}

func renderEntities(d *diagram.ERDiagram, l *layout.Result, pad, fontSize float64, th Theme, ruler *textmeasure.Ruler) []any {
	var elems []any
	for _, e := range d.Entities {
		nl, ok := l.Nodes[e.Name]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(y),
			Width: svgFloat(w), Height: svgFloat(h),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.EntityFill, th.EntityStroke),
		})
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(y + headerH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.EntityText, fontSize),
			Content: e.Name,
		})

		if len(e.Attributes) == 0 {
			continue
		}

		dividerStyle := fmt.Sprintf("stroke:%s;stroke-width:1", th.EntityStroke)
		typeColW, _ := attrColumnWidths(e.Attributes, ruler, fontSize-1)
		colDividerX := x + entityPadX + typeColW + attrCellGap/2

		headerSepY := y + headerH
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(headerSepY),
			X2: svgFloat(x + w), Y2: svgFloat(headerSepY),
			Style: dividerStyle,
		})
		// Vertical divider between type and name columns spans the
		// whole attribute section.
		attrSectionH := float64(len(e.Attributes)) * attrRowH
		elems = append(elems, &line{
			X1: svgFloat(colDividerX), Y1: svgFloat(headerSepY),
			X2: svgFloat(colDividerX), Y2: svgFloat(headerSepY + attrSectionH),
			Style: dividerStyle,
		})

		nameCellX := x + entityPadX + typeColW + attrCellGap
		attrTextStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EntityText, fontSize-1)
		for i, a := range e.Attributes {
			rowTop := headerSepY + float64(i)*attrRowH
			rowMid := rowTop + attrRowH/2
			if i > 0 {
				elems = append(elems, &line{
					X1: svgFloat(x), Y1: svgFloat(rowTop),
					X2: svgFloat(x + w), Y2: svgFloat(rowTop),
					Style: dividerStyle,
				})
			}
			elems = append(elems, &text{
				X: svgFloat(x + entityPadX), Y: svgFloat(rowMid),
				Anchor: "start", Dominant: "central",
				Style:   attrTextStyle,
				Content: a.Type,
			})
			elems = append(elems, &text{
				X: svgFloat(nameCellX), Y: svgFloat(rowMid),
				Anchor: "start", Dominant: "central",
				Style:   attrTextStyle,
				Content: nameCellText(a),
			})
		}
	}
	return elems
}

func renderEdges(d *diagram.ERDiagram, l *layout.Result, pad, fontSize float64, th Theme, ruler *textmeasure.Ruler) []any {
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

	relQueue := make(map[string][]diagram.ERRelationship)
	for _, r := range d.Relationships {
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
		if len(candidates) == 0 {
			continue
		}
		rel := candidates[0]
		relQueue[key] = candidates[1:]

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}
		// pts[1] and pts[len-2] alias for 2-point edges; cache before
		// mutating either endpoint, or the dst clip reads the already-
		// clipped src as its direction reference.
		srcDir := pts[1]
		dstDir := pts[len(pts)-2]
		if src, ok := l.Nodes[eid.From]; ok {
			x, y := svgutil.ClipToRectEdge(src.X+pad, src.Y+pad, src.Width, src.Height, srcDir.X, srcDir.Y)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			last := len(pts) - 1
			x, y := svgutil.ClipToRectEdge(dst.X+pad, dst.Y+pad, dst.Width, dst.Height, dstDir.X, dstDir.Y)
			pts[last] = layout.Point{X: x, Y: y}
		}

		style := fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke)
		endRef := markerRef(markerEndID(rel.ToCard))
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style:     style,
				MarkerEnd: endRef,
			})
		} else {
			elems = append(elems, &path{
				D:         svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension),
				Style:     style,
				MarkerEnd: endRef,
			})
		}
		if g := startMarkerGroup(rel.FromCard, pts[0], srcDir); g != nil {
			elems = append(elems, g)
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			labelFont := fontSize - 1
			labelW, labelH := ruler.Measure(rel.Label, labelFont)
			const labelPad = 4.0
			// Chip backdrop tinted with the theme background — same
			// pattern as flowchart (PR #73) and class (PR #74). Without
			// it, ER labels overlap cardinality markers and crossing
			// lines (most visible in examples/er/blog.svg).
			elems = append(elems, &rect{
				X: svgFloat(lx - labelW/2 - labelPad), Y: svgFloat(ly - labelH/2 - labelPad),
				Width:  svgFloat(labelW + 2*labelPad),
				Height: svgFloat(labelH + 2*labelPad),
				RX:     3, RY: 3,
				Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
			})
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EdgeText, labelFont),
				Content: rel.Label,
			})
		}
	}
	return elems
}
