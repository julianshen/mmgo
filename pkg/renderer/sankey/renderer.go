// Package sankey renders a SankeyDiagram to SVG. Nodes are drawn as
// vertical bars arranged in columns by longest-path rank; links are
// thick cubic Bezier ribbons whose width is proportional to the flow
// value, with an opacity below 1 so overlapping flows blend visibly.
package sankey

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// LinkColorMode controls how ribbon colors are picked. Source uses
// the source node's palette color; Target uses the target's;
// Gradient interpolates between the two via an SVG <linearGradient>;
// Hex paints every ribbon a literal #rrggbb color from
// Options.LinkColorHex. Source is the default.
type LinkColorMode int8

const (
	LinkColorSource LinkColorMode = iota
	LinkColorTarget
	LinkColorGradient
	LinkColorHex
)

// NodeAlignmentMode picks one of the d3-sankey column-ranking
// strategies. Left places each node as close to its sources as the
// longest-path depth allows; Right places it as close to the sinks
// (length-from-sinks); Center averages the two; Justify is Left
// with sinks pinned to the rightmost column (the historical default).
type NodeAlignmentMode int8

const (
	NodeAlignJustify NodeAlignmentMode = iota
	NodeAlignLeft
	NodeAlignRight
	NodeAlignCenter
)

type Options struct {
	FontSize float64
	Theme    Theme
	// LinkColor selects how ribbon fill is computed.
	LinkColor LinkColorMode
	// LinkColorHex supplies the literal color when LinkColor is
	// LinkColorHex. Ignored otherwise.
	LinkColorHex string
	// NodeAlignment controls horizontal column ranking. See
	// NodeAlignmentMode for caveats.
	NodeAlignment NodeAlignmentMode
	// ShowValues=false suppresses the magnitude appended to each
	// node label (`Name 5` → `Name`). Default true.
	ShowValues *bool
	// Prefix and Suffix wrap the formatted value when ShowValues
	// is on (e.g. Prefix="$", Suffix=" kW").
	Prefix string
	Suffix string
}

const (
	defaultFontSize = 13.0
	nodeW           = 18.0
	columnSpacing   = 160.0
	verticalPadding = 8.0
	marginX         = 40.0
	marginY         = 40.0
	minCanvasH      = 300.0
	labelGap        = 6.0

	ribbonOpacity = 0.45
)

func Render(d *diagram.SankeyDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("sankey render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	nodes := d.Nodes()
	nodeIdx := make(map[string]int, len(nodes))
	for i, n := range nodes {
		nodeIdx[n] = i
	}

	mode := NodeAlignJustify
	if opts != nil {
		mode = opts.NodeAlignment
	}
	col, maxCol := assignColumns(nodes, d.Flows, mode)

	// Node height = max(sumIn, sumOut) so flow is visually conserved:
	// a node that receives 10 and emits 10 has a bar tall enough for
	// either side, not both.
	sumIn := make(map[string]float64, len(nodes))
	sumOut := make(map[string]float64, len(nodes))
	for _, f := range d.Flows {
		sumOut[f.Source] += f.Value
		sumIn[f.Target] += f.Value
	}
	magnitude := make(map[string]float64, len(nodes))
	for _, n := range nodes {
		m := sumIn[n]
		if sumOut[n] > m {
			m = sumOut[n]
		}
		magnitude[n] = m
	}

	columns := make([][]string, maxCol+1)
	for _, n := range nodes {
		columns[col[n]] = append(columns[col[n]], n)
	}

	canvasH := minCanvasH
	for _, colNodes := range columns {
		sum := 0.0
		for _, n := range colNodes {
			sum += magnitude[n]
		}
		sum += float64(max(len(colNodes)-1, 0)) * verticalPadding
		if sum > canvasH {
			canvasH = sum
		}
	}

	// Compose "Name Value" labels so the total flow through each node
	// is visible next to the name (matches Mermaid's default rendering).
	// `showValues=false` suppresses the magnitude entirely; `prefix`/
	// `suffix` wrap the formatted number so authors can render
	// `$1.5M` or `5 kW`.
	showValues := true
	if opts != nil && opts.ShowValues != nil {
		showValues = *opts.ShowValues
	}
	prefix := ""
	suffix := ""
	if opts != nil {
		prefix, suffix = opts.Prefix, opts.Suffix
	}
	labelOf := func(n string) string {
		m := magnitude[n]
		if !showValues || m <= 0 {
			return n
		}
		return fmt.Sprintf("%s %s%s%s", n, prefix, svgutil.FormatNumber(m, 2), suffix)
	}

	// Labels anchor leftward (text-anchor=end) for every column except
	// the rightmost when maxCol > 0; the rightmost column anchors them
	// rightward (text-anchor=start). Reserve pad on both sides so long
	// labels don't clip outside the viewBox.
	var leftPad, rightPad float64
	for _, n := range nodes {
		w := textmeasure.EstimateWidth(labelOf(n), fontSize)
		if col[n] == maxCol && maxCol > 0 {
			if w > rightPad {
				rightPad = w
			}
		} else {
			if w > leftPad {
				leftPad = w
			}
		}
	}

	originX := marginX + leftPad
	viewW := originX + float64(maxCol)*columnSpacing + nodeW + rightPad + marginX
	viewH := canvasH + 2*marginY

	nodeY := make(map[string]float64, len(nodes))
	nodeH := make(map[string]float64, len(nodes))
	nodeX := make(map[string]float64, len(nodes))
	for c, colNodes := range columns {
		y := marginY
		for _, n := range colNodes {
			h := magnitude[n]
			if h < 1 {
				h = 1 // ensure a visible stub for zero-value leaves
			}
			nodeX[n] = originX + float64(c)*columnSpacing
			nodeY[n] = y
			nodeH[n] = h
			y += h + verticalPadding
		}
	}

	// Each node's outgoing ribbons stack top-to-bottom on the source
	// side; incoming ribbons stack top-to-bottom on the target side.
	// Ordering follows d.Flows so output is deterministic.
	srcOffset := make(map[string]float64, len(nodes))
	tgtOffset := make(map[string]float64, len(nodes))

	children := make([]any, 0, 1+len(d.Flows)+2*len(nodes))
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
		// Frontmatter `title:` renders as a centered caption above
		// the diagram body so a screen reader and a human eye see
		// the same heading text.
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(14),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   fmt.Sprintf("fill:%s;font-size:14px;font-weight:bold", th.LabelText),
			Content: d.Title,
		})
	}

	// Resolve the link-color mode once so the ribbon loop stays
	// branch-free. Gradient mode also needs a <defs> entry per
	// flow; we collect them as we go and emit a single Defs
	// element later.
	linkMode := LinkColorSource
	linkHex := ""
	if opts != nil {
		linkMode = opts.LinkColor
		linkHex = opts.LinkColorHex
	}
	colorOf := func(idx int) string {
		return th.NodeColors[idx%len(th.NodeColors)]
	}
	var gradientDefs []any

	// Ribbons before bars so bars paint over the ribbon edges.
	for i, f := range d.Flows {
		sx := nodeX[f.Source] + nodeW
		tx := nodeX[f.Target]
		syTop := nodeY[f.Source] + srcOffset[f.Source]
		tyTop := nodeY[f.Target] + tgtOffset[f.Target]
		srcOffset[f.Source] += f.Value
		tgtOffset[f.Target] += f.Value

		var fill string
		switch linkMode {
		case LinkColorTarget:
			fill = colorOf(nodeIdx[f.Target])
		case LinkColorHex:
			if linkHex != "" {
				fill = linkHex
			} else {
				fill = colorOf(nodeIdx[f.Source])
			}
		case LinkColorGradient:
			id := fmt.Sprintf("sankey-grad-%d", i)
			srcCol := colorOf(nodeIdx[f.Source])
			dstCol := colorOf(nodeIdx[f.Target])
			gradientDefs = append(gradientDefs, &linearGradient{
				ID: id, X1: "0%", Y1: "0%", X2: "100%", Y2: "0%",
				Stops: []gradientStop{
					{Offset: "0%", StopColor: srcCol},
					{Offset: "100%", StopColor: dstCol},
				},
			})
			fill = "url(#" + id + ")"
		default: // LinkColorSource
			fill = colorOf(nodeIdx[f.Source])
		}
		children = append(children, &path{
			D:     ribbonPath(sx, syTop, tx, tyTop, f.Value),
			Style: fmt.Sprintf("fill:%s;stroke:none;opacity:%.2f", fill, ribbonOpacity),
		})
	}
	if len(gradientDefs) > 0 {
		children = append(children, &sankeyDefs{Children: gradientDefs})
	}

	for _, n := range nodes {
		color := th.NodeColors[nodeIdx[n]%len(th.NodeColors)]
		children = append(children, &rect{
			X: svgFloat(nodeX[n]), Y: svgFloat(nodeY[n]),
			Width:  svgFloat(nodeW),
			Height: svgFloat(nodeH[n]),
			Style:  fmt.Sprintf("fill:%s;stroke:none", color),
		})

		labelX := nodeX[n] - labelGap
		anchor := "end"
		if col[n] == maxCol && maxCol > 0 {
			labelX = nodeX[n] + nodeW + labelGap
			anchor = "start"
		}
		children = append(children, &text{
			X:        svgFloat(labelX),
			Y:        svgFloat(nodeY[n] + nodeH[n]/2),
			Anchor:   anchor,
			Dominant: svgutil.BaselineCentral,
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LabelText, fontSize),
			Content:  labelOf(n),
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

// assignColumns returns each node's column index plus the largest
// assigned index, dispatching on NodeAlignmentMode. The fixed-point
// relaxation is capped at len(nodes) so cyclic input terminates —
// the render may end up wide but does not hang.
func assignColumns(nodes []string, flows []diagram.SankeyFlow, mode NodeAlignmentMode) (map[string]int, int) {
	depth := longestFromSources(nodes, flows)
	maxCol := 0
	for _, c := range depth {
		if c > maxCol {
			maxCol = c
		}
	}
	switch mode {
	case NodeAlignLeft:
		return depth, maxCol
	case NodeAlignRight:
		// Length-from-sinks: column = maxCol − heightFromSinks.
		height := longestToSinks(nodes, flows)
		col := make(map[string]int, len(nodes))
		for _, n := range nodes {
			col[n] = maxCol - height[n]
		}
		return col, maxCol
	case NodeAlignCenter:
		// Float each node midway between its Left and Right columns.
		height := longestToSinks(nodes, flows)
		col := make(map[string]int, len(nodes))
		for _, n := range nodes {
			left := depth[n]
			right := maxCol - height[n]
			col[n] = (left + right) / 2
		}
		return col, maxCol
	default:
		// Justify: Left assignment, but sinks (no outgoing flows)
		// are pinned to maxCol so the right edge is filled.
		hasOut := make(map[string]bool, len(nodes))
		for _, f := range flows {
			hasOut[f.Source] = true
		}
		col := make(map[string]int, len(nodes))
		for _, n := range nodes {
			c := depth[n]
			if !hasOut[n] {
				c = maxCol
			}
			col[n] = c
		}
		return col, maxCol
	}
}

// longestFromSources returns the longest-path distance from any
// source node to each node, by fixed-point relaxation along flow
// direction. Sources end up at 0; downstream nodes accumulate.
func longestFromSources(nodes []string, flows []diagram.SankeyFlow) map[string]int {
	d := make(map[string]int, len(nodes))
	for _, n := range nodes {
		d[n] = 0
	}
	for iter := 0; iter < len(nodes); iter++ {
		changed := false
		for _, f := range flows {
			if d[f.Target] < d[f.Source]+1 {
				d[f.Target] = d[f.Source] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return d
}

// longestToSinks returns the longest-path distance from each node to
// any sink — the same relaxation as longestFromSources but walking
// flows in reverse. A pure sink ends up at 0.
func longestToSinks(nodes []string, flows []diagram.SankeyFlow) map[string]int {
	h := make(map[string]int, len(nodes))
	for _, n := range nodes {
		h[n] = 0
	}
	for iter := 0; iter < len(nodes); iter++ {
		changed := false
		for _, f := range flows {
			if h[f.Source] < h[f.Target]+1 {
				h[f.Source] = h[f.Target] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return h
}

// ribbonPath returns a filled SVG path describing the ribbon between
// two vertical node faces. Both curves are cubic Beziers with
// horizontal tangents so each end enters the bar perpendicular.
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
