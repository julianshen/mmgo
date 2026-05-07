// Package quadrant renders a QuadrantChartDiagram to SVG as a square
// plot split into four labeled quadrants by a horizontal and vertical
// center line. Points are placed by normalized (x, y) in [0, 1] where
// (0, 0) is the bottom-left corner.
package quadrant

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
)

type Options struct {
	FontSize float64
}

const (
	defaultFontSize = 13.0
	marginX         = 60.0
	marginY         = 40.0
	titleGap        = 28.0
	axisLabelGap    = 20.0
	plotSide        = 400.0
	pointRadius     = 7.0
	pointLabelGap   = 4.0

	bgFill         = "#fff"
	plotFill       = "#f7f7fa"
	dividerStroke  = "#bbb"
	borderStroke   = "#888"
	quadrantFill   = "#555"
	labelFill      = "#333"
	pointFill      = "#5470c6"
	pointStroke    = "#fff"
	pointLabelFill = "#222"
)

func Render(d *diagram.QuadrantChartDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("quadrant render: diagram is nil")
	}
	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	titleH := 0.0
	if d.Title != "" {
		titleH = titleGap
	}
	// Leave room for axis labels on the left and bottom.
	leftPad := marginX + axisLabelGap
	bottomPad := marginY + axisLabelGap
	if d.YAxisLow != "" || d.YAxisHigh != "" {
		leftPad += titleGap
	}
	if d.XAxisLow != "" || d.XAxisHigh != "" {
		bottomPad += titleGap
	}

	plotX0 := leftPad
	plotY0 := marginY + titleH
	plotX1 := plotX0 + plotSide
	plotY1 := plotY0 + plotSide
	viewW := plotX1 + marginX
	viewH := plotY1 + bottomPad

	children := make([]any, 0, 16+2*len(d.Points))
	children = append(children, &rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewW),
		Height: svgFloat(viewH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", bgFill),
	})
	children = append(children, &rect{
		X: svgFloat(plotX0), Y: svgFloat(plotY0),
		Width:  svgFloat(plotSide),
		Height: svgFloat(plotSide),
		Style:  fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", plotFill, borderStroke),
	})

	if d.Title != "" {
		children = append(children, &text{
			X:        svgFloat((plotX0 + plotX1) / 2),
			Y:        svgFloat(marginY + titleH/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", labelFill, fontSize+2),
			Content:  d.Title,
		})
	}

	midX := (plotX0 + plotX1) / 2
	midY := (plotY0 + plotY1) / 2
	children = append(children, &line{
		X1: svgFloat(plotX0), Y1: svgFloat(midY),
		X2: svgFloat(plotX1), Y2: svgFloat(midY),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4 3", dividerStroke),
	})
	children = append(children, &line{
		X1: svgFloat(midX), Y1: svgFloat(plotY0),
		X2: svgFloat(midX), Y2: svgFloat(plotY1),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1;stroke-dasharray:4 3", dividerStroke),
	})

	// Quadrant labels — Mermaid uses math-convention numbering:
	// Q1 top-right, Q2 top-left, Q3 bottom-left, Q4 bottom-right.
	quadCenter := func(x, y float64) (float64, float64) {
		return (plotX0+midX)/2 + x*(plotSide/2), (plotY0+midY)/2 + y*(plotSide/2)
	}
	type qLabel struct {
		text string
		x, y float64 // 0 = left/top, 1 = right/bottom within its half
	}
	labels := []qLabel{
		{d.Quadrant2, 0, 0}, // top-left
		{d.Quadrant1, 1, 0}, // top-right
		{d.Quadrant3, 0, 1}, // bottom-left
		{d.Quadrant4, 1, 1}, // bottom-right
	}
	for _, q := range labels {
		if q.text == "" {
			continue
		}
		cx, cy := quadCenter(q.x, q.y)
		children = append(children, &text{
			X:        svgFloat(cx),
			Y:        svgFloat(cy),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", quadrantFill, fontSize),
			Content:  q.text,
		})
	}

	// Clamp derived sizes so tiny custom FontSize values don't produce
	// zero- or negative-pixel labels.
	axisFontSize := max(fontSize-1, 1)
	pointLabelFontSize := max(fontSize-2, 1)
	axisFontStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", labelFill, axisFontSize)
	if d.XAxisLow != "" {
		children = append(children, &text{
			X:        svgFloat(plotX0 + plotSide/4),
			Y:        svgFloat(plotY1 + axisLabelGap),
			Anchor:   "middle",
			Dominant: "hanging",
			Style:    axisFontStyle,
			Content:  d.XAxisLow,
		})
	}
	if d.XAxisHigh != "" {
		children = append(children, &text{
			X:        svgFloat(plotX0 + 3*plotSide/4),
			Y:        svgFloat(plotY1 + axisLabelGap),
			Anchor:   "middle",
			Dominant: "hanging",
			Style:    axisFontStyle,
			Content:  d.XAxisHigh,
		})
	}
	if d.YAxisLow != "" {
		cx := plotX0 - axisLabelGap
		cy := plotY0 + 3*plotSide/4 // low end is bottom-quarter of the plot
		children = append(children, &text{
			X:         svgFloat(cx),
			Y:         svgFloat(cy),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     axisFontStyle,
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", cx, cy),
			Content:   d.YAxisLow,
		})
	}
	if d.YAxisHigh != "" {
		cx := plotX0 - axisLabelGap
		cy := plotY0 + plotSide/4 // high end is top-quarter of the plot
		children = append(children, &text{
			X:         svgFloat(cx),
			Y:         svgFloat(cy),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     axisFontStyle,
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", cx, cy),
			Content:   d.YAxisHigh,
		})
	}

	// Y is inverted: our coords put (0, 0) at bottom-left but SVG's
	// origin is top-left.
	pointLabelStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", pointLabelFill, pointLabelFontSize)
	for _, p := range d.Points {
		px := plotX0 + p.X*plotSide
		py := plotY1 - p.Y*plotSide
		// Resolve point styling: inline `color: …` etc. wins over
		// the referenced classDef, which wins over the theme
		// defaults the constants supply.
		fill, stroke, width, radius := resolvePointStyle(p, d)
		children = append(children, &circle{
			CX: svgFloat(px), CY: svgFloat(py), R: svgFloat(radius),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", fill, stroke, width),
		})
		if p.Label != "" {
			children = append(children, &text{
				X:        svgFloat(px),
				Y:        svgFloat(py - radius - pointLabelGap),
				Anchor:   "middle",
				Dominant: "baseline",
				Style:    pointLabelStyle,
				Content:  p.Label,
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
		return nil, fmt.Errorf("quadrant render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), b...), nil
}


// resolvePointStyle layers inline > class > theme defaults to
// produce the final (fill, stroke, stroke-width, radius) for a
// quadrant point. Zero-valued numeric fields fall through to the
// next layer rather than masking the theme as 0px circles.
func resolvePointStyle(p diagram.QuadrantPoint, d *diagram.QuadrantChartDiagram) (fill, stroke string, width, radius float64) {
	fill = pointFill
	stroke = pointStroke
	width = 2
	radius = pointRadius

	if p.Class != "" {
		if cls, ok := d.Classes[p.Class]; ok {
			if cls.Color != "" {
				fill = cls.Color
			}
			if cls.StrokeColor != "" {
				stroke = cls.StrokeColor
			}
			if cls.Radius > 0 {
				radius = cls.Radius
			}
			if cls.StrokeWidth > 0 {
				width = cls.StrokeWidth
			}
		}
	}
	if p.Style.Color != "" {
		fill = p.Style.Color
	}
	if p.Style.StrokeColor != "" {
		stroke = p.Style.StrokeColor
	}
	if p.Style.Radius > 0 {
		radius = p.Style.Radius
	}
	if p.Style.StrokeWidth > 0 {
		width = p.Style.StrokeWidth
	}
	return fill, stroke, width, radius
}
