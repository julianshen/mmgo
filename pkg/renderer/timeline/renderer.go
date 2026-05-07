package timeline

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
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

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.TimelineDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("timeline render: diagram is nil")
	}

	fontSize := defaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)
	// An empty diagram (no sections, no events) renders a tiny
	// placeholder regardless of direction — neither layout has
	// columns or rows to draw, so dispatch is moot.
	if len(d.Sections) == 0 && len(d.Events) == 0 {
		return renderEmpty(d, fontSize, th)
	}
	// Mermaid spec defaults timelines to LR (horizontal); only an
	// explicit `direction TD` flips to the vertical layout this
	// renderer originally shipped with.
	if d.Direction == "TD" {
		return renderTD(d, fontSize, th)
	}
	return renderLR(d, fontSize, th)
}

func renderTD(d *diagram.TimelineDiagram, fontSize float64, th Theme) ([]byte, error) {
	pad := defaultPadding

	rows := countRows(d)
	totalH := pad
	if d.Title != "" {
		totalH += titleH
	}
	totalH += rowGap + float64(rows)*(eventBoxH+rowGap) + float64(len(d.Sections))*sectionGap + pad
	viewW := pad + axisX + axisW + 30 + eventBoxW + pad

	children := emitChrome(d, viewW, totalH, fontSize, th)

	startY := pad
	if d.Title != "" {
		startY += titleH
	}

	axis := pad + axisX
	children = append(children, &line{
		X1: svgFloat(axis), Y1: svgFloat(startY),
		X2: svgFloat(axis), Y2: svgFloat(totalH - pad),
		Style: fmt.Sprintf("stroke:%s;stroke-width:2", th.AxisStroke),
	})

	curY := startY + rowGap
	if len(d.Sections) > 0 {
		for i, sec := range d.Sections {
			color := th.SectionColors[i%len(th.SectionColors)]
			curY += sectionGap / 2
			children = append(children, &text{
				X: svgFloat(axis - 40), Y: svgFloat(curY - 4),
				Anchor: "end", Dominant: "auto",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", color, fontSize),
				Content: sec.Name,
			})
			curY += sectionGap / 2
			for _, ev := range sec.Events {
				elems, dy := renderEvent(ev, axis, curY, color, fontSize, th)
				children = append(children, elems...)
				curY += dy
			}
		}
	} else {
		color := th.SectionColors[0]
		for _, ev := range d.Events {
			elems, dy := renderEvent(ev, axis, curY, color, fontSize, th)
			children = append(children, elems...)
			curY += dy
		}
	}

	return finalizeSVG(0, 0, viewW, totalH, children)
}

func countRows(d *diagram.TimelineDiagram) int {
	rows := 0
	periods := d.Events
	if len(d.Sections) > 0 {
		for _, s := range d.Sections {
			for _, p := range s.Events {
				rows += eventRowCount(p)
			}
		}
		return rows
	}
	for _, p := range periods {
		rows += eventRowCount(p)
	}
	return rows
}

// eventRowCount is the number of vertically stacked event boxes a
// period contributes. A period with no Events list is still one
// row (so the Time + axis dot has somewhere to live).
func eventRowCount(p diagram.TimelineEvent) int {
	if n := len(p.Events); n > 0 {
		return n
	}
	return 1
}

// renderEvent emits one period's stack of events: the Time label
// and axis dot anchor at the first row, followed by one filled
// rounded-rect+label per event in the period. Returns the
// rendered elements and the y-delta consumed (so the caller can
// advance its cursor across multi-event periods).
func renderEvent(ev diagram.TimelineEvent, axis, y float64, color string, fontSize float64, th Theme) ([]any, float64) {
	var elems []any
	rowAdvance := eventBoxH + rowGap
	elems = append(elems, &text{
		X: svgFloat(axis - 20), Y: svgFloat(y + eventBoxH/2),
		Anchor: "end", Dominant: "central",
		Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.SectionText, fontSize),
		Content: ev.Time,
	})
	elems = append(elems, &circle{
		CX: svgFloat(axis), CY: svgFloat(y + eventBoxH/2), R: svgFloat(dotR),
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", color, th.Background),
	})
	boxX := axis + 20
	events := ev.Events
	if len(events) == 0 {
		// Period without any events still draws an empty box so
		// the time label has a visual anchor.
		events = []string{""}
	}
	for i, content := range events {
		ey := y + float64(i)*rowAdvance
		elems = append(elems, &rect{
			X: svgFloat(boxX), Y: svgFloat(ey),
			Width: svgFloat(eventBoxW), Height: svgFloat(eventBoxH),
			RX: 5, RY: 5,
			Style: fmt.Sprintf("fill:%s;stroke:none", color),
		})
		if content != "" {
			elems = append(elems, &text{
				X: svgFloat(boxX + eventBoxW/2), Y: svgFloat(ey + eventBoxH/2),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EventText, fontSize-1),
				Content: content,
			})
		}
	}
	return elems, float64(len(events)) * rowAdvance
}

const (
	// LR layout metrics. Period columns are slightly wider than
	// TD's eventBoxW because the LR header (time label) sits
	// above the column instead of beside the axis dot.
	lrColW           = 180.0
	lrColGap         = 16.0
	lrSectionBandH   = 28.0
	lrTimeRowH       = 26.0
	lrEventBoxH      = 40.0
	lrEventGap       = 8.0
	lrAxisStrokeW    = 2.0
	lrAxisDotR       = 6.0
	lrSectionBandPad = 4.0
)

// renderLR draws the spec-default horizontal timeline layout:
// section bands sit at the top spanning the columns they own,
// period time labels run as a row beneath the bands, a single
// horizontal axis cuts across all columns, and events stack
// vertically below each period column.
func renderLR(d *diagram.TimelineDiagram, fontSize float64, th Theme) ([]byte, error) {
	pad := defaultPadding

	// Flatten section→period structure into a single column list
	// so layout can address every period uniformly. sectionRanges
	// records each section's first/last column (inclusive) so the
	// header band can span them.
	type col struct {
		ev      diagram.TimelineEvent
		section int // -1 for section-less periods
		color   string
	}
	var cols []col
	type sectionRange struct {
		name       string
		startCol   int
		endCol     int
		colorIndex int
	}
	var ranges []sectionRange
	maxEvents := 0
	tally := func(ev diagram.TimelineEvent) {
		if n := len(ev.Events); n > maxEvents {
			maxEvents = n
		}
	}
	if len(d.Sections) > 0 {
		for i, sec := range d.Sections {
			color := th.SectionColors[i%len(th.SectionColors)]
			startCol := len(cols)
			for _, ev := range sec.Events {
				cols = append(cols, col{ev: ev, section: i, color: color})
				tally(ev)
			}
			endCol := len(cols) - 1
			if endCol >= startCol {
				ranges = append(ranges, sectionRange{
					name:       sec.Name,
					startCol:   startCol,
					endCol:     endCol,
					colorIndex: i,
				})
			}
		}
	} else {
		color := th.SectionColors[0]
		for _, ev := range d.Events {
			cols = append(cols, col{ev: ev, section: -1, color: color})
			tally(ev)
		}
	}
	if len(cols) == 0 {
		// Render dispatched here only when sections/events list
		// existed but every section's Events slice was empty.
		// Fall through to the empty-diagram chrome.
		return renderEmpty(d, fontSize, th)
	}

	// Layout pass.
	titleOffset := 0.0
	if d.Title != "" {
		titleOffset = titleH
	}
	bandRowH := 0.0
	if len(ranges) > 0 {
		bandRowH = lrSectionBandH + lrSectionBandPad
	}
	if maxEvents == 0 {
		maxEvents = 1
	}
	eventsRowH := float64(maxEvents) * (lrEventBoxH + lrEventGap)

	bandY := pad + titleOffset
	timeRowY := bandY + bandRowH
	axisY := timeRowY + lrTimeRowH
	eventsY := axisY + lrEventGap

	viewW := pad + float64(len(cols))*(lrColW+lrColGap) + pad
	viewH := eventsY + eventsRowH + pad

	colCenterX := func(i int) float64 {
		return pad + float64(i)*(lrColW+lrColGap) + lrColW/2
	}

	children := emitChrome(d, viewW, viewH, fontSize, th)

	// Section bands span their column range with a tinted rect
	// and centered title.
	for _, r := range ranges {
		startX := pad + float64(r.startCol)*(lrColW+lrColGap)
		endX := pad + float64(r.endCol)*(lrColW+lrColGap) + lrColW
		color := th.SectionColors[r.colorIndex%len(th.SectionColors)]
		children = append(children, &rect{
			X: svgFloat(startX), Y: svgFloat(bandY),
			Width: svgFloat(endX - startX), Height: svgFloat(lrSectionBandH),
			RX: 4, RY: 4,
			Style: fmt.Sprintf("fill:%s;fill-opacity:0.18;stroke:%s;stroke-width:1;stroke-opacity:0.5", color, color),
		})
		if r.name != "" {
			children = append(children, &text{
				X: svgFloat((startX + endX) / 2), Y: svgFloat(bandY + lrSectionBandH/2),
				Anchor: "middle", Dominant: "central",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", color, fontSize),
				Content: r.name,
			})
		}
	}

	// Horizontal axis: a single line across the entire column
	// range; each column gets a dot at its center.
	axisX1 := pad
	axisX2 := pad + float64(len(cols))*(lrColW+lrColGap) - lrColGap
	if axisX2 < axisX1 {
		axisX2 = axisX1
	}
	children = append(children, &line{
		X1: svgFloat(axisX1), Y1: svgFloat(axisY),
		X2: svgFloat(axisX2), Y2: svgFloat(axisY),
		Style: fmt.Sprintf("stroke:%s;stroke-width:%g", th.AxisStroke, lrAxisStrokeW),
	})

	for i, c := range cols {
		cx := colCenterX(i)
		// Time label sits just above the axis.
		children = append(children, &text{
			X: svgFloat(cx), Y: svgFloat(timeRowY + lrTimeRowH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.SectionText, fontSize),
			Content: c.ev.Time,
		})
		// Axis dot anchors the column to the timeline.
		children = append(children, &circle{
			CX: svgFloat(cx), CY: svgFloat(axisY), R: svgFloat(lrAxisDotR),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", c.color, th.Background),
		})
		events := c.ev.Events
		if len(events) == 0 {
			events = []string{""}
		}
		for j, content := range events {
			ey := eventsY + float64(j)*(lrEventBoxH+lrEventGap)
			children = append(children, &rect{
				X: svgFloat(cx - lrColW/2), Y: svgFloat(ey),
				Width: svgFloat(lrColW), Height: svgFloat(lrEventBoxH),
				RX: 5, RY: 5,
				Style: fmt.Sprintf("fill:%s;stroke:none", c.color),
			})
			if content != "" {
				children = append(children, &text{
					X: svgFloat(cx), Y: svgFloat(ey + lrEventBoxH/2),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.EventText, fontSize-1),
					Content: content,
				})
			}
		}
	}

	return finalizeSVG(0, 0, viewW, viewH, children)
}

// renderEmpty draws a minimal placeholder when the diagram has
// no periods at all — shared by both LR and TD dispatch paths
// since neither layout has anything to draw without periods.
// Title + accessibility are still honoured.
func renderEmpty(d *diagram.TimelineDiagram, fontSize float64, th Theme) ([]byte, error) {
	pad := defaultPadding
	w := 200.0
	h := 2 * pad
	if d.Title != "" {
		h += titleH
	}
	children := emitChrome(d, w, h, fontSize, th)
	return finalizeSVG(0, 0, w, h, children)
}

// emitChrome builds the always-on opening SVG children every
// timeline render shares: optional `<title>` / `<desc>` for
// screen readers, the full-canvas background rect, and the
// optional centered diagram title.
func emitChrome(d *diagram.TimelineDiagram, viewW, viewH, fontSize float64, th Theme) []any {
	var children []any
	if d.AccTitle != "" {
		children = append(children, &svgutil.Title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &svgutil.Desc{Content: d.AccDescr})
	}
	children = append(children, &rect{
		X: 0, Y: 0, Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})
	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(viewW / 2), Y: svgFloat(defaultPadding + titleH/2),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize+2),
			Content: d.Title,
		})
	}
	return children
}

// finalizeSVG marshals the root <svg> with the supplied viewBox
// and prepends the XML declaration. The minX/minY parameters let
// callers shift the origin (currently always 0) without further
// cosmetics in the call site.
func finalizeSVG(minX, minY, viewW, viewH float64, children []any) ([]byte, error) {
	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("%g %g %.2f %.2f", minX, minY, viewW, viewH),
		Children: children,
	}
	out, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("timeline render: %w", err)
	}
	return append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), out...), nil
}
