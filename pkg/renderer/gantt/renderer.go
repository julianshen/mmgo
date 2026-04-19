package gantt

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/julianshen/mmgo/pkg/diagram"
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
)

type Options struct {
	FontSize float64
}

func Render(d *diagram.GanttDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("gantt render: diagram is nil")
	}
	if len(d.Tasks) == 0 {
		return renderEmpty(d)
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

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

	rows := len(d.Tasks) + len(d.Sections)
	chartH := bodyY + float64(rows)*(barH+barGap) + pad

	// Reserve room for outside-the-bar labels on the rightmost task
	// so they don't clip past the viewBox edge. avgCharWidth is the
	// same heuristic used for the in-bar fit decision below.
	maxLabelLen := 0
	for _, task := range d.Tasks {
		if n := len(task.Name); n > maxLabelLen {
			maxLabelLen = n
		}
	}
	rightLabelPad := float64(maxLabelLen)*(fontSize-1)*0.55 + 12
	viewW := pad + sectionLabelW + chartW + rightLabelPad + pad
	viewH := chartH

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:#fff;stroke:none",
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize+2),
			Content: d.Title,
		})
	}

	chartX := pad + sectionLabelW
	children = append(children, renderAxis(minDate, totalDays, chartX, axisY, chartW, fontSize)...)

	curY := bodyY
	curSection := ""
	for _, task := range d.Tasks {
		if task.Section != curSection && task.Section != "" {
			curSection = task.Section
			children = append(children, &text{
				X: svgFloat(pad + sectionLabelW - 8), Y: svgFloat(curY + barH/2),
				Anchor: "end", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize),
				Content: curSection,
			})
			curY += barH + barGap
		}

		startOffset := task.Start.Sub(minDate).Hours() / 24 * dayWidth
		endOffset := task.End.Sub(minDate).Hours() / 24 * dayWidth
		barW := endOffset - startOffset
		if barW < 2 {
			barW = 2
		}
		bx := chartX + startOffset
		by := curY

		color := taskColor(task.Status)
		children = append(children, &rect{
			X: svgFloat(bx), Y: svgFloat(by),
			Width: svgFloat(barW), Height: svgFloat(barH),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:none", color),
		})
		// Estimate label width and place the text outside the bar
		// (right side, dark color) when the bar is too narrow to hold
		// it without spillover. Inside-bar labels stay white-on-fill.
		labelW := float64(len(task.Name)) * (fontSize - 1) * 0.55
		if labelW+8 > barW {
			children = append(children, &text{
				X: svgFloat(bx + barW + 4), Y: svgFloat(by + barH/2),
				Anchor: "start", Dominant: "central",
				Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
				Content: task.Name,
			})
		} else {
			children = append(children, &text{
				X: svgFloat(bx + barW/2), Y: svgFloat(by + barH/2),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:white;font-size:%.0fpx", fontSize-1),
				Content: task.Name,
			})
		}
		curY += barH + barGap
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

func renderEmpty(d *diagram.GanttDiagram) ([]byte, error) {
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
		Style: "fill:#fff;stroke:none",
	})
	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(w / 2), Y: svgFloat(pad + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   "fill:#333;font-size:15px;font-weight:bold",
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

func renderAxis(minDate time.Time, totalDays float64, x, y, w, fontSize float64) []any {
	var elems []any
	elems = append(elems, &line{
		X1: svgFloat(x), Y1: svgFloat(y + axisH),
		X2: svgFloat(x + w), Y2: svgFloat(y + axisH),
		Style: "stroke:#ccc;stroke-width:1",
	})

	interval := axisInterval(totalDays)
	for day := 0; float64(day) <= totalDays; day += interval {
		dx := x + float64(day)*dayWidth
		date := minDate.AddDate(0, 0, day)
		label := date.Format("Jan 02")
		elems = append(elems, &text{
			X: svgFloat(dx), Y: svgFloat(y + axisH - 6),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:#666;font-size:%.0fpx", fontSize-2),
			Content: label,
		})
		elems = append(elems, &line{
			X1: svgFloat(dx), Y1: svgFloat(y + axisH - 3),
			X2: svgFloat(dx), Y2: svgFloat(y + axisH + 3),
			Style: "stroke:#ccc;stroke-width:1",
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

func taskColor(s diagram.TaskStatus) string {
	switch s {
	case diagram.TaskStatusDone:
		return "#9370DB"
	case diagram.TaskStatusActive:
		return "#4e79a7"
	case diagram.TaskStatusCrit:
		return "#e15759"
	default:
		return "#76b7b2"
	}
}

