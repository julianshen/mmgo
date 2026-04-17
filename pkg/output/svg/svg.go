// Package svg is the end-to-end Mermaid → SVG entry point. It wires
// the parser, layout engine, and renderer behind a single Render call:
//
//	svgBytes, err := svg.Render(strings.NewReader(input), nil)
//
// Currently supports flowchart/graph, sequence, and pie diagrams.
package svg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/config"
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
	classparser "github.com/julianshen/mmgo/pkg/parser/class"
	erparser "github.com/julianshen/mmgo/pkg/parser/er"
	ganttparser "github.com/julianshen/mmgo/pkg/parser/gantt"
	mindmapparser "github.com/julianshen/mmgo/pkg/parser/mindmap"
	blockparser "github.com/julianshen/mmgo/pkg/parser/block"
	c4parser "github.com/julianshen/mmgo/pkg/parser/c4"
	timelineparser "github.com/julianshen/mmgo/pkg/parser/timeline"
	flowchartparser "github.com/julianshen/mmgo/pkg/parser/flowchart"
	pieparser "github.com/julianshen/mmgo/pkg/parser/pie"
	sequenceparser "github.com/julianshen/mmgo/pkg/parser/sequence"
	stateparser "github.com/julianshen/mmgo/pkg/parser/state"
	classrenderer "github.com/julianshen/mmgo/pkg/renderer/class"
	errenderer "github.com/julianshen/mmgo/pkg/renderer/er"
	ganttrenderer "github.com/julianshen/mmgo/pkg/renderer/gantt"
	mindmaprenderer "github.com/julianshen/mmgo/pkg/renderer/mindmap"
	blockrenderer "github.com/julianshen/mmgo/pkg/renderer/block"
	c4renderer "github.com/julianshen/mmgo/pkg/renderer/c4"
	timelinerenderer "github.com/julianshen/mmgo/pkg/renderer/timeline"
	flowchartrenderer "github.com/julianshen/mmgo/pkg/renderer/flowchart"
	pierenderer "github.com/julianshen/mmgo/pkg/renderer/pie"
	sequencerenderer "github.com/julianshen/mmgo/pkg/renderer/sequence"
	staterenderer "github.com/julianshen/mmgo/pkg/renderer/state"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// Options configures the end-to-end pipeline. All fields are optional;
// nil opts uses defaults end-to-end.
type Options struct {
	// Layout.RankDir is intentionally ignored — direction comes from
	// the diagram header.
	Layout layout.Options
	Theme  config.ThemeName
	Flowchart *flowchartrenderer.Options
	Sequence  *sequencerenderer.Options
	Pie       *pierenderer.Options
}

// Sizing constants for nodes when no caller-specified theme overrides
// are present. Padding chosen to leave breathing room around the
// label; minimums chosen so empty/short labels still render at a
// readable size.
const (
	nodePaddingX     = 30.0
	nodePaddingY     = 20.0
	minNodeWidth     = 60.0
	minNodeHeight    = 40.0
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
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("svg render: read input: %w", err)
	}

	src, initCfg := extractInitDirective(raw)
	opts = mergeInitTheme(opts, initCfg)

	kind, err := detectDiagramKind(src)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	switch kind {
	case kindFlowchart:
		return renderFlowchart(src, opts)
	case kindSequence:
		return renderSequence(src, opts)
	case kindPie:
		return renderPie(src, opts)
	case kindClass:
		return renderClass(src, opts)
	case kindState:
		return renderState(src, opts)
	case kindER:
		return renderER(src, opts)
	case kindGantt:
		return renderGantt(src, opts)
	case kindMindmap:
		return renderMindmap(src, opts)
	case kindTimeline:
		return renderTimeline(src, opts)
	case kindC4:
		return renderC4(src, opts)
	case kindBlock:
		return renderBlock(src, opts)
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
	kindSequence
	kindPie
	kindClass
	kindState
	kindER
	kindGantt
	kindMindmap
	kindTimeline
	kindC4
	kindBlock
)

func (k diagramKind) String() string {
	switch k {
	case kindFlowchart:
		return "flowchart"
	case kindSequence:
		return "sequence"
	case kindPie:
		return "pie"
	case kindClass:
		return "class"
	case kindState:
		return "state"
	case kindER:
		return "er"
	case kindGantt:
		return "gantt"
	case kindMindmap:
		return "mindmap"
	case kindTimeline:
		return "timeline"
	case kindC4:
		return "c4"
	case kindBlock:
		return "block"
	default:
		return "unknown"
	}
}

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
		if hasHeaderKeyword(line, "sequenceDiagram") {
			return kindSequence, nil
		}
		if hasHeaderKeyword(line, "pie") {
			return kindPie, nil
		}
		if hasHeaderKeyword(line, "classDiagram") {
			return kindClass, nil
		}
		if hasHeaderKeyword(line, "stateDiagram-v2") || hasHeaderKeyword(line, "stateDiagram") {
			return kindState, nil
		}
		if hasHeaderKeyword(line, "erDiagram") {
			return kindER, nil
		}
		if hasHeaderKeyword(line, "gantt") {
			return kindGantt, nil
		}
		if hasHeaderKeyword(line, "mindmap") {
			return kindMindmap, nil
		}
		if hasHeaderKeyword(line, "timeline") {
			return kindTimeline, nil
		}
		switch line {
		case "C4Context", "C4Container", "C4Component", "C4Dynamic", "C4Deployment":
			return kindC4, nil
		}
		if hasHeaderKeyword(line, "block-beta") {
			return kindBlock, nil
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
// extractInitDirective strips `%%{init: {...}}%%` lines from src and
// returns the cleaned source plus the parsed JSON config (nil if none).
func extractInitDirective(src []byte) ([]byte, *config.Config) {
	scanner := bufio.NewScanner(bytes.NewReader(src))
	var cleaned []byte
	var cfg *config.Config
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "%%{init:") && strings.HasSuffix(trimmed, "}%%") {
			inner := strings.TrimPrefix(trimmed, "%%{init:")
			inner = strings.TrimSuffix(inner, "}%%")
			inner = strings.TrimSpace(inner)
			var c config.Config
			if json.Unmarshal([]byte(inner), &c) == nil {
				if c.Theme == "" {
					c.Theme = config.ThemeDefault
				}
				cfg = &c
			}
			continue
		}
		if len(cleaned) > 0 {
			cleaned = append(cleaned, '\n')
		}
		cleaned = append(cleaned, []byte(line)...)
	}
	if err := scanner.Err(); err != nil {
		return src, nil
	}
	return cleaned, cfg
}

func mergeInitTheme(opts *Options, initCfg *config.Config) *Options {
	if initCfg == nil && (opts == nil || opts.Theme == "") {
		return opts
	}
	theme := config.ThemeDefault
	if opts != nil && opts.Theme != "" {
		theme = opts.Theme
	}
	if initCfg != nil && initCfg.Theme != "" {
		theme = initCfg.Theme
	}
	tc, err := config.BuiltInTheme(theme)
	if err != nil {
		return opts
	}
	merged := &Options{}
	if opts != nil {
		*merged = *opts
	}
	// Clone pointer fields so we don't mutate the caller's structs.
	if merged.Flowchart != nil {
		clone := *merged.Flowchart
		merged.Flowchart = &clone
	} else {
		merged.Flowchart = &flowchartrenderer.Options{}
	}
	merged.Flowchart.Theme = toFlowchartTheme(tc)

	if merged.Sequence != nil {
		clone := *merged.Sequence
		merged.Sequence = &clone
	} else {
		merged.Sequence = &sequencerenderer.Options{}
	}
	merged.Sequence.Theme = toSequenceTheme(tc)

	return merged
}

func toFlowchartTheme(tc *config.ThemeColors) flowchartrenderer.Theme {
	return flowchartrenderer.Theme{
		NodeFill:       tc.Primary,
		NodeStroke:     tc.LineColor,
		NodeText:       tc.Text,
		EdgeStroke:     tc.LineColor,
		EdgeText:       tc.Text,
		SubgraphFill:   tc.Tertiary,
		SubgraphStroke: tc.LineColor,
		SubgraphText:   tc.Text,
		Background:     tc.Background,
	}
}

func toSequenceTheme(tc *config.ThemeColors) sequencerenderer.Theme {
	return sequencerenderer.Theme{
		Background:        tc.Background,
		ParticipantFill:   tc.Secondary,
		ParticipantStroke: tc.LineColor,
		ParticipantText:   tc.Text,
		LifelineStroke:    tc.LineColor,
		MessageStroke:     tc.LineColor,
		MessageText:       tc.Text,
		NoteFill:          tc.NoteFill,
	}
}

func renderFlowchart(src []byte, opts *Options) ([]byte, error) {
	d, err := flowchartparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("svg render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

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

	fcopts := &flowchartrenderer.Options{}
	if opts != nil && opts.Flowchart != nil {
		clone := *opts.Flowchart
		fcopts = &clone
	}
	// Share the ruler we already built for node sizing so the renderer
	// doesn't re-parse the bundled TTF.
	fcopts.Ruler = ruler
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
func renderSequence(src []byte, opts *Options) ([]byte, error) {
	d, err := sequenceparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	var sopts *sequencerenderer.Options
	if opts != nil && opts.Sequence != nil {
		clone := *opts.Sequence
		sopts = &clone
	}
	out, err := sequencerenderer.Render(d, sopts)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderClass(src []byte, opts *Options) ([]byte, error) {
	d, err := classparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := classrenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderState(src []byte, opts *Options) ([]byte, error) {
	d, err := stateparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := staterenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderER(src []byte, opts *Options) ([]byte, error) {
	d, err := erparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := errenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderGantt(src []byte, opts *Options) ([]byte, error) {
	d, err := ganttparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := ganttrenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderBlock(src []byte, opts *Options) ([]byte, error) {
	d, err := blockparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := blockrenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderC4(src []byte, opts *Options) ([]byte, error) {
	d, err := c4parser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := c4renderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderTimeline(src []byte, opts *Options) ([]byte, error) {
	d, err := timelineparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := timelinerenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderMindmap(src []byte, opts *Options) ([]byte, error) {
	d, err := mindmapparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	out, err := mindmaprenderer.Render(d, nil)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func renderPie(src []byte, opts *Options) ([]byte, error) {
	d, err := pieparser.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("svg render: parse: %w", err)
	}
	var popts *pierenderer.Options
	if opts != nil && opts.Pie != nil {
		clone := *opts.Pie
		popts = &clone
	}
	out, err := pierenderer.Render(d, popts)
	if err != nil {
		return nil, fmt.Errorf("svg render: %w", err)
	}
	return out, nil
}

func flowchartFontSize(opts *Options) float64 {
	if opts != nil && opts.Flowchart != nil && opts.Flowchart.FontSize > 0 {
		return opts.Flowchart.FontSize
	}
	return flowchartrenderer.DefaultFontSize
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
// DirectionUnknown and DirectionTB both map to RankDirTB (top-to-bottom)
// because TB is the Mermaid default when no direction is specified.
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
