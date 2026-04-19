package pie

import (
	"encoding/xml"
	"fmt"
	"math"

	"github.com/julianshen/mmgo/pkg/diagram"
)

const (
	DefaultFontSize = 14.0
	defaultRadius   = 120.0
	defaultPadding  = 30.0
	defaultLegendW  = 150.0
	defaultLegendH  = 20.0
	legendGap       = 5.0
	legendSwatchW   = 14.0
)

type Options struct {
	FontSize float64
}

var defaultColors = []string{
	"#4e79a7", "#f28e2b", "#e15759", "#76b7b2",
	"#59a14f", "#edc948", "#b07aa1", "#ff9da7",
	"#9c755f", "#bab0ac",
}

func Render(d *diagram.PieDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("pie render: diagram is nil")
	}

	fontSize := DefaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}

	total := 0.0
	for _, s := range d.Slices {
		total += s.Value
	}

	pad := defaultPadding
	cx := pad + defaultRadius
	cy := pad + defaultRadius
	if d.Title != "" {
		cy += fontSize + 10
	}

	legendX := cx + defaultRadius + pad
	legendY := cy - defaultRadius

	viewW := legendX + defaultLegendW + pad
	viewH := cy + defaultRadius + pad

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0,
		Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: "fill:white;stroke:none",
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(cx), Y: svgFloat(pad + fontSize),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx;font-weight:bold", fontSize+2),
			Content: d.Title,
		})
	}

	if len(d.Slices) == 1 {
		s := d.Slices[0]
		children = append(children, &circle{
			CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(defaultRadius),
			Style: fmt.Sprintf("fill:%s;stroke:white;stroke-width:2", colorFor(0)),
		})
		label := "100.0%"
		if d.ShowData {
			label = fmt.Sprintf("100.0%% (%.0f)", s.Value)
		}
		children = append(children, &text{
			X: svgFloat(cx), Y: svgFloat(cy),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:white;font-size:%.0fpx;font-weight:bold", fontSize-1),
			Content: label,
		})
	} else if total > 0 {
		startAngle := -math.Pi / 2
		for i, s := range d.Slices {
			frac := s.Value / total
			sweep := frac * 2 * math.Pi
			endAngle := startAngle + sweep
			children = append(children, arcPath(cx, cy, defaultRadius, startAngle, endAngle, colorFor(i)))
			startAngle = endAngle
		}
		// Slices below this fraction get an outside label connected
		// by a short leader line — centered in-slice text would
		// collide with adjacent thin slices.
		const smallSliceThreshold = 0.06
		startAngle = -math.Pi / 2
		for _, s := range d.Slices {
			frac := s.Value / total
			sweep := frac * 2 * math.Pi
			midAngle := startAngle + sweep/2
			pct := fmt.Sprintf("%.1f%%", frac*100)
			label := pct
			if d.ShowData {
				label = fmt.Sprintf("%s (%.0f)", pct, s.Value)
			}
			if frac >= smallSliceThreshold {
				lx := cx + defaultRadius*0.65*math.Cos(midAngle)
				ly := cy + defaultRadius*0.65*math.Sin(midAngle)
				children = append(children, &text{
					X: svgFloat(lx), Y: svgFloat(ly),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:white;font-size:%.0fpx;font-weight:bold", fontSize-1),
					Content: label,
				})
			} else {
				// Leader line: from the slice edge out to the label.
				inX := cx + defaultRadius*0.95*math.Cos(midAngle)
				inY := cy + defaultRadius*0.95*math.Sin(midAngle)
				outX := cx + defaultRadius*1.12*math.Cos(midAngle)
				outY := cy + defaultRadius*1.12*math.Sin(midAngle)
				children = append(children, &line{
					X1: svgFloat(inX), Y1: svgFloat(inY),
					X2: svgFloat(outX), Y2: svgFloat(outY),
					Style: "stroke:#666;stroke-width:1",
				})
				anchor := "start"
				if math.Cos(midAngle) < 0 {
					anchor = "end"
				}
				children = append(children, &text{
					X: svgFloat(outX), Y: svgFloat(outY),
					Anchor: anchor, Dominant: "central",
					Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize-1),
					Content: label,
				})
			}
			startAngle += sweep
		}
	}

	for i, s := range d.Slices {
		y := legendY + float64(i)*(defaultLegendH+legendGap)
		children = append(children, &rect{
			X: svgFloat(legendX), Y: svgFloat(y),
			Width: svgFloat(legendSwatchW), Height: svgFloat(legendSwatchW),
			Style: fmt.Sprintf("fill:%s", colorFor(i)),
		})
		children = append(children, &text{
			X: svgFloat(legendX + legendSwatchW + 6), Y: svgFloat(y + legendSwatchW/2),
			Anchor: "start", Dominant: "central",
			Style:   fmt.Sprintf("fill:#333;font-size:%.0fpx", fontSize),
			Content: s.Label,
		})
	}

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", viewW, viewH),
		Children: children,
	}
	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("pie render: %w", err)
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

func arcPath(cx, cy, r, startAngle, endAngle float64, color string) *path {
	x1 := cx + r*math.Cos(startAngle)
	y1 := cy + r*math.Sin(startAngle)
	x2 := cx + r*math.Cos(endAngle)
	y2 := cy + r*math.Sin(endAngle)
	largeArc := 0
	if endAngle-startAngle > math.Pi {
		largeArc = 1
	}
	d := fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 %d,1 %.2f,%.2f Z",
		cx, cy, x1, y1, r, r, largeArc, x2, y2)
	return &path{
		D:     d,
		Style: fmt.Sprintf("fill:%s;stroke:white;stroke-width:2", color),
	}
}

func colorFor(i int) string {
	return defaultColors[i%len(defaultColors)]
}
