package state

import (
	"encoding/xml"
	"fmt"
	"math"
	"sort"
	"strconv"
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
	historyR          = 12.0
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

	// Recursive scope-aware layout: each composite is its own dagre
	// sub-graph, with pseudo-state `[*]` nodes placed in the scope
	// where they were written. The flatten pass projects every node /
	// edge / composite rect into the diagram's global coordinate frame
	// so the existing render primitives still work.
	root := layoutScope("", d.States, d.Transitions, ruler, fontSize,
		layout.Options{RankDir: svgutil.RankDirFor(d.Direction)})
	flat := flattenScopedLayout(root)

	l := &layout.Result{
		Nodes:  flat.Nodes,
		Edges:  flat.Edges,
		Width:  flat.Width,
		Height: flat.Height,
	}
	g := graph.New()
	for id, attrs := range flat.NodeAttrs {
		g.SetNode(id, attrs)
	}
	allStates := collectAllStates(d.States)
	leafStates := leafStatesOnly(allStates)
	composites := buildPlacedComposites(d.States, flat.Composites)
	// Coords from flatten are already in the global frame; the renderer
	// helpers add pad to every coord, so feed them pad=0 to avoid a
	// double shift.
	pad := 0.0

	contentW := flat.Width
	contentH := flat.Height
	notes := layoutStateNotes(d, l, pad, fontSize, ruler)

	// Notes can extend outside the class layout's bounding box
	// (left of leftmost state, etc.); expand the viewBox to include
	// every note rect so labels and connector lines stay visible.
	viewMinX, viewMinY := 0.0, 0.0
	viewMaxX, viewMaxY := contentW, contentH
	for _, p := range notes {
		if p.x-pad < viewMinX {
			viewMinX = p.x - pad
		}
		if p.y-pad < viewMinY {
			viewMinY = p.y - pad
		}
		if right := p.x + p.w + pad; right > viewMaxX {
			viewMaxX = right
		}
		if bottom := p.y + p.h + pad; bottom > viewMaxY {
			viewMaxY = bottom
		}
	}
	for _, c := range composites {
		if c.x-pad < viewMinX {
			viewMinX = c.x - pad
		}
		if c.y-pad < viewMinY {
			viewMinY = c.y - pad
		}
		if right := c.x + c.w + pad; right > viewMaxX {
			viewMaxX = right
		}
		if bottom := c.y + c.h + pad; bottom > viewMaxY {
			viewMaxY = bottom
		}
	}
	// Reserve a band above the content for the visible title (Mermaid
	// renders frontmatter `title:` as a centered heading).
	titleBandH := 0.0
	if d.Title != "" {
		titleBandH = fontSize*1.4 + pad
		viewMinY -= titleBandH
	}
	viewW := viewMaxX - viewMinX
	viewH := viewMaxY - viewMinY

	var children []any
	// Accessibility metadata first (SVG 1.1 §5.4 convention).
	if d.AccTitle != "" {
		children = append(children, &svgTitle{Content: d.AccTitle})
	} else if d.Title != "" {
		children = append(children, &svgTitle{Content: d.Title})
	}
	if d.AccDescr != "" {
		children = append(children, &svgDesc{Content: d.AccDescr})
	}
	children = append(children, &defs{Markers: []marker{buildArrowMarker(th)}})
	children = append(children, &rect{
		X: svgFloat(viewMinX), Y: svgFloat(viewMinY),
		Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})
	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat((viewMinX + viewMaxX) / 2),
			Y: svgFloat(viewMinY + titleBandH/2),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.StateText, fontSize+2),
			Content: d.Title,
		})
	}

	// Composite boxes go behind edges + leaf states so the frame
	// doesn't occlude inner content.
	children = append(children, renderCompositeBoxes(composites, fontSize, th)...)
	children = append(children, renderEdges(d, l, pad, fontSize, ruler, th, g)...)
	children = append(children, renderNodes(d, leafStates, l, pad, fontSize, th)...)
	children = append(children, renderStateNotes(notes, l, pad, fontSize, th)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("%.2f %.2f %.2f %.2f", viewMinX, viewMinY, viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("state render: %w", err)
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

// placedStateNote is a sized + positioned StateNote ready to emit.
type placedStateNote struct {
	note  diagram.StateNote
	lines []string
	x, y  float64
	w, h  float64
}

// stackKey identifies a (target, side) bucket for stacking same-side
// notes vertically. Typed (rather than `target+"|"+sprintf`) so a
// state ID containing the separator can't cause a key collision.
type stackKey struct {
	target string
	side   diagram.NoteSide
}

// layoutStateNotes places each note relative to its target state.
// `note left of S` sits to the left of S; `note right of S` to the
// right. Multiple notes attached to the same side of the same state
// stack vertically downward (later ones below earlier ones) so they
// don't overlap.
func layoutStateNotes(d *diagram.StateDiagram, l *layout.Result, pad, fontSize float64, ruler *textmeasure.Ruler) []placedStateNote {
	if len(d.Notes) == 0 {
		return nil
	}
	out := make([]placedStateNote, 0, len(d.Notes))
	stackY := make(map[stackKey]float64)
	for _, n := range d.Notes {
		target, ok := l.Nodes[n.Target]
		if !ok {
			continue
		}
		lines := strings.Split(n.Text, "\n")
		w := 0.0
		for _, line := range lines {
			lw, _ := ruler.Measure(line, fontSize-1)
			if lw > w {
				w = lw
			}
		}
		w += 2 * svgutil.NotePadX
		h := float64(len(lines))*svgutil.NoteLineH + 2*svgutil.NotePadY
		var x float64
		switch n.Side {
		case diagram.NoteSideLeft:
			x = (target.X + pad) - target.Width/2 - svgutil.NoteGap - w
		default:
			x = (target.X + pad) + target.Width/2 + svgutil.NoteGap
		}
		// First note on this side anchors at the target's vertical
		// midpoint; subsequent ones stack below by their height +
		// gap so notes don't overlap. y can go negative when a note
		// at the top edge of the diagram extends above the layout
		// origin — the viewBox-extents pass picks that up so the
		// rect isn't clipped.
		key := stackKey{target: n.Target, side: n.Side}
		baseY := (target.Y + pad) - h/2
		y := baseY + stackY[key]
		stackY[key] += h + svgutil.NoteGap
		out = append(out, placedStateNote{note: n, lines: lines, x: x, y: y, w: w, h: h})
	}
	return out
}

// renderStateNotes emits a yellow rect + multi-line text per note,
// plus a dashed connector from the target state's nearest edge to
// the note's near edge.
func renderStateNotes(notes []placedStateNote, l *layout.Result, pad, fontSize float64, th Theme) []any {
	if len(notes) == 0 {
		return nil
	}
	noteStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.NoteFill, th.NoteStroke)
	textStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.NoteText, fontSize-1)
	connStyle := fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4,3;fill:none", th.NoteStroke)
	var elems []any
	for _, p := range notes {
		elems = append(elems, &rect{
			X: svgFloat(p.x), Y: svgFloat(p.y),
			Width: svgFloat(p.w), Height: svgFloat(p.h),
			Style: noteStyle,
		})
		for i, ln := range p.lines {
			elems = append(elems, &text{
				X:        svgFloat(p.x + svgutil.NotePadX),
				Y:        svgFloat(p.y + svgutil.NotePadY + float64(i)*svgutil.NoteLineH + svgutil.NoteLineH/2),
				Anchor:   svgutil.AnchorStart,
				Dominant: svgutil.BaselineCentral,
				Style:    textStyle,
				Content:  ln,
			})
		}
		target, ok := l.Nodes[p.note.Target]
		if !ok {
			continue
		}
		stateMidY := target.Y + pad
		var stateX, noteX float64
		if p.note.Side == diagram.NoteSideLeft {
			stateX = (target.X + pad) - target.Width/2
			noteX = p.x + p.w
		} else {
			stateX = (target.X + pad) + target.Width/2
			noteX = p.x
		}
		elems = append(elems, &line{
			X1:    svgFloat(stateX),
			Y1:    svgFloat(stateMidY),
			X2:    svgFloat(noteX),
			Y2:    svgFloat(p.y + p.h/2),
			Style: connStyle,
		})
	}
	return elems
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

// stateRectStyle returns the merged CSS overrides for a state —
// classDef references first (source order), then any `style ID …`
// rules, joined with `;` so later values win per CSS cascade.
// stylesByID is a pre-built index over d.Styles so this runs O(1)
// per state instead of O(states * style-rules).
func stateRectStyle(d *diagram.StateDiagram, s diagram.StateDef, stylesByID map[string][]string) string {
	var parts []string
	for _, name := range s.CSSClasses {
		if css := d.CSSClasses[name]; css != "" {
			parts = append(parts, css)
		}
	}
	parts = append(parts, stylesByID[s.ID]...)
	return strings.Join(parts, ";")
}

// textPropsFromCSS extracts the CSS declarations from a state's
// classDef override that affect the <text> element rather than the
// containing <rect>: typography (font-*), text colour (color, mapped
// to fill), and text alignment. The full override is already applied
// to the rect via stateRectStyle; this returns the subset that has
// to be merged into the label style separately so rect-only fill /
// stroke don't bleed onto the text.
func textPropsFromCSS(css string) string {
	if css == "" {
		return ""
	}
	var parts []string
	for _, decl := range strings.Split(css, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		colon := strings.IndexByte(decl, ':')
		if colon < 1 {
			continue
		}
		prop := strings.TrimSpace(decl[:colon])
		val := strings.TrimSpace(decl[colon+1:])
		switch prop {
		case "color":
			parts = append(parts, "fill:"+val)
		case "font-style", "font-weight", "font-family", "font-size",
			"text-decoration", "text-transform", "letter-spacing":
			parts = append(parts, prop+":"+val)
		}
	}
	return strings.Join(parts, ";")
}

// stateClicksByID indexes click defs by state id; last-seen wins.
func stateClicksByID(clicks []diagram.StateClickDef) map[string]diagram.StateClickDef {
	return svgutil.IndexByID(clicks, func(c diagram.StateClickDef) string { return c.StateID })
}

// stateStylesByID indexes per-state style declarations so
// stateRectStyle is O(1) per state.
func stateStylesByID(styles []diagram.StateStyleDef) map[string][]string {
	grouped := svgutil.GroupByID(styles, func(s diagram.StateStyleDef) string { return s.StateID })
	if grouped == nil {
		return nil
	}
	out := make(map[string][]string, len(grouped))
	for id, defs := range grouped {
		css := make([]string, len(defs))
		for i, d := range defs {
			css[i] = d.CSS
		}
		out[id] = css
	}
	return out
}

// leafStatesOnly returns the states that don't have a composite
// body. Composite states are rendered as labelled bounding boxes
// around their children rather than as nodes in the dagre layout.
func leafStatesOnly(states []diagram.StateDef) []diagram.StateDef {
	out := make([]diagram.StateDef, 0, len(states))
	for _, s := range states {
		if len(s.Children) == 0 {
			out = append(out, s)
		}
	}
	return out
}

// placedComposite is a sized + positioned composite state ready to
// emit. regionRects is non-nil for multi-region composites; each
// entry is the bbox of one region's children, in source order.
type placedComposite struct {
	def  diagram.StateDef
	x, y float64
	w, h float64
	// depth is the nesting level (0 = top-level composite, 1 = child
	// of a composite, etc.). Used to vary fill colour and padding.
	depth int
}

const (
	compositePadX    = 14.0
	compositePadY    = 12.0
	compositeLabelH  = 22.0
	compositeCornerR = 8.0
)

// darkenHex reduces each RGB channel of a 6-digit hex colour by
// factor (0–1). factor=0.92 darkens by 8 %. Returns the original
// string unchanged if parsing fails.
func darkenHex(hex string, factor float64) string {
	if len(hex) != 7 || hex[0] != '#' {
		return hex
	}
	r, err1 := strconv.ParseInt(hex[1:3], 16, 0)
	g, err2 := strconv.ParseInt(hex[3:5], 16, 0)
	b, err3 := strconv.ParseInt(hex[5:7], 16, 0)
	if err1 != nil || err2 != nil || err3 != nil {
		return hex
	}
	darken := func(v int64) int {
		c := int(float64(v) * factor)
		if c < 0 {
			c = 0
		}
		return c
	}
	return fmt.Sprintf("#%02x%02x%02x", darken(r), darken(g), darken(b))
}

// compositeFillForDepth returns a fill colour that darkens slightly
// with each nesting level so nested composites are visually distinct
// from their parents.
func compositeFillForDepth(base string, depth int) string {
	const darkenPerLevel = 0.92
	factor := math.Pow(darkenPerLevel, float64(depth))
	return darkenHex(base, factor)
}

// renderCompositeBoxes emits the labelled rounded rect plus optional
// dashed region dividers for each pre-placed composite state.
func renderCompositeBoxes(composites []placedComposite, fontSize float64, th Theme) []any {
	if len(composites) == 0 {
		return nil
	}
	textStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.CompositeText, fontSize-1)
	var elems []any
	for _, p := range composites {
		fill := compositeFillForDepth(th.CompositeFill, p.depth)
		boxStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", fill, th.CompositeStroke)
		elems = append(elems,
			&rect{
				X: svgFloat(p.x), Y: svgFloat(p.y),
				Width: svgFloat(p.w), Height: svgFloat(p.h),
				RX: svgFloat(compositeCornerR), RY: svgFloat(compositeCornerR),
				Style: boxStyle,
			},
			&text{
				X: svgFloat(p.x + compositePadX), Y: svgFloat(p.y + compositeLabelH/2 + compositePadY/2),
				Anchor: svgutil.AnchorStart, Dominant: svgutil.BaselineCentral,
				Style:   textStyle,
				Content: p.def.Label,
			},
		)
		// Title-bar divider: a subtle line separating the label band
		// from the composite body so nested boxes read as framed
		// containers rather than plain rounded rects.
		titleY := p.y + compositeLabelH
		elems = append(elems, &line{
			X1: svgFloat(p.x + compositePadX), Y1: svgFloat(titleY),
			X2: svgFloat(p.x + p.w - compositePadX), Y2: svgFloat(titleY),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.CompositeStroke),
		})
	}
	return elems
}

// titleBandHeight is the vertical band reserved for the state's
// title row inside its rounded rect. Shared between sizing and
// rendering so the description divider lands on the same y as
// stateNodeSize accounted for.
func titleBandHeight(fontSize float64) float64 {
	return fontSize + 2*statePadY
}

func stateNodeSize(s diagram.StateDef, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	switch s.Kind {
	case diagram.StateKindFork, diagram.StateKindJoin:
		return forkBarW, forkBarH
	case diagram.StateKindChoice:
		return choiceSize, choiceSize
	case diagram.StateKindHistory, diagram.StateKindDeepHistory:
		return historyR * 2, historyR * 2
	}
	tw, _ := ruler.Measure(s.Label, fontSize)
	w = tw + 2*statePadX
	h = titleBandHeight(fontSize)
	if w < minStateW {
		w = minStateW
	}
	if h < minStateH {
		h = minStateH
	}
	return w, h
}

func renderNodes(d *diagram.StateDiagram, states []diagram.StateDef, l *layout.Result, pad, fontSize float64, th Theme) []any {
	clicks := stateClicksByID(d.Clicks)
	styles := stateStylesByID(d.Styles)
	var elems []any
	for _, s := range states {
		nl, ok := l.Nodes[s.ID]
		if !ok {
			continue
		}
		cx := nl.X + pad
		cy := nl.Y + pad

		// Most states append directly to elems; states with a URL
		// click action divert to a per-state buffer that's wrapped
		// in <a> at the end so a click anywhere inside activates
		// the link.
		click, hasClick := clicks[s.ID]
		var stateBuf []any
		buf := &elems
		if hasClick && click.URL != "" {
			stateBuf = make([]any, 0, 4)
			buf = &stateBuf
		}

		switch s.Kind {
		case diagram.StateKindFork, diagram.StateKindJoin:
			*buf = append(*buf, &rect{
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
			*buf = append(*buf, &polygon{
				Points: pts,
				Style:  fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.StateFill, th.ChoiceFill),
			})
		case diagram.StateKindHistory, diagram.StateKindDeepHistory:
			label := "H"
			if s.Kind == diagram.StateKindDeepHistory {
				label = "H*"
			}
			*buf = append(*buf, &circle{
				CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(historyR),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.Background, th.StateStroke),
			})
			*buf = append(*buf, &text{
				X: svgFloat(cx), Y: svgFloat(cy),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.StateText, fontSize-1),
				Content: label,
			})
		default:
			w := nl.Width
			h := nl.Height
			x := cx - w/2
			y := cy - h/2
			rectStyle := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", th.StateFill, th.StateStroke)
			textStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.StateText, fontSize)
			if override := stateRectStyle(d, s, styles); override != "" {
				rectStyle = rectStyle + ";" + override
				textStyle = textStyle + ";" + textPropsFromCSS(override)
			}
			*buf = append(*buf, &rect{
				X: svgFloat(x), Y: svgFloat(y),
				Width: svgFloat(w), Height: svgFloat(h),
				RX: 8, RY: 8,
				Style: rectStyle,
			})
			*buf = append(*buf, &text{
				X: svgFloat(cx), Y: svgFloat(cy),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   textStyle,
				Content: s.Label,
			})
		}

		if hasClick && click.URL != "" {
			a := &anchor{Href: click.URL, Target: click.Target}
			if click.Tooltip != "" {
				a.Children = append(a.Children, &svgTitle{Content: click.Tooltip})
			}
			a.Children = append(a.Children, stateBuf...)
			elems = append(elems, a)
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

func renderEdges(d *diagram.StateDiagram, l *layout.Result, pad, fontSize float64, ruler *textmeasure.Ruler, th Theme, g *graph.Graph) []any {
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

	transMap := make(map[string][]diagram.StateTransition, len(d.Transitions))
	for _, t := range d.Transitions {
		key := t.From + "->" + t.To
		transMap[key] = append(transMap[key], t)
	}

	// Detect anti-parallel edge pairs (e.g. `A --> B` plus `B --> A`)
	// so we can bow them apart visually. Without this, both lines
	// trace the same straight segment and the two labels collide at
	// the same midpoint (the start/stop, deactivate/activate cases
	// in the composite example).
	hasEdge := make(map[[2]string]bool, len(l.Edges))
	for eid := range l.Edges {
		hasEdge[[2]string{eid.From, eid.To}] = true
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
			x, y := clipNodeEdge(eid.From, src, pad, srcDir, g)
			pts[0] = layout.Point{X: x, Y: y}
		}
		if dst, ok := l.Nodes[eid.To]; ok {
			x, y := clipNodeEdge(eid.To, dst, pad, dstDir, g)
			pts[len(pts)-1] = layout.Point{X: x, Y: y}
		}

		// For an anti-parallel straight edge, inject a perpendicular
		// midpoint deflection so the line bows away from its sibling.
		// The sign of the offset is fixed relative to the edge tangent
		// (always 90° CCW of the forward direction in SVG's Y-down
		// frame), so a forward edge bows to one side and its reverse
		// bows to the other — visually disambiguating both lines.
		// Anchor the label to the deflected midpoint so the chips
		// follow each curve instead of stacking on the straight
		// centre with the partner edge.
		labelAnchor := layout.Point{X: el.LabelPos.X + pad, Y: el.LabelPos.Y + pad}
		if len(pts) == 2 && hasEdge[[2]string{eid.To, eid.From}] {
			a, b := pts[0], pts[1]
			dx, dy := b.X-a.X, b.Y-a.Y
			length := math.Sqrt(dx*dx + dy*dy)
			if length > 0 {
				const arcOffset = 18.0
				mx, my := (a.X+b.X)/2, (a.Y+b.Y)/2
				// Perpendicular direction (rotate tangent 90° CCW in
				// the math frame, i.e. CW in SVG's Y-down frame).
				px, py := -dy/length*arcOffset, dx/length*arcOffset
				pts = []layout.Point{a, {X: mx + px, Y: my + py}, b}
				labelAnchor = layout.Point{X: mx + px, Y: my + py}
			}
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
				p := labelPosition(pts, labelAnchor)
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
						Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
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
// nodes are circles; choice nodes are diamonds. Clipping to the visible
// outline keeps the arrowhead tucked against the glyph instead of
// floating inside the layout box reserved around it.
func clipNodeEdge(id string, n layout.NodeLayout, pad float64, dir layout.Point, g *graph.Graph) (float64, float64) {
	cx := n.X + pad
	cy := n.Y + pad
	if isStartNode(id) {
		return svgutil.ClipToCircleEdge(cx, cy, startDotR, dir.X, dir.Y)
	}
	if isEndNode(id) {
		return svgutil.ClipToCircleEdge(cx, cy, endRingR, dir.X, dir.Y)
	}
	if attrs, ok := g.NodeAttrs(id); ok {
		switch attrs.Shape {
		case graph.ShapeDiamond:
			return svgutil.ClipToDiamondEdge(cx, cy, n.Width, n.Height, dir.X, dir.Y)
		case graph.ShapeCircle:
			// History glyphs are drawn at the fixed historyR radius,
			// not at n.Width/2 — the latter is whatever dagre reserved
			// (which can be inflated to the default 100×40 if the
			// state ID collides with a phantom reference). Pin to the
			// visible glyph radius so the arrowhead lands on the
			// circle outline rather than well outside it.
			r := historyR
			if n.Width/2 < r {
				r = n.Width / 2
			}
			return svgutil.ClipToCircleEdge(cx, cy, r, dir.X, dir.Y)
		}
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
