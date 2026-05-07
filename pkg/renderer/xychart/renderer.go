// Package xychart renders an XYChartDiagram to SVG.
package xychart

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type Options struct {
	FontSize float64
	Theme    Theme
	Config   Config
}

const (
	defaultFontSize = 13.0
	marginX         = 50.0
	marginY         = 40.0
	barInsetRatio   = 0.15
	yRangeHeadroom  = 0.10
	yTickTarget     = 6
)

func Render(d *diagram.XYChartDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("xychart render: diagram is nil")
	}
	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)
	cfg := resolveConfig(opts)

	horizontal := isHorizontal(d, cfg)

	categories := d.XAxis.Categories
	if len(categories) == 0 {
		// Even in continuous-X mode (HasRange) the parser does not
		// capture per-point x values, so series get laid out across
		// synthetic index slots; the x-axis ticks render numeric
		// values from the explicit range while the series still
		// project onto these slot positions.
		n := maxSeriesLen(d.Series)
		categories = make([]string, n)
		for i := range categories {
			categories[i] = strconv.Itoa(i + 1)
		}
	}
	yMin, yMax := valueRange(d)
	if yMin == yMax {
		yMax = yMin + 1
	}

	titleH := 0.0
	if d.Title != "" && flag(cfg.ShowTitle, true) {
		titleH = cfg.TitlePadding + cfg.TitleFontSize + cfg.TitlePadding
	}
	// Which axis ends up on which edge depends on orientation. cfg.XAxis
	// always describes the AST x-axis (categories in standard usage);
	// in horizontal mode that axis appears on the left of the canvas.
	leftAxisCfg, bottomAxisCfg := cfg.YAxis, cfg.XAxis
	leftAxisDef, bottomAxisDef := d.YAxis, d.XAxis
	if horizontal {
		leftAxisCfg, bottomAxisCfg = cfg.XAxis, cfg.YAxis
		leftAxisDef, bottomAxisDef = d.XAxis, d.YAxis
	}
	leftAxisPad := axisLabelGap(leftAxisCfg)
	if leftAxisDef.Title != "" && flag(leftAxisCfg.ShowTitle, true) {
		leftAxisPad += leftAxisCfg.TitleFontSize + leftAxisCfg.TitlePadding
	}
	bottomAxisPad := axisLabelGap(bottomAxisCfg)
	if bottomAxisDef.Title != "" && flag(bottomAxisCfg.ShowTitle, true) {
		bottomAxisPad += bottomAxisCfg.TitleFontSize + bottomAxisCfg.TitlePadding
	}

	plotX0 := marginX + leftAxisPad
	plotY0 := marginY + titleH
	plotX1 := cfg.Width - marginX
	plotY1 := cfg.Height - marginY - bottomAxisPad
	if plotX1 <= plotX0 {
		plotX1 = plotX0 + 1
	}
	if plotY1 <= plotY0 {
		plotY1 = plotY0 + 1
	}
	viewW := cfg.Width
	viewH := cfg.Height

	children := []any{}
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

	if d.Title != "" && flag(cfg.ShowTitle, true) {
		children = append(children, &text{
			X:        svgFloat((plotX0 + plotX1) / 2),
			Y:        svgFloat(marginY + cfg.TitlePadding + cfg.TitleFontSize/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleColor, cfg.TitleFontSize),
			Content:  d.Title,
		})
	}

	_ = fontSize // FontSize broadcasts via resolveConfig; not needed here.
	if horizontal {
		children = append(children, renderHorizontal(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, cfg, th)...)
	} else {
		children = append(children, renderVertical(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, cfg, th)...)
	}

	doc := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	b, err := xml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("xychart render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), b...), nil
}

// isHorizontal resolves the orientation: an explicit Config override
// always wins; otherwise the AST's Horizontal flag (set by the parser
// when the source is `xychart-beta horizontal`) decides.
func isHorizontal(d *diagram.XYChartDiagram, cfg Config) bool {
	switch cfg.ChartOrientation {
	case OrientationHorizontal:
		return true
	case OrientationVertical:
		return false
	default:
		return d.Horizontal
	}
}

func axisLabelGap(a AxisConfig) float64 {
	gap := 0.0
	if flag(a.ShowTick, true) {
		gap += a.TickLength
	}
	if flag(a.ShowLabel, true) {
		gap += a.LabelFontSize + a.LabelPadding
	}
	return gap
}

// valueRange returns the value-axis [min, max]. The clamp-to-zero
// floor for all-non-negative data keeps bars anchored to a visible
// baseline rather than floating mid-plot.
func valueRange(d *diagram.XYChartDiagram) (float64, float64) {
	if d.YAxis.HasRange {
		return d.YAxis.Min, d.YAxis.Max
	}
	lo := math.Inf(1)
	hi := math.Inf(-1)
	for _, s := range d.Series {
		for _, v := range s.Data {
			if v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
		}
	}
	if math.IsInf(lo, 0) {
		return 0, 1
	}
	if lo > 0 {
		lo = 0
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	hi += span * yRangeHeadroom
	return lo, hi
}

func maxSeriesLen(series []diagram.XYSeries) int {
	n := 0
	for _, s := range series {
		if len(s.Data) > n {
			n = len(s.Data)
		}
	}
	return n
}

type tickEdge int

const (
	edgeBottom tickEdge = iota
	edgeLeft
)

type tickItem struct {
	pos   float64
	label string
}

// gridSpec bundles the three coupled gridline fields. A zero stroke
// disables grid emission; lo/hi are then ignored.
type gridSpec struct {
	lo, hi float64
	stroke string
}

type tickRow struct {
	edge tickEdge
	// axisOrigin is y1 for edgeBottom, x0 for edgeLeft — the coupling
	// is non-obvious because the field name doesn't carry the edge.
	axisOrigin float64
	items      []tickItem
	grid       gridSpec
	axisCfg    AxisConfig
	tickColor  string
	labelColor string
}

func renderTickRow(r tickRow) []any {
	var out []any
	showTick := flag(r.axisCfg.ShowTick, true)
	showLabel := flag(r.axisCfg.ShowLabel, true)
	for _, it := range r.items {
		if r.grid.stroke != "" {
			switch r.edge {
			case edgeBottom:
				out = append(out, &line{
					X1: svgFloat(it.pos), Y1: svgFloat(r.grid.lo),
					X2: svgFloat(it.pos), Y2: svgFloat(r.grid.hi),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", r.grid.stroke),
				})
			case edgeLeft:
				out = append(out, &line{
					X1: svgFloat(r.grid.lo), Y1: svgFloat(it.pos),
					X2: svgFloat(r.grid.hi), Y2: svgFloat(it.pos),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", r.grid.stroke),
				})
			default:
				panic(fmt.Sprintf("xychart: unknown tickEdge %d", r.edge))
			}
		}
		if showTick {
			switch r.edge {
			case edgeBottom:
				out = append(out, &line{
					X1: svgFloat(it.pos), Y1: svgFloat(r.axisOrigin),
					X2: svgFloat(it.pos), Y2: svgFloat(r.axisOrigin + r.axisCfg.TickLength),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", r.tickColor),
				})
			case edgeLeft:
				out = append(out, &line{
					X1: svgFloat(r.axisOrigin - r.axisCfg.TickLength), Y1: svgFloat(it.pos),
					X2: svgFloat(r.axisOrigin), Y2: svgFloat(it.pos),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", r.tickColor),
				})
			default:
				panic(fmt.Sprintf("xychart: unknown tickEdge %d", r.edge))
			}
		}
		if showLabel {
			switch r.edge {
			case edgeBottom:
				out = append(out, &text{
					X:        svgFloat(it.pos),
					Y:        svgFloat(r.axisOrigin + r.axisCfg.TickLength + r.axisCfg.LabelPadding),
					Anchor:   "middle",
					Dominant: "hanging",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", r.labelColor, r.axisCfg.LabelFontSize),
					Content:  it.label,
				})
			case edgeLeft:
				out = append(out, &text{
					X:        svgFloat(r.axisOrigin - r.axisCfg.TickLength - r.axisCfg.LabelPadding),
					Y:        svgFloat(it.pos),
					Anchor:   "end",
					Dominant: "central",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", r.labelColor, r.axisCfg.LabelFontSize),
					Content:  it.label,
				})
			default:
				panic(fmt.Sprintf("xychart: unknown tickEdge %d", r.edge))
			}
		}
	}
	return out
}

// continuousTicks projects niceTicks values onto pixel positions.
func continuousTicks(lo, hi, pxLo, pxHi float64) []tickItem {
	vals := niceTicks(lo, hi, yTickTarget)
	items := make([]tickItem, 0, len(vals))
	for _, v := range vals {
		t := (v - lo) / (hi - lo)
		items = append(items, tickItem{pos: pxLo + t*(pxHi-pxLo), label: formatTick(v)})
	}
	return items
}

// categoricalTicks centres labels in their slot.
func categoricalTicks(categories []string, pxLo, pxHi float64) []tickItem {
	items := make([]tickItem, len(categories))
	for i, c := range categories {
		items[i] = tickItem{pos: categoryCenter(i, len(categories), pxLo, pxHi), label: c}
	}
	return items
}

func renderVertical(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1 float64, cfg Config, th Theme) []any {
	var elems []any

	elems = append(elems, renderTickRow(tickRow{
		edge:       edgeLeft,
		items:      continuousTicks(yMin, yMax, y1, y0),
		axisOrigin: x0,
		grid:       gridSpec{lo: x0, hi: x1, stroke: th.GridStroke},
		axisCfg:    cfg.YAxis,
		tickColor:  th.YAxisTickColor,
		labelColor: th.YAxisLabelColor,
	})...)

	var xItems []tickItem
	if d.XAxis.HasRange && len(d.XAxis.Categories) == 0 {
		xItems = continuousTicks(d.XAxis.Min, d.XAxis.Max, x0, x1)
	} else {
		xItems = categoricalTicks(categories, x0, x1)
	}
	elems = append(elems, renderTickRow(tickRow{
		edge:       edgeBottom,
		items:      xItems,
		axisOrigin: y1,
		axisCfg:    cfg.XAxis,
		tickColor:  th.XAxisTickColor,
		labelColor: th.XAxisLabelColor,
	})...)

	elems = append(elems, axisLinesOriented(x0, y0, x1, y1, cfg.XAxis, cfg.YAxis, th.XAxisLineColor, th.YAxisLineColor)...)
	elems = append(elems, axisTitlesOriented(d.XAxis, d.YAxis, x0, y0, x1, y1, cfg.XAxis, cfg.YAxis, th.XAxisTitleColor, th.YAxisTitleColor)...)
	elems = append(elems, renderSeriesVertical(d, categories, yMin, yMax, x0, y0, x1, y1, cfg, th)...)
	return elems
}

// axisLinesOriented draws the bottom + left axis lines using the
// AxisConfig that describes each visible edge. Vertical mode passes
// (XAxis, YAxis); horizontal mode passes (YAxis, XAxis) since the
// AST roles of each edge are swapped.
func axisLinesOriented(x0, y0, x1, y1 float64, bottomCfg, leftCfg AxisConfig, bottomColor, leftColor string) []any {
	var elems []any
	if flag(bottomCfg.ShowAxisLine, true) {
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(y1),
			X2: svgFloat(x1), Y2: svgFloat(y1),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%g", bottomColor, bottomCfg.AxisLineWidth),
		})
	}
	if flag(leftCfg.ShowAxisLine, true) {
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(y0),
			X2: svgFloat(x0), Y2: svgFloat(y1),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%g", leftColor, leftCfg.AxisLineWidth),
		})
	}
	return elems
}

// axisTitlesOriented places the bottom axis title centred below the
// plot and the left axis title rotated -90 to the left. The
// presentation `transform` attribute (not CSS) is used so the
// rotation renders in non-browser SVG consumers like tdewolff/canvas.
func axisTitlesOriented(bottomDef, leftDef diagram.XYAxis, x0, y0, x1, y1 float64, bottomCfg, leftCfg AxisConfig, bottomColor, leftColor string) []any {
	var elems []any
	if bottomDef.Title != "" && flag(bottomCfg.ShowTitle, true) {
		elems = append(elems, &text{
			X:        svgFloat((x0 + x1) / 2),
			Y:        svgFloat(y1 + axisLabelGap(bottomCfg) + bottomCfg.TitlePadding + bottomCfg.TitleFontSize/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", bottomColor, bottomCfg.TitleFontSize),
			Content:  bottomDef.Title,
		})
	}
	if leftDef.Title != "" && flag(leftCfg.ShowTitle, true) {
		midY := (y0 + y1) / 2
		tx := x0 - axisLabelGap(leftCfg) - leftCfg.TitlePadding - leftCfg.TitleFontSize/2
		elems = append(elems, &text{
			X:         svgFloat(tx),
			Y:         svgFloat(midY),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     fmt.Sprintf("fill:%s;font-size:%.0fpx", leftColor, leftCfg.TitleFontSize),
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", tx, midY),
			Content:   leftDef.Title,
		})
	}
	return elems
}

func renderSeriesVertical(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1 float64, cfg Config, th Theme) []any {
	var elems []any
	nCols := len(categories)
	if nCols == 0 {
		return elems
	}
	slotW := (x1 - x0) / float64(nCols)
	bandW := slotW * (1 - 2*barInsetRatio)
	barIndexes, nBars := barSeriesIndexes(d.Series)
	bw := bandW
	if nBars > 1 {
		bw = bandW / float64(nBars)
	}
	showLabel := flag(cfg.ShowDataLabel, false)
	outside := flag(cfg.ShowDataLabelOutsideBar, false)
	// Vertical: values plot against the Y-axis; data labels carry the
	// value, so derive their font from the Y-axis label size.
	labelFontSize := cfg.YAxis.LabelFontSize - 2

	// Bars grow from the v=0 pixel on the value axis, clamped into
	// the plot range. When the range straddles zero, positive values
	// grow toward yMax and negative toward yMin; otherwise the
	// baseline lands at the plot edge and the bar fills from there.
	baselineY := axisPos(0, yMin, yMax, y1, y0)

	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				bx := cx - bandW/2 + float64(barSlot)*bw
				by := axisPos(s.Data[i], yMin, yMax, y1, y0)
				topY := math.Min(by, baselineY)
				height := math.Abs(by - baselineY)
				elems = append(elems, &rect{
					X: svgFloat(bx), Y: svgFloat(topY),
					Width:  svgFloat(bw),
					Height: svgFloat(height),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
				if showLabel {
					var ly float64
					var dom string
					switch {
					case outside && s.Data[i] >= 0:
						ly, dom = by-2, "auto"
					case outside:
						ly, dom = by+2+labelFontSize, "auto"
					default:
						ly, dom = topY+height/2, "central"
					}
					elems = append(elems, &text{
						X:        svgFloat(bx + bw/2),
						Y:        svgFloat(ly),
						Anchor:   "middle",
						Dominant: dom,
						Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content:  formatTick(s.Data[i]),
					})
				}
			}
		case diagram.XYSeriesLine:
			if len(s.Data) == 0 {
				continue
			}
			var sb strings.Builder
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				py := axisPos(s.Data[i], yMin, yMax, y1, y0)
				if i > 0 {
					sb.WriteByte(' ')
				}
				fmt.Fprintf(&sb, "%.2f,%.2f", cx, py)
			}
			elems = append(elems, &polyline{
				Points: sb.String(),
				Style:  fmt.Sprintf("fill:none;stroke:%s;stroke-width:2", color),
			})
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				py := axisPos(s.Data[i], yMin, yMax, y1, y0)
				elems = append(elems, &circle{
					CX: svgFloat(cx), CY: svgFloat(py), R: 3,
					Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", color, th.MarkerStroke),
				})
				if showLabel {
					elems = append(elems, &text{
						X:        svgFloat(cx),
						Y:        svgFloat(py - 6),
						Anchor:   "middle",
						Dominant: "auto",
						Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content:  formatTick(s.Data[i]),
					})
				}
			}
		}
	}
	return elems
}

func renderHorizontal(d *diagram.XYChartDiagram, categories []string, vMin, vMax, x0, y0, x1, y1 float64, cfg Config, th Theme) []any {
	var elems []any

	// In horizontal mode the AST y-axis (values, range) is the bottom
	// axis and the AST x-axis (categories) is the left axis. Cfg/theme
	// fields named after each AST axis still apply to that axis — only
	// its visual position has changed.
	elems = append(elems, renderTickRow(tickRow{
		edge:       edgeBottom,
		items:      continuousTicks(vMin, vMax, x0, x1),
		axisOrigin: y1,
		grid:       gridSpec{lo: y0, hi: y1, stroke: th.GridStroke},
		axisCfg:    cfg.YAxis,
		tickColor:  th.YAxisTickColor,
		labelColor: th.YAxisLabelColor,
	})...)
	elems = append(elems, renderTickRow(tickRow{
		edge:       edgeLeft,
		items:      categoricalTicks(categories, y0, y1),
		axisOrigin: x0,
		axisCfg:    cfg.XAxis,
		tickColor:  th.XAxisTickColor,
		labelColor: th.XAxisLabelColor,
	})...)

	elems = append(elems, axisLinesOriented(x0, y0, x1, y1, cfg.YAxis, cfg.XAxis, th.YAxisLineColor, th.XAxisLineColor)...)
	elems = append(elems, axisTitlesOriented(d.YAxis, d.XAxis, x0, y0, x1, y1, cfg.YAxis, cfg.XAxis, th.YAxisTitleColor, th.XAxisTitleColor)...)
	elems = append(elems, renderSeriesHorizontal(d, categories, vMin, vMax, x0, y0, x1, y1, cfg, th)...)
	return elems
}

func renderSeriesHorizontal(d *diagram.XYChartDiagram, categories []string, vMin, vMax, x0, y0, x1, y1 float64, cfg Config, th Theme) []any {
	var elems []any
	nCats := len(categories)
	if nCats == 0 {
		return elems
	}
	slotH := (y1 - y0) / float64(nCats)
	bandH := slotH * (1 - 2*barInsetRatio)
	barIndexes, nBars := barSeriesIndexes(d.Series)
	bh := bandH
	if nBars > 1 {
		bh = bandH / float64(nBars)
	}
	showLabel := flag(cfg.ShowDataLabel, false)
	outside := flag(cfg.ShowDataLabelOutsideBar, false)
	// Horizontal: values plot against the AST y-axis (bottom edge);
	// data labels match that font.
	labelFontSize := cfg.YAxis.LabelFontSize - 2

	// Bars grow from the v=0 pixel on the value axis, clamped into
	// the plot range. Positive values grow toward vMax (rightward),
	// negative toward vMin (leftward); otherwise the baseline lands
	// at the plot edge and the bar fills from there.
	baselineX := axisPos(0, vMin, vMax, x0, x1)

	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				cy := y0 + slotH*(float64(i)+0.5)
				by := cy - bandH/2 + float64(barSlot)*bh
				bx := axisPos(s.Data[i], vMin, vMax, x0, x1)
				leftX := math.Min(bx, baselineX)
				width := math.Abs(bx - baselineX)
				elems = append(elems, &rect{
					X: svgFloat(leftX), Y: svgFloat(by),
					Width:  svgFloat(width),
					Height: svgFloat(bh),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
				if showLabel {
					var lx float64
					var anchor string
					switch {
					case outside && s.Data[i] >= 0:
						lx, anchor = bx+3, "start"
					case outside:
						lx, anchor = bx-3, "end"
					default:
						lx, anchor = leftX+width/2, "middle"
					}
					elems = append(elems, &text{
						X:        svgFloat(lx),
						Y:        svgFloat(by + bh/2),
						Anchor:   anchor,
						Dominant: "central",
						Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content:  formatTick(s.Data[i]),
					})
				}
			}
		case diagram.XYSeriesLine:
			if len(s.Data) == 0 {
				continue
			}
			var sb strings.Builder
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				cy := y0 + slotH*(float64(i)+0.5)
				px := axisPos(s.Data[i], vMin, vMax, x0, x1)
				if i > 0 {
					sb.WriteByte(' ')
				}
				fmt.Fprintf(&sb, "%.2f,%.2f", px, cy)
			}
			elems = append(elems, &polyline{
				Points: sb.String(),
				Style:  fmt.Sprintf("fill:none;stroke:%s;stroke-width:2", color),
			})
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				cy := y0 + slotH*(float64(i)+0.5)
				px := axisPos(s.Data[i], vMin, vMax, x0, x1)
				elems = append(elems, &circle{
					CX: svgFloat(px), CY: svgFloat(cy), R: 3,
					Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", color, th.MarkerStroke),
				})
				if showLabel {
					elems = append(elems, &text{
						X:        svgFloat(px + 5),
						Y:        svgFloat(cy),
						Anchor:   "start",
						Dominant: "central",
						Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content:  formatTick(s.Data[i]),
					})
				}
			}
		}
	}
	return elems
}

func barSeriesIndexes(series []diagram.XYSeries) ([]int, int) {
	idx := make([]int, len(series))
	n := 0
	for i, s := range series {
		if s.Type == diagram.XYSeriesBar {
			idx[i] = n
			n++
		} else {
			idx[i] = -1
		}
	}
	return idx, n
}

func categoryCenter(i, n int, lo, hi float64) float64 {
	slot := (hi - lo) / float64(n)
	return lo + slot*(float64(i)+0.5)
}

// axisPos maps a data value to a pixel along an axis, where lo/hi are
// the pixel coordinates of (vMin, vMax). For the y-axis vMin maps to
// y1 (bottom) and vMax to y0 (top), so callers pass (y1, y0). For the
// x-axis vMin → x0 and vMax → x1.
func axisPos(v, vMin, vMax, lo, hi float64) float64 {
	if v < vMin {
		v = vMin
	}
	if v > vMax {
		v = vMax
	}
	t := (v - vMin) / (vMax - vMin)
	return lo + t*(hi-lo)
}

// niceTicks returns axis tick values at round intervals via the 1/2/5
// × 10^k step selector — the standard "nice ticks" heuristic. Used to
// avoid awkward decimal stops like 2.4, 4.8, 7.2 that naive uniform
// spacing produces.
func niceTicks(lo, hi float64, target int) []float64 {
	if hi <= lo || target < 2 {
		return []float64{lo, hi}
	}
	rawStep := (hi - lo) / float64(target-1)
	if rawStep <= 0 || math.IsInf(rawStep, 0) || math.IsNaN(rawStep) {
		return []float64{lo, hi}
	}
	mag := math.Pow(10, math.Floor(math.Log10(rawStep)))
	norm := rawStep / mag
	var step float64
	switch {
	case norm < 1.5:
		step = 1 * mag
	case norm < 3:
		step = 2 * mag
	case norm < 7:
		step = 5 * mag
	default:
		step = 10 * mag
	}
	start := math.Ceil(lo/step) * step
	ticks := []float64{}
	for v := start; v <= hi+step*1e-9; v += step {
		ticks = append(ticks, v)
	}
	return ticks
}

func formatTick(v float64) string {
	return svgutil.FormatNumber(v, 2)
}
