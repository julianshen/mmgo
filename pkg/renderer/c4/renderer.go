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

	var children []any
	children = append(children, &defs{Markers: []marker{buildArrowMarker(th)}})
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize+2),
			Content: d.Title,
		})
	}

	children = append(children, renderEdges(d, l, pad, titleOffset, fontSize, th, ruler)...)
	children = append(children, renderElements(d, l, pad, titleOffset, fontSize, th)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
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
		shapeStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", p.Fill, p.Stroke)

		switch e.Kind {
		case diagram.C4ElementSystemDB, diagram.C4ElementContainerDB:
			// Database elements render as a cylinder (top/bottom
			// ellipse + sides) — matches mmdc's "DB" glyph.
			elems = append(elems, &path{D: cylinderPath(cx, cy, w, h), Style: shapeStyle})
		case diagram.C4ElementPerson, diagram.C4ElementPersonExt:
			// Strongly rounded corners hint at a "person" shape
			// without requiring an embedded icon.
			rx := svgFloat(h / 4)
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: rx, RY: rx, Style: shapeStyle,
			})
		default:
			elems = append(elems, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				Style: shapeStyle,
			})
		}

		curY := y + kindLabelH
		kindLabel := kindDisplayLabel(e.Kind)
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(y + kindLabelH/2 + 2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-style:italic;opacity:0.85", p.Text, fontSize-3),
			Content: kindLabel,
		})
		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(curY + fontSize/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", p.Text, fontSize),
			Content: e.Label,
		})
		curY += fontSize + 4
		if e.Technology != "" {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", p.Text, fontSize-2),
				Content: "[" + e.Technology + "]",
			})
			curY += fontSize - 2
		}
		if e.Description != "" {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(curY),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;opacity:0.9", p.Text, fontSize-2),
				Content: e.Description,
			})
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
	case diagram.C4ElementContainer:
		name = "container"
	case diagram.C4ElementContainerDB:
		name = "container_db"
	case diagram.C4ElementComponent:
		name = "component"
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

		style := fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke)
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style, MarkerEnd: "url(#c4-arrow)",
			})
		} else {
			elems = append(elems, &path{
				D:         svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension),
				Style:     style,
				MarkerEnd: "url(#c4-arrow)",
			})
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad + titleOff
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
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EdgeText, labelFont),
					Content: ln,
				})
			}
		}
	}
	return elems
}

// cylinderEllipseRY governs how "tall" the top/bottom caps look
// relative to the body — same value the flowchart cylinder shape uses.
const cylinderEllipseRY = 0.1

func cylinderPath(cx, cy, w, h float64) string {
	ry := h * cylinderEllipseRY
	top := cy - h/2 + ry
	bot := cy + h/2 - ry
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"L%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f "+
		"M%.2f,%.2f A%.2f,%.2f 0 0,0 %.2f,%.2f",
		cx-w/2, top, cx-w/2, bot, w/2, ry, cx+w/2, bot,
		cx+w/2, top, w/2, ry, cx-w/2, top,
		cx-w/2, top, w/2, ry, cx+w/2, top)
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
