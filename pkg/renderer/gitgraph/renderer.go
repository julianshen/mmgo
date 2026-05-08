// Package gitgraph renders a GitGraphDiagram to SVG as horizontal
// swim lanes: one row per branch, commits as dots along the X axis in
// declaration order, and cross-branch parent links as curved connectors.
package gitgraph

import (
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

type Options struct {
	FontSize float64
	Theme    Theme
	Config   Config
}

const (
	defaultFontSize = 14.0
	commitStride    = 60.0
	laneHeight      = 60.0
	commitRadius    = 8.0
	highlightRadius = 11.0
	marginX         = 40.0
	marginY         = 40.0
	labelGap        = 6.0
	branchLabelPadX = 10.0
	branchLabelPadY = 5.0
	branchLabelR    = 10.0 // pill corner radius
	branchPathW     = 4.0  // thick colored path between a branch's own commits
	branchPathOp    = 1.0  // mmdc's branch paths are fully opaque
	tagPadX         = 6.0
	tagPadY         = 3.0
	tagGap          = 14.0 // vertical gap from commit center to tag callout
	tagCornerR      = 4.0  // tag callout corner radius
)

func Render(d *diagram.GitGraphDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("gitgraph render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)
	cfg := resolveConfig(opts)

	lanes := orderedLanes(d, cfg.MainBranchOrder)
	laneOf := make(map[string]int, len(lanes))
	for i, b := range lanes {
		laneOf[b] = i
	}

	// LR's gutter is the leftmost pill column (label width + pill
	// padding); TB/BT put the gutter at the top, where the relevant
	// dimension is pill height (one line of text + vertical pad).
	maxPillW := branchGutterW(lanes, fontSize) + 2*branchLabelPadX
	pillH := fontSize + 2*branchLabelPadY
	gutter := maxPillW
	if d.Direction == diagram.GitGraphDirTB || d.Direction == diagram.GitGraphDirBT {
		gutter = pillH
	}
	if !svgutil.BoolOr(cfg.ShowBranches, true) {
		gutter = 0
		maxPillW = 0 // pills aren't drawn, so they shouldn't reserve viewport room
	}
	g := newGeom(d.Direction, len(lanes), len(d.Commits), gutter)
	g.maxPillW = maxPillW

	cx := make(map[string]float64, len(d.Commits))
	cy := make(map[string]float64, len(d.Commits))
	branchRange := make(map[string][2]float64, len(lanes))
	for i, c := range d.Commits {
		x, y := g.commitXY(i, laneOf[c.Branch])
		cx[c.ID] = x
		cy[c.ID] = y
		// branchRange records the min/max along the commit axis only.
		mainPx := g.commitMain(i)
		if r, ok := branchRange[c.Branch]; ok {
			if mainPx < r[0] {
				r[0] = mainPx
			}
			if mainPx > r[1] {
				r[1] = mainPx
			}
			branchRange[c.Branch] = r
		} else {
			branchRange[c.Branch] = [2]float64{mainPx, mainPx}
		}
	}

	viewW, viewH := g.viewSize()

	var children []any
	if d.AccTitle != "" {
		children = append(children, &svgutil.Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgutil.Desc{Content: d.AccDescr})
	}
	children = append(children, &rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewW),
		Height: svgFloat(viewH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})
	if d.Title != "" {
		children = append(children, &text{
			X:        svgFloat(viewW / 2),
			Y:        svgFloat(marginY / 2),
			Anchor:   svgutil.AnchorMiddle,
			Dominant: svgutil.BaselineCentral,
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.Text, fontSize+2),
			Content:  d.Title,
		})
	}

	// Swimlane baselines drawn first so colored branch paths and
	// commit dots paint on top. Dashed neutral line spans the full
	// commit-axis range of the diagram.
	for i := range lanes {
		x1, y1, x2, y2 := g.laneGuide(i, viewW, viewH)
		children = append(children, &line{
			X1: svgFloat(x1), Y1: svgFloat(y1),
			X2: svgFloat(x2), Y2: svgFloat(y2),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4,4;fill:none", th.LaneGuide),
		})
	}

	showBranches := svgutil.BoolOr(cfg.ShowBranches, true)
	for i, b := range lanes {
		color := colorFor(th, i)
		if showBranches {
			labelW := textmeasure.EstimateWidth(b, fontSize)
			pillW := labelW + 2*branchLabelPadX
			pillH := fontSize + 2*branchLabelPadY
			pillX, pillY := g.pillTopLeft(i, pillW, pillH)
			textX, textY := g.pillTextXY(i, pillW, pillH)
			children = append(children, &rect{
				X:      svgFloat(pillX),
				Y:      svgFloat(pillY),
				Width:  svgFloat(pillW),
				Height: svgFloat(pillH),
				RX:     svgFloat(branchLabelR), RY: svgFloat(branchLabelR),
				Style: fmt.Sprintf("fill:%s;stroke:none", color),
			})
			children = append(children, &text{
				X:        svgFloat(textX),
				Y:        svgFloat(textY),
				Anchor:   svgutil.AnchorMiddle,
				Dominant: svgutil.BaselineCentral,
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.BranchLabelText, fontSize),
				Content:  b,
			})
		}
		if r, ok := branchRange[b]; ok && showBranches {
			x1, y1, x2, y2 := g.branchPath(i, r[0], r[1])
			children = append(children, &line{
				X1: svgFloat(x1), Y1: svgFloat(y1),
				X2: svgFloat(x2), Y2: svgFloat(y2),
				Style: fmt.Sprintf("stroke:%s;stroke-width:%g;fill:none;opacity:%g", color, branchPathW, branchPathOp),
			})
		}
	}

	for _, c := range d.Commits {
		childX, childY := cx[c.ID], cy[c.ID]
		color := colorFor(th, laneOf[c.Branch])
		for _, pid := range c.Parents {
			px, ok := cx[pid]
			if !ok {
				continue
			}
			py := cy[pid]
			// Same-lane parents are already covered by the colored
			// branch path line; skip drawing a redundant connector
			// over them. Compare on the lane axis (y for LR, x for
			// TB/BT) rather than equality of both coords.
			if g.isVertical() && px == childX {
				continue
			}
			if !g.isVertical() && py == childY {
				continue
			}
			children = append(children, &path{
				D:     curvePath(px, py, childX, childY, g.isVertical()),
				Style: fmt.Sprintf("stroke:%s;stroke-width:2;fill:none;opacity:0.8", color),
			})
		}
	}

	// Dots and labels drawn last so they sit above the branch lines and
	// curves (SVG paints in document order).
	for _, c := range d.Commits {
		color := colorFor(th, laneOf[c.Branch])
		x, y := cx[c.ID], cy[c.ID]
		children = append(children, commitDot(c, x, y, color, th)...)

		// Tag callout suppresses the id label when both are set —
		// the callout already carries the visible text, so an id
		// underneath would just clutter the lane.
		if c.Tag != "" {
			// HIGHLIGHT commits are squares reaching ±highlightRadius from
			// the center; the default tagGap (14) leaves only 3px over a
			// highlight square. Widen the gap for those so the callout
			// doesn't kiss the glyph.
			gap := tagGap
			if c.Type == diagram.GitCommitHighlight {
				gap = highlightRadius + labelGap
			}
			children = append(children, tagCallout(c.Tag, x, y, gap, fontSize, th, g.isVertical())...)
		} else if c.ID != "" && svgutil.BoolOr(cfg.ShowCommitLabel, true) {
			// In vertical layouts the label sits to the right of the
			// dot (free space is along the cross axis, not above);
			// LR's "above the dot" position would push into the
			// neighbouring lane.
			var labelX, labelY float64
			anchor := svgutil.AnchorMiddle
			dom := svgutil.BaselineBaseline
			if g.isVertical() {
				labelX = x + commitRadius + labelGap
				labelY = y
				anchor = svgutil.AnchorStart
				dom = svgutil.BaselineCentral
			} else {
				labelX = x
				labelY = y - commitRadius - labelGap
			}
			t := &text{
				X:        svgFloat(labelX),
				Y:        svgFloat(labelY),
				Anchor:   anchor,
				Dominant: dom,
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.Text, fontSize-2),
				Content:  c.ID,
			}
			if svgutil.BoolOr(cfg.RotateCommitLabel, true) && !g.isVertical() {
				// LR-only rotation: long ids would overlap the next
				// commit along the same lane. In TB/BT the cross axis
				// already gives the label room, so rotation hurts more
				// than it helps.
				t.Anchor = svgutil.AnchorStart
				t.Transform = fmt.Sprintf("rotate(-45 %.2f %.2f)", labelX, labelY)
			}
			children = append(children, t)
		}
	}

	doc := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	b, err := xml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("gitgraph render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), b...), nil
}

// geom encapsulates the orientation-dependent coordinate math.
// Direction LR (default) keeps the historical "commits along x,
// lanes along y" layout. TB / BT swap the roles so commits flow
// vertically; BT additionally inverts so commit 0 sits at the
// bottom and the latest commit at the top.
type geom struct {
	dir         diagram.GitGraphDirection
	nLanes      int
	nCommits    int
	gutter      float64
	originMain  float64 // start position along the commit axis
	originCross float64 // start position along the lane axis
	// maxPillW is the widest branch pill in pixels — relevant in
	// vertical layouts because the pill, centred on its lane, can
	// extend past the lane spacing into the chart's left/right
	// margins. Used when sizing the viewport.
	maxPillW float64
}

func newGeom(dir diagram.GitGraphDirection, nLanes, nCommits int, gutter float64) geom {
	g := geom{dir: dir, nLanes: nLanes, nCommits: nCommits, gutter: gutter}
	switch dir {
	case diagram.GitGraphDirTB, diagram.GitGraphDirBT:
		// Commits flow along Y, lanes along X. The pill gutter sits
		// at the top so commitRadius keeps the first commit clear.
		g.originMain = marginY + gutter + commitRadius
		g.originCross = marginX
	default:
		g.originMain = marginX + gutter + commitRadius
		g.originCross = marginY
	}
	return g
}

// commitMain returns the pixel coord along the commit axis for index i,
// honoring BT inversion.
func (g geom) commitMain(i int) float64 {
	idx := i
	if g.dir == diagram.GitGraphDirBT {
		idx = g.nCommits - 1 - i
	}
	return g.originMain + float64(idx)*commitStride
}

// laneCross returns the pixel coord along the lane axis for lane idx.
func (g geom) laneCross(lane int) float64 {
	return g.originCross + float64(lane)*laneHeight
}

// commitXY maps (commitIdx, laneIdx) to (x, y) in the rendered SVG.
func (g geom) commitXY(i, lane int) (x, y float64) {
	main := g.commitMain(i)
	cross := g.laneCross(lane)
	if g.isVertical() {
		return cross, main
	}
	return main, cross
}

func (g geom) isVertical() bool {
	return g.dir == diagram.GitGraphDirTB || g.dir == diagram.GitGraphDirBT
}

// viewSize returns the SVG viewBox width and height. Commit axis
// extends nCommits-1 strides past origin; lane axis extends
// nLanes laneHeight units. Both axes get marginX/Y on the far edge.
//
// Vertical layouts have two extra concerns: the half-pill on either
// side of the leftmost / rightmost lane (since pills are centred on
// their lane column), and commit-id labels that sit to the right of
// each dot — both can spill past the lane-spacing extent.
func (g geom) viewSize() (w, h float64) {
	cols := g.nCommits
	if cols > 0 {
		cols--
	}
	mainExtent := g.originMain + float64(cols)*commitStride
	crossExtent := float64(max(g.nLanes, 1)) * laneHeight
	if g.isVertical() {
		// The rightmost lane sits at originCross + (nLanes-1)*laneHeight;
		// allow half-pill on its right edge plus right-of-dot label
		// budget for commit ids. The left half-pill is absorbed by
		// marginX (the canvas's natural left padding).
		halfPill := g.maxPillW / 2
		extraRight := halfPill
		if commitLabelBudget > extraRight {
			extraRight = commitLabelBudget
		}
		return g.originCross + crossExtent + extraRight + marginX,
			mainExtent + marginY
	}
	return mainExtent + marginX, g.originCross + crossExtent + marginY
}

// commitLabelBudget reserves room for the commit-id text in vertical
// layouts (where labels sit to the right of the dot). Conservative
// 80px covers most short ids without measuring each one — exact
// measurement would require threading the ruler through geom.
const commitLabelBudget = 80.0

// pillTopLeft returns the pill rect's top-left for branch lane idx.
// Width / height come from the caller.
func (g geom) pillTopLeft(lane int, pillW, pillH float64) (x, y float64) {
	cross := g.laneCross(lane)
	if g.isVertical() {
		// Pill sits in the top gutter, centred horizontally on the
		// lane's commit column.
		return cross - pillW/2, marginY + (g.gutter-pillH)/2
	}
	return marginX, cross - pillH/2
}

// pillTextXY returns (x, y) for the centred text inside the pill.
func (g geom) pillTextXY(lane int, pillW, pillH float64) (x, y float64) {
	px, py := g.pillTopLeft(lane, pillW, pillH)
	return px + pillW/2, py + pillH/2
}

// laneGuide returns the two endpoints of the dashed lane guide for
// lane idx. The guide spans the full commit range of the diagram.
func (g geom) laneGuide(lane int, viewW, viewH float64) (x1, y1, x2, y2 float64) {
	cross := g.laneCross(lane)
	if g.isVertical() {
		return cross, g.originMain - commitStride/2, cross, viewH - marginY
	}
	return g.originMain - commitStride/2, cross, viewW - marginX, cross
}

// branchPath returns the two endpoints of the colored thick line
// connecting a branch's first and last commit.
func (g geom) branchPath(lane int, mainLo, mainHi float64) (x1, y1, x2, y2 float64) {
	cross := g.laneCross(lane)
	if g.isVertical() {
		return cross, mainLo, cross, mainHi
	}
	return mainLo, cross, mainHi, cross
}

// orderedLanes returns the branch list sorted by BranchOrder ascending.
// SliceStable is required so two branches at the same order keep the
// declaration sequence — golden-file tests depend on that determinism.
func orderedLanes(d *diagram.GitGraphDiagram, mainBranchOrder int) []string {
	// Mermaid's "main branch" is whatever name MainBranchName configures;
	// the literal "main" is the spec default when the directive is
	// unset. Falling back to d.Branches[0] would mis-target the order
	// shift if the user declared a non-main branch first without
	// configuring MainBranchName.
	mainName := d.MainBranchName
	if mainName == "" {
		mainName = "main"
	}
	type lane struct {
		name  string
		order int
	}
	all := make([]lane, len(d.Branches))
	for i, b := range d.Branches {
		ord, ok := d.BranchOrder[b]
		if !ok && b == mainName {
			ord = mainBranchOrder
		}
		all[i] = lane{name: b, order: ord}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].order < all[j].order })
	out := make([]string, len(all))
	for i, l := range all {
		out[i] = l.name
	}
	return out
}

func colorFor(th Theme, lane int) string {
	return th.BranchColors[lane%len(th.BranchColors)]
}

func branchGutterW(branches []string, fontSize float64) float64 {
	var max float64
	for _, b := range branches {
		if w := textmeasure.EstimateWidth(b, fontSize); w > max {
			max = w
		}
	}
	return max
}

type dotStyle struct {
	r       float64
	fill    string
	stroke  string
	strokeW float64
	innerR  float64 // non-zero for merge commits — a small white dot on top
}

// dotStyleFor returns the circle style for a commit. Highlight and
// CherryPick commits are handled separately in commitDot (they
// render as alternative glyphs) and never reach this function.
func dotStyleFor(c diagram.GitCommit, color string, th Theme) dotStyle {
	switch c.Type {
	case diagram.GitCommitMerge:
		return dotStyle{r: commitRadius, fill: color, stroke: th.DotStrokeFill, strokeW: 2, innerR: commitRadius * 0.4}
	case diagram.GitCommitReverse:
		return dotStyle{r: commitRadius, fill: "none", stroke: color, strokeW: 3}
	default:
		return dotStyle{r: commitRadius, fill: color, stroke: th.Text, strokeW: 1.5}
	}
}

func commitDot(c diagram.GitCommit, x, y float64, color string, th Theme) []any {
	// HIGHLIGHT commits render as an outlined square instead of a
	// bigger circle — matches mmdc's "callout" glyph for emphasized
	// commits and makes them distinct from merge/reverse dots.
	if c.Type == diagram.GitCommitHighlight {
		side := highlightRadius * 2
		return []any{&rect{
			X:      svgFloat(x - highlightRadius),
			Y:      svgFloat(y - highlightRadius),
			Width:  svgFloat(side),
			Height: svgFloat(side),
			Style:  fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", th.DotStrokeFill, color),
		}}
	}
	if c.Type == diagram.GitCommitCherryPick {
		// Cherry-pick: hollow circle with a chevron-style bisecting
		// line, the closest stable approximation to mmdc's notched
		// glyph using only stroke primitives.
		r := commitRadius
		return []any{
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(r),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", th.DotStrokeFill, color),
			},
			&line{
				X1: svgFloat(x - r*0.6), Y1: svgFloat(y),
				X2: svgFloat(x + r*0.6), Y2: svgFloat(y),
				Style: fmt.Sprintf("stroke:%s;stroke-width:2", color),
			},
		}
	}
	s := dotStyleFor(c, color, th)
	elems := []any{&circle{
		CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(s.r),
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", s.fill, s.stroke, s.strokeW),
	}}
	if s.innerR > 0 {
		elems = append(elems, &circle{
			CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(s.innerR),
			Style: fmt.Sprintf("fill:%s;stroke:none", th.DotStrokeFill),
		})
	}
	return elems
}

// tagCallout draws a small rounded rect above (x,y) filled with the
// theme's tag color, containing the tag text. mmdc shows the tag as a
// floating callout distinct from the plain commit-id text.
// tagCallout places the rounded callout above the dot in LR layouts
// and to the right in TB/BT — putting it above in vertical mode
// would land it inside the previous commit's lane.
func tagCallout(tag string, x, y, gap, fontSize float64, th Theme, vertical bool) []any {
	tagFont := fontSize - 2
	tw := textmeasure.EstimateWidth(tag, tagFont)
	w := tw + 2*tagPadX
	h := tagFont + 2*tagPadY
	var bx, by, textX, textY float64
	if vertical {
		bx = x + gap
		by = y - h/2
		textX = bx + w/2
		textY = y
	} else {
		bx = x - w/2
		by = y - gap - h
		textX = x
		textY = by + h/2
	}
	return []any{
		&rect{
			X: svgFloat(bx), Y: svgFloat(by),
			Width: svgFloat(w), Height: svgFloat(h),
			RX: svgFloat(tagCornerR), RY: svgFloat(tagCornerR),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.TagFill, th.TagStroke),
		},
		&text{
			X:        svgFloat(textX),
			Y:        svgFloat(textY),
			Anchor:   svgutil.AnchorMiddle,
			Dominant: svgutil.BaselineCentral,
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.TagText, tagFont),
			Content:  tag,
		},
	}
}

// curvePath draws a cubic Bezier from parent (x1,y1) to child
// (x2,y2). In LR layouts the curve leaves horizontally and arrives
// vertically; vertical layouts swap so the curve leaves vertically
// and arrives horizontally — matches mmdc's branch-into-lane glyph
// in either orientation.
func curvePath(x1, y1, x2, y2 float64, vertical bool) string {
	if vertical {
		midY := (y1 + y2) / 2
		return fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			x1, y1, x1, midY, x2, midY, x2, y2)
	}
	midX := (x1 + x2) / 2
	return fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		x1, y1, midX, y1, midX, y2, x2, y2)
}
