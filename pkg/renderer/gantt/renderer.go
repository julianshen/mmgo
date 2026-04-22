package gantt

import (
	"encoding/xml"
	"fmt"
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
	// Each task gets one row; rowY(i) returns its top. Section bands
	// and grid lines are drawn first so bars paint on top of them.
	rowY := func(i int) float64 { return bodyY + float64(i)*rowH }

	// Section bands: group tasks by contiguous section and tint the
	// full-width row range behind them so the eye can cluster related
	// bars without tracing the section label across the chart.
	if len(th.SectionBands) > 0 {
		sectionIdx := -1
		prev := "__unset__"
		bandStart := 0
		flush := func(end int) {
			if sectionIdx < 0 || end <= bandStart {
				return
			}
			color := th.SectionBands[sectionIdx%len(th.SectionBands)]
			children = append(children, &rect{
				X: svgFloat(pad), Y: svgFloat(rowY(bandStart)),
				Width: svgFloat(viewW - 2*pad), Height: svgFloat(float64(end-bandStart) * rowH),
				Style: fmt.Sprintf("fill:%s;stroke:none", color),
			})
		}
		for i, task := range d.Tasks {
			if task.Section != prev {
				flush(i)
				prev = task.Section
				bandStart = i
				sectionIdx++
			}
		}
		flush(len(d.Tasks))
	}

	children = append(children, renderAxis(minDate, totalDays, chartX, axisY, chartW, rowH*float64(rows), fontSize, th)...)

	// Bars + labels. The per-section label is emitted once per section,
	// vertically centered across the section's row range.
	sectionStart := -1
	sectionName := "__unset__"
	flushSectionLabel := func(endRow int) {
		if sectionStart < 0 || sectionName == "" {
			return
		}
		centerY := rowY(sectionStart) + float64(endRow-sectionStart)*rowH/2
		children = append(children, &text{
			X: svgFloat(pad + sectionLabelW - 8), Y: svgFloat(centerY),
			Anchor: "end", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.SectionText, fontSize),
			Content: sectionName,
		})
	}
	for i, task := range d.Tasks {
		if task.Section != sectionName {
			flushSectionLabel(i)
			sectionName = task.Section
			sectionStart = i
		}

		startOffset := task.Start.Sub(minDate).Hours() / 24 * dayWidth
		endOffset := task.End.Sub(minDate).Hours() / 24 * dayWidth
		barW := endOffset - startOffset
		if barW < 2 {
			barW = 2
		}
		bx := chartX + startOffset
		by := rowY(i) + barGap/2

		color := th.taskColor(task.Status)
		children = append(children, &rect{
			X: svgFloat(bx), Y: svgFloat(by),
			Width: svgFloat(barW), Height: svgFloat(barH),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:none", color),
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
	flushSectionLabel(len(d.Tasks))

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
func renderAxis(minDate time.Time, totalDays float64, x, y, w, bodyH, fontSize float64, th Theme) []any {
	var elems []any
	elems = append(elems, &line{
		X1: svgFloat(x), Y1: svgFloat(y + axisH),
		X2: svgFloat(x + w), Y2: svgFloat(y + axisH),
		Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.AxisStroke),
	})

	interval := axisInterval(totalDays)
	gridStyle := fmt.Sprintf("stroke:%s;stroke-width:1", th.GridStroke)
	for day := 0; float64(day) <= totalDays; day += interval {
		dx := x + float64(day)*dayWidth
		date := minDate.AddDate(0, 0, day)
		// ISO-8601 (yyyy-mm-dd) matches mmdc's axis format and
		// disambiguates year/century compared to "Jan 02".
		label := date.Format("2006-01-02")
		elems = append(elems, &text{
			X: svgFloat(dx), Y: svgFloat(y + axisH - 6),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.AxisLabel, fontSize-2),
			Content: label,
		})
		elems = append(elems, &line{
			X1: svgFloat(dx), Y1: svgFloat(y + axisH),
			X2: svgFloat(dx), Y2: svgFloat(y + axisH + bodyH),
			Style: gridStyle,
		})
	}
	return elems
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


