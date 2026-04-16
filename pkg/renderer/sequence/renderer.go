package sequence

import (
	"encoding/xml"
	"fmt"
	"unicode/utf8"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func Render(d *diagram.SequenceDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("sequence render: diagram is nil")
	}

	pad := resolvePadding(opts)
	th := DefaultTheme()
	fontSize := resolveFontSize(opts)

	lay := computeLayout(d, fontSize, pad)

	var children []any

	children = append(children, &rect{
		X: 0, Y: 0,
		Width: svgFloat(lay.width), Height: svgFloat(lay.height),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	children = append(children, renderLifelines(d, lay, th)...)
	children = append(children, renderParticipants(d, lay, th, fontSize)...)

	svg := svgDoc{
		XMLNS:    "http://www.w3.org/2000/svg",
		ViewBox:  fmt.Sprintf("0 0 %.2f %.2f", lay.width, lay.height),
		Children: children,
	}

	svgBytes, err := xml.Marshal(svg)
	if err != nil {
		return nil, fmt.Errorf("sequence render: %w", err)
	}
	xmlDecl := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	return append(xmlDecl, svgBytes...), nil
}

type seqLayout struct {
	participantX []float64
	topY         float64
	bodyStartY   float64
	bodyEndY     float64
	width        float64
	height       float64
}

func computeLayout(d *diagram.SequenceDiagram, fontSize, pad float64) seqLayout {
	n := len(d.Participants)
	if n == 0 {
		return seqLayout{
			width:  2 * pad,
			height: 2 * pad,
		}
	}

	// Compute per-participant widths so spacing adapts to labels.
	widths := make([]float64, n)
	maxHeaderH := defaultBoxHeight
	for i, p := range d.Participants {
		label := p.Alias
		if label == "" {
			label = p.ID
		}
		widths[i] = estimateTextWidth(label, fontSize) + 2*defaultBoxPadX
		if widths[i] < defaultParticipantGap*0.6 {
			widths[i] = defaultParticipantGap * 0.6
		}
		if p.Kind == diagram.ParticipantKindActor {
			h := actorHeight(fontSize)
			if h > maxHeaderH {
				maxHeaderH = h
			}
		}
	}

	xs := make([]float64, n)
	xs[0] = pad + widths[0]/2
	for i := 1; i < n; i++ {
		gap := (widths[i-1] + widths[i]) / 2
		if gap < defaultParticipantGap {
			gap = defaultParticipantGap
		}
		xs[i] = xs[i-1] + gap
	}

	topY := pad
	bodyStart := topY + maxHeaderH + 10
	rows := countRows(d)
	bodyEnd := bodyStart + float64(rows)*defaultRowHeight
	if rows == 0 {
		bodyEnd = bodyStart + defaultRowHeight
	}

	lastHalfW := widths[n-1] / 2
	totalW := xs[n-1] + lastHalfW + pad
	totalH := bodyEnd + pad

	return seqLayout{
		participantX: xs,
		topY:         topY,
		bodyStartY:   bodyStart,
		bodyEndY:     bodyEnd,
		width:  totalW,
		height: totalH,
	}
}

func actorHeight(fontSize float64) float64 {
	return 2*defaultActorHeadR + defaultActorBodyH + fontSize + 2
}

func countRows(d *diagram.SequenceDiagram) int {
	return countItemRows(d.Items)
}

func countItemRows(items []diagram.SequenceItem) int {
	count := 0
	for _, item := range items {
		switch {
		case item.Message != nil:
			count++
		case item.Note != nil:
			count++
		case item.Block != nil:
			count += 1 + countBlockRows(item.Block)
		}
	}
	return count
}

func countBlockRows(b *diagram.Block) int {
	count := countItemRows(b.Items)
	for _, br := range b.Branches {
		count += 1 + countItemRows(br.Items)
	}
	return count
}

func renderParticipants(d *diagram.SequenceDiagram, lay seqLayout, th Theme, fontSize float64) []any {
	var elems []any
	for i, p := range d.Participants {
		x := lay.participantX[i]
		label := p.Alias
		if label == "" {
			label = p.ID
		}

		if p.Kind == diagram.ParticipantKindActor {
			elems = append(elems, renderActor(x, lay.topY, label, th, fontSize)...)
		} else {
			elems = append(elems, renderParticipantBox(x, lay.topY, label, th, fontSize)...)
		}
	}
	return elems
}

func renderParticipantBox(cx, topY float64, label string, th Theme, fontSize float64) []any {
	w := estimateTextWidth(label, fontSize) + 2*defaultBoxPadX
	h := defaultBoxHeight
	rx := cx - w/2
	ry := topY
	return []any{
		&rect{
			X: svgFloat(rx), Y: svgFloat(ry),
			Width: svgFloat(w), Height: svgFloat(h),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f", th.ParticipantFill, th.ParticipantStroke, defaultStrokeWidth),
		},
		&text{
			X: svgFloat(cx), Y: svgFloat(ry + h/2),
			Anchor: "middle", Dominant: "central",
			Style:   labelStyle(th, fontSize),
			Content: label,
		},
	}
}

func labelStyle(th Theme, fontSize float64) string {
	return fmt.Sprintf("fill:%s;font-size:%.0fpx", th.ParticipantText, fontSize)
}

func renderActor(cx, topY float64, label string, th Theme, fontSize float64) []any {
	headCY := topY + defaultActorHeadR
	bodyTop := headCY + defaultActorHeadR
	bodyBot := bodyTop + defaultActorBodyH
	strokeStyle := fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", th.ParticipantStroke, defaultStrokeWidth)

	return []any{
		&circle{
			CX: svgFloat(cx), CY: svgFloat(headCY), R: svgFloat(defaultActorHeadR),
			Style: strokeStyle,
		},
		&line{X1: svgFloat(cx), Y1: svgFloat(bodyTop), X2: svgFloat(cx), Y2: svgFloat(bodyBot - 10), Style: strokeStyle},
		&line{X1: svgFloat(cx - 15), Y1: svgFloat(bodyTop + 8), X2: svgFloat(cx + 15), Y2: svgFloat(bodyTop + 8), Style: strokeStyle},
		&line{X1: svgFloat(cx), Y1: svgFloat(bodyBot - 10), X2: svgFloat(cx - 10), Y2: svgFloat(bodyBot), Style: strokeStyle},
		&line{X1: svgFloat(cx), Y1: svgFloat(bodyBot - 10), X2: svgFloat(cx + 10), Y2: svgFloat(bodyBot), Style: strokeStyle},
		&text{
			X: svgFloat(cx), Y: svgFloat(bodyBot + fontSize + 2),
			Anchor: "middle", Dominant: "auto",
			Style:   labelStyle(th, fontSize),
			Content: label,
		},
	}
}

func renderLifelines(d *diagram.SequenceDiagram, lay seqLayout, th Theme) []any {
	var elems []any
	for i := range d.Participants {
		x := lay.participantX[i]
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(lay.bodyStartY),
			X2: svgFloat(x), Y2: svgFloat(lay.bodyEndY),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5", th.LifelineStroke, defaultStrokeWidth),
		})
	}
	return elems
}

func estimateTextWidth(s string, fontSize float64) float64 {
	return float64(utf8.RuneCountInString(s)) * fontSize * 0.6
}
