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
	elems = append(elems, renderSeries(d, len(categories), yMin, yMax, x0, y0, x1, y1, cfg, th, false)...)
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

// seriesLayout captures the orientation-dependent geometry shared by
// the bar/line emit loops. Vertical mode places categories along x;
// horizontal mode places them along y. The val-axis pixel range is
// inverted for vertical (valLo=y1, valHi=y0) so that valMin maps to
// the plot's bottom edge in both modes.
type seriesLayout struct {
	horizontal     bool
	nCats          int
	catLo, catHi   float64
	valLo, valHi   float64
	valMin, valMax float64
	catSlot        float64
	bandSize       float64
	barBandSize    float64
	baseline       float64
}

func newSeriesLayout(d *diagram.XYChartDiagram, nCats int, valMin, valMax, x0, y0, x1, y1 float64, horizontal bool) seriesLayout {
	catLo, catHi := x0, x1
	valLo, valHi := y1, y0
	if horizontal {
		catLo, catHi = y0, y1
		valLo, valHi = x0, x1
	}
	slot := (catHi - catLo) / float64(nCats)
	bandSize := slot * (1 - 2*barInsetRatio)
	_, nBars := barSeriesIndexes(d.Series)
	barBandSize := bandSize
	if nBars > 1 {
		barBandSize = bandSize / float64(nBars)
	}
	return seriesLayout{
		horizontal: horizontal, nCats: nCats,
		catLo: catLo, catHi: catHi, valLo: valLo, valHi: valHi,
		valMin: valMin, valMax: valMax,
		catSlot: slot, bandSize: bandSize, barBandSize: barBandSize,
		baseline: axisPos(0, valMin, valMax, valLo, valHi),
	}
}

func (m seriesLayout) catCenter(i int) float64 {
	return m.catLo + m.catSlot*(float64(i)+0.5)
}

func (m seriesLayout) valPx(v float64) float64 {
	return axisPos(v, m.valMin, m.valMax, m.valLo, m.valHi)
}

// barRect places bar i of band-position bp at value v, returning the
// SVG rect bounds. Bars grow from the value-axis baseline (v=0
// pixel, clamped) toward the data point.
func (m seriesLayout) barRect(i, bp int, v float64) (x, y, w, h float64) {
	bandStart := m.catCenter(i) - m.bandSize/2 + float64(bp)*m.barBandSize
	valP := m.valPx(v)
	valStart := math.Min(valP, m.baseline)
	valLen := math.Abs(valP - m.baseline)
	if m.horizontal {
		return valStart, bandStart, valLen, m.barBandSize
	}
	return bandStart, valStart, m.barBandSize, valLen
}

func (m seriesLayout) linePoint(i int, v float64) (px, py float64) {
	c := m.catCenter(i)
	v2 := m.valPx(v)
	if m.horizontal {
		return v2, c
	}
	return c, v2
}

// labelPlacement is the (x, y, text-anchor, dominant-baseline) tuple
// for a data label.
type labelPlacement struct {
	x, y     float64
	anchor   string
	dominant string
}

// barLabel returns where to place a bar's data label. Inside placement
// is always the bar's geometric centre. Outside placement anchors to
// the bar's value-axis edge and follows the value's sign so labels
// for negative bars sit on the correct side.
func (m seriesLayout) barLabel(rx, ry, rw, rh, v float64, outside bool) labelPlacement {
	if !outside {
		return labelPlacement{x: rx + rw/2, y: ry + rh/2, anchor: "middle", dominant: "central"}
	}
	valP := m.valPx(v)
	if m.horizontal {
		anchor, x := "start", valP+3
		if v < 0 {
			anchor, x = "end", valP-3
		}
		return labelPlacement{x: x, y: ry + rh/2, anchor: anchor, dominant: "central"}
	}
	dom, y := "auto", valP-2
	if v < 0 {
		dom, y = "hanging", valP+2
	}
	return labelPlacement{x: rx + rw/2, y: y, anchor: "middle", dominant: dom}
}

// linePointLabel sits the label clear of the marker on the side
// opposite the value axis: above in vertical, right in horizontal.
func (m seriesLayout) linePointLabel(px, py float64) labelPlacement {
	if m.horizontal {
		return labelPlacement{x: px + 5, y: py, anchor: "start", dominant: "central"}
	}
	return labelPlacement{x: px, y: py - 6, anchor: "middle", dominant: "auto"}
}

func renderSeries(d *diagram.XYChartDiagram, nCats int, valMin, valMax, x0, y0, x1, y1 float64, cfg Config, th Theme, horizontal bool) []any {
	if nCats == 0 {
		return nil
	}
	m := newSeriesLayout(d, nCats, valMin, valMax, x0, y0, x1, y1, horizontal)
	barIndexes, _ := barSeriesIndexes(d.Series)
	showLabel := flag(cfg.ShowDataLabel, false)
	outside := flag(cfg.ShowDataLabelOutsideBar, false)
	// Data labels describe value-axis quantities, so derive their
	// font from the Y-axis label size in both orientations (the AST
	// y-axis is the value axis even when drawn at the bottom in
	// horizontal mode).
	labelFontSize := cfg.YAxis.LabelFontSize - 2

	var elems []any
	for seriesIdx, s := range d.Series {
		color := th.SeriesColors[seriesIdx%len(th.SeriesColors)]
		switch s.Type {
		case diagram.XYSeriesBar:
			barSlot := barIndexes[seriesIdx]
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				rx, ry, rw, rh := m.barRect(i, barSlot, s.Data[i])
				elems = append(elems, &rect{
					X: svgFloat(rx), Y: svgFloat(ry),
					Width:  svgFloat(rw),
					Height: svgFloat(rh),
					Style:  fmt.Sprintf("fill:%s;stroke:none", color),
				})
				if showLabel {
					p := m.barLabel(rx, ry, rw, rh, s.Data[i], outside)
					elems = append(elems, &text{
						X: svgFloat(p.x), Y: svgFloat(p.y),
						Anchor: p.anchor, Dominant: p.dominant,
						Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content: formatTick(s.Data[i]),
					})
				}
			}
		case diagram.XYSeriesLine:
			if len(s.Data) == 0 {
				continue
			}
			var sb strings.Builder
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				px, py := m.linePoint(i, s.Data[i])
				if i > 0 {
					sb.WriteByte(' ')
				}
				fmt.Fprintf(&sb, "%.2f,%.2f", px, py)
			}
			elems = append(elems, &polyline{
				Points: sb.String(),
				Style:  fmt.Sprintf("fill:none;stroke:%s;stroke-width:2", color),
			})
			for i := 0; i < len(s.Data) && i < nCats; i++ {
				px, py := m.linePoint(i, s.Data[i])
				elems = append(elems, &circle{
					CX: svgFloat(px), CY: svgFloat(py), R: 3,
					Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", color, th.MarkerStroke),
				})
				if showLabel {
					p := m.linePointLabel(px, py)
					elems = append(elems, &text{
						X: svgFloat(p.x), Y: svgFloat(p.y),
						Anchor: p.anchor, Dominant: p.dominant,
						Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.DataLabelColor, labelFontSize),
						Content: formatTick(s.Data[i]),
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
	elems = append(elems, renderSeries(d, len(categories), vMin, vMax, x0, y0, x1, y1, cfg, th, true)...)
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
