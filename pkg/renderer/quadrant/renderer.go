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
	Theme    Theme
	Config   Config
}

const (
	defaultFontSize = 13.0
	// Outer page-margin around the chart area. These aren't part
	// of the spec's quadrantChart config (which only governs
	// inside-the-plot geometry); they live in the renderer so
	// labels and titles always have breathing room outside the
	// plot rect.
	pageMarginX = 60.0
	pageMarginY = 40.0
	pointLabelGap = 4.0
)

func Render(d *diagram.QuadrantChartDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("quadrant render: diagram is nil")
	}
	th := resolveTheme(opts)
	cfg := resolveConfig(opts)
	// Options.FontSize is the legacy "scale every label" knob —
	// keep it working by deriving per-element sizes when callers
	// haven't set the granular Config fields. Matches the original
	// `fontSize+2 / fontSize-1 / fontSize-2` formulas.
	if opts != nil && opts.FontSize > 0 && opts.FontSize != defaultFontSize {
		if opts.Config.TitleFontSize == 0 {
			cfg.TitleFontSize = opts.FontSize + 2
		}
		if opts.Config.QuadrantLabelFontSize == 0 {
			cfg.QuadrantLabelFontSize = opts.FontSize
		}
		if opts.Config.XAxisLabelFontSize == 0 {
			cfg.XAxisLabelFontSize = max(opts.FontSize-1, 1)
		}
		if opts.Config.YAxisLabelFontSize == 0 {
			cfg.YAxisLabelFontSize = max(opts.FontSize-1, 1)
		}
		if opts.Config.PointLabelFontSize == 0 {
			cfg.PointLabelFontSize = max(opts.FontSize-2, 1)
		}
	}
	// Plot side: square of min(ChartWidth, ChartHeight) sans
	// per-side padding so squashed configs still produce a
	// proportional plot.
	plotSide := cfg.ChartWidth
	if cfg.ChartHeight < plotSide {
		plotSide = cfg.ChartHeight
	}

	titleH := 0.0
	if d.Title != "" {
		titleH = cfg.TitleFontSize + 2*cfg.TitlePadding
	}
	xAxisAtTop := cfg.XAxisPosition == AxisPositionTop || (cfg.XAxisPosition == AxisPositionAuto && onlyBottomQuadrantsPopulated(d))
	yAxisAtRight := cfg.YAxisPosition == AxisPositionRight || (cfg.YAxisPosition == AxisPositionAuto && onlyRightQuadrantsPopulated(d))

	// Each axis has its own gap because the documented config
	// supplies separate font-size + padding values for X and Y.
	xAxisGap := cfg.XAxisLabelFontSize + 2*cfg.XAxisLabelPadding
	yAxisGap := cfg.YAxisLabelFontSize + 2*cfg.YAxisLabelPadding
	leftPad := pageMarginX
	rightPad := pageMarginX
	topPad := pageMarginY + titleH
	bottomPad := pageMarginY
	if d.YAxisLow != "" || d.YAxisHigh != "" {
		if yAxisAtRight {
			rightPad += yAxisGap
		} else {
			leftPad += yAxisGap
		}
	}
	if d.XAxisLow != "" || d.XAxisHigh != "" {
		if xAxisAtTop {
			topPad += xAxisGap
		} else {
			bottomPad += xAxisGap
		}
	}

	plotX0 := leftPad
	plotY0 := topPad
	plotX1 := plotX0 + plotSide
	plotY1 := plotY0 + plotSide
	viewW := plotX1 + rightPad
	viewH := plotY1 + bottomPad

	children := make([]any, 0, 20+2*len(d.Points))
	if d.AccTitle != "" {
		children = append(children, &svgTitle{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgDesc{Content: d.AccDescr})
	}
	children = append(children, &rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewW),
		Height: svgFloat(viewH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", th.BackgroundColor),
	})

	midX := (plotX0 + plotX1) / 2
	midY := (plotY0 + plotY1) / 2
	// Plot body: a single PlotFill rect when no per-quadrant
	// fills are supplied (the typical case), else four
	// per-quadrant rects. Keeping the single-rect path preserves
	// SVG output stability when callers don't override the
	// quadrant palette.
	// Per-quadrant rect bounds in math-convention order
	// (Q1=top-right, Q2=top-left, Q3=bottom-left, Q4=bottom-right).
	type quadRect struct{ x0, y0, x1, y1 float64 }
	quadRects := [4]quadRect{
		{midX, plotY0, plotX1, midY},   // Q1 top-right
		{plotX0, plotY0, midX, midY},   // Q2 top-left
		{plotX0, midY, midX, plotY1},   // Q3 bottom-left
		{midX, midY, plotX1, plotY1},   // Q4 bottom-right
	}
	hasPerQuadrant := false
	for _, q := range th.Quadrants {
		if q.Fill != "" {
			hasPerQuadrant = true
			break
		}
	}
	if hasPerQuadrant {
		for i, qr := range quadRects {
			fill := th.Quadrants[i].Fill
			if fill == "" {
				fill = th.PlotFill
			}
			children = append(children, &rect{
				X: svgFloat(qr.x0), Y: svgFloat(qr.y0),
				Width: svgFloat(qr.x1 - qr.x0), Height: svgFloat(qr.y1 - qr.y0),
				Style: fmt.Sprintf("fill:%s;stroke:none", fill),
			})
		}
		// Outer border drawn on top of the four fills.
		children = append(children, &rect{
			X: svgFloat(plotX0), Y: svgFloat(plotY0),
			Width:  svgFloat(plotSide),
			Height: svgFloat(plotSide),
			Style:  fmt.Sprintf("fill:none;stroke:%s;stroke-width:%g", th.QuadrantExternalBorderStrokeFill, cfg.QuadrantExternalBorderStroke),
		})
	} else {
		children = append(children, &rect{
			X: svgFloat(plotX0), Y: svgFloat(plotY0),
			Width:  svgFloat(plotSide),
			Height: svgFloat(plotSide),
			Style:  fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%g", th.PlotFill, th.QuadrantExternalBorderStrokeFill, cfg.QuadrantExternalBorderStroke),
		})
	}

	if d.Title != "" {
		children = append(children, &text{
			X:        svgFloat((plotX0 + plotX1) / 2),
			Y:        svgFloat(pageMarginY + titleH/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleColor, cfg.TitleFontSize),
			Content:  d.Title,
		})
	}

	dividerStyle := fmt.Sprintf("stroke:%s;stroke-width:%g;stroke-dasharray:4 3", th.QuadrantInternalBorderStrokeFill, cfg.QuadrantInternalBorderStroke)
	children = append(children, &line{
		X1: svgFloat(plotX0), Y1: svgFloat(midY),
		X2: svgFloat(plotX1), Y2: svgFloat(midY),
		Style: dividerStyle,
	})
	children = append(children, &line{
		X1: svgFloat(midX), Y1: svgFloat(plotY0),
		X2: svgFloat(midX), Y2: svgFloat(plotY1),
		Style: dividerStyle,
	})

	// Quadrant labels — Mermaid uses math-convention numbering:
	// Q1 top-right, Q2 top-left, Q3 bottom-left, Q4 bottom-right.
	// Labels iterate over the 4 quadrants in the same index order
	// as Theme.Quadrants so text color picks straight off the
	// matching palette.
	type qLabel struct {
		text string
		x, y float64 // 0 = left/top, 1 = right/bottom within its half
	}
	labels := [4]qLabel{
		{d.Quadrant1, 1, 0}, // Q1 top-right
		{d.Quadrant2, 0, 0}, // Q2 top-left
		{d.Quadrant3, 0, 1}, // Q3 bottom-left
		{d.Quadrant4, 1, 1}, // Q4 bottom-right
	}
	quadCenter := func(x, y float64) (float64, float64) {
		return (plotX0+midX)/2 + x*(plotSide/2), (plotY0+midY)/2 + y*(plotSide/2)
	}
	for i, q := range labels {
		if q.text == "" {
			continue
		}
		fill := th.Quadrants[i].TextFill
		if fill == "" {
			fill = th.QuadrantTitleFill
		}
		cx, cy := quadCenter(q.x, q.y)
		children = append(children, &text{
			X:        svgFloat(cx),
			Y:        svgFloat(cy),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", fill, cfg.QuadrantLabelFontSize),
			Content:  q.text,
		})
	}

	xAxisStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.XAxisLabelColor, cfg.XAxisLabelFontSize)
	yAxisStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.YAxisLabelColor, cfg.YAxisLabelFontSize)
	xAxisY := plotY1 + cfg.XAxisLabelPadding
	xAxisDom := "hanging"
	if xAxisAtTop {
		xAxisY = plotY0 - cfg.XAxisLabelPadding
		xAxisDom = "auto"
	}
	if d.XAxisLow != "" {
		children = append(children, &text{
			X:        svgFloat(plotX0 + plotSide/4),
			Y:        svgFloat(xAxisY),
			Anchor:   "middle",
			Dominant: xAxisDom,
			Style:    xAxisStyle,
			Content:  d.XAxisLow,
		})
	}
	if d.XAxisHigh != "" {
		children = append(children, &text{
			X:        svgFloat(plotX0 + 3*plotSide/4),
			Y:        svgFloat(xAxisY),
			Anchor:   "middle",
			Dominant: xAxisDom,
			Style:    xAxisStyle,
			Content:  d.XAxisHigh,
		})
	}
	yAxisX := plotX0 - cfg.YAxisLabelPadding
	if yAxisAtRight {
		yAxisX = plotX1 + cfg.YAxisLabelPadding
	}
	if d.YAxisLow != "" {
		cy := plotY0 + 3*plotSide/4 // low end is bottom-quarter of the plot
		children = append(children, &text{
			X:         svgFloat(yAxisX),
			Y:         svgFloat(cy),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     yAxisStyle,
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", yAxisX, cy),
			Content:   d.YAxisLow,
		})
	}
	if d.YAxisHigh != "" {
		cy := plotY0 + plotSide/4 // high end is top-quarter of the plot
		children = append(children, &text{
			X:         svgFloat(yAxisX),
			Y:         svgFloat(cy),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     yAxisStyle,
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", yAxisX, cy),
			Content:   d.YAxisHigh,
		})
	}

	// Y is inverted: our coords put (0, 0) at bottom-left but SVG's
	// origin is top-left.
	pointLabelStyle := fmt.Sprintf("fill:%s;font-size:%.0fpx", th.QuadrantPointTextFill, cfg.PointLabelFontSize)
	for _, p := range d.Points {
		px := plotX0 + p.X*plotSide
		py := plotY1 - p.Y*plotSide
		// Resolve point styling: inline `color: …` etc. wins over
		// the referenced classDef, which wins over the theme
		// defaults Theme/Config supply.
		fill, stroke, width, radius := resolvePointStyle(p, d, th, cfg)
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
func resolvePointStyle(p diagram.QuadrantPoint, d *diagram.QuadrantChartDiagram, th Theme, cfg Config) (fill, stroke string, width, radius float64) {
	fill = th.QuadrantPointFill
	stroke = th.QuadrantPointStroke
	width = 2
	radius = cfg.PointRadius

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

// quadrantsPopulated walks the diagram once and reports whether
// the four "halves" (top / bottom / left / right) carry any
// content — either a non-empty quadrant label or a data point on
// that side. The boundary value 0.5 is treated as "high" on both
// axes (point at exactly y=0.5 → top; x=0.5 → right) so the same
// edge-case rule applies symmetrically.
func quadrantsPopulated(d *diagram.QuadrantChartDiagram) (top, bottom, left, right bool) {
	top = d.Quadrant1 != "" || d.Quadrant2 != ""
	bottom = d.Quadrant3 != "" || d.Quadrant4 != ""
	left = d.Quadrant2 != "" || d.Quadrant3 != ""
	right = d.Quadrant1 != "" || d.Quadrant4 != ""
	for _, p := range d.Points {
		if p.Y >= 0.5 {
			top = true
		} else {
			bottom = true
		}
		if p.X >= 0.5 {
			right = true
		} else {
			left = true
		}
	}
	return top, bottom, left, right
}

// onlyBottomQuadrantsPopulated reports whether the upper half of
// the plot carries no labels or points. The X-axis auto-flip
// kicks in only in that "no-content-up-top" case so the labels
// don't collide with data.
func onlyBottomQuadrantsPopulated(d *diagram.QuadrantChartDiagram) bool {
	top, bottom, _, _ := quadrantsPopulated(d)
	return bottom && !top
}

// onlyRightQuadrantsPopulated is the mirror condition for the
// Y-axis label auto-flip — left half empty means the rotated
// title can move to the right side without colliding.
func onlyRightQuadrantsPopulated(d *diagram.QuadrantChartDiagram) bool {
	_, _, left, right := quadrantsPopulated(d)
	return right && !left
}
