package sequence

import (
	"encoding/xml"
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func Render(d *diagram.SequenceDiagram, opts *Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("sequence render: diagram is nil")
	}

	pad := resolvePadding(opts)
	th := resolveTheme(opts)
	fontSize := resolveFontSize(opts)

	lay := computeLayout(d, fontSize, pad)

	mr := newMessageRenderer(d, lay, th, fontSize)
	msgElems := mr.renderItems(d.Items, true)

	var children []any

	// <title>/<desc> must be the first children of <svg> for assistive
	// tech to expose them as the document's accessible name/description.
	if d.AccTitle != "" {
		children = append(children, &title{Content: d.AccTitle})
	}
	if d.AccDescr != "" {
		children = append(children, &desc{Content: d.AccDescr})
	}
	children = append(children, &defs{Markers: buildSequenceMarkers(th)})
	children = append(children, &rect{
		X: 0, Y: 0,
		Width: svgFloat(lay.width), Height: svgFloat(lay.height),
		Style: fmt.Sprintf("fill:%s;stroke:none", th.Background),
	})

	if d.Title != "" {
		children = append(children, renderTitle(d.Title, lay, th, fontSize)...)
	}
	children = append(children, renderBoxes(d, lay, th, fontSize)...)
	children = append(children, renderLifelines(d, lay, th, mr.createY, mr.destroyY)...)
	children = append(children, mr.flushActivations()...)
	children = append(children, msgElems...)
	children = append(children, renderParticipants(d, lay, th, fontSize, mr.createY, mr.destroyY)...)

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
	participantX  []float64
	participantW  []float64
	participantIx map[string]int
	topY          float64
	bodyStartY    float64
	bodyEndY      float64
	bottomY       float64
	width         float64
	height        float64
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
		widths[i] = textmeasure.EstimateWidth(p.Label(), fontSize) + 2*defaultBoxPadX
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
	if d.Title != "" {
		topY += titleHeight(fontSize)
	}
	bodyStart := topY + maxHeaderH + 10
	rows := countRows(d)
	bodyEnd := bodyStart + float64(rows)*defaultRowHeight
	if rows == 0 {
		bodyEnd = bodyStart + defaultRowHeight
	}

	lastHalfW := widths[n-1] / 2
	totalW := xs[n-1] + lastHalfW + pad
	// Notes anchored "right of" the last participant (and "left of"
	// the first) extend past the participant boxes; reserve room so
	// the note rect doesn't clip at the viewBox edge.
	pIndex := make(map[string]int, n)
	for i, p := range d.Participants {
		pIndex[p.ID] = i
	}
	leftBleed, rightBleed := noteBleed(d.Items, pIndex, n)
	if extra := leftBleed - pad; extra > 0 {
		// Shift everything right so left-side notes fit.
		for i := range xs {
			xs[i] += extra
		}
		totalW += extra
	}
	if extra := rightBleed - pad; extra > 0 {
		totalW += extra
	}
	// Mermaid renders participant/actor boxes at both ends of every
	// lifeline. bottomGap separates the lifelines from the bottom row.
	const bottomGap = 10.0
	bottomY := bodyEnd + bottomGap
	totalH := bottomY + maxHeaderH + pad

	return seqLayout{
		participantX:  xs,
		participantW:  widths,
		participantIx: pIndex,
		topY:          topY,
		bodyStartY:    bodyStart,
		bodyEndY:      bodyEnd,
		bottomY:       bottomY,
		width:         totalW,
		height:        totalH,
	}
}

func actorHeight(fontSize float64) float64 {
	return 2*defaultActorHeadR + defaultActorBodyH + fontSize + 2
}

func titleHeight(fontSize float64) float64 {
	return fontSize + 12
}

func renderTitle(title string, lay seqLayout, th Theme, fontSize float64) []any {
	return []any{&text{
		X: svgFloat(lay.width / 2), Y: svgFloat(titleHeight(fontSize) / 2),
		Anchor: "middle", Dominant: "central",
		Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.MessageText, fontSize+2),
		Content: title,
	}}
}

// noteBleed returns the pixel extent past the leftmost and rightmost
// participants needed to fit any "Note left of" / "Note right of"
// items. Mirrors the geometry in messageRenderer.renderNote.
func noteBleed(items []diagram.SequenceItem, pIndex map[string]int, n int) (left, right float64) {
	const noteHalfW = 60.0 // half of noteW
	const noteOff = 10.0   // matches noteOffset
	for _, item := range items {
		switch {
		case item.Note != nil && len(item.Note.Participants) > 0:
			idx, ok := pIndex[item.Note.Participants[0]]
			if !ok {
				continue
			}
			switch item.Note.Position {
			case diagram.NotePositionLeft:
				if idx == 0 {
					if w := noteOff + 2*noteHalfW; w > left {
						left = w
					}
				}
			case diagram.NotePositionRight:
				if idx == n-1 {
					if w := noteOff + 2*noteHalfW; w > right {
						right = w
					}
				}
			}
		case item.Block != nil:
			l, r := noteBleed(item.Block.Items, pIndex, n)
			if l > left {
				left = l
			}
			if r > right {
				right = r
			}
			for _, br := range item.Block.Branches {
				bl, br_ := noteBleed(br.Items, pIndex, n)
				if bl > left {
					left = bl
				}
				if br_ > right {
					right = br_
				}
			}
		}
	}
	return
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
		case item.Destroy != nil:
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

func renderParticipants(d *diagram.SequenceDiagram, lay seqLayout, th Theme, fontSize float64, createY, destroyY map[string]float64) []any {
	var elems []any
	for i, p := range d.Participants {
		x := lay.participantX[i]
		w := lay.participantW[i]
		label := p.Label()
		_, isCreated := createY[p.ID]
		_, isDestroyed := destroyY[p.ID]
		if !isCreated {
			elems = append(elems, drawParticipant(p.Kind, x, lay.topY, w, label, th, fontSize)...)
		}
		if !isDestroyed {
			elems = append(elems, drawParticipant(p.Kind, x, lay.bottomY, w, label, th, fontSize)...)
		}
	}
	return elems
}

func drawParticipant(kind diagram.ParticipantKind, cx, topY, w float64, label string, th Theme, fontSize float64) []any {
	if kind == diagram.ParticipantKindActor {
		return renderActor(cx, topY, label, th, fontSize)
	}
	return renderParticipantBox(cx, topY, w, label, th, fontSize)
}

func renderParticipantBox(cx, topY, w float64, label string, th Theme, fontSize float64) []any {
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

func renderLifelines(d *diagram.SequenceDiagram, lay seqLayout, th Theme, createY, destroyY map[string]float64) []any {
	var elems []any
	for i, p := range d.Participants {
		x := lay.participantX[i]
		startY := lay.bodyStartY
		endY := lay.bodyEndY
		if y, ok := createY[p.ID]; ok {
			startY = y - defaultRowHeight/2
		}
		if y, ok := destroyY[p.ID]; ok {
			endY = y
		}
		if startY < endY {
			elems = append(elems, &line{
				X1: svgFloat(x), Y1: svgFloat(startY),
				X2: svgFloat(x), Y2: svgFloat(endY),
				Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5", th.LifelineStroke, defaultStrokeWidth),
			})
		}
	}
	return elems
}

func renderBoxes(d *diagram.SequenceDiagram, lay seqLayout, th Theme, fontSize float64) []any {
	if len(d.Boxes) == 0 || len(d.Participants) == 0 {
		return nil
	}
	pIndex := lay.participantIx

	var elems []any
	for _, bx := range d.Boxes {
		if len(bx.Members) == 0 {
			continue
		}
		leftIdx, ok := pIndex[bx.Members[0]]
		if !ok {
			continue
		}
		rightIdx := leftIdx
		for _, m := range bx.Members[1:] {
			if idx, ok := pIndex[m]; ok {
				if idx < leftIdx {
					leftIdx = idx
				}
				if idx > rightIdx {
					rightIdx = idx
				}
			}
		}

		const boxPad = 10.0
		x := lay.participantX[leftIdx] - lay.participantW[leftIdx]/2 - boxPad
		right := lay.participantX[rightIdx] + lay.participantW[rightIdx]/2 + boxPad
		w := right - x
		h := lay.bodyEndY - lay.topY + boxPad
		y := lay.topY - boxPad/2

		fill := th.ParticipantFill
		if bx.Fill != "" {
			fill = bx.Fill
		}
		style := fmt.Sprintf("fill:%s;fill-opacity:0.15;stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5",
			fill, th.ParticipantStroke, defaultStrokeWidth)

		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(y),
			Width: svgFloat(w), Height: svgFloat(h),
			RX: 3, RY: 3,
			Style: style,
		})

		if bx.Label != "" {
			elems = append(elems, &text{
				X: svgFloat(x + boxPad), Y: svgFloat(y + fontSize + 2),
				Anchor: "start", Dominant: "auto",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.ParticipantText, fontSize-2),
				Content: bx.Label,
			})
		}
	}
	return elems
}
