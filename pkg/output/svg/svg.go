// Package svg is the end-to-end Mermaid → SVG entry point. It ties
// together the parser, layout engine, and renderer so callers don't
// need to orchestrate the individual phases:
//
//	svgBytes, err := svg.Render(strings.NewReader(input), nil)
//
// Currently supports flowchart/graph diagrams. Sequence and pie support
// will be added in the corresponding renderer steps.
package svg

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	flowchartparser "github.com/julianshen/mmgo/pkg/parser/flowchart"
	flowchartrenderer "github.com/julianshen/mmgo/pkg/renderer/flowchart"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// Options configures the end-to-end pipeline.
type Options struct {
	// Layout configures the layout engine (rank direction, spacing).
	// If RankDir is the zero value, it is auto-derived from the parsed
	// diagram's Direction header (`graph LR` → RankDirLR, etc).
	Layout layout.Options
	// Flowchart is forwarded to the flowchart renderer when the input
	// is a flowchart diagram. Nil uses renderer defaults.
	Flowchart *flowchartrenderer.Options
	// FontSize sets the default font size for node labels (used both
	// for sizing nodes and for the renderer's text metrics). When 0 it
	// falls back to defaultFontSize.
	FontSize float64
}

// Default sizing constants. Node dimensions are computed from the
// label's measured bounding box plus padding, then clamped to a
// readable minimum.
const (
	defaultFontSize  = 16.0
	nodePaddingX     = 30.0
	nodePaddingY     = 20.0
	minNodeWidth     = 60.0
	minNodeHeight    = 40.0
	nodeLineHeightK  = 1.2
)

// Render reads a Mermaid diagram from r, runs the full
// parse → measure → layout → render pipeline, and returns the SVG
// document bytes. The diagram type is detected from the first
// non-comment, non-blank line.
func Render(r io.Reader, opts *Options) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("svg render: reader is nil")
	}
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("svg render: read input: %w", err)
	}

	kind, err := detectDiagramKind(src)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	switch kind {
	case kindFlowchart:
		return renderFlowchart(src, opts)
	default:
		return nil, fmt.Errorf("svg render: %s diagrams are not yet supported", kind)
	}
}

// diagramKind is a coarse classification of supported Mermaid headers.
// More entries land alongside their renderer (sequence in Step 13, pie
// in Step 14, etc.).
type diagramKind string

const (
	kindFlowchart diagramKind = "flowchart"
	kindUnknown   diagramKind = "unknown"
)

// detectDiagramKind scans the first non-blank, non-comment line of src
// for a recognized header keyword. This avoids the cost of trying to
// fully parse with each parser to find the matching one.
func detectDiagramKind(src []byte) (diagramKind, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(src)))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		if hasHeaderKeyword(line, "graph") || hasHeaderKeyword(line, "flowchart") {
			return kindFlowchart, nil
		}
		return kindUnknown, fmt.Errorf("unrecognized diagram header: %q", line)
	}
	if err := scanner.Err(); err != nil {
		return kindUnknown, fmt.Errorf("scan input: %w", err)
	}
	return kindUnknown, fmt.Errorf("empty input: no diagram header found")
}

// hasHeaderKeyword reports whether line begins with kw followed by
// either end-of-string or whitespace. Mirrors the parser's word-
// boundary rule so `grapha` is not misread as `graph`.
func hasHeaderKeyword(line, kw string) bool {
	if !strings.HasPrefix(line, kw) {
		return false
	}
	if len(line) == len(kw) {
		return true
	}
	c := line[len(kw)]
	return c == ' ' || c == '\t'
}

// renderFlowchart runs the flowchart pipeline: parse, build the layout
// graph (sizing nodes via textmeasure), layout, render. The ruler is
// initialized once and shared between node sizing and the renderer's
// edge-label measurement, so a font init failure is reported exactly
// once with a clean error.
func renderFlowchart(src []byte, opts *Options) ([]byte, error) {
	d, err := flowchartparser.Parse(strings.NewReader(string(src)))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("svg render: text measurer: %w", err)
	}
	defer ruler.Close()

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	g := buildFlowchartGraph(d, ruler, fontSize)

	lopts := layout.Options{}
	if opts != nil {
		lopts = opts.Layout
	}
	if lopts.RankDir == layout.RankDirTB && d.Direction != diagram.DirectionTB {
		// Caller didn't pin a RankDir; honor the diagram header.
		lopts.RankDir = directionToRankDir(d.Direction)
	}

	l := layout.Layout(g, lopts)

	var fcopts *flowchartrenderer.Options
	if opts != nil {
		fcopts = opts.Flowchart
	}
	out, err := flowchartrenderer.Render(d, l, fcopts)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

// buildFlowchartGraph converts a parsed flowchart AST into a layout
// graph. It walks d.Subgraphs recursively so that nodes nested inside
// `subgraph ... end` blocks (which the AST stores ONLY in the
// Subgraph.Nodes slice) are included, and so are their scoped edges.
//
// Node dimensions come from textmeasure: the label's widest line
// drives width, the line count drives height, and both are clamped to
// a readable minimum. This produces SVG output where a node always
// fits its label without truncation.
func buildFlowchartGraph(d *diagram.FlowchartDiagram, ruler *textmeasure.Ruler, fontSize float64) *graph.Graph {
	g := graph.New()

	addNode := func(n diagram.Node) {
		w, h := nodeSize(n.Label, ruler, fontSize)
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: w, Height: h})
	}
	addEdge := func(e diagram.Edge) {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}

	for _, n := range d.Nodes {
		addNode(n)
	}
	for _, e := range d.Edges {
		addEdge(e)
	}
	var walk func(sgs []diagram.Subgraph)
	walk = func(sgs []diagram.Subgraph) {
		for i := range sgs {
			for _, n := range sgs[i].Nodes {
				addNode(n)
			}
			for _, e := range sgs[i].Edges {
				addEdge(e)
			}
			walk(sgs[i].Children)
		}
	}
	walk(d.Subgraphs)

	return g
}

// nodeSize measures label and returns padded (width, height) for the
// layout engine. Empty labels still get a readable minimum box so
// bare `A --> B` style references render visibly.
func nodeSize(label string, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	if label == "" {
		return minNodeWidth, minNodeHeight
	}
	mw, mh := ruler.Measure(label, fontSize)
	w = mw + nodePaddingX
	h = mh*nodeLineHeightK + nodePaddingY
	if w < minNodeWidth {
		w = minNodeWidth
	}
	if h < minNodeHeight {
		h = minNodeHeight
	}
	return w, h
}

// directionToRankDir maps the parsed diagram's Direction enum to the
// layout engine's RankDir. The two enums are intentionally separate
// (one is a parser concept, one is a layout concept), so we need a
// translation here.
func directionToRankDir(d diagram.Direction) layout.RankDir {
	switch d {
	case diagram.DirectionBT:
		return layout.RankDirBT
	case diagram.DirectionLR:
		return layout.RankDirLR
	case diagram.DirectionRL:
		return layout.RankDirRL
	default:
		return layout.RankDirTB
	}
}
