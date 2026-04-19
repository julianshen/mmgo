package class

import (
	"encoding/xml"
	"fmt"
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
	memberRowH         = 20.0
	classPadX          = 15.0
	classPadY          = 10.0
	headerH            = 30.0
	minClassW          = 120.0
)

type Options struct {
	FontSize float64
}

func Render(d *diagram.ClassDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("class render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("class render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	g := graph.New()
	for _, c := range d.Classes {
		w, h := classNodeSize(c, ruler, fontSize)
		g.SetNode(c.ID, graph.NodeAttrs{Label: c.Label, Width: w, Height: h})
	}
	for _, r := range d.Relations {
		g.SetEdge(r.From, r.To, graph.EdgeAttrs{Label: r.Label})
	}

	l := layout.Layout(g, layout.Options{RankDir: layout.RankDirTB})
	pad := defaultPadding

	viewW := svgutil.Sanitize(l.Width) + 2*pad
	viewH := svgutil.Sanitize(l.Height) + 2*pad

	var children []any
	if defs := buildDefs(d); defs != nil {
		children = append(children, defs)
	}
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:#fff;stroke:none",
	})

	children = append(children, renderEdges(d, l, pad, fontSize)...)
	children = append(children, renderClasses(d, l, pad, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("class render: %w", err)
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

func classNodeSize(c diagram.ClassDef, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, _ := ruler.Measure(c.Label, fontSize)
	w = tw + 2*classPadX
	h = headerH
	if c.Annotation != diagram.AnnotationNone {
		h += memberRowH
	}

	fields, methods := splitMembers(c.Members)
	if len(fields) > 0 {
		h += classPadY + float64(len(fields))*memberRowH
		for _, f := range fields {
			fw, _ := ruler.Measure(memberText(f), fontSize-1)
			if fw+2*classPadX > w {
				w = fw + 2*classPadX
			}
		}
	}
	if len(methods) > 0 {
		h += classPadY + float64(len(methods))*memberRowH
		for _, m := range methods {
			mw, _ := ruler.Measure(memberText(m), fontSize-1)
			if mw+2*classPadX > w {
				w = mw + 2*classPadX
			}
		}
	}
	if w < minClassW {
		w = minClassW
	}
	h += classPadY
	return w, h
}

func splitMembers(members []diagram.ClassMember) (fields, methods []diagram.ClassMember) {
	for _, m := range members {
		if m.IsMethod {
			methods = append(methods, m)
		} else {
			fields = append(fields, m)
		}
	}
	return
}

func memberText(m diagram.ClassMember) string {
	prefix := ""
	switch m.Visibility {
	case diagram.VisibilityPublic:
		prefix = "+"
	case diagram.VisibilityPrivate:
		prefix = "-"
	case diagram.VisibilityProtected:
		prefix = "#"
	case diagram.VisibilityPackage:
		prefix = "~"
	}
	name := m.Name
	if m.IsMethod {
		name += "()"
	}
	if m.ReturnType != "" {
		return prefix + name + " : " + m.ReturnType
	}
	return prefix + name
}

func renderClasses(d *diagram.ClassDiagram, l *layout.Result, pad, fontSize float64) []any {
	var elems []any
	for _, c := range d.Classes {
		nl, ok := l.Nodes[c.ID]
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
			Style: "fill:#ECECFF;stroke:#9370DB;stroke-width:1.5",
		})

		curY := y + headerH/2
		if c.Annotation != diagram.AnnotationNone {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(y + 14),
				Anchor: "middle", Dominant: "auto",
				Style:   fmt.Sprintf("fill:#999;font-size:%.0fpx;font-style:italic", fontSize-2),
				Content: "«" + c.Annotation.String() + "»",
			})
			curY = y + headerH/2 + memberRowH/2
		}

		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(curY),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize),
			Content: c.Label,
		})

		sectionY := y + headerH
		if c.Annotation != diagram.AnnotationNone {
			sectionY += memberRowH
		}

		fields, methods := splitMembers(c.Members)
		if len(fields) > 0 {
			elems, sectionY = appendMemberSection(elems, fields, x, w, sectionY, fontSize)
		}
		if len(methods) > 0 {
			elems, _ = appendMemberSection(elems, methods, x, w, sectionY, fontSize)
		}
	}
	return elems
}

func appendMemberSection(elems []any, members []diagram.ClassMember, x, w, sectionY, fontSize float64) ([]any, float64) {
	elems = append(elems, &line{
		X1: svgFloat(x), Y1: svgFloat(sectionY),
		X2: svgFloat(x + w), Y2: svgFloat(sectionY),
		Style: "stroke:#9370DB;stroke-width:1",
	})
	for i, m := range members {
		my := sectionY + classPadY/2 + float64(i)*memberRowH + memberRowH/2
		elems = append(elems, &text{
			X: svgFloat(x + classPadX), Y: svgFloat(my),
			Anchor: "start", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
			Content: memberText(m),
		})
	}
	return elems, sectionY + classPadY + float64(len(members))*memberRowH
}

func renderEdges(d *diagram.ClassDiagram, l *layout.Result, pad, fontSize float64) []any {
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

	relQueue := make(map[string][]diagram.ClassRelation)
	for _, r := range d.Relations {
		key := r.From + "->" + r.To
		relQueue[key] = append(relQueue[key], r)
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		key := eid.From + "->" + eid.To
		candidates := relQueue[key]
		if len(candidates) == 0 {
			continue
		}
		rel := candidates[0]
		relQueue[key] = candidates[1:]

		if len(el.Points) < 2 {
			continue
		}

		style := "stroke:#333;stroke-width:1.5;fill:none"
		if relationIsDashed(rel.RelationType) {
			style += ";stroke-dasharray:5,5"
		}

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

		endRef := ""
		if m, ok := endMarkers[rel.RelationType]; ok {
			endRef = fmt.Sprintf("url(#%s)", m.ID)
		}
		if len(pts) == 2 {
			elems = append(elems, &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style:     style,
				MarkerEnd: endRef,
			})
		} else {
			elems = append(elems, &path{
				D:         polylineD(pts),
				Style:     style,
				MarkerEnd: endRef,
			})
		}
		if g := startMarkerGroup(rel.RelationType, pts[0], srcDir); g != nil {
			elems = append(elems, g)
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			elems = append(elems, &text{
				X: svgFloat(lx), Y: svgFloat(ly),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
				Content: rel.Label,
			})
		}
	}
	return elems
}

func polylineD(pts []layout.Point) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "M%.2f,%.2f", pts[0].X, pts[0].Y)
	for _, p := range pts[1:] {
		fmt.Fprintf(&sb, " L%.2f,%.2f", p.X, p.Y)
	}
	return sb.String()
}

func relationIsDashed(rt diagram.RelationType) bool {
	switch rt {
	case diagram.RelationTypeDependency,
		diagram.RelationTypeRealization,
		diagram.RelationTypeDashedLink:
		return true
	}
	return false
}

func startMarkerGroup(rt diagram.RelationType, start, next layout.Point) *group {
	children, refX, refY, ok := startMarkerGeom(rt)
	if !ok {
		return nil
	}
	return svgutil.InlineMarkerAt(start.X, start.Y, next.X, next.Y, refX, refY, children)
}

type startGeom struct {
	children   []any
	refX, refY float64
}

// Start glyphs point into the "parent" end of the edge (e.g. the
// hollow triangle of <|--, the diamond of *-- and o--). refX places
// the glyph's tip on the From-side class boundary after rotation.
var startGeoms = map[diagram.RelationType]startGeom{
	diagram.RelationTypeInheritance: {
		children: []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:white;stroke:#333;stroke-width:1.5"}},
		refX:     0, refY: 10,
	},
	diagram.RelationTypeComposition: {
		children: []any{&polygon{Points: "0,10 10,0 20,10 10,20", Style: "fill:#333;stroke:#333;stroke-width:1"}},
		refX:     0, refY: 10,
	},
	diagram.RelationTypeAggregation: {
		children: []any{&polygon{Points: "0,10 10,0 20,10 10,20", Style: "fill:white;stroke:#333;stroke-width:1"}},
		refX:     0, refY: 10,
	},
}

func startMarkerGeom(rt diagram.RelationType) (children []any, refX, refY float64, ok bool) {
	g, ok := startGeoms[rt]
	if !ok {
		return nil, 0, 0, false
	}
	return g.children, g.refX, g.refY, true
}

// endMarkers covers the arrow-on-right relation types (A --> B,
// A ..> B, A ..|> B). Inheritance/composition/aggregation place
// their glyph on the From end and appear in startGeoms instead;
// see pkg/renderer/er/markers.go for why those start glyphs are
// inlined rather than referenced via SVG marker-start.
var endMarkers = map[diagram.RelationType]marker{
	diagram.RelationTypeRealization: {
		ID: "cls-realization", ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:white;stroke:#333;stroke-width:1.5"}},
	},
	diagram.RelationTypeDependency: {
		ID: "cls-dependency", ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:#333;stroke:#333;stroke-width:1"}},
	},
	diagram.RelationTypeAssociation: {
		ID: "cls-association", ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:#333;stroke:#333;stroke-width:1"}},
	},
}

func buildDefs(d *diagram.ClassDiagram) *defs {
	needed := make(map[diagram.RelationType]bool)
	for _, r := range d.Relations {
		if _, ok := endMarkers[r.RelationType]; ok {
			needed[r.RelationType] = true
		}
	}
	if len(needed) == 0 {
		return nil
	}

	types := make([]diagram.RelationType, 0, len(needed))
	for rt := range needed {
		types = append(types, rt)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	markers := make([]marker, 0, len(types))
	for _, rt := range types {
		markers = append(markers, endMarkers[rt])
	}
	return &defs{Markers: markers}
}
