package flowchart

import (
	"encoding/xml"
	"fmt"
	"math"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

// sanitizeDimension clamps NaN, Inf, and negative values to 0 so they
// can never reach SVG numeric attributes. A degenerate layout (empty
// diagram, all-missing nodes) becomes a 0×0 viewBox plus padding.
func sanitizeDimension(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return v
}

func Render(d *diagram.FlowchartDiagram, l *layout.Result, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("flowchart render: diagram is nil")
	}
	if l == nil {
		return nil, fmt.Errorf("flowchart render: layout is nil")
	}

	pad := resolvePadding(opts)
	th := resolveTheme(opts)
	fontSize := resolveFontSize(opts)
	bg := resolveBackground(opts, th)

	// Subgraph title bands extend above the layout's top-most node
	// rect by depth × titleBand. Instead of inflating `pad` on all
	// four sides, only grow the viewBox downward by topInset and shift
	// all rendered content by the same amount through a transform
	// group — keeps the left/right/bottom margins lean.
	topInset := float64(maxSubgraphDepth(d.Subgraphs)) * subgraphTitleBand(fontSize)

	ruler := rulerFromOpts(opts)
	if ruler == nil {
		r, err := textmeasure.NewDefaultRuler()
		if err != nil {
			return nil, fmt.Errorf("flowchart render: text measurer init: %w", err)
		}
		defer func() { _ = r.Close() }()
		ruler = r
	}

	// A nil/zero/negative layout size still produces a valid SVG: clamp
	// to a non-negative minimum so a NaN/Inf layout (degenerate input)
	// never reaches the formatter.
	viewBoxW := sanitizeDimension(l.Width) + 2*pad
	viewBoxH := sanitizeDimension(l.Height) + 2*pad + topInset

	children := []any{
		buildDefs(d, th),
	}

	if d.Title != "" {
		children = append(children, &Title{Content: d.Title})
	}
	if d.AccTitle != "" {
		children = append(children, &Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &Desc{Content: d.AccDescr})
	}

	classCSS := buildClassCSS(d)
	extraCSS := ""
	if opts != nil {
		extraCSS = opts.ExtraCSS
	}
	if classCSS != "" || extraCSS != "" {
		parts := []string{}
		if classCSS != "" {
			parts = append(parts, classCSS)
		}
		if extraCSS != "" {
			parts = append(parts, extraCSS)
		}
		children = append(children, &StyleEl{Content: strings.Join(parts, "\n")})
	}

	children = append(children, &Rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewBoxW),
		Height: svgFloat(viewBoxH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", bg),
	})

	if topInset > 0 {
		content := &Group{Transform: fmt.Sprintf("translate(0,%.2f)", topInset)}
		content.Children = append(content.Children, renderSubgraphs(d, l, pad, th, fontSize)...)
		content.Children = append(content.Children, renderEdges(d, l, pad, th, fontSize, ruler)...)
		content.Children = append(content.Children, renderNodes(d, l, pad, th, fontSize)...)
		children = append(children, content)
	} else {
		children = append(children, renderSubgraphs(d, l, pad, th, fontSize)...)
		children = append(children, renderEdges(d, l, pad, th, fontSize, ruler)...)
		children = append(children, renderNodes(d, l, pad, th, fontSize)...)
	}

	svg := SVG{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewBoxW, viewBoxH),
		Children: children,
	}

	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("flowchart render: %w", err)
	}

	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

func buildDefs(d *diagram.FlowchartDiagram, th Theme) *Defs {
	return &Defs{Markers: buildMarkers(d, th)}
}

func renderNodes(d *diagram.FlowchartDiagram, l *layout.Result, pad float64, th Theme, fontSize float64) []any {
	var elems []any
	for _, n := range d.AllNodes() {
		nl, ok := l.Nodes[n.ID]
		if !ok {
			continue
		}
		nodeElems := renderNode(n, nl, pad, th, fontSize)
		applyStyleOverrides(nodeElems, n, d.Styles)
		applyClassAttr(nodeElems, n)
		elems = append(elems, nodeElems...)
	}
	return elems
}

// applyStyleOverrides appends the user-supplied per-node CSS to the
// shape's existing style (fill/stroke/stroke-width from the theme).
// CSS later-wins semantics mean the override wins for any property it
// sets while leaving the rest of the base style intact.
func applyStyleOverrides(elems []any, n diagram.Node, styles []diagram.StyleDef) {
	css := nodeStyleCSS(n, styles)
	if css == "" {
		return
	}
	setFirstElement(elems, func(s styleClassSetter) { s.appendStyle(css) })
}

func applyClassAttr(elems []any, n diagram.Node) {
	if len(n.Classes) == 0 {
		return
	}
	classVal := strings.Join(n.Classes, " ")
	setFirstElement(elems, func(s styleClassSetter) { s.setClass(classVal) })
}

type styleClassSetter interface {
	appendStyle(string)
	setClass(string)
}

// setFirstElement walks elems and applies fn to the first element that
// implements styleClassSetter. Walking (rather than indexing [0]) means
// the style/class apply correctly even if a future shape emits a
// non-stylable element first (e.g., a leading <line>).
func setFirstElement(elems []any, fn func(styleClassSetter)) {
	for _, e := range elems {
		if s, ok := e.(styleClassSetter); ok {
			fn(s)
			return
		}
	}
}

// mergeStyle joins two CSS declaration lists with `;`, taking care
// not to introduce a leading or duplicate separator. Later declarations
// win under standard CSS later-wins semantics.
func mergeStyle(base, extra string) string {
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	if strings.HasSuffix(base, ";") {
		return base + extra
	}
	return base + ";" + extra
}

func (r *Rect) appendStyle(v string)    { r.Style = mergeStyle(r.Style, v) }
func (r *Rect) setClass(v string)       { r.Class = v }
func (p *Polygon) appendStyle(v string) { p.Style = mergeStyle(p.Style, v) }
func (p *Polygon) setClass(v string)    { p.Class = v }
func (c *Circle) appendStyle(v string)  { c.Style = mergeStyle(c.Style, v) }
func (c *Circle) setClass(v string)     { c.Class = v }
func (p *Path) appendStyle(v string)    { p.Style = mergeStyle(p.Style, v) }
func (p *Path) setClass(v string)       { p.Class = v }
