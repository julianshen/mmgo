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
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type Options struct {
	FontSize float64
	Theme    Theme
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
	barInsetRatio  = 0.15 // fraction of category slot left blank on each side
	yRangeHeadroom = 0.10 // 10% visual padding above the max data value
	tickFontDelta  = 2.0  // tick/category labels render this many px below base
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

	categories := d.XAxis.Categories
	if len(categories) == 0 {
		n := maxSeriesLen(d.Series)
		categories = make([]string, n)
		for i := range categories {
			categories[i] = strconv.Itoa(i + 1)
		}
	}
	yMin, yMax := yRange(d)
	if yMin == yMax {
		yMax = yMin + 1 // flat data: widen so the line/bar is visible
	}

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

	// Conservative preallocation: background + optional title + axis
	// lines/ticks + per-series elements.
	nYTicks := 6
	size := 3 + 2*nYTicks + 2*len(categories) + 4
	for _, s := range d.Series {
		if s.Type == diagram.XYSeriesBar {
			size += len(s.Data)
		} else {
			size += 1 + len(s.Data)
		}
	}
	children := make([]any, 0, size)
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

	if d.Title != "" {
		children = append(children, &text{
			X:        svgFloat((plotX0 + plotX1) / 2),
			Y:        svgFloat(marginY + titleH/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.LabelFill, fontSize+2),
			Content:  d.Title,
		})
	}

	children = append(children, renderAxes(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, fontSize, th)...)
	children = append(children, renderSeries(d, categories, yMin, yMax, plotX0, plotY0, plotX1, plotY1, th)...)

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

// yRange returns the y-axis [min, max], honoring an explicit range
// when set or deriving from data. Lower bound clamps to 0 when all
// data is non-negative; upper bound gets a yRangeHeadroom nudge so
// top values aren't flush against the axis.
func yRange(d *diagram.XYChartDiagram) (float64, float64) {
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

// renderAxes draws gridlines, tick marks/labels, axis lines, and
// axis titles. Axis lines come last so the ticks/grid don't overdraw
// them.
func renderAxes(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1, fontSize float64, th Theme) []any {
	var elems []any
	for _, val := range niceYTicks(yMin, yMax, 6) {
		t := (val - yMin) / (yMax - yMin)
		yPix := y1 - t*(y1-y0)
		elems = append(elems, &line{
			X1: svgFloat(x0), Y1: svgFloat(yPix),
			X2: svgFloat(x1), Y2: svgFloat(yPix),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.GridStroke),
		})
		elems = append(elems, &text{
			X:        svgFloat(x0 - tickSize - 2),
			Y:        svgFloat(yPix),
			Anchor:   "end",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LabelFill, fontSize-tickFontDelta),
			Content:  formatTick(val),
		})
	}

	for i, c := range categories {
		x := categoryCenter(i, len(categories), x0, x1)
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(y1),
			X2: svgFloat(x), Y2: svgFloat(y1 + tickSize),
			Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.AxisStroke),
		})
		elems = append(elems, &text{
			X:        svgFloat(x),
			Y:        svgFloat(y1 + tickSize + float64(fontSize)),
			Anchor:   "middle",
			Dominant: "hanging",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LabelFill, fontSize-tickFontDelta),
			Content:  c,
		})
	}

	elems = append(elems, &line{
		X1: svgFloat(x0), Y1: svgFloat(y1),
		X2: svgFloat(x1), Y2: svgFloat(y1),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", th.AxisStroke),
	})
	elems = append(elems, &line{
		X1: svgFloat(x0), Y1: svgFloat(y0),
		X2: svgFloat(x0), Y2: svgFloat(y1),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1.5", th.AxisStroke),
	})

	if d.XAxis.Title != "" {
		elems = append(elems, &text{
			X:        svgFloat((x0 + x1) / 2),
			Y:        svgFloat(y1 + axisLabelGap + titleGap/2),
			Anchor:   "middle",
			Dominant: "central",
			Style:    fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LabelFill, fontSize),
			Content:  d.XAxis.Title,
		})
	}
	if d.YAxis.Title != "" {
		// Use the SVG `transform` presentation attribute (not CSS
		// transform) so the rotation renders correctly in non-browser
		// SVG consumers like tdewolff/canvas (used for PNG/PDF).
		midY := (y0 + y1) / 2
		tx := x0 - axisLabelGap - titleGap/2
		elems = append(elems, &text{
			X:         svgFloat(tx),
			Y:         svgFloat(midY),
			Anchor:    "middle",
			Dominant:  "central",
			Style:     fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LabelFill, fontSize),
			Transform: fmt.Sprintf("rotate(-90 %.2f %.2f)", tx, midY),
			Content:   d.YAxis.Title,
		})
	}
	return elems
}

// renderSeries draws each series. Multiple bar series share the
// category slot and split it into equal bands; line series overlay.
func renderSeries(d *diagram.XYChartDiagram, categories []string, yMin, yMax, x0, y0, x1, y1 float64, th Theme) []any {
	var elems []any
	nCols := len(categories)
	slotW := (x1 - x0) / float64(nCols)
	bandW := slotW * (1 - 2*barInsetRatio)

	barIndexes, nBars := barSeriesIndexes(d.Series)
	bw := bandW
	if nBars > 1 {
		bw = bandW / float64(nBars)
	}

	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				bx := cx - bandW/2 + float64(barSlot)*bw
				by := yPix(s.Data[i], yMin, yMax, y0, y1)
				elems = append(elems, &rect{
					X: svgFloat(bx), Y: svgFloat(by),
					Width:  svgFloat(bw),
					Height: svgFloat(y1 - by),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
			}
		case diagram.XYSeriesLine:
			if len(s.Data) == 0 {
				continue
			}
			var sb strings.Builder
			for i := 0; i < len(s.Data) && i < nCols; i++ {
				cx := x0 + slotW*(float64(i)+0.5)
				py := yPix(s.Data[i], yMin, yMax, y0, y1)
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
				py := yPix(s.Data[i], yMin, yMax, y0, y1)
				elems = append(elems, &circle{
					CX: svgFloat(cx), CY: svgFloat(py), R: 3,
					Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", color, th.MarkerStroke),
				})
			}
		}
	}
	return elems
}

// barSeriesIndexes returns a slice (index-per-series) giving each bar
// series its band position within a category slot, plus the total bar
// count. Line series map to -1.
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

// niceYTicks returns axis tick values at round intervals covering
// [yMin, yMax]. Aims for roughly `target` ticks using a 1/2/5 × 10^k
// step selector — the conventional "nice ticks" heuristic. For e.g.
// [0, 12] and target=6 it returns {0,2,4,6,8,10,12}, not {0, 2.4, 4.8,
// 7.2, 9.6, 12} which is what naive uniform spacing produces.
func niceYTicks(yMin, yMax float64, target int) []float64 {
	if yMax <= yMin || target < 2 {
		return []float64{yMin, yMax}
	}
	rawStep := (yMax - yMin) / float64(target-1)
	if rawStep <= 0 || math.IsInf(rawStep, 0) || math.IsNaN(rawStep) {
		return []float64{yMin, yMax}
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
	start := math.Ceil(yMin/step) * step
	ticks := []float64{}
	for v := start; v <= yMax+step*1e-9; v += step {
		ticks = append(ticks, v)
	}
	return ticks
}

// formatTick returns a compact numeric string: integer form for
// whole-number ticks, otherwise up to 2 decimals with trailing zeros
// trimmed.
func formatTick(v float64) string {
	return svgutil.FormatNumber(v, 2)
}
