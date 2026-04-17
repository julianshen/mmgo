// Package sankey renders a SankeyDiagram to SVG. Nodes are drawn as
// vertical bars arranged in columns by longest-path rank; links are
// thick cubic Bezier ribbons whose width is proportional to the flow
// value, with an opacity below 1 so overlapping flows blend visibly.
package sankey

import (
	"encoding/xml"
	"fmt"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
)

type Options struct {
	FontSize float64
}

const (
	defaultFontSize  = 13.0
	nodeW            = 18.0
	columnSpacing    = 160.0
	verticalPadding  = 8.0
	marginX          = 40.0
	marginY          = 40.0
	minCanvasH       = 300.0
	labelGap         = 6.0
	avgCharWidth     = 0.6
)

// palette cycles by node index (stable first-appearance order) so the
// output is deterministic.
var palette = []string{
	"#5470c6", "#91cc75", "#fac858", "#ee6666",
	"#73c0de", "#3ba272", "#fc8452", "#9a60b4",
}

func Render(d *diagram.SankeyDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("sankey render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	nodes := d.Nodes()
	nodeIdx := make(map[string]int, len(nodes))
	for i, n := range nodes {
		nodeIdx[n] = i
	}

	// --- 1. Column assignment via longest-path rank.
	col := assignColumns(nodes, d.Flows)

	// --- 2. Compute per-node magnitude = max(sumIn, sumOut). Bar
	// height is proportional to this, so flow is visually conserved.
	sumIn := make(map[string]float64, len(nodes))
	sumOut := make(map[string]float64, len(nodes))
	for _, f := range d.Flows {
		sumOut[f.Source] += f.Value
		sumIn[f.Target] += f.Value
	}
	magnitude := make(map[string]float64, len(nodes))
	totalValue := 0.0
	for _, n := range nodes {
		m := sumIn[n]
		if sumOut[n] > m {
			m = sumOut[n]
		}
		magnitude[n] = m
		totalValue += m
	}

	// --- 3. Vertical layout per column. Unit scale so the tallest
	// column fits the canvas minus margins.
	columns := make(map[int][]string)
	maxCol := 0
	for _, n := range nodes {
		c := col[n]
		columns[c] = append(columns[c], n)
		if c > maxCol {
			maxCol = c
		}
	}
	var canvasH float64 = minCanvasH
	for c := 0; c <= maxCol; c++ {
		sum := 0.0
		for _, n := range columns[c] {
			sum += magnitude[n]
		}
		sum += float64(max(len(columns[c])-1, 0)) * verticalPadding
		if sum > canvasH {
			canvasH = sum
		}
	}

	// --- 4. Longest label width for outer margin on the right side.
	maxLabel := 0
	for _, n := range nodes {
		if len(n) > maxLabel {
			maxLabel = len(n)
		}
	}
	labelPad := fontSize * avgCharWidth * float64(maxLabel)

	viewW := 2*marginX + float64(maxCol)*columnSpacing + nodeW + labelPad
	viewH := canvasH + 2*marginY

	// --- 5. Position nodes: y is cumulative stack top of each column.
	nodeY := make(map[string]float64, len(nodes))
	nodeH := make(map[string]float64, len(nodes))
	nodeX := make(map[string]float64, len(nodes))
	for c := 0; c <= maxCol; c++ {
		y := marginY
		for _, n := range columns[c] {
			h := magnitude[n]
			if h < 1 {
				h = 1 // minimum bar height for visibility
			}
			nodeX[n] = marginX + float64(c)*columnSpacing
			nodeY[n] = y
			nodeH[n] = h
			y += h + verticalPadding
		}
	}

	// --- 6. Link ribbons. For each node, stack outgoing ribbons top-
	// to-bottom on the source side and incoming ribbons top-to-bottom
	// on the target side in the order they appear in d.Flows — this
	// keeps output deterministic and visually consistent.
	srcOffset := make(map[string]float64, len(nodes))
	tgtOffset := make(map[string]float64, len(nodes))

	children := []any{
		&rect{
			X: 0, Y: 0,
			Width:  svgFloat(viewW),
			Height: svgFloat(viewH),
			Style:  "fill:#fff;stroke:none",
		},
	}

	// Ribbons first so nodes paint over the ribbon edges.
	for _, f := range d.Flows {
		sx := nodeX[f.Source] + nodeW
		tx := nodeX[f.Target]
		syTop := nodeY[f.Source] + srcOffset[f.Source]
		tyTop := nodeY[f.Target] + tgtOffset[f.Target]
		srcOffset[f.Source] += f.Value
		tgtOffset[f.Target] += f.Value

		color := palette[nodeIdx[f.Source]%len(palette)]
		children = append(children, &path{
			D:     ribbonPath(sx, syTop, tx, tyTop, f.Value),
			Style: fmt.Sprintf("fill:%s;stroke:none;opacity:0.45", color),
		})
	}

	for i, n := range nodes {
		color := palette[i%len(palette)]
		children = append(children, &rect{
			X: svgFloat(nodeX[n]), Y: svgFloat(nodeY[n]),
			Width:  svgFloat(nodeW),
			Height: svgFloat(nodeH[n]),
			Style:  fmt.Sprintf("fill:%s;stroke:none", color),
		})

		// Label anchored left or right of the bar depending on column
		// so labels don't overlap adjacent ribbons.
		labelX := nodeX[n] - labelGap
		anchor := "end"
		if col[n] == 0 {
			labelX = nodeX[n] - labelGap
			anchor = "end"
		}
		if col[n] == maxCol {
			labelX = nodeX[n] + nodeW + labelGap
			anchor = "start"
		}
		children = append(children, &text{
			X:        svgFloat(labelX),
			Y:        svgFloat(nodeY[n] + nodeH[n]/2),
			Anchor:   anchor,
			Dominant: "central",
			Style:    fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize),
			Content:  n,
		})
	}

	doc := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	b, err := xml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("sankey render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), b...), nil
}

// assignColumns returns a column index per node using longest-path
// rank from any source (node with no incoming flow). Cycles are
// broken by capping iteration at len(nodes) — a flow graph should be
// a DAG in practice, but the cap keeps pathological input bounded.
func assignColumns(nodes []string, flows []diagram.SankeyFlow) map[string]int {
	col := make(map[string]int, len(nodes))
	for _, n := range nodes {
		col[n] = 0
	}
	// Iterate until no change or we hit the cap.
	for iter := 0; iter < len(nodes); iter++ {
		changed := false
		for _, f := range flows {
			if col[f.Target] < col[f.Source]+1 {
				col[f.Target] = col[f.Source] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	// Stabilize the per-column node order by first-appearance in
	// `nodes` (already the case), but sort by column as well so the
	// outer loop visits columns in order.
	sort.SliceStable(nodes, func(i, j int) bool { return col[nodes[i]] < col[nodes[j]] })
	return col
}

// ribbonPath builds a filled SVG path for a flow ribbon between two
// rectangular endpoints. The curve is a cubic Bezier with horizontal
// tangents at both ends so the ribbon enters/exits each node bar
// perpendicular to the bar face.
func ribbonPath(sx, syTop, tx, tyTop, value float64) string {
	midX := (sx + tx) / 2
	syBot := syTop + value
	tyBot := tyTop + value
	return fmt.Sprintf(
		"M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f L%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f Z",
		sx, syTop,
		midX, syTop, midX, tyTop, tx, tyTop,
		tx, tyBot,
		midX, tyBot, midX, syBot, sx, syBot,
	)
}
