package class

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
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

	viewW := sanitize(l.Width) + 2*pad
	viewH := sanitize(l.Height) + 2*pad

	var children []any
	children = append(children, buildDefs(d))
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

func sanitize(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
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
			elems = append(elems, &line{
				X1: svgFloat(x), Y1: svgFloat(sectionY),
				X2: svgFloat(x + w), Y2: svgFloat(sectionY),
				Style: "stroke:#9370DB;stroke-width:1",
			})
			for i, f := range fields {
				fy := sectionY + classPadY/2 + float64(i)*memberRowH + memberRowH/2
				elems = append(elems, &text{
					X: svgFloat(x + classPadX), Y: svgFloat(fy),
					Anchor: "start", Dominant: "central",
					Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
					Content: memberText(f),
				})
			}
			sectionY += classPadY + float64(len(fields))*memberRowH
		}
		if len(methods) > 0 {
			elems = append(elems, &line{
				X1: svgFloat(x), Y1: svgFloat(sectionY),
				X2: svgFloat(x + w), Y2: svgFloat(sectionY),
				Style: "stroke:#9370DB;stroke-width:1",
			})
			for i, m := range methods {
				my := sectionY + classPadY/2 + float64(i)*memberRowH + memberRowH/2
				elems = append(elems, &text{
					X: svgFloat(x + classPadX), Y: svgFloat(my),
					Anchor: "start", Dominant: "central",
					Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
					Content: memberText(m),
				})
			}
		}
	}
	return elems
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

	relMap := make(map[string]diagram.ClassRelation)
	for _, r := range d.Relations {
		relMap[r.From+"->"+r.To] = r
	}

	var elems []any
	for _, eid := range edgeKeys {
		el := l.Edges[eid]
		rel := relMap[eid.From+"->"+eid.To]

		if len(el.Points) < 2 {
			continue
		}

		style := "stroke:#333;stroke-width:1.5;fill:none"
		switch rel.RelationType {
		case diagram.RelationTypeDependency, diagram.RelationTypeRealization,
			diagram.RelationTypeDashedLink:
			style += ";stroke-dasharray:5,5"
		}

		pts := make([]layout.Point, len(el.Points))
		for i, p := range el.Points {
			pts[i] = layout.Point{X: p.X + pad, Y: p.Y + pad}
		}

		if len(pts) == 2 {
			ln := &line{
				X1: svgFloat(pts[0].X), Y1: svgFloat(pts[0].Y),
				X2: svgFloat(pts[1].X), Y2: svgFloat(pts[1].Y),
				Style: style,
			}
			ln.MarkerEnd = fmt.Sprintf("url(#%s)", markerID(rel.RelationType))
			elems = append(elems, ln)
		} else {
			d := fmt.Sprintf("M%.2f,%.2f", pts[0].X, pts[0].Y)
			for _, p := range pts[1:] {
				d += fmt.Sprintf(" L%.2f,%.2f", p.X, p.Y)
			}
			elems = append(elems, &path{
				D: d, Style: style,
				MarkerEnd: fmt.Sprintf("url(#%s)", markerID(rel.RelationType)),
			})
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

func markerID(rt diagram.RelationType) string {
	return fmt.Sprintf("cls-%s", rt.String())
}

func buildDefs(d *diagram.ClassDiagram) *defs {
	needed := make(map[diagram.RelationType]bool)
	for _, r := range d.Relations {
		needed[r.RelationType] = true
	}

	types := make([]diagram.RelationType, 0, len(needed))
	for rt := range needed {
		types = append(types, rt)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })

	var markers []marker
	for _, rt := range types {
		markers = append(markers, buildMarker(rt))
	}
	return &defs{Markers: markers}
}

func buildMarker(rt diagram.RelationType) marker {
	m := marker{
		ID: markerID(rt), ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
	}
	switch rt {
	case diagram.RelationTypeInheritance:
		m.Children = []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:white;stroke:#333;stroke-width:1.5"}}
	case diagram.RelationTypeRealization:
		m.Children = []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:white;stroke:#333;stroke-width:1.5"}}
	case diagram.RelationTypeComposition:
		m.RefX = 20
		m.Children = []any{&polygon{Points: "0,10 10,0 20,10 10,20", Style: "fill:#333;stroke:#333;stroke-width:1"}}
	case diagram.RelationTypeAggregation:
		m.RefX = 20
		m.Children = []any{&polygon{Points: "0,10 10,0 20,10 10,20", Style: "fill:white;stroke:#333;stroke-width:1"}}
	case diagram.RelationTypeDependency, diagram.RelationTypeAssociation:
		m.Children = []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:#333;stroke:#333;stroke-width:1"}}
	default:
		m.Children = []any{&polygon{Points: "0,0 20,10 0,20", Style: "fill:#333;stroke:#333;stroke-width:1"}}
	}
	return m
}
