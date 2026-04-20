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
	branchLabelPadX = 8.0

	labelFill     = "#333"
	dotStrokeFill = "#fff"
)

// branchPalette is cycled by branch declaration order. Lane 0 (main)
// gets the first color; additional branches cycle through the rest.
var branchPalette = []string{
	"#0f62fe", "#24a148", "#f1c21b",
	"#8a3ffc", "#ff7eb6", "#6fdc8c",
}

func Render(d *diagram.GitGraphDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("gitgraph render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	laneOf := make(map[string]int, len(d.Branches))
	for i, b := range d.Branches {
		laneOf[b] = i
	}

	originX := marginX + branchGutterW(d.Branches, fontSize) + 2*branchLabelPadX

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
			Style:  "fill:#fff;stroke:none",
		},
	}

	for i, b := range d.Branches {
		color := branchPalette[i%len(branchPalette)]
		y := laneY(i)
		children = append(children, &text{
			X:        svgFloat(marginX + branchLabelPadX),
			Y:        svgFloat(y),
			Anchor:   "start",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", color, fontSize),
			Content:  b,
		})
		if r, ok := branchRange[b]; ok {
			children = append(children, &line{
				X1: svgFloat(r[0]), Y1: svgFloat(y),
				X2: svgFloat(r[1]), Y2: svgFloat(y),
				Style: fmt.Sprintf("stroke:%s;stroke-width:3;fill:none;opacity:0.6", color),
			})
		}
	}

	for _, c := range d.Commits {
		childX, childY := cx[c.ID], cy[c.ID]
		color := colorFor(laneOf[c.Branch])
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
		color := colorFor(laneOf[c.Branch])
		x, y := cx[c.ID], cy[c.ID]
		children = append(children, commitDot(c, x, y, color)...)

		label := c.Tag
		if label == "" {
			label = c.ID
		}
		if label != "" {
			children = append(children, &text{
				X:        svgFloat(x),
				Y:        svgFloat(y - commitRadius - labelGap),
				Anchor:   "middle",
				Dominant: "baseline",
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", labelFill, fontSize-2),
				Content:  label,
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

func laneY(lane int) float64   { return marginY + float64(lane)*laneHeight }
func colorFor(lane int) string { return branchPalette[lane%len(branchPalette)] }

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

func dotStyleFor(c diagram.GitCommit, color string) dotStyle {
	switch c.Type {
	case diagram.GitCommitMerge:
		return dotStyle{r: commitRadius, fill: color, stroke: dotStrokeFill, strokeW: 2, innerR: commitRadius * 0.4}
	case diagram.GitCommitHighlight:
		return dotStyle{r: highlightRadius, fill: color, stroke: labelFill, strokeW: 2}
	case diagram.GitCommitReverse:
		return dotStyle{r: commitRadius, fill: "none", stroke: color, strokeW: 3}
	default:
		return dotStyle{r: commitRadius, fill: color, stroke: labelFill, strokeW: 1.5}
	}
}

func commitDot(c diagram.GitCommit, x, y float64, color string) []any {
	s := dotStyleFor(c, color)
	elems := []any{&circle{
		CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(s.r),
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", s.fill, s.stroke, s.strokeW),
	}}
	if s.innerR > 0 {
		elems = append(elems, &circle{
			CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(s.innerR),
			Style: fmt.Sprintf("fill:%s;stroke:none", dotStrokeFill),
		})
	}
	return elems
}

// curvePath draws a cubic Bezier that leaves parent (x1,y1) horizontal
// and arrives at child (x2,y2) vertical — the Mermaid gitgraph style.
func curvePath(x1, y1, x2, y2 float64) string {
	midX := (x1 + x2) / 2
	return fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		x1, y1, midX, y1, midX, y2, x2, y2)
}
