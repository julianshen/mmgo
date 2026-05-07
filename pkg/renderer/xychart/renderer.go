// Package xychart renders an XYChartDiagram to SVG. The plot area is
// a rectangle bounded by X and Y axes; bars and lines are positioned
// by column index (categorical x) and data value (y, scaled to the
// y-axis range). Multiple bar series in the same chart share each
// category's horizontal slot and split it into equal-width bands.
//
// Vertical layout: categories along x-axis, data scaled to y-axis.
// Horizontal layout (`xychart-beta horizontal`): the value-axis is
// the x-axis and categories run down the y-axis; bars grow rightward.
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
	if len(categories) == 0 && !d.XAxis.HasRange {
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
	leftAxisPad := axisLabelGap(cfg.YAxis)
	if d.YAxis.Title != "" && flag(cfg.YAxis.ShowTitle, true) {
		leftAxisPad += cfg.YAxis.TitleFontSize + cfg.YAxis.TitlePadding
	}
	bottomAxisPad := axisLabelGap(cfg.XAxis)
	if d.XAxis.Title != "" && flag(cfg.XAxis.ShowTitle, true) {
		bottomAxisPad += cfg.XAxis.TitleFontSize + cfg.XAxis.TitlePadding
	}

	// Plot area sized off the spec width/height with the chrome
	// (title + axis pads + outer margins) carved out.
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

	if horizontal {
		children = append(children, renderHorizontal(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, fontSize, cfg, th)...)
	} else {
		children = append(children, renderVertical(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, fontSize, cfg, th)...)
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

// axisLabelGap is the room reserved for one axis's label row + tick
// length, observing the per-axis show* flags.
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

// valueRange returns the value-axis [min, max] honoring an explicit
// y-range when set, otherwise deriving from the data with a 10% top
// headroom and a clamp-to-zero floor for non-negative data.
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

// renderVertical draws the original layout: categories along x-axis,
// data scaled to y-axis. When d.XAxis.HasRange and Categories is
// empty, the x-axis renders continuous numeric ticks instead of
// per-category labels.
func renderVertical(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1, fontSize float64, cfg Config, th Theme) []any {
	var elems []any

	// Y gridlines + tick labels.
	yTicks := niceTicks(yMin, yMax, yTickTarget)
	if flag(cfg.YAxis.ShowTick, true) {
		for _, val := range yTicks {
			t := (val - yMin) / (yMax - yMin)
			yPx := y1 - t*(y1-y0)
			elems = append(elems, &line{
				X1: svgFloat(x0), Y1: svgFloat(yPx),
				X2: svgFloat(x1), Y2: svgFloat(yPx),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.GridStroke),
			})
			if flag(cfg.YAxis.ShowLabel, true) {
				elems = append(elems, &text{
					X:        svgFloat(x0 - cfg.YAxis.TickLength - cfg.YAxis.LabelPadding),
					Y:        svgFloat(yPx),
					Anchor:   "end",
					Dominant: "central",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.YAxisLabelColor, cfg.YAxis.LabelFontSize),
					Content:  formatTick(val),
				})
			}
		}
	}

	// X axis: continuous when HasRange w/o categories, else categorical.
	if d.XAxis.HasRange && len(d.XAxis.Categories) == 0 {
		elems = append(elems, renderContinuousXTicks(d.XAxis.Min, d.XAxis.Max, x0, y1, x1, cfg, th)...)
	} else {
		elems = append(elems, renderCategoricalXTicks(categories, x0, y1, x1, cfg, th)...)
	}

	elems = append(elems, axisLines(x0, y0, x1, y1, cfg, th)...)
	elems = append(elems, axisTitlesVertical(d, x0, y0, x1, y1, fontSize, cfg, th)...)
	elems = append(elems, renderSeriesVertical(d, categories, yMin, yMax, x0, y0, x1, y1, cfg, th)...)
	return elems
}

func renderContinuousXTicks(xMin, xMax, x0, y1, x1 float64, cfg Config, th Theme) []any {
	var elems []any
	if !flag(cfg.XAxis.ShowTick, true) && !flag(cfg.XAxis.ShowLabel, true) {
		return elems
	}
	for _, val := range niceTicks(xMin, xMax, yTickTarget) {
		t := (val - xMin) / (xMax - xMin)
		xPx := x0 + t*(x1-x0)
		if flag(cfg.XAxis.ShowTick, true) {
			elems = append(elems, &line{
				X1: svgFloat(xPx), Y1: svgFloat(y1),
				X2: svgFloat(xPx), Y2: svgFloat(y1 + cfg.XAxis.TickLength),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.XAxisTickColor),
			})
		}
		if flag(cfg.XAxis.ShowLabel, true) {
			elems = append(elems, &text{
				X:        svgFloat(xPx),
				Y:        svgFloat(y1 + cfg.XAxis.TickLength + cfg.XAxis.LabelPadding),
				Anchor:   "middle",
				Dominant: "hanging",
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.XAxisLabelColor, cfg.XAxis.LabelFontSize),
				Content:  formatTick(val),
			})
		}
	}
	return elems
}

func renderCategoricalXTicks(categories []string, x0, y1, x1 float64, cfg Config, th Theme) []any {
	var elems []any
	for i, c := range categories {
		x := categoryCenter(i, len(categories), x0, x1)
		if flag(cfg.XAxis.ShowTick, true) {
			elems = append(elems, &line{
				X1: svgFloat(x), Y1: svgFloat(y1),
				X2: svgFloat(x), Y2: svgFloat(y1 + cfg.XAxis.TickLength),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.XAxisTickColor),
			})
		}
		if flag(cfg.XAxis.ShowLabel, true) {
			elems = append(elems, &text{
				X:        svgFloat(x),
				Y:        svgFloat(y1 + cfg.XAxis.TickLength + cfg.XAxis.LabelPadding),
				Anchor:   "middle",
				Dominant: "hanging",
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.XAxisLabelColor, cfg.XAxis.LabelFontSize),
				Content:  c,
			})
		}
	}
	return elems
}

func axisLines(x0, y0, x1, y1 float64, cfg Config, th Theme) []any {
	var elems []any
	if flag(cfg.XAxis.ShowAxisLine, true) {
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(y1),
			X2: svgFloat(x1), Y2: svgFloat(y1),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%g", th.XAxisLineColor, cfg.XAxis.AxisLineWidth),
		})
	}
	if flag(cfg.YAxis.ShowAxisLine, true) {
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(y0),
			X2: svgFloat(x0), Y2: svgFloat(y1),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%g", th.YAxisLineColor, cfg.YAxis.AxisLineWidth),
		})
	}
	return elems
}

func axisTitlesVertical(d *diagram.XYChartDiagram, x0, y0, x1, y1, _ float64, cfg Config, th Theme) []any {
	var elems []any
	if d.XAxis.Title != "" && flag(cfg.XAxis.ShowTitle, true) {
		elems = append(elems, &text{
			X:        svgFloat((x0 + x1) / 2),
			Y:        svgFloat(y1 + axisLabelGap(cfg.XAxis) + cfg.XAxis.TitlePadding + cfg.XAxis.TitleFontSize/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.XAxisTitleColor, cfg.XAxis.TitleFontSize),
			Content:  d.XAxis.Title,
		})
	}
	if d.YAxis.Title != "" && flag(cfg.YAxis.ShowTitle, true) {
		// SVG `transform` (presentation attribute, not CSS) so the
		// rotation renders in non-browser SVG consumers like
		// tdewolff/canvas (used for our PNG/PDF output).
		midY := (y0 + y1) / 2
		tx := x0 - axisLabelGap(cfg.YAxis) - cfg.YAxis.TitlePadding - cfg.YAxis.TitleFontSize/2
		elems = append(elems, &text{
			X:         svgFloat(tx),
			Y:         svgFloat(midY),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     fmt.Sprintf("fill:%s;font-size:%.0fpx", th.YAxisTitleColor, cfg.YAxis.TitleFontSize),
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", tx, midY),
			Content:   d.YAxis.Title,
		})
	}
	return elems
}

// renderSeriesVertical draws bar/line series for the vertical layout,
// honoring cfg.ShowDataLabel / ShowDataLabelOutsideBar.
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
	labelFontSize := cfg.XAxis.LabelFontSize - 2

	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				bx := cx - bandW/2 + float64(barSlot)*bw
				by := axisPos(s.Data[i], yMin, yMax, y1, y0)
				elems = append(elems, &rect{
					X: svgFloat(bx), Y: svgFloat(by),
					Width:  svgFloat(bw),
					Height: svgFloat(y1 - by),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
				if showLabel {
					ly := by - 2
					dom := "auto"
					if !outside {
						ly = by + (y1-by)/2
						dom = "central"
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

// renderHorizontal draws the chart with category-axis on the left
// (vertical) and value-axis along the bottom (horizontal). Bars grow
// rightward from x0, lines run top-to-bottom.
func renderHorizontal(d *diagram.XYChartDiagram, categories []string, vMin, vMax, x0, y0, x1, y1, fontSize float64, cfg Config, th Theme) []any {
	var elems []any

	// X axis: continuous value scale.
	xTicks := niceTicks(vMin, vMax, yTickTarget)
	if flag(cfg.XAxis.ShowTick, true) {
		for _, val := range xTicks {
			t := (val - vMin) / (vMax - vMin)
			xPx := x0 + t*(x1-x0)
			elems = append(elems, &line{
				X1: svgFloat(xPx), Y1: svgFloat(y0),
				X2: svgFloat(xPx), Y2: svgFloat(y1),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.GridStroke),
			})
			if flag(cfg.XAxis.ShowLabel, true) {
				elems = append(elems, &text{
					X:        svgFloat(xPx),
					Y:        svgFloat(y1 + cfg.XAxis.TickLength + cfg.XAxis.LabelPadding),
					Anchor:   "middle",
					Dominant: "hanging",
					Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.XAxisLabelColor, cfg.XAxis.LabelFontSize),
					Content:  formatTick(val),
				})
			}
		}
	}

	// Y axis: categorical, top-to-bottom.
	nCats := len(categories)
	for i, c := range categories {
		yc := categoryCenter(i, nCats, y0, y1)
		if flag(cfg.YAxis.ShowTick, true) {
			elems = append(elems, &line{
				X1: svgFloat(x0 - cfg.YAxis.TickLength), Y1: svgFloat(yc),
				X2: svgFloat(x0), Y2: svgFloat(yc),
				Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.YAxisTickColor),
			})
		}
		if flag(cfg.YAxis.ShowLabel, true) {
			elems = append(elems, &text{
				X:        svgFloat(x0 - cfg.YAxis.TickLength - cfg.YAxis.LabelPadding),
				Y:        svgFloat(yc),
				Anchor:   "end",
				Dominant: "central",
				Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.YAxisLabelColor, cfg.YAxis.LabelFontSize),
				Content:  c,
			})
		}
	}

	elems = append(elems, axisLines(x0, y0, x1, y1, cfg, th)...)
	// Axis titles flipped: x-title at the bottom (value axis),
	// y-title rotated on the left (category axis) — same placement
	// as vertical, but the underlying axes have swapped meaning.
	elems = append(elems, axisTitlesVertical(d, x0, y0, x1, y1, fontSize, cfg, th)...)
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
	labelFontSize := cfg.XAxis.LabelFontSize - 2

	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				cy := y0 + slotH*(float64(i)+0.5)
				by := cy - bandH/2 + float64(barSlot)*bh
				bx := axisPos(s.Data[i], vMin, vMax, x0, x1)
				elems = append(elems, &rect{
					X: svgFloat(x0), Y: svgFloat(by),
					Width:  svgFloat(bx - x0),
					Height: svgFloat(bh),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
				if showLabel {
					lx := bx + 3
					anchor := "start"
					if !outside {
						lx = (x0 + bx) / 2
						anchor = "middle"
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

// niceTicks returns axis tick values at round intervals covering
// [lo, hi]. 1/2/5 × 10^k step selector — the standard "nice ticks"
// heuristic. For [0, 12] / target=6 it produces {0,2,4,6,8,10,12}.
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
