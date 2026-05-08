package c4

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
	defaultFontSize = 13.0
	defaultPadding  = 20.0
	titleH          = 30.0
	boxPadX         = 15.0
	boxPadY         = 10.0
	minBoxW         = 160.0
	minBoxH         = 80.0
	kindLabelH      = 18.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.C4Diagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("c4 render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("c4 render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	for _, e := range d.Elements {
		w, h := elementSize(e, ruler, fontSize)
		g.SetNode(e.ID, graph.NodeAttrs{Label: e.Label, Width: w, Height: h})
	}
	for _, r := range d.Relations {
		g.SetEdge(r.From, r.To, graph.EdgeAttrs{Label: r.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
	pad := defaultPadding

	titleOffset := 0.0
	if d.Title != "" {
		titleOffset = titleH
	}
	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad + titleOffset
	// Boundary frames extend beyond the layout's element bbox by
	// boundaryPad + boundaryHeadingPad, so a frame around an
	// element at (~0, ~0) would clip outside the SVG viewBox.
	// Walk the top-level frames once to find the worst-case
	// overflow on each side and grow the viewport (using negative
	// origin where needed) so every frame stays inside.
	viewMinX, viewMinY := 0.0, 0.0
	for _, b := range d.Boundaries {
		bb := boundaryBBox(d, b, l, pad, titleOffset)
		if bb.Empty() {
			continue
		}
		left := bb.MinX - boundaryPad
		top := bb.MinY - boundaryPad - boundaryHeadingPad
		right := bb.MaxX + boundaryPad
		bottom := bb.MaxY + boundaryPad
		if left < viewMinX {
			viewMinX = left
		}
		if top < viewMinY {
			viewMinY = top
		}
		if right+pad > viewW {
			viewW = right + pad
		}
		if bottom+pad > viewH {
			viewH = bottom + pad
		}
	}
	// Translate negative origins into extra width/height so the
	// background rect at (0,0) still covers the full viewBox.
	if viewMinX < 0 {
		viewW -= viewMinX
	}
	if viewMinY < 0 {
		viewH -= viewMinY
	}

	var children []any
	if d.AccTitle != "" {
		children = append(children, &svgutil.Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgutil.Desc{Content: d.AccDescr})
	}
	children = append(children, &defs{Markers: []marker{buildArrowMarker(th)}})
	children = append(children, &rect{
		X: svgFloat(viewMinX), Y: svgFloat(viewMinY),
		Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(pad + titleH/2),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize+2),
			Content: d.Title,
		})
	}

	// Boundaries paint behind edges + elements so the dashed
	// frame reads as a backdrop. Outermost-first emission is fine
	// because every frame uses fill:none.
	children = append(children, renderBoundaries(d, l, pad, titleOffset, fontSize, th)...)
	children = append(children, renderEdges(d, l, pad, titleOffset, fontSize, th, ruler)...)
	children = append(children, renderElements(d, l, pad, titleOffset, fontSize, th)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("%.2f %.2f %.2f %.2f", viewMinX, viewMinY, viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("c4 render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func elementSize(e diagram.C4Element, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	lw, _ := ruler.Measure(e.Label, fontSize)
	w = lw + 2*boxPadX
	h = kindLabelH + fontSize + 2*boxPadY
	if e.Technology != "" {
		tw, _ := ruler.Measure("["+e.Technology+"]", fontSize-2)
		if tw+2*boxPadX > w {
			w = tw + 2*boxPadX
		}
		h += fontSize - 2
	}
	if e.Description != "" {
		dw, _ := ruler.Measure(e.Description, fontSize-2)
		if dw+2*boxPadX > w {
			w = dw + 2*boxPadX
		}
		h += fontSize - 2
	}
	if w < minBoxW {
		w = minBoxW
	}
	if h < minBoxH {
		h = minBoxH
	}
	return w, h
}

func renderElements(d *diagram.C4Diagram, l *layout.Result, pad, titleOff, fontSize float64, th Theme) []any {
	var elems []any
	for _, e := range d.Elements {
		nl, ok := l.Nodes[e.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad + titleOff
		w := nl.Width
		h := nl.Height
		x := cx - w/2
		y := cy - h/2

		p := th.roleOf(e.Kind)
		// UpdateElementStyle layered on top of the resolved theme
		// palette. Empty fields fall through.
		if ov, ok := d.ElementStyles[e.Kind.String()]; ok {
			if ov.BgColor != "" {
				p.Fill = ov.BgColor
			}
			if ov.BorderColor != "" {
				p.Stroke = ov.BorderColor
			}
			if ov.FontColor != "" {
				p.Text = ov.FontColor
			}
		}
		shapeStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", p.Fill, p.Stroke)
		// Collect per-element children so a single <a href> can wrap
		// the whole group when $link= is set; otherwise they flatten
		// into elems directly.
		var node []any

		switch {
		case IsDBKind(e.Kind):
			// Cylinder glyph signals "this is a datastore" — the
			// shape mmdc uses for every DB-kind variant
			// (system / container / component, plain or _Ext).
			node = append(node, &path{D: svgutil.CylinderPath(cx, cy, w, h), Style: shapeStyle})
		case IsQueueKind(e.Kind):
			// Stadium pill — fully rounded ends — for every queue
			// variant. The half-height radius matches mmdc's
			// queue glyph.
			rx := svgFloat(h / 2)
			node = append(node, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: rx, RY: rx, Style: shapeStyle,
			})
		case e.Kind == diagram.C4ElementPerson || e.Kind == diagram.C4ElementPersonExt:
			// Strongly rounded corners hint at a "person" shape
			// without requiring an embedded icon.
			rx := svgFloat(h / 4)
			node = append(node, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: rx, RY: rx, Style: shapeStyle,
			})
		case e.Kind == diagram.C4ElementDeploymentNode:
			// Deployment_Node renders as a dashed-border rect so
			// it reads as a container of nested elements rather
			// than a leaf.
			node = append(node, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5;stroke-dasharray:6 4", p.Fill, p.Stroke),
			})
		default:
			node = append(node, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				Style: shapeStyle,
			})
		}

		curY := y + kindLabelH
		kindLabel := kindDisplayLabel(e.Kind)
		node = append(node, &text{
			X: svgFloat(cx), Y: svgFloat(y + kindLabelH/2 + 2),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-style:italic;opacity:0.85", p.Text, fontSize-3),
			Content: kindLabel,
		})
		node = append(node, &text{
			X: svgFloat(cx), Y: svgFloat(curY + fontSize/2),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", p.Text, fontSize),
			Content: e.Label,
		})
		curY += fontSize + 4
		if e.Technology != "" {
			node = append(node, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", p.Text, fontSize-2),
				Content: "[" + e.Technology + "]",
			})
			curY += fontSize - 2
		}
		if e.Description != "" {
			node = append(node, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;opacity:0.9", p.Text, fontSize-2),
				Content: e.Description,
			})
		}
		if e.Link != "" {
			elems = append(elems, &svgutil.Anchor{Href: e.Link, Children: node})
		} else {
			elems = append(elems, node...)
		}
	}
	return elems
}

// kindDisplayLabel returns the stereotype tag shown above an element's
// label. Format mirrors mmdc: lowercase snake_case wrapped in `<<...>>`
// (e.g. `<<container_db>>`, `<<external_system>>`) instead of the
// French-quote `«Person»` style mmgo originally used.
func kindDisplayLabel(k diagram.C4ElementKind) string {
	name := ""
	switch k {
	case diagram.C4ElementPerson:
		name = "person"
	case diagram.C4ElementPersonExt:
		name = "external_person"
	case diagram.C4ElementSystem:
		name = "system"
	case diagram.C4ElementSystemExt:
		name = "external_system"
	case diagram.C4ElementSystemDB:
		name = "system_db"
	case diagram.C4ElementSystemDBExt:
		name = "external_system_db"
	case diagram.C4ElementSystemQueue:
		name = "system_queue"
	case diagram.C4ElementSystemQueueExt:
		name = "external_system_queue"
	case diagram.C4ElementContainer:
		name = "container"
	case diagram.C4ElementContainerExt:
		name = "external_container"
	case diagram.C4ElementContainerDB:
		name = "container_db"
	case diagram.C4ElementContainerDBExt:
		name = "external_container_db"
	case diagram.C4ElementContainerQueue:
		name = "container_queue"
	case diagram.C4ElementContainerQueueExt:
		name = "external_container_queue"
	case diagram.C4ElementComponent:
		name = "component"
	case diagram.C4ElementComponentExt:
		name = "external_component"
	case diagram.C4ElementComponentDB:
		name = "component_db"
	case diagram.C4ElementComponentDBExt:
		name = "external_component_db"
	case diagram.C4ElementComponentQueue:
		name = "component_queue"
	case diagram.C4ElementComponentQueueExt:
		name = "external_component_queue"
	case diagram.C4ElementDeploymentNode:
		name = "deployment_node"
	default:
		return ""
	}
	return "<<" + name + ">>"
}

func renderEdges(d *diagram.C4Diagram, l *layout.Result, pad, titleOff, fontSize float64, th Theme, ruler *textmeasure.Ruler) []any {
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

	relQueue := make(map[string][]diagram.C4Relation)
	for _, r := range d.Relations {
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
		var rel diagram.C4Relation
		if len(candidates) > 0 {
			rel = candidates[0]
			relQueue[key] = candidates[1:]
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad + titleOff}
		}
		// Cache direction references before clipping (pts[1] and
		// pts[len-2] alias for 2-point edges) so the destination clip
		// doesn't read the already-clipped source as its direction.
		srcDir := pts[1]
		dstDir := pts[len(pts)-2]
		if src, ok := l.Nodes[eid.From]; ok {
			x, y := svgutil.ClipToRectEdge(src.X+pad, src.Y+pad+titleOff, src.Width, src.Height, srcDir.X, srcDir.Y)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			x, y := svgutil.ClipToRectEdge(dst.X+pad, dst.Y+pad+titleOff, dst.Width, dst.Height, dstDir.X, dstDir.Y)
			pts[len(pts)-1] = layout.Point{X: x, Y: y}
		}

		// UpdateRelStyle override (per from->to pair). Empty fields
		// fall through to the resolved theme.
		relStyle := d.RelStyles[eid.From+"->"+eid.To]
		lineColor := th.EdgeStroke
		if relStyle.LineColor != "" {
			lineColor = relStyle.LineColor
		}
		textColor := th.EdgeText
		if relStyle.TextColor != "" {
			textColor = relStyle.TextColor
		}
		style := fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", lineColor)
		// BiRel adds an arrowhead at the source side too so the
		// edge reads as bidirectional. Other directions still get
		// a single end-marker.
		markerStart := ""
		if rel.Direction == diagram.C4RelBi {
			markerStart = "url(#c4-arrow)"
		}
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: "url(#c4-arrow)", MarkerStart: markerStart,
			})
		} else {
			elems = append(elems, &path{
				D:           svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension),
				Style:       style,
				MarkerEnd:   "url(#c4-arrow)",
				MarkerStart: markerStart,
			})
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad + titleOff
			// $offsetX/Y on the rel itself plus any UpdateRelStyle
			// override stack additively — both shift the label off
			// the default mid-curve anchor.
			lx += rel.OffsetX + relStyle.OffsetX
			ly += rel.OffsetY + relStyle.OffsetY
			labelFont := fontSize - 2
			// Label on first line, optional [technology] on the second
			// — matches how mmdc breaks long edge captions.
			lines := []string{rel.Label}
			if rel.Technology != "" {
				lines = append(lines, "["+rel.Technology+"]")
			}
			lineH := labelFont * 1.2
			totalH := lineH * float64(len(lines))
			var maxW float64
			for _, ln := range lines {
				w, _ := ruler.Measure(ln, labelFont)
				if w > maxW {
					maxW = w
				}
			}
			elems = append(elems, svgutil.LabelChip(lx, ly, maxW, totalH, 4, th.Background, 3))
			startY := ly - totalH/2 + lineH/2
			for i, ln := range lines {
				elems = append(elems, &text{
					X: svgFloat(lx), Y: svgFloat(startY + float64(i)*lineH),
					Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", textColor, labelFont),
					Content: ln,
				})
			}
		}
	}
	return elems
}

// buildArrowMarker — width/height 12 matches state, class, ER. The
// previous 8×8 was barely visible against the 1.5px edge stroke.
func buildArrowMarker(th Theme) marker {
	return marker{
		ID: "c4-arrow", ViewBox: "0 0 10 10",
		RefX: 9, RefY: 5, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", th.EdgeStroke)}},
	}
}

// boundaryPad is the breathing room a boundary frame leaves around
// its tightest child bounding box. Picked empirically — large
// enough that the dashed border doesn't kiss the inner shapes,
// small enough that nested boundaries still nest visibly.
const boundaryPad = 18.0

// renderBoundaries emits one frame per top-level boundary,
// recursing into nested boundaries inside renderBoundary. A
// boundary with no resolvable children (every id missed the
// layout) is skipped silently — this is degenerate input that
// shouldn't crash the render but doesn't have a sensible frame
// to draw either.
func renderBoundaries(d *diagram.C4Diagram, l *layout.Result, pad, titleOff, fontSize float64, th Theme) []any {
	var elems []any
	for _, b := range d.Boundaries {
		elems = append(elems, renderBoundary(d, b, l, pad, titleOff, fontSize, th)...)
	}
	return elems
}

func renderBoundary(d *diagram.C4Diagram, b *diagram.C4Boundary, l *layout.Result, pad, titleOff, fontSize float64, th Theme) []any {
	bbox := boundaryBBox(d, b, l, pad, titleOff)
	if bbox.Empty() {
		return nil
	}
	x0 := bbox.MinX - boundaryPad
	y0 := bbox.MinY - boundaryPad - boundaryHeadingPad
	x1 := bbox.MaxX + boundaryPad
	y1 := bbox.MaxY + boundaryPad
	stroke := th.EdgeStroke
	if stroke == "" {
		stroke = "#666"
	}
	frame := []any{
		&rect{
			X: svgFloat(x0), Y: svgFloat(y0),
			Width:  svgFloat(x1 - x0),
			Height: svgFloat(y1 - y0),
			RX:     6, RY: 6,
			Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:1.2;stroke-dasharray:6 4", stroke),
		},
		&text{
			X: svgFloat(x0 + 8), Y: svgFloat(y0 + 6),
			Anchor: svgutil.AnchorStart, Dominant: svgutil.BaselineHanging,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize-1),
			Content: boundaryHeading(b),
		},
	}
	var out []any
	if b.Link != "" {
		out = append(out, &svgutil.Anchor{Href: b.Link, Children: frame})
	} else {
		out = append(out, frame...)
	}
	for _, child := range b.Boundaries {
		out = append(out, renderBoundary(d, child, l, pad, titleOff, fontSize, th)...)
	}
	return out
}

// boundaryHeadingPad is the extra vertical room reserved at the
// top of a boundary frame so the `Name <<kind>>` stereotype label
// has somewhere to sit without overlapping the children's bbox.
const boundaryHeadingPad = 12.0

// boundaryHeading uses the same `<<kind>>` stereotype style
// element labels do (kindDisplayLabel) so the two read as one
// vocabulary in the rendered SVG. A user-supplied TypeHint
// (`Boundary(b, "Label", "service")`) overrides the kind-derived
// stereotype on a generic Boundary — it doesn't override the
// dedicated System_/Enterprise_/Container_Boundary keywords,
// since those carry their stereotype in the keyword itself.
func boundaryHeading(b *diagram.C4Boundary) string {
	name := b.Label
	if name == "" {
		name = b.ID
	}
	stereotype := b.Kind.String()
	if b.Kind == diagram.C4BoundaryGeneric && b.TypeHint != "" {
		stereotype = b.TypeHint
	}
	return fmt.Sprintf("%s <<%s>>", name, stereotype)
}

// boundaryBBox unions every child element's and nested boundary's
// rect into one bounding box, including the per-frame padding +
// heading slot for nested frames. Returns an Empty() bbox when
// no descendant resolves in the layout — that happens when a
// Boundary( ) parses successfully but every child id failed
// lookup, which is degenerate input we skip rather than crash.
func boundaryBBox(d *diagram.C4Diagram, b *diagram.C4Boundary, l *layout.Result, pad, titleOff float64) svgutil.BBox {
	// Resolve element indexes to ids once so we can reuse the
	// shared BBoxOver helper for the leaf union.
	ids := make([]string, 0, len(b.Elements))
	for _, idx := range b.Elements {
		if idx < 0 || idx >= len(d.Elements) {
			continue
		}
		ids = append(ids, d.Elements[idx].ID)
	}
	bb := svgutil.BBoxOver(ids, l.Nodes, pad)
	// l.Nodes coordinates don't yet include the chart's titleOff
	// — that's added at element-render time. Translate the bbox
	// once so direct elements end up in the same coord space the
	// recursive child boundaries already produce.
	if !bb.Empty() {
		bb.MinY += titleOff
		bb.MaxY += titleOff
	}
	for _, child := range b.Boundaries {
		cb := boundaryBBox(d, child, l, pad, titleOff)
		if cb.Empty() {
			continue
		}
		// Union the child's *frame* rect (its own bbox grown by
		// boundaryPad + heading slot) so the parent sits outside
		// the child's drawn frame.
		w := cb.MaxX - cb.MinX + 2*boundaryPad
		h := cb.MaxY - cb.MinY + 2*boundaryPad + boundaryHeadingPad
		cx := (cb.MinX + cb.MaxX) / 2
		cy := (cb.MinY+cb.MaxY)/2 - boundaryHeadingPad/2
		bb.Expand(cx, cy, w, h)
	}
	return bb
}
