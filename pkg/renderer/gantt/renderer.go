package gantt

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultFontSize = 13.0
	defaultPadding  = 20.0
	titleH          = 30.0
	axisH           = 25.0
	barH            = 22.0
	barGap          = 6.0
	sectionLabelW   = 120.0
	dayWidth        = 20.0

	labelInsideSlack = 8.0
	labelOutsideGap  = 4.0
	labelEdgeMargin  = 12.0
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.GanttDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("gantt render: diagram is nil")
	}
	th := resolveTheme(opts)
	if len(d.Tasks) == 0 {
		return renderEmpty(d, th)
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("gantt render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	minDate, maxDate := dateRange(d.Tasks)
	totalDays := maxDate.Sub(minDate).Hours() / 24
	if totalDays < 1 {
		totalDays = 1
	}

	chartW := totalDays * dayWidth
	pad := defaultPadding

	headerY := pad
	if d.Title != "" {
		headerY += titleH
	}
	axisY := headerY
	bodyY := axisY + axisH

	// One row per task; sections no longer get their own empty row —
	// the label is vertically centered in the section's first bar row
	// (mmdc-style), so a bare section header line wastes vertical
	// space.
	rows := len(d.Tasks)
	chartH := bodyY + float64(rows)*(barH+barGap) + pad

	// Reserve right-edge room only for tasks whose label actually
	// overflows their bar — adding the worst-case label width to
	// every chart wastes whitespace on most diagrams.
	// Measured once and reused by the placement loop below so the
	// viewBox-sizing pass and the inside-vs-outside-bar decision can't
	// disagree if one site's inputs ever change.
	taskLabelW := make([]float64, len(d.Tasks))
	for i, task := range d.Tasks {
		taskLabelW[i], _ = ruler.Measure(task.Name, fontSize-1)
	}

	chartX := pad + sectionLabelW
	rightExtent := chartX + chartW
	for i, task := range d.Tasks {
		startOffset := task.Start.Sub(minDate).Hours() / 24 * dayWidth
		endOffset := task.End.Sub(minDate).Hours() / 24 * dayWidth
		barW := endOffset - startOffset
		if barW < 2 {
			barW = 2
		}
		if taskLabelW[i]+labelInsideSlack > barW {
			if e := chartX + endOffset + labelOutsideGap + taskLabelW[i]; e > rightExtent {
				rightExtent = e
			}
		}
	}
	viewW := rightExtent + labelEdgeMargin + pad
	viewH := chartH

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize+2),
			Content: d.Title,
		})
	}

	rowH := barH + barGap
	rowY := func(i int) float64 { return bodyY + float64(i)*rowH }
	spans := sectionSpans(d.Tasks)

	// Bands drawn first so the axis grid lines and bars paint on top.
	// Band X/width extend from `pad` to `viewW - pad` — wider than the
	// chart body — so the section label column and the overflow-label
	// margin both sit on the tint, matching mmdc. Unnamed spans get no
	// band (and no palette slot, via sectionSpans.index=-1), consistent
	// with unnamed spans also getting no section label.
	if len(th.SectionBands) > 0 {
		for _, sp := range spans {
			if sp.index < 0 {
				continue
			}
			color := th.SectionBands[sp.index%len(th.SectionBands)]
			children = append(children, &rect{
				X: svgFloat(pad), Y: svgFloat(rowY(sp.start)),
				Width: svgFloat(viewW - 2*pad), Height: svgFloat(float64(sp.end-sp.start) * rowH),
				Style: fmt.Sprintf("fill:%s;stroke:none", color),
			})
		}
	}

	children = append(children, renderAxis(d, minDate, maxDate, totalDays, chartX, axisY, chartW, rowH*float64(rows), fontSize, th)...)
	if marker, ok := todayMarkerLine(d, minDate, maxDate, chartX, axisY+axisH, rowH*float64(rows), th); ok {
		children = append(children, marker)
	}

	// Per-section label vertically centered across its bar rows.
	for _, sp := range spans {
		if sp.name == "" {
			continue
		}
		centerY := rowY(sp.start) + float64(sp.end-sp.start)*rowH/2
		children = append(children, &text{
			X: svgFloat(pad + sectionLabelW - 8), Y: svgFloat(centerY),
			Anchor: "end", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.SectionText, fontSize),
			Content: sp.name,
		})
	}
	for i, task := range d.Tasks {
		startOffset := task.Start.Sub(minDate).Hours() / 24 * dayWidth
		endOffset := task.End.Sub(minDate).Hours() / 24 * dayWidth
		barW := endOffset - startOffset
		if barW < 2 {
			barW = 2
		}
		bx := chartX + startOffset
		by := rowY(i) + barGap/2

		color := th.taskColor(task.Status)

		// Milestone tasks render as a diamond at the task's start
		// rather than as a rectangle. The label always sits to the
		// right of the glyph since its width is a small fixed
		// constant regardless of the task's parsed duration.
		if task.Status.Has(diagram.TaskStatusMilestone) {
			cx, cy := bx, by+barH/2
			half := barH / 2
			children = append(children, &polygon{
				Points: fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f %.2f,%.2f",
					cx, cy-half,
					cx+half, cy,
					cx, cy+half,
					cx-half, cy),
				Style: fmt.Sprintf("fill:%s;stroke:none", color),
			})
			children = append(children, &text{
				X: svgFloat(cx + half + labelOutsideGap), Y: svgFloat(cy),
				Anchor: "start", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.OutsideBarText, fontSize-1),
				Content: task.Name,
			})
			continue
		}

		// Crit tasks get a strong outline on top of the fill so a
		// blocker stands out even when its bar sits next to other
		// red bars. The fill priority already used the crit color;
		// the stroke darkens that.
		barStyle := fmt.Sprintf("fill:%s;stroke:none", color)
		if task.Status.Has(diagram.TaskStatusCrit) {
			barStyle = fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1.5", color, th.CritStroke)
		}
		children = append(children, &rect{
			X: svgFloat(bx), Y: svgFloat(by),
			Width: svgFloat(barW), Height: svgFloat(barH),
			RX: 3, RY: 3,
			Style: barStyle,
		})
		if taskLabelW[i]+labelInsideSlack > barW {
			children = append(children, &text{
				X: svgFloat(bx + barW + labelOutsideGap), Y: svgFloat(by + barH/2),
				Anchor: "start", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.OutsideBarText, fontSize-1),
				Content: task.Name,
			})
		} else {
			children = append(children, &text{
				X: svgFloat(bx + barW/2), Y: svgFloat(by + barH/2),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.InsideBarText, fontSize-1),
				Content: task.Name,
			})
		}
	}

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("gantt render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func renderEmpty(d *diagram.GanttDiagram, th Theme) ([]byte, error) {
	pad := defaultPadding
	h := 2 * pad
	w := 2 * pad
	if d.Title != "" {
		h += titleH
		w = 200
	}
	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(w), Height: svgFloat(h),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})
	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(w / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:15px;font-weight:bold", th.TitleText),
			Content: d.Title,
		})
	}
	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", w, h),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("gantt render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

// renderAxis draws the axis baseline, tick labels, and full-height
// vertical grid lines at each tick so bars read against a timeline.
// bodyH is the vertical extent of the task area below the axis.
//
// `axisFormat` (d3-strftime) and `tickInterval` (`<N><unit>`)
// override the defaults; an empty/invalid spec falls back to ISO
// labels and an auto-chosen interval based on chart span.
func renderAxis(d *diagram.GanttDiagram, minDate, maxDate time.Time, totalDays, x, y, w, bodyH, fontSize float64, th Theme) []any {
	var elems []any
	elems = append(elems, &line{
		X1: svgFloat(x), Y1: svgFloat(y + axisH),
		X2: svgFloat(x + w), Y2: svgFloat(y + axisH),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.AxisStroke),
	})

	layout := "2006-01-02"
	if d.AxisFormat != "" {
		layout = d3StrftimeToGoLayout(d.AxisFormat)
	}
	gridStyle := fmt.Sprintf("stroke:%s;stroke-width:1", th.GridStroke)

	if step, ok := parseTickInterval(d.TickInterval); ok {
		// Calendar-aware stepping: walk the full date range with
		// the user-requested cadence rather than the day-bucket
		// auto-interval.
		for tick := minDate; !tick.After(maxDate); tick = step.advance(tick) {
			dx := x + tick.Sub(minDate).Hours()/24*dayWidth
			elems = append(elems, axisTick(tick, dx, y, bodyH, fontSize, layout, gridStyle, th)...)
		}
		return elems
	}

	interval := axisInterval(totalDays)
	for day := 0; float64(day) <= totalDays; day += interval {
		dx := x + float64(day)*dayWidth
		date := minDate.AddDate(0, 0, day)
		elems = append(elems, axisTick(date, dx, y, bodyH, fontSize, layout, gridStyle, th)...)
	}
	return elems
}

// axisTick is one tick label + its grid line, factored out so the
// `tickInterval` and auto-interval branches above stay symmetric.
func axisTick(date time.Time, dx, y, bodyH, fontSize float64, layout, gridStyle string, th Theme) []any {
	return []any{
		&text{
			X: svgFloat(dx), Y: svgFloat(y + axisH - 6),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.AxisLabel, fontSize-2),
			Content: date.Format(layout),
		},
		&line{
			X1: svgFloat(dx), Y1: svgFloat(y + axisH),
			X2: svgFloat(dx), Y2: svgFloat(y + axisH + bodyH),
			Style: gridStyle,
		},
	}
}

// todayMarkerLine returns a vertical rule at the current day
// position when (a) the diagram opted in via `todayMarker` (any
// non-empty value other than `off`) AND (b) today's date sits
// within the chart's date range. Style is parsed from the user's
// directive value when it looks like a CSS string; otherwise a
// red dashed line is used.
func todayMarkerLine(d *diagram.GanttDiagram, minDate, maxDate time.Time, x, y, bodyH float64, th Theme) (*line, bool) {
	if d.TodayMarker == "" || d.TodayMarker == "off" {
		return nil, false
	}
	now := time.Now().UTC().Truncate(24 * time.Hour)
	if now.Before(minDate) || now.After(maxDate) {
		return nil, false
	}
	dx := x + now.Sub(minDate).Hours()/24*dayWidth
	style := "stroke:#d62728;stroke-width:2;stroke-dasharray:4 2"
	if strings.Contains(d.TodayMarker, ":") {
		// User supplied a CSS-ish snippet like
		// `stroke-width:3px,stroke:red`; pass it through after
		// normalising commas to semicolons.
		style = strings.ReplaceAll(d.TodayMarker, ",", ";")
	}
	return &line{
		X1: svgFloat(dx), Y1: svgFloat(y),
		X2: svgFloat(dx), Y2: svgFloat(y + bodyH),
		Style: style,
	}, true
}

func axisInterval(totalDays float64) int {
	switch {
	case totalDays <= 14:
		return 1
	case totalDays <= 60:
		return 7
	case totalDays <= 365:
		return 30
	default:
		return 90
	}
}

// sectionSpan describes one contiguous run of tasks sharing the same
// Section. index counts named spans only — unnamed runs (Section="")
// are represented in the slice but carry index=-1 so callers can skip
// painting bands or labels for them without desynchronizing the
// palette cycle across the diagram.
type sectionSpan struct {
	start, end, index int
	name              string
}

func sectionSpans(tasks []diagram.GanttTask) []sectionSpan {
	if len(tasks) == 0 {
		return nil
	}
	spans := []sectionSpan{{start: 0, name: tasks[0].Section}}
	for i := 1; i < len(tasks); i++ {
		if tasks[i].Section != spans[len(spans)-1].name {
			spans[len(spans)-1].end = i
			spans = append(spans, sectionSpan{start: i, name: tasks[i].Section})
		}
	}
	spans[len(spans)-1].end = len(tasks)
	// Assign palette indices only to named spans so a leading run of
	// unsectioned tasks doesn't consume palette[0].
	idx := 0
	for i := range spans {
		if spans[i].name == "" {
			spans[i].index = -1
			continue
		}
		spans[i].index = idx
		idx++
	}
	return spans
}

func dateRange(tasks []diagram.GanttTask) (min, max time.Time) {
	min = tasks[0].Start
	max = tasks[0].End
	for _, t := range tasks[1:] {
		if t.Start.Before(min) {
			min = t.Start
		}
		if t.End.After(max) {
			max = t.End
		}
	}
	return min, max
}


