package pie

import (
	"encoding/xml"
	"fmt"
	"math"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	DefaultFontSize = 14.0
	defaultRadius   = 120.0
	defaultPadding  = 30.0
	defaultLegendW  = 150.0
	defaultLegendH  = 20.0
	legendGap       = 5.0
	legendSwatchW   = 14.0

	// smallSliceThreshold: slices below this fraction of the pie get
	// an outside leader-line label rather than centered text, since
	// thin sectors don't have room for inside labels without
	// colliding with adjacent thin slices.
	smallSliceThreshold = 0.06
)

type Options struct {
	FontSize float64
	Theme    Theme
}

func Render(d *diagram.PieDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("pie render: diagram is nil")
	}

	fontSize := DefaultFontSize
	if opts != nil && opts.FontSize > 0 {
		fontSize = opts.FontSize
	}
	th := resolveTheme(opts)

	ruler, err := textmeasure.NewDefaultRuler()
	if err != nil {
		return nil, fmt.Errorf("pie render: text measurer: %w", err)
	}
	defer func() { _ = ruler.Close() }()

	total := 0.0
	for _, s := range d.Slices {
		total += s.Value
	}

	pad := defaultPadding
	// Reserve a side gutter wide enough to fit the worst-case outside
	// label for any small slice. Without this, labels at angles near
	// 180° (anchor=end) clip past x=0, and labels near 0° (anchor=start)
	// can collide with the legend.
	outsideGutter := 0.0
	if total > 0 && len(d.Slices) > 1 {
		for _, s := range d.Slices {
			if s.Value/total < smallSliceThreshold {
				w, _ := ruler.Measure(formatSliceLabel(s, total, d.ShowData), fontSize-1)
				if w > outsideGutter {
					outsideGutter = w
				}
			}
		}
		if outsideGutter > 0 {
			outsideGutter += 6 // tiny breathing room
		}
	}
	cx := pad + outsideGutter + defaultRadius
	cy := pad + defaultRadius
	if d.Title != "" {
		cy += fontSize + 10
	}

	legendX := cx + defaultRadius + outsideGutter + pad
	legendY := cy - defaultRadius

	viewW := legendX + defaultLegendW + pad
	viewH := cy + defaultRadius + pad

	var children []any
	children = append(children, &rect{
		X: 0, Y: 0,
		Width: svgFloat(viewW), Height: svgFloat(viewH),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		children = append(children, &text{
			X: svgFloat(cx), Y: svgFloat(pad + fontSize),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.TitleText, fontSize+2),
			Content: d.Title,
		})
	}

	if len(d.Slices) == 1 {
		s := d.Slices[0]
		children = append(children, &circle{
			CX: svgFloat(cx), CY: svgFloat(cy), R: svgFloat(defaultRadius),
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", colorFor(th, 0), th.Background),
		})
		label := "100.0%"
		if d.ShowData {
			label = fmt.Sprintf("100.0%% (%.0f)", s.Value)
		}
		children = append(children, &text{
			X: svgFloat(cx), Y: svgFloat(cy),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.InsideText, fontSize-1),
			Content: label,
		})
	} else if total > 0 {
		startAngle := -math.Pi / 2
		for i, s := range d.Slices {
			frac := s.Value / total
			sweep := frac * 2 * math.Pi
			endAngle := startAngle + sweep
			children = append(children, arcPath(cx, cy, defaultRadius, startAngle, endAngle, colorFor(th, i), th.Background))
			startAngle = endAngle
		}
		startAngle = -math.Pi / 2
		for _, s := range d.Slices {
			frac := s.Value / total
			sweep := frac * 2 * math.Pi
			midAngle := startAngle + sweep/2
			cosA, sinA := math.Cos(midAngle), math.Sin(midAngle)
			label := formatSliceLabel(s, total, d.ShowData)
			if frac >= smallSliceThreshold {
				lx := cx + defaultRadius*0.65*cosA
				ly := cy + defaultRadius*0.65*sinA
				children = append(children, &text{
					X: svgFloat(lx), Y: svgFloat(ly),
					Anchor: "middle", Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.InsideText, fontSize-1),
					Content: label,
				})
			} else {
				inX := cx + defaultRadius*0.95*cosA
				inY := cy + defaultRadius*0.95*sinA
				outX := cx + defaultRadius*1.12*cosA
				outY := cy + defaultRadius*1.12*sinA
				children = append(children, &line{
					X1: svgFloat(inX), Y1: svgFloat(inY),
					X2: svgFloat(outX), Y2: svgFloat(outY),
					Style: fmt.Sprintf("stroke:%s;stroke-width:1", th.LeaderStroke),
				})
				anchor := "start"
				if cosA < 0 {
					anchor = "end"
				}
				children = append(children, &text{
					X: svgFloat(outX), Y: svgFloat(outY),
					Anchor: anchor, Dominant: "central",
					Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.OutsideText, fontSize-1),
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
			Style: fmt.Sprintf("fill:%s", colorFor(th, i)),
		})
		children = append(children, &text{
			X: svgFloat(legendX + legendSwatchW + 6), Y: svgFloat(y + legendSwatchW/2),
			Anchor: "start", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", th.LegendText, fontSize),
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

func arcPath(cx, cy, r, startAngle, endAngle float64, color, stroke string) *path {
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
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", color, stroke),
	}
}

func colorFor(th Theme, i int) string {
	return th.SliceColors[i%len(th.SliceColors)]
}

// formatSliceLabel returns the percentage (and optional count) text
// shown on a pie slice. Used both at render time and during the
// pre-pass that sizes the outside-label gutter.
func formatSliceLabel(s diagram.Slice, total float64, showData bool) string {
	pct := fmt.Sprintf("%.1f%%", (s.Value/total)*100)
	if showData {
		return fmt.Sprintf("%s (%.0f)", pct, s.Value)
	}
	return pct
}

