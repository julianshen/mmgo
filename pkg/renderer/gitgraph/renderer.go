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

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("gitgraph render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	laneOf := make(map[string]int, len(d.Branches))
	for i, b := range d.Branches {
		laneOf[b] = i
	}

	// Reserve left-margin space for the longest branch label so the
	// first commit column doesn't overlap.
	branchLabelW := 0.0
	for _, b := range d.Branches {
		if w, _ := ruler.Measure(b, fontSize); w > branchLabelW {
			branchLabelW = w
		}
	}
	originX := marginX + branchLabelW + 2*branchLabelPadX

	cx := make(map[string]float64, len(d.Commits))
	cy := make(map[string]float64, len(d.Commits))
	branchRange := make(map[string][2]float64, len(d.Branches))
	for i, c := range d.Commits {
		x := originX + float64(i)*commitStride
		y := laneY(laneOf[c.Branch])
		cx[c.ID] = x
		cy[c.ID] = y
		if r, ok := branchRange[c.Branch]; ok {
			if x < r[0] {
				r[0] = x
			}
			if x > r[1] {
				r[1] = x
			}
			branchRange[c.Branch] = r
		} else {
			branchRange[c.Branch] = [2]float64{x, x}
		}
	}

	viewW := originX + float64(maxCommits(d.Commits))*commitStride + marginX
	viewH := marginY + float64(max(len(d.Branches), 1))*laneHeight + marginY

	children := []any{
		&rect{
			X: 0, Y: 0,
			Width:  svgFloat(viewW),
			Height: svgFloat(viewH),
			Style:  "fill:#fff;stroke:none",
		},
	}

	// Branch labels + horizontal branch lines. Declaration order is
	// deterministic, so SVG output is stable.
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

	// Cross-branch parent links (merge/fork). Same-branch links are
	// covered by the branch line.
	for _, c := range d.Commits {
		childX, childY := cx[c.ID], cy[c.ID]
		color := branchPalette[laneOf[c.Branch]%len(branchPalette)]
		for _, pid := range c.Parents {
			px, ok := cx[pid]
			if !ok {
				continue
			}
			py := cy[pid]
			if py == childY {
				continue
			}
			children = append(children, &path{
				D:     curvePath(px, py, childX, childY),
				Style: fmt.Sprintf("stroke:%s;stroke-width:2;fill:none;opacity:0.8", color),
			})
		}
	}

	// Commit dots and labels drawn last so they sit above the branch
	// lines and curves.
	for _, c := range d.Commits {
		lane := laneOf[c.Branch]
		color := branchPalette[lane%len(branchPalette)]
		x, y := cx[c.ID], cy[c.ID]
		children = append(children, commitShape(c, x, y, color)...)

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
				Style:    fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-2),
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

func laneY(lane int) float64 { return marginY + float64(lane)*laneHeight }

// maxCommits returns len(cs)-1 clamped to >=0 so viewBox width accounts
// for the final column, which lives at `origin + (n-1)*stride`.
func maxCommits(cs []diagram.GitCommit) int {
	if len(cs) == 0 {
		return 0
	}
	return len(cs) - 1
}

// commitShape returns the SVG element(s) for a commit dot, varied by
// commit type so Highlight / Reverse / Merge are visually distinct.
func commitShape(c diagram.GitCommit, x, y float64, color string) []any {
	switch c.Type {
	case diagram.GitCommitMerge:
		return []any{
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(commitRadius),
				Style: fmt.Sprintf("fill:%s;stroke:#fff;stroke-width:2", color),
			},
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(commitRadius * 0.4),
				Style: "fill:#fff;stroke:none",
			},
		}
	case diagram.GitCommitHighlight:
		return []any{
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(highlightRadius),
				Style: fmt.Sprintf("fill:%s;stroke:#333;stroke-width:2", color),
			},
		}
	case diagram.GitCommitReverse:
		return []any{
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(commitRadius),
				Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:3", color),
			},
		}
	default:
		return []any{
			&circle{
				CX: svgFloat(x), CY: svgFloat(y), R: svgFloat(commitRadius),
				Style: fmt.Sprintf("fill:%s;stroke:#333;stroke-width:1.5", color),
			},
		}
	}
}

// curvePath draws a short cubic Bezier that leaves parent (x1,y1)
// horizontally and arrives at child (x2,y2) vertically, matching
// Mermaid's style for branch/merge connectors.
func curvePath(x1, y1, x2, y2 float64) string {
	midX := (x1 + x2) / 2
	return fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
		x1, y1, midX, y1, midX, y2, x2, y2)
}
