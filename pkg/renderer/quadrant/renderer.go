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
)

func Render(d *diagram.QuadrantChartDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("quadrant render: diagram is nil")
	}
	th := resolveTheme(opts)
	cfg := resolveConfig(opts)
	// Options.FontSize remains the single-knob shorthand for
	// callers that predate per-element Config fields; the
	// granular Config values win when set.
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
	// Plot is always square; pick the smaller of the two
	// configured dimensions so a non-square Config still produces
	// a proportional plot rather than skewing.
	plotSide := cfg.ChartWidth
	if cfg.ChartHeight < plotSide {
		plotSide = cfg.ChartHeight
	}

	titleH := 0.0
	if d.Title != "" {
		titleH = cfg.TitleFontSize + 2*cfg.TitlePadding
	}
	xAxisAtTop := cfg.XAxisPosition == XAxisTop || (cfg.XAxisPosition == XAxisAuto && onlyBottomQuadrantsPopulated(d))
	yAxisAtRight := cfg.YAxisPosition == YAxisRight || (cfg.YAxisPosition == YAxisAuto && onlyRightQuadrantsPopulated(d))

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

	// Inner edges split QuadrantPadding (½ each) so the visible gap
	// between adjacent rects equals pad; outer edges take the full
	// pad against the plot border. Each rect's width is therefore
	// plotSide/2 − 1.5·pad; clamping at plotSide/4 keeps the rect
	// at least plotSide/8 wide so a misconfigured pad can't degenerate
	// the layout.
	type quadRect struct{ x0, y0, x1, y1 float64 }
	pad := cfg.QuadrantPadding
	if maxPad := plotSide / 4; pad > maxPad {
		pad = maxPad
	}
	half := pad / 2
	quadRects := [4]quadRect{
		{midX + half, plotY0 + pad, plotX1 - pad, midY - half},
		{plotX0 + pad, plotY0 + pad, midX - half, midY - half},
		{plotX0 + pad, midY + half, midX - half, plotY1 - pad},
		{midX + half, midY + half, plotX1 - pad, plotY1 - pad},
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

	// Top-anchored rather than centred so titles match Mermaid's
	// title-at-top-of-quadrant convention. Cap the inset at half
	// the rect height so a misconfigured padding can't push the
	// text past the rect's bottom edge into the neighbour.
	for i, q := range [4]struct {
		text string
		rect quadRect
	}{
		{d.Quadrant1, quadRects[0]},
		{d.Quadrant2, quadRects[1]},
		{d.Quadrant3, quadRects[2]},
		{d.Quadrant4, quadRects[3]},
	} {
		if q.text == "" {
			continue
		}
		fill := th.Quadrants[i].TextFill
		if fill == "" {
			fill = th.QuadrantTitleFill
		}
		textTopPad := cfg.QuadrantTextTopPadding
		if maxPad := (q.rect.y1 - q.rect.y0) / 2; textTopPad > maxPad {
			textTopPad = maxPad
		}
		cx := (q.rect.x0 + q.rect.x1) / 2
		cy := q.rect.y0 + textTopPad
		children = append(children, &text{
			X:        svgFloat(cx),
			Y:        svgFloat(cy),
			Anchor:   "middle",
			Dominant: "hanging",
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
				Y:        svgFloat(py - radius - cfg.PointTextPadding),
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

// quadrantsPopulated reports which "halves" (top / bottom /
// left / right) carry quadrant LABELS. Data points are not
// considered — Mermaid auto-flips axis labels only based on the
// quadrant captions, since a chart with points is presumed to
// have meaningful data on every side and shouldn't have its
// axes rearranged by point placement.
func quadrantsPopulated(d *diagram.QuadrantChartDiagram) (top, bottom, left, right bool) {
	top = d.Quadrant1 != "" || d.Quadrant2 != ""
	bottom = d.Quadrant3 != "" || d.Quadrant4 != ""
	left = d.Quadrant2 != "" || d.Quadrant3 != ""
	right = d.Quadrant1 != "" || d.Quadrant4 != ""
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
