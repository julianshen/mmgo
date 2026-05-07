// Package gitgraph renders a GitGraphDiagram to SVG as horizontal
// swim lanes: one row per branch, commits as dots along the X axis in
// declaration order, and cross-branch parent links as curved connectors.
package gitgraph

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

type Options struct {
	FontSize float64
	Theme    Theme
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

	laneOf := make(map[string]int, len(d.Branches))
	for i, b := range d.Branches {
		laneOf[b] = i
	}

	// `2*branchLabelPadX` covers the pill's own horizontal padding;
	// `commitRadius` keeps the first commit's dot clear of the pill
	// even when that commit sits on the branch with the longest name
	// (otherwise the dot's left edge would overlap the pill's right).
	originX := marginX + branchGutterW(d.Branches, fontSize) + 2*branchLabelPadX + commitRadius

	cx := make(map[string]float64, len(d.Commits))
	cy := make(map[string]float64, len(d.Commits))
	// x is monotonic in commit index, so only the first and last x per
	// branch are ever needed; no min/max reduction required.
	branchRange := make(map[string][2]float64, len(d.Branches))
	for i, c := range d.Commits {
		x := originX + float64(i)*commitStride
		cx[c.ID] = x
		cy[c.ID] = laneY(laneOf[c.Branch])
		if r, ok := branchRange[c.Branch]; ok {
			r[1] = x
			branchRange[c.Branch] = r
		} else {
			branchRange[c.Branch] = [2]float64{x, x}
		}
	}

	cols := len(d.Commits)
	if cols > 0 {
		cols--
	}
	viewW := originX + float64(cols)*commitStride + marginX
	viewH := marginY + float64(max(len(d.Branches), 1))*laneHeight + marginY

	children := []any{
		&rect{
			X: 0, Y: 0,
			Width:  svgFloat(viewW),
			Height: svgFloat(viewH),
			Style:  fmt.Sprintf("fill:%s;stroke:none", th.Background),
		},
	}

	// Swimlane baselines drawn first so colored branch paths and
	// commit dots paint on top. Dashed neutral line runs full-width
	// across each lane, matching mmdc's faint lane guide.
	baselineX1 := originX - commitStride/2
	baselineX2 := viewW - marginX
	for i := range d.Branches {
		y := laneY(i)
		children = append(children, &line{
			X1: svgFloat(baselineX1), Y1: svgFloat(y),
			X2: svgFloat(baselineX2), Y2: svgFloat(y),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4,4;fill:none", th.LaneGuide),
		})
	}

	for i, b := range d.Branches {
		color := colorFor(th, i)
		y := laneY(i)
		// Pill label: rounded rect filled with the branch color, white
		// text centered inside — matches mmdc's branch-tag affordance.
		labelW := textmeasure.EstimateWidth(b, fontSize)
		pillW := labelW + 2*branchLabelPadX
		pillH := fontSize + 2*branchLabelPadY
		children = append(children, &rect{
			X:      svgFloat(marginX),
			Y:      svgFloat(y - pillH/2),
			Width:  svgFloat(pillW),
			Height: svgFloat(pillH),
			RX:     svgFloat(branchLabelR), RY: svgFloat(branchLabelR),
			Style: fmt.Sprintf("fill:%s;stroke:none", color),
		})
		children = append(children, &text{
			X:        svgFloat(marginX + pillW/2),
			Y:        svgFloat(y),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.BranchLabelText, fontSize),
			Content:  b,
		})
		if r, ok := branchRange[b]; ok {
			children = append(children, &line{
				X1: svgFloat(r[0]), Y1: svgFloat(y),
				X2: svgFloat(r[1]), Y2: svgFloat(y),
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
			if py == childY {
				continue // same-branch link is covered by the branch line
			}
			children = append(children, &path{
				D:     curvePath(px, py, childX, childY),
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

		// Tag takes precedence over id when both exist: mmdc shows the
		// tag as a rounded callout above the commit, the commit id as
		// plain text below.
		if c.Tag != "" {
			// HIGHLIGHT commits are squares reaching ±highlightRadius from
			// the center; the default tagGap (14) leaves only 3px over a
			// highlight square. Widen the gap for those so the callout
			// doesn't kiss the glyph.
			gap := tagGap
			if c.Type == diagram.GitCommitHighlight {
				gap = highlightRadius + labelGap
			}
			children = append(children, tagCallout(c.Tag, x, y, gap, fontSize, th)...)
		} else if c.ID != "" {
			children = append(children, &text{
				X:        svgFloat(x),
				Y:        svgFloat(y - commitRadius - labelGap),
				Anchor:   "middle",
				Dominant: "baseline",
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.Text, fontSize-2),
				Content:  c.ID,
			})
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

func laneY(lane int) float64 { return marginY + float64(lane)*laneHeight }

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
func tagCallout(tag string, x, y, gap, fontSize float64, th Theme) []any {
	tagFont := fontSize - 2
	tw := textmeasure.EstimateWidth(tag, tagFont)
	w := tw + 2*tagPadX
	h := tagFont + 2*tagPadY
	bx := x - w/2
	by := y - gap - h
	return []any{
		&rect{
			X: svgFloat(bx), Y: svgFloat(by),
			Width: svgFloat(w), Height: svgFloat(h),
			RX: svgFloat(tagCornerR), RY: svgFloat(tagCornerR),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.TagFill, th.TagStroke),
		},
		&text{
			X:        svgFloat(x),
			Y:        svgFloat(by + h/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.TagText, tagFont),
			Content:  tag,
		},
	}
}

// curvePath draws a cubic Bezier that leaves parent (x1,y1) horizontal
// and arrives at child (x2,y2) vertical — the Mermaid gitgraph style.
func curvePath(x1, y1, x2, y2 float64) string {
	midX := (x1 + x2) / 2
	return fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		x1, y1, midX, y1, midX, y2, x2, y2)
}
