package flowchart

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

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

	viewBoxW := l.Width + 2*pad
	viewBoxH := l.Height + 2*pad

	children := []any{
		buildDefs(d, th),
	}

	classCSS := buildClassCSS(d)
	if classCSS != "" || (opts != nil && opts.CSSFile != "") {
		cssContent := classCSS
		if opts != nil {
			cssContent += opts.CSSFile
		}
		children = append(children, &StyleEl{Content: cssContent})
	}

	children = append(children, &Rect{
		X: 0, Y: 0,
		Width:  viewBoxW,
		Height: viewBoxH,
		Style:  fmt.Sprintf("fill:%s;stroke:none", bg),
	})

	children = append(children, renderSubgraphs(d, l, pad, th, fontSize)...)
	children = append(children, renderEdges(d, l, pad, th, fontSize)...)
	children = append(children, renderNodes(d, l, pad, th, fontSize)...)

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
	for _, n := range d.Nodes {
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

func applyStyleOverrides(elems []any, n diagram.Node, styles []diagram.StyleDef) {
	css := nodeStyleCSS(n, styles)
	if css == "" || len(elems) == 0 {
		return
	}
	switch e := elems[0].(type) {
	case *Rect:
		e.Style = css
	case *Polygon:
		e.Style = css
	case *Circle:
		e.Style = css
	case *Path:
		e.Style = css
	}
}

func applyClassAttr(elems []any, n diagram.Node) {
	if len(n.Classes) == 0 || len(elems) == 0 {
		return
	}
	classVal := strings.Join(n.Classes, " ")
	switch e := elems[0].(type) {
	case *Rect:
		e.Class = classVal
	case *Polygon:
		e.Class = classVal
	case *Circle:
		e.Class = classVal
	case *Path:
		e.Class = classVal
	}
}
