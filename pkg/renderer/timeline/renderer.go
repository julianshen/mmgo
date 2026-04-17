package timeline

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
)

const (
	defaultFontSize = 14.0
	defaultPadding  = 20.0
	titleH          = 30.0
	timeColW        = 120.0
	eventBoxW       = 180.0
	eventBoxH       = 36.0
	rowGap          = 14.0
	axisX           = 150.0 // relative to left padding
	axisW           = 3.0
	dotR            = 7.0
	sectionGap      = 18.0
)

var sectionColors = []string{"#4e79a7", "#f28e2b", "#59a14f", "#e15759", "#76b7b2", "#edc948"}

type Options struct {
	FontSize float64
}

func Render(d *diagram.TimelineDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("timeline render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	pad := defaultPadding

	rows, sectionStarts := countRows(d)
	totalH := pad
	if d.Title != "" {
		totalH += titleH
	}
	totalH += float64(rows)*(eventBoxH+rowGap) + float64(len(d.Sections))*sectionGap + pad
	viewW := pad + axisX + axisW + 30 + eventBoxW + pad
	_ = sectionStarts

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(totalH),
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

	startY := pad
	if d.Title != "" {
		startY += titleH
	}

	axis := pad + axisX
	children = append(children, &line{
		X1: svgFloat(axis), Y1: svgFloat(startY),
		X2: svgFloat(axis), Y2: svgFloat(totalH - pad),
		Style: "stroke:#999;stroke-width:2",
	})

	curY := startY + rowGap
	if len(d.Sections) > 0 {
		for i, sec := range d.Sections {
			color := sectionColors[i%len(sectionColors)]
			curY += sectionGap / 2
			children = append(children, &text{
				X: svgFloat(axis - 40), Y: svgFloat(curY - 4),
				Anchor: "end", Dominant: "auto",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", color, fontSize),
				Content: sec.Name,
			})
			curY += sectionGap / 2
			for _, ev := range sec.Events {
				children = append(children, renderEvent(ev, axis, curY, color, fontSize)...)
				curY += eventBoxH + rowGap
			}
		}
	} else {
		color := sectionColors[0]
		for _, ev := range d.Events {
			children = append(children, renderEvent(ev, axis, curY, color, fontSize)...)
			curY += eventBoxH + rowGap
		}
	}

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, totalH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("timeline render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), svgBytes...), nil
}

func countRows(d *diagram.TimelineDiagram) (int, []int) {
	starts := []int{}
	rows := 0
	if len(d.Sections) > 0 {
		for _, s := range d.Sections {
			starts = append(starts, rows)
			rows += len(s.Events)
		}
	} else {
		rows = len(d.Events)
	}
	return rows, starts
}

func renderEvent(ev diagram.TimelineEvent, axis, y float64, color string, fontSize float64) []any {
	var elems []any
	elems = append(elems, &text{
		X: svgFloat(axis - 20), Y: svgFloat(y + eventBoxH/2),
		Anchor: "end", Dominant: "central",
		Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize),
		Content: ev.Time,
	})
	elems = append(elems, &circle{
		CX: svgFloat(axis), CY: svgFloat(y + eventBoxH/2), R: svgFloat(dotR),
		Style: fmt.Sprintf("fill:%s;stroke:white;stroke-width:2", color),
	})
	boxX := axis + 20
	content := ev.Events[0]
	for _, e := range ev.Events[1:] {
		content += ", " + e
	}
	elems = append(elems, &rect{
		X: svgFloat(boxX), Y: svgFloat(y),
		Width: svgFloat(eventBoxW), Height: svgFloat(eventBoxH),
		RX: 5, RY: 5,
		Style: fmt.Sprintf("fill:%s;stroke:none", color),
	})
	elems = append(elems, &text{
		X: svgFloat(boxX + eventBoxW/2), Y: svgFloat(y + eventBoxH/2),
		Anchor: "middle", Dominant: "central",
		Style:   fmt.Sprintf("fill:white;font-size:%.0fpx", fontSize-1),
		Content: content,
	})
	return elems
}
