// Package xychart renders an XYChartDiagram to SVG. The plot area is
// a rectangle bounded by X and Y axes; bars and lines are positioned
// by column index (categorical x) and data value (y, scaled to the
// y-axis range). Multiple bar series in the same chart share each
// category's horizontal slot and split it into equal-width bands.
package xychart

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

type Options struct {
	FontSize float64
}

const (
	defaultFontSize = 13.0
	marginX         = 50.0
	marginY         = 40.0
	titleGap        = 24.0
	axisLabelGap    = 32.0
	tickSize        = 5.0
	plotW           = 520.0
	plotH           = 320.0
	labelFill       = "#333"
	axisStroke      = "#999"
	gridStroke      = "#e5e5e5"
	bgFill          = "#fff"
	barInsetRatio   = 0.15 // fraction of category slot left blank on each side
)

// seriesPalette cycles by series index so output is deterministic
// without depending on map iteration.
var seriesPalette = []string{
	"#5470c6", "#91cc75", "#fac858", "#ee6666",
	"#73c0de", "#3ba272", "#fc8452", "#9a60b4",
}

func Render(d *diagram.XYChartDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("xychart render: diagram is nil")
	}
	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	// Categorical x-axis is the common case; generate synthetic
	// category labels ("1".."n") when none are supplied so every
	// series still maps to X positions.
	categories := d.XAxis.Categories
	if len(categories) == 0 {
		n := maxSeriesLen(d.Series)
		categories = make([]string, n)
		for i := range categories {
			categories[i] = strconv.Itoa(i + 1)
		}
	}
	yMin, yMax := yRange(d, 0.0)
	if yMin == yMax {
		// Flat data: widen the range so the line/bar is visible.
		yMax = yMin + 1
	}

	// Plot rectangle.
	titleH := 0.0
	if d.Title != "" {
		titleH = titleGap
	}
	leftAxisPad := axisLabelGap
	if d.YAxis.Title != "" {
		leftAxisPad += titleGap
	}
	bottomAxisPad := axisLabelGap
	if d.XAxis.Title != "" {
		bottomAxisPad += titleGap
	}
	plotX0 := marginX + leftAxisPad
	plotY0 := marginY + titleH
	plotX1 := plotX0 + plotW
	plotY1 := plotY0 + plotH
	viewW := plotX1 + marginX
	viewH := plotY1 + bottomAxisPad + marginY

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0,
		Width:  svgFloat(viewW),
		Height: svgFloat(viewH),
		Style:  fmt.Sprintf("fill:%s;stroke:none", bgFill),
	})

	// Title centered over the plot.
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

	children = append(children, renderAxes(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, fontSize)...)
	children = append(children, renderSeries(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1)...)

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

// yRange returns [min, max] for the y-axis, honoring an explicit range
// if set, otherwise deriving from series data with a small headroom
// extension so top values aren't pressed against the axis.
func yRange(d *diagram.XYChartDiagram, _ float64) (float64, float64) {
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
	// Nudge upper bound for visual breathing room; clamp lower to 0
	// when all data is non-negative (the common case).
	if lo > 0 {
		lo = 0
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	hi += span * 0.1
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

// renderAxes draws the X and Y axis lines, tick marks, gridlines,
// tick labels, and optional axis titles.
func renderAxes(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1, fontSize float64) []any {
	var elems []any
	// Y gridlines + tick labels (5 ticks).
	const nYTicks = 5
	for i := 0; i <= nYTicks; i++ {
		t := float64(i) / float64(nYTicks)
		yPix := y1 - t*(y1-y0)
		val := yMin + t*(yMax-yMin)
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(yPix),
			X2: svgFloat(x1), Y2: svgFloat(yPix),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1", gridStroke),
		})
		elems = append(elems, &text{
			X:        svgFloat(x0 - tickSize - 2),
			Y:        svgFloat(yPix),
			Anchor:   "end",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", labelFill, fontSize-2),
			Content:  formatTick(val),
		})
	}

	// X tick labels at each category center.
	for i, c := range categories {
		x := categoryCenter(i, len(categories), x0, x1)
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(y1),
			X2: svgFloat(x), Y2: svgFloat(y1 + tickSize),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1", axisStroke),
		})
		elems = append(elems, &text{
			X:        svgFloat(x),
			Y:        svgFloat(y1 + tickSize + float64(fontSize)),
			Anchor:   "middle",
			Dominant: "hanging",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", labelFill, fontSize-2),
			Content:  c,
		})
	}

	// Axis lines drawn last so ticks/grid don't overlap them.
	elems = append(elems, &line{
		X1: svgFloat(x0), Y1: svgFloat(y1),
		X2: svgFloat(x1), Y2: svgFloat(y1),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", axisStroke),
	})
	elems = append(elems, &line{
		X1: svgFloat(x0), Y1: svgFloat(y0),
		X2: svgFloat(x0), Y2: svgFloat(y1),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", axisStroke),
	})

	if d.XAxis.Title != "" {
		elems = append(elems, &text{
			X:        svgFloat((x0 + x1) / 2),
			Y:        svgFloat(y1 + axisLabelGap + titleGap/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", labelFill, fontSize),
			Content:  d.XAxis.Title,
		})
	}
	if d.YAxis.Title != "" {
		// Rotated y-axis title, anchored at the plot midpoint.
		midY := (y0 + y1) / 2
		tx := x0 - axisLabelGap - titleGap/2
		elems = append(elems, &text{
			X:        svgFloat(tx),
			Y:        svgFloat(midY),
			Anchor:   "middle",
			Dominant: "central",
			Style: fmt.Sprintf("fill:%s;font-size:%.0fpx;transform:rotate(-90deg);transform-origin:%.2fpx %.2fpx",
				labelFill, fontSize, tx, midY),
			Content: d.YAxis.Title,
		})
	}
	return elems
}

// renderSeries draws each series. Multiple bar series share the
// category slot and split it into equal bands; line series overlay
// connecting their category centers.
func renderSeries(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1 float64) []any {
	var elems []any
	nCols := len(categories)

	barIndexes := barSeriesIndexes(d.Series)
	nBars := len(barIndexes)

	for seriesIdx, s := range d.Series {
		color := seriesPalette[seriesIdx%len(seriesPalette)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := categoryCenter(i, nCols, x0, x1)
				slotW := (x1 - x0) / float64(nCols)
				bandW := slotW * (1 - 2*barInsetRatio)
				bw := bandW
				if nBars > 1 {
					bw = bandW / float64(nBars)
				}
				bx := cx - bandW/2 + float64(barSlot)*bw
				by := yPix(s.Data[i], yMin, yMax, y0, y1)
				bh := y1 - by
				if bh < 0 {
					bh = 0
				}
				elems = append(elems, &rect{
					X: svgFloat(bx), Y: svgFloat(by),
					Width:  svgFloat(bw),
					Height: svgFloat(bh),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
			}
		case diagram.XYSeriesLine:
			if len(s.Data) == 0 {
				continue
			}
			points := make([]string, 0, len(s.Data))
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := categoryCenter(i, nCols, x0, x1)
				py := yPix(s.Data[i], yMin, yMax, y0, y1)
				points = append(points, fmt.Sprintf("%.2f,%.2f", cx, py))
			}
			elems = append(elems, &polyline{
				Points: strings.Join(points, " "),
				Style:  fmt.Sprintf("fill:none;stroke:%s;stroke-width:2", color),
			})
			// Small dots at each data point for readability.
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := categoryCenter(i, nCols, x0, x1)
				py := yPix(s.Data[i], yMin, yMax, y0, y1)
				elems = append(elems, &circle{
					CX: svgFloat(cx), CY: svgFloat(py), R: 3,
					Style: fmt.Sprintf("fill:%s;stroke:#fff;stroke-width:1", color),
				})
			}
		}
	}
	return elems
}

// barSeriesIndexes assigns each bar series a band index within a
// category slot. Line series get -1 (unused).
func barSeriesIndexes(series []diagram.XYSeries) []int {
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
	return idx
}

func categoryCenter(i, n int, x0, x1 float64) float64 {
	slotW := (x1 - x0) / float64(n)
	return x0 + slotW*(float64(i)+0.5)
}

// yPix maps a data value to a pixel Y coordinate within the plot
// rectangle. Values outside [yMin, yMax] are clamped so they don't
// draw outside the plot area.
func yPix(v, yMin, yMax, y0, y1 float64) float64 {
	if v < yMin {
		v = yMin
	}
	if v > yMax {
		v = yMax
	}
	t := (v - yMin) / (yMax - yMin)
	return y1 - t*(y1-y0)
}

// formatTick returns a compact numeric string: integer form for
// whole-number ticks, otherwise up to 2 decimals with trailing zeros
// trimmed.
func formatTick(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	if math.Abs(v-math.Round(v)) < 1e-9 {
		return strconv.FormatFloat(math.Round(v), 'f', 0, 64)
	}
	s := strconv.FormatFloat(v, 'f', 2, 64)
	// Trim trailing zeros then a trailing dot.
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
