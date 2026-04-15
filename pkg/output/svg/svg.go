// Package svg is the end-to-end Mermaid → SVG entry point. It wires
// the parser, layout engine, and renderer behind a single Render call:
//
//	svgBytes, err := svg.Render(strings.NewReader(input), nil)
//
// Currently supports flowchart/graph diagrams; sequence and pie land
// alongside their renderers.
package svg

import (
	"bufio"
	"bytes"
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

// Options configures the end-to-end pipeline. All fields are optional;
// nil opts uses defaults end-to-end.
type Options struct {
	// Layout forwards spacing knobs (NodeSep, RankSep) to the layout
	// engine. RankDir is intentionally ignored — direction comes from
	// the parsed diagram header (`graph LR`, `flowchart TB`, ...) so
	// the rendered output matches the input verbatim.
	Layout layout.Options
	// Flowchart is forwarded to the flowchart renderer (theme, padding,
	// font size, ExtraCSS). Nil uses renderer defaults.
	Flowchart *flowchartrenderer.Options
}

// Sizing constants for nodes when no caller-specified theme overrides
// are present. Padding chosen to leave breathing room around the
// label; minimums chosen so empty/short labels still render at a
// readable size.
const (
	defaultFontSize = 16.0
	nodePaddingX    = 30.0
	nodePaddingY    = 20.0
	minNodeWidth    = 60.0
	minNodeHeight   = 40.0
	lineHeightFactor = 1.2
)

// Render reads a Mermaid diagram from r, runs the full
// parse → measure → layout → render pipeline, and returns the SVG
// document bytes. The diagram type is sniffed from the first
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
		return nil, fmt.Errorf("svg render: %v diagrams are not yet supported", kind)
	}
}

// diagramKind is a coarse classification of supported Mermaid headers.
// More entries land alongside their renderer.
type diagramKind int8

const (
	kindUnknown diagramKind = iota
	kindFlowchart
)

// detectDiagramKind sniffs the first non-blank, non-comment line of
// src for a recognized header keyword. This pre-check exists so we
// can return a clean "X diagrams not yet supported" error before
// invoking a parser that doesn't know about X.
func detectDiagramKind(src []byte) (diagramKind, error) {
	scanner := bufio.NewScanner(bytes.NewReader(src))
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
// either end-of-string or whitespace, mirroring the parser's
// word-boundary rule so `grapha` is not mis-matched as `graph`.
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

// renderFlowchart runs parse → size → layout → render for a flowchart
// diagram. The font size used for node sizing is read from the
// flowchart renderer's Options so node boxes and rendered text always
// agree, even when the caller customizes it.
func renderFlowchart(src []byte, opts *Options) ([]byte, error) {
	d, err := flowchartparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("svg render: text measurer: %w", err)
	}
	defer ruler.Close()

	fontSize := flowchartFontSize(opts)
	g := buildFlowchartGraph(d, ruler, fontSize)

	lopts := layout.Options{}
	if opts != nil {
		lopts = opts.Layout
	}
	// Direction always comes from the diagram header — ignore any
	// caller-supplied RankDir to keep the output faithful to the input.
	lopts.RankDir = directionToRankDir(d.Direction)

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

// flowchartFontSize returns the font size used for both node sizing
// and the renderer. Reads from opts.Flowchart.FontSize so a single
// caller setting flows end-to-end; falls back to defaultFontSize when
// the caller hasn't specified one.
func flowchartFontSize(opts *Options) float64 {
	if opts != nil && opts.Flowchart != nil && opts.Flowchart.FontSize > 0 {
		return opts.Flowchart.FontSize
	}
	return defaultFontSize
}

// buildFlowchartGraph converts a parsed flowchart AST into a layout
// graph. Uses the AST walkers in pkg/diagram so subgraph-nested nodes
// and scoped edges (which the AST stores ONLY in Subgraph.Nodes /
// Subgraph.Edges) are included automatically.
func buildFlowchartGraph(d *diagram.FlowchartDiagram, ruler *textmeasure.Ruler, fontSize float64) *graph.Graph {
	g := graph.New()
	for _, n := range d.AllNodes() {
		w, h := nodeSize(n.Label, ruler, fontSize)
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: w, Height: h})
	}
	for _, e := range d.AllEdges() {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}
	return g
}

// nodeSize returns the padded (width, height) for a node label,
// clamped to a readable minimum so empty/short labels still render
// visibly.
func nodeSize(label string, ruler *textmeasure.Ruler, fontSize float64) (w, h float64) {
	if label == "" {
		return minNodeWidth, minNodeHeight
	}
	mw, mh := ruler.Measure(label, fontSize)
	w = mw + nodePaddingX
	h = mh*lineHeightFactor + nodePaddingY
	if w < minNodeWidth {
		w = minNodeWidth
	}
	if h < minNodeHeight {
		h = minNodeHeight
	}
	return w, h
}

// directionToRankDir maps the parsed Direction enum to the layout
// RankDir enum. They are intentionally separate types (parser concept
// vs. layout concept) so this translator is the seam.
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
