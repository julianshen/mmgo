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
	notePadX           = 10.0
	notePadY           = 8.0
	noteGap            = 16.0
	noteLineH          = 18.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.ClassDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("class render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

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

	l := layout.Layout(g, layout.Options{RankDir: rankDirFor(d.Direction)})
	pad := defaultPadding

	contentW := svgutil.Sanitize(l.Width) + 2*pad
	contentH := svgutil.Sanitize(l.Height) + 2*pad

	notes := layoutNotes(d, contentW, pad, fontSize, ruler)
	viewW, viewH := contentW, contentH
	if len(notes) > 0 {
		viewW = notes[len(notes)-1].x + notes[len(notes)-1].w + pad
		if last := notes[len(notes)-1].y + notes[len(notes)-1].h + pad; last > viewH {
			viewH = last
		}
	}

	var children []any
	// Accessibility metadata comes first per SVG 1.1 §5.4: screen
	// readers announce the document name from <title> and any longer
	// description from <desc>. accTitle / accDescr are mermaid's
	// dedicated accessibility keywords; `title:` falls back to a
	// general document title.
	if d.AccTitle != "" {
		children = append(children, &svgTitle{Content: d.AccTitle})
	} else if d.Title != "" {
		children = append(children, &svgTitle{Content: d.Title})
	}
	if d.AccDescr != "" {
		children = append(children, &svgDesc{Content: d.AccDescr})
	}
	if defs := buildDefs(d); defs != nil {
		children = append(children, defs)
	}
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	children = append(children, renderEdges(d, l, pad, fontSize, th, ruler)...)
	children = append(children, renderClasses(d, l, pad, fontSize, th)...)
	children = append(children, renderNotes(notes, l, pad, fontSize, th)...)

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

func rankDirFor(d diagram.Direction) layout.RankDir {
	switch d {
	case diagram.DirectionBT:
		return layout.RankDirBT
	case diagram.DirectionLR:
		return layout.RankDirLR
	case diagram.DirectionRL:
		return layout.RankDirRL
	}
	return layout.RankDirTB
}

func classNodeSize(c diagram.ClassDef, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	tw, _ := ruler.Measure(headerText(c), fontSize)
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

// classRectStyle returns the merged CSS overrides for a class —
// classDef declarations referenced via CSSClasses first, then any
// `style ID …` rules in source order. Later declarations win because
// they're appended to the style string verbatim and SVG honors
// later-declared values.
func classRectStyle(d *diagram.ClassDiagram, c diagram.ClassDef) string {
	var parts []string
	for _, name := range c.CSSClasses {
		if css := d.CSSClasses[name]; css != "" {
			parts = append(parts, css)
		}
	}
	for _, s := range d.Styles {
		if s.ClassID == c.ID {
			parts = append(parts, s.CSS)
		}
	}
	return strings.Join(parts, ";")
}

// headerText is the rendered class header — the label (custom or
// defaulted to the ID) optionally followed by `<Generic>`. Mermaid
// renders generics with angle brackets, mirroring TypeScript / Java.
func headerText(c diagram.ClassDef) string {
	if c.Generic == "" {
		return c.Label
	}
	return c.Label + "<" + c.Generic + ">"
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
	if m.IsMethod {
		body := m.Name + "(" + m.Args + ")"
		if m.ReturnType != "" {
			body += " : " + m.ReturnType
		}
		return prefix + body
	}
	return prefix + m.Name
}

func renderClasses(d *diagram.ClassDiagram, l *layout.Result, pad, fontSize float64, th Theme) []any {
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

		rectStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.NodeFill, th.NodeStroke)
		if override := classRectStyle(d, c); override != "" {
			rectStyle = rectStyle + ";" + override
		}
		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(y),
			Width: svgFloat(w), Height: svgFloat(h),
			Style: rectStyle,
		})

		curY := y + headerH/2
		if c.Annotation != diagram.AnnotationNone {
			elems = append(elems, &text{
				X: svgFloat(cx), Y: svgFloat(y + 14),
				Anchor: "middle", Dominant: "auto",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-style:italic", th.AnnotationText, fontSize-2),
				Content: "«" + c.Annotation.String() + "»",
			})
			curY = y + headerH/2 + memberRowH/2
		}

		elems = append(elems, &text{
			X: svgFloat(cx), Y: svgFloat(curY),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.NodeText, fontSize),
			Content: headerText(c),
		})

		sectionY := y + headerH
		if c.Annotation != diagram.AnnotationNone {
			sectionY += memberRowH
		}

		fields, methods := splitMembers(c.Members)
		if len(fields) > 0 {
			elems, sectionY = appendMemberSection(elems, fields, x, w, sectionY, fontSize, th)
		}
		if len(methods) > 0 {
			elems, _ = appendMemberSection(elems, methods, x, w, sectionY, fontSize, th)
		}
	}
	return elems
}

func appendMemberSection(elems []any, members []diagram.ClassMember, x, w, sectionY, fontSize float64, th Theme) ([]any, float64) {
	elems = append(elems, &line{
		X1: svgFloat(x), Y1: svgFloat(sectionY),
		X2: svgFloat(x + w), Y2: svgFloat(sectionY),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.NodeStroke),
	})
	for i, m := range members {
		my := sectionY + classPadY/2 + float64(i)*memberRowH + memberRowH/2
		style := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.NodeText, fontSize-1)
		if m.IsStatic {
			style += ";text-decoration:underline"
		}
		if m.IsAbstract {
			style += ";font-style:italic"
		}
		elems = append(elems, &text{
			X: svgFloat(x + classPadX), Y: svgFloat(my),
			Anchor: "start", Dominant: "central",
			Style:   style,
			Content: memberText(m),
		})
	}
	return elems, sectionY + classPadY + float64(len(members))*memberRowH
}

// placedNote is a sized + positioned ClassNote ready to emit.
type placedNote struct {
	note    diagram.ClassNote
	lines   []string
	x, y    float64
	w, h    float64
}

// layoutNotes sizes each note from its text and stacks the lot in a
// column to the right of the class diagram. Source order, single
// column, no collision avoidance — keeps notes out of the dagre graph
// so they can't distort class placement.
func layoutNotes(d *diagram.ClassDiagram, contentW, pad, fontSize float64, ruler *textmeasure.Ruler) []placedNote {
	if len(d.Notes) == 0 {
		return nil
	}
	out := make([]placedNote, 0, len(d.Notes))
	colX := contentW + noteGap
	cursorY := pad
	for _, n := range d.Notes {
		lines := strings.Split(n.Text, "\n")
		w := 0.0
		for _, line := range lines {
			lw, _ := ruler.Measure(line, fontSize-1)
			if lw > w {
				w = lw
			}
		}
		w += 2 * notePadX
		h := float64(len(lines))*noteLineH + 2*notePadY
		out = append(out, placedNote{
			note: n, lines: lines,
			x: colX, y: cursorY, w: w, h: h,
		})
		cursorY += h + noteGap
	}
	return out
}

// renderNotes emits the rect + text per note, plus a dashed connector
// line from each `note for X` to the class X box.
func renderNotes(notes []placedNote, l *layout.Result, pad, fontSize float64, th Theme) []any {
	var elems []any
	noteStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.NoteFill, th.NoteStroke)
	textStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.NoteText, fontSize-1)
	connStyle := fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4,3;fill:none", th.NoteStroke)
	for _, p := range notes {
		elems = append(elems, &rect{
			X: svgFloat(p.x), Y: svgFloat(p.y),
			Width: svgFloat(p.w), Height: svgFloat(p.h),
			Style: noteStyle,
		})
		for i, ln := range p.lines {
			elems = append(elems, &text{
				X:        svgFloat(p.x + notePadX),
				Y:        svgFloat(p.y + notePadY + float64(i)*noteLineH + noteLineH/2),
				Anchor:   "start",
				Dominant: "central",
				Style:    textStyle,
				Content:  ln,
			})
		}
		if p.note.For == "" {
			continue
		}
		target, ok := l.Nodes[p.note.For]
		if !ok {
			continue
		}
		// Notes column sits to the right of the diagram, so the natural
		// connector path is class right-middle → note left-middle.
		classRightX := target.X + pad + target.Width/2
		classMidY := target.Y + pad
		elems = append(elems, &line{
			X1:    svgFloat(classRightX),
			Y1:    svgFloat(classMidY),
			X2:    svgFloat(p.x),
			Y2:    svgFloat(p.y + p.h/2),
			Style: connStyle,
		})
	}
	return elems
}

func renderEdges(d *diagram.ClassDiagram, l *layout.Result, pad, fontSize float64, th Theme, ruler *textmeasure.Ruler) []any {
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

		style := fmt.Sprintf("stroke:%s;stroke-width:1.5;fill:none", th.EdgeStroke)
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

		// Forward asymmetric edges use SVG marker-end for the To-end
		// glyph; reverse/bidirectional edges inline-place because
		// marker-end can't address the start.
		atFrom, atTo := edgeGlyphs(rel)
		canUseEndMarker := rel.Direction == diagram.RelationForward
		endRef := ""
		if canUseEndMarker && atTo != glyphNoneKind {
			if m, ok := endMarkerForGlyph(atTo); ok {
				endRef = fmt.Sprintf("url(#%s)", m.ID)
			}
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
				D:         svgutil.CatmullRomPath(pts, svgutil.CatmullRomTension),
				Style:     style,
				MarkerEnd: endRef,
			})
		}
		// Inline-place the start glyph (canonical forward path) and any
		// glyph the SVG marker-end couldn't carry (reverse / bidirectional
		// or non-end-marker glyphs at the To end).
		if atFrom != glyphNoneKind {
			if g := inlineGlyphAt(atFrom, pts[0], srcDir); g != nil {
				elems = append(elems, g)
			}
		}
		if atTo != glyphNoneKind && endRef == "" {
			last := len(pts) - 1
			if g := inlineGlyphAt(atTo, pts[last], dstDir); g != nil {
				elems = append(elems, g)
			}
		}

		if rel.Label != "" {
			lx := el.LabelPos.X + pad
			ly := el.LabelPos.Y + pad
			labelFont := fontSize - 1
			labelW, labelH := ruler.Measure(rel.Label, labelFont)
			// Chip backdrop tinted with the theme background, mirroring
			// the flowchart edge-label treatment (PR #73). Without it,
			// long association labels visually merge with the edge line
			// and any class boxes they cross.
			elems = append(elems, svgutil.LabelChip(lx, ly, labelW, labelH, 4, th.Background, 3))
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

func relationIsDashed(rt diagram.RelationType) bool {
	switch rt {
	case diagram.RelationTypeDependency,
		diagram.RelationTypeRealization,
		diagram.RelationTypeDashedLink:
		return true
	}
	return false
}

// glyphKind enumerates the visual end-glyphs the renderer can emit.
// It collapses the (RelationType, Reverse, Bidirectional) tuple from the
// AST into a small palette of shapes.
type glyphKind int8

const (
	glyphNoneKind       glyphKind = iota
	glyphTriangleHollow           // inheritance + realization heads
	glyphDiamondFilled            // composition
	glyphDiamondHollow            // aggregation
	glyphArrowhead                // association + dependency
	glyphLollipop                 // provided-interface circle
)

// SVG marker / inline polygon shapes. Hoisted to package-level so the
// arrowhead `0,0 20,10 0,20` and triangle `20,0 0,10 20,20` each have
// one source of truth across `inlineGeoms` and `endMarkersByGlyph`.
const (
	arrowheadPoints = "0,0 20,10 0,20"
	trianglePoints  = "20,0 0,10 20,20"
	diamondPoints   = "0,10 10,0 20,10 10,20"
)

// edgeGlyphs returns the glyph (if any) to draw at each end of rel.
// Reverse swaps the canonical side; Bidirectional mirrors the glyph
// onto both ends.
func edgeGlyphs(rel diagram.ClassRelation) (atFrom, atTo glyphKind) {
	g, primaryAtTo := glyphForRelation(rel.RelationType)
	if g == glyphNoneKind {
		return glyphNoneKind, glyphNoneKind
	}
	switch rel.Direction {
	case diagram.RelationBidirectional:
		return g, g
	case diagram.RelationReverse:
		primaryAtTo = !primaryAtTo
	}
	if primaryAtTo {
		return glyphNoneKind, g
	}
	return g, glyphNoneKind
}

// glyphForRelation returns the glyph and whether it sits at the To end
// in the canonical (forward, non-bidirectional) form.
func glyphForRelation(rt diagram.RelationType) (g glyphKind, atTo bool) {
	switch rt {
	case diagram.RelationTypeInheritance:
		return glyphTriangleHollow, false
	case diagram.RelationTypeRealization:
		return glyphTriangleHollow, true
	case diagram.RelationTypeComposition:
		return glyphDiamondFilled, false
	case diagram.RelationTypeAggregation:
		return glyphDiamondHollow, false
	case diagram.RelationTypeAssociation, diagram.RelationTypeDependency:
		return glyphArrowhead, true
	case diagram.RelationTypeLollipop:
		// Canonical literal `bar ()-- foo` puts the circle on the
		// LEFT (From end), so atTo=false is the canonical-forward.
		return glyphLollipop, false
	}
	return glyphNoneKind, false
}

func inlineGlyphAt(g glyphKind, anchor, dirRef layout.Point) *group {
	geom, ok := inlineGeoms[g]
	if !ok {
		return nil
	}
	return svgutil.InlineMarkerAt(anchor.X, anchor.Y, dirRef.X, dirRef.Y, geom.refX, geom.refY, geom.children)
}

func endMarkerForGlyph(g glyphKind) (marker, bool) {
	m, ok := endMarkersByGlyph[g]
	return m, ok
}

type startGeom struct {
	children   []any
	refX, refY float64
}

// inlineGeoms holds the shape for each glyph when drawn inline. refX=0
// pins the glyph's inner vertex (the side that touches the class box)
// at the edge endpoint; the rest of the glyph flares into the gap.
var inlineGeoms = map[glyphKind]startGeom{
	glyphTriangleHollow: {
		children: []any{&polygon{Points: trianglePoints, Style: "fill:white;stroke:#333;stroke-width:1.5"}},
		refX:     0, refY: 10,
	},
	glyphDiamondFilled: {
		children: []any{&polygon{Points: diamondPoints, Style: "fill:#333;stroke:#333;stroke-width:1"}},
		refX:     0, refY: 10,
	},
	glyphDiamondHollow: {
		children: []any{&polygon{Points: diamondPoints, Style: "fill:white;stroke:#333;stroke-width:1"}},
		refX:     0, refY: 10,
	},
	glyphArrowhead: {
		children: []any{&polygon{Points: arrowheadPoints, Style: "fill:#333;stroke:#333;stroke-width:1"}},
		refX:     20, refY: 10,
	},
	// Lollipop: a small hollow circle on a short stub. The circle's
	// outer edge sits at refX (touching the class box); the stub
	// extends from refX outward toward the line interior so the line
	// terminates at the back of the lollipop.
	glyphLollipop: {
		children: []any{
			&circle{CX: 14, CY: 10, R: 5, Style: "fill:white;stroke:#333;stroke-width:1.5"},
		},
		refX: 0, refY: 10,
	},
}

// endMarkersByGlyph is the SVG `<marker>` form of the same glyph table,
// used on the canonical-forward path via `marker-end`. The arrowhead
// marker is shared by Association (solid line) and Dependency (dashed
// line) — the dash pattern is set on the path, not the marker.
var endMarkersByGlyph = map[glyphKind]marker{
	glyphArrowhead: {
		ID: "cls-association", ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: arrowheadPoints, Style: "fill:#333;stroke:#333;stroke-width:1"}},
	},
	glyphTriangleHollow: {
		ID: "cls-realization", ViewBox: "0 0 20 20",
		RefX: 18, RefY: 10, Width: 12, Height: 12, Orient: "auto",
		Children: []any{&polygon{Points: arrowheadPoints, Style: "fill:white;stroke:#333;stroke-width:1.5"}},
	},
}

func buildDefs(d *diagram.ClassDiagram) *defs {
	needed := make(map[glyphKind]bool)
	for _, r := range d.Relations {
		if r.Direction != diagram.RelationForward {
			continue // reverse / bidirectional inline-place; no <marker> def needed
		}
		// Only forward edges with a glyph on the To end use marker-end.
		g, atTo := glyphForRelation(r.RelationType)
		if !atTo {
			continue
		}
		if _, ok := endMarkersByGlyph[g]; ok {
			needed[g] = true
		}
	}
	if len(needed) == 0 {
		return nil
	}

	kinds := make([]glyphKind, 0, len(needed))
	for k := range needed {
		kinds = append(kinds, k)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	markers := make([]marker, 0, len(kinds))
	for _, k := range kinds {
		markers = append(markers, endMarkersByGlyph[k])
	}
	return &defs{Markers: markers}
}
