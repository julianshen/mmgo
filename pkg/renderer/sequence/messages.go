package sequence

import (
	"fmt"

	"github.com/julianshen/mmgo/pkg/diagram"
)

const (
	defaultActivationW = 10.0
	selfLoopW          = 30.0
	selfLoopH          = 20.0
)

type messageRenderer struct {
	lay       seqLayout
	th        Theme
	fontSize  float64
	pIndex    map[string]int
	curY      float64
	msgNum    int
	autoNum   bool
	actStack  map[string][]float64
	actElems  []any
}

func newMessageRenderer(d *diagram.SequenceDiagram, lay seqLayout, th Theme, fontSize float64) *messageRenderer {
	pix := make(map[string]int, len(d.Participants))
	for i, p := range d.Participants {
		pix[p.ID] = i
	}
	return &messageRenderer{
		lay:      lay,
		th:       th,
		fontSize: fontSize,
		pIndex:   pix,
		curY:     lay.bodyStartY + defaultRowHeight/2,
		autoNum:  d.AutoNumber,
		actStack: make(map[string][]float64),
	}
}

func (mr *messageRenderer) renderItems(items []diagram.SequenceItem) []any {
	var elems []any
	for _, item := range items {
		switch {
		case item.Message != nil:
			elems = append(elems, mr.renderMessage(*item.Message)...)
			mr.curY += defaultRowHeight
		case item.Note != nil:
			mr.curY += defaultRowHeight
		case item.Block != nil:
			mr.curY += defaultRowHeight / 2
			elems = append(elems, mr.renderItems(item.Block.Items)...)
			for _, br := range item.Block.Branches {
				mr.curY += defaultRowHeight / 2
				elems = append(elems, mr.renderItems(br.Items)...)
			}
			mr.curY += defaultRowHeight / 2
		}
	}
	return elems
}

func (mr *messageRenderer) renderMessage(m diagram.Message) []any {
	fromIdx, fromOK := mr.pIndex[m.From]
	toIdx, toOK := mr.pIndex[m.To]
	if !fromOK || !toOK {
		return nil
	}

	mr.handleLifeline(m)

	fromX := mr.lay.participantX[fromIdx]
	toX := mr.lay.participantX[toIdx]
	y := mr.curY

	mr.msgNum++
	var elems []any

	if fromIdx == toIdx {
		elems = append(elems, mr.renderSelfMessage(fromX, y, m)...)
	} else {
		elems = append(elems, mr.renderStraightMessage(fromX, toX, y, m)...)
	}

	if mr.autoNum {
		midX := (fromX + toX) / 2
		if fromIdx == toIdx {
			midX = fromX + selfLoopW/2
		}
		elems = append(elems, &text{
			X: svgFloat(midX), Y: svgFloat(y - 18),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", mr.th.MessageText, mr.fontSize-2),
			Content: fmt.Sprintf("%d", mr.msgNum),
		})
	}

	return elems
}

func (mr *messageRenderer) renderStraightMessage(fromX, toX, y float64, m diagram.Message) []any {
	style := messageLineStyle(mr.th, m.ArrowType)
	mid := (fromX + toX) / 2

	var elems []any
	l := &line{
		X1: svgFloat(fromX), Y1: svgFloat(y),
		X2: svgFloat(toX), Y2: svgFloat(y),
		Style: style,
	}
	if hasArrowHead(m.ArrowType) {
		l.MarkerEnd = fmt.Sprintf("url(#%s)", arrowMarkerID(m.ArrowType))
	}
	elems = append(elems, l)

	if m.Label != "" {
		elems = append(elems, &text{
			X: svgFloat(mid), Y: svgFloat(y - 6),
			Anchor: "middle", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", mr.th.MessageText, mr.fontSize),
			Content: m.Label,
		})
	}
	return elems
}

func (mr *messageRenderer) renderSelfMessage(x, y float64, m diagram.Message) []any {
	style := messageLineStyle(mr.th, m.ArrowType)
	p := &path{
		D: fmt.Sprintf("M%.2f,%.2f h%.2f v%.2f h%.2f",
			x, y, selfLoopW, selfLoopH, -selfLoopW),
		Style: style,
	}
	if hasArrowHead(m.ArrowType) {
		p.MarkerEnd = fmt.Sprintf("url(#%s)", arrowMarkerID(m.ArrowType))
	}

	var elems []any
	elems = append(elems, p)
	if m.Label != "" {
		elems = append(elems, &text{
			X: svgFloat(x + selfLoopW + 4), Y: svgFloat(y + selfLoopH/2),
			Anchor: "start", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", mr.th.MessageText, mr.fontSize),
			Content: m.Label,
		})
	}
	return elems
}

func (mr *messageRenderer) handleLifeline(m diagram.Message) {
	switch m.Lifeline {
	case diagram.LifelineEffectActivate:
		mr.actStack[m.To] = append(mr.actStack[m.To], mr.curY)
	case diagram.LifelineEffectDeactivate:
		stack := mr.actStack[m.To]
		if len(stack) > 0 {
			startY := stack[len(stack)-1]
			mr.actStack[m.To] = stack[:len(stack)-1]
			idx := mr.pIndex[m.To]
			x := mr.lay.participantX[idx]
			mr.actElems = append(mr.actElems, &rect{
				X: svgFloat(x - defaultActivationW/2), Y: svgFloat(startY),
				Width: svgFloat(defaultActivationW), Height: svgFloat(mr.curY - startY),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
					mr.th.ParticipantFill, mr.th.ParticipantStroke, defaultStrokeWidth),
			})
		}
	}
}

func (mr *messageRenderer) flushActivations() []any {
	var elems []any
	for id, stack := range mr.actStack {
		for _, startY := range stack {
			idx := mr.pIndex[id]
			x := mr.lay.participantX[idx]
			elems = append(elems, &rect{
				X: svgFloat(x - defaultActivationW/2), Y: svgFloat(startY),
				Width: svgFloat(defaultActivationW), Height: svgFloat(mr.curY - startY),
				Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
					mr.th.ParticipantFill, mr.th.ParticipantStroke, defaultStrokeWidth),
			})
		}
	}
	elems = append(elems, mr.actElems...)
	return elems
}

func messageLineStyle(th Theme, at diagram.ArrowType) string {
	base := fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", th.MessageStroke, defaultStrokeWidth)
	switch at {
	case diagram.ArrowTypeDashed, diagram.ArrowTypeDashedNoHead,
		diagram.ArrowTypeDashedCross, diagram.ArrowTypeDashedOpen:
		return base + ";stroke-dasharray:5,5"
	default:
		return base
	}
}

func hasArrowHead(at diagram.ArrowType) bool {
	switch at {
	case diagram.ArrowTypeSolidNoHead, diagram.ArrowTypeDashedNoHead:
		return false
	default:
		return true
	}
}

func arrowMarkerID(at diagram.ArrowType) string {
	return fmt.Sprintf("seq-arrow-%s", at.String())
}

func buildSequenceMarkers(th Theme) []marker {
	stroke := th.MessageStroke
	sw := defaultStrokeWidth

	return []marker{
		{
			ID: arrowMarkerID(diagram.ArrowTypeSolid), ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", stroke)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeDashed), ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polygon{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("fill:%s", stroke)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeSolidCross), ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polyline{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeDashedCross), ViewBox: "0 0 10 10",
			RefX: 9, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polyline{Points: "0,0 10,5 0,10", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeSolidOpen), ViewBox: "0 0 10 10",
			RefX: 10, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polyline{Points: "0,1 10,5 0,9", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeDashedOpen), ViewBox: "0 0 10 10",
			RefX: 10, RefY: 5, Width: 8, Height: 8, Orient: "auto",
			Children: []any{&polyline{Points: "0,1 10,5 0,9", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
		},
	}
}
