package sequence

import (
	"fmt"
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/textmeasure"
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
			elems = append(elems, mr.renderNote(*item.Note)...)
			mr.curY += defaultRowHeight
		case item.Block != nil:
			elems = append(elems, mr.renderBlock(*item.Block)...)
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

const (
	noteW      = 120.0
	noteH      = 30.0
	notePad    = 8.0
	noteOffset = 10.0
	blockPad   = 15.0
)

func (mr *messageRenderer) renderNote(n diagram.Note) []any {
	if len(n.Participants) == 0 {
		return nil
	}
	idx0, ok := mr.pIndex[n.Participants[0]]
	if !ok {
		return nil
	}
	y := mr.curY
	x0 := mr.lay.participantX[idx0]

	var cx float64
	w := noteW
	switch n.Position {
	case diagram.NotePositionLeft:
		cx = x0 - noteOffset - w/2
	case diagram.NotePositionRight:
		cx = x0 + noteOffset + w/2
	case diagram.NotePositionOver:
		if len(n.Participants) == 2 {
			idx1, ok2 := mr.pIndex[n.Participants[1]]
			if !ok2 {
				return nil
			}
			x1 := mr.lay.participantX[idx1]
			cx = (x0 + x1) / 2
			w = math.Abs(x1-x0) + 2*notePad
		} else {
			cx = x0
		}
	}

	rx := cx - w/2
	return []any{
		&rect{
			X: svgFloat(rx), Y: svgFloat(y - noteH/2),
			Width: svgFloat(w), Height: svgFloat(noteH),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f", mr.th.NoteFill, mr.th.MessageStroke, defaultStrokeWidth),
		},
		&text{
			X: svgFloat(cx), Y: svgFloat(y),
			Anchor: "middle", Dominant: "central",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", mr.th.MessageText, mr.fontSize),
			Content: n.Text,
		},
	}
}

func (mr *messageRenderer) renderBlock(b diagram.Block) []any {
	startY := mr.curY
	mr.curY += defaultRowHeight / 2

	var elems []any
	elems = append(elems, mr.renderItems(b.Items)...)

	var branchYs []float64
	for _, br := range b.Branches {
		branchYs = append(branchYs, mr.curY)
		mr.curY += defaultRowHeight / 2
		elems = append(elems, mr.renderItems(br.Items)...)
	}
	mr.curY += defaultRowHeight / 2
	endY := mr.curY

	x := blockPad
	if len(mr.lay.participantX) > 0 {
		x = mr.lay.participantX[0] - defaultParticipantGap/3
	}
	w := mr.lay.width - 2*x
	if w < 0 {
		w = mr.lay.width - 2*blockPad
		x = blockPad
	}

	elems = append(elems, &rect{
		X: svgFloat(x), Y: svgFloat(startY - defaultRowHeight/4),
		Width: svgFloat(w), Height: svgFloat(endY - startY + defaultRowHeight/4),
		RX: 3, RY: 3,
		Style: fmt.Sprintf("fill:none;stroke:%s;stroke-width:%.1f", mr.th.MessageStroke, defaultStrokeWidth),
	})

	kindLabel := b.Kind.String()
	kindLabelW := textmeasure.EstimateWidth(kindLabel, mr.fontSize)
	elems = append(elems, &rect{
		X: svgFloat(x), Y: svgFloat(startY - defaultRowHeight/4),
		Width: svgFloat(kindLabelW + 2*notePad),
		Height: svgFloat(20),
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
			mr.th.ParticipantFill, mr.th.MessageStroke, defaultStrokeWidth),
	})
	elems = append(elems, &text{
		X: svgFloat(x + notePad), Y: svgFloat(startY - defaultRowHeight/4 + 14),
		Anchor: "start", Dominant: "auto",
		Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", mr.th.MessageText, mr.fontSize-1),
		Content: kindLabel,
	})

	if b.Label != "" {
		elems = append(elems, &text{
			X: svgFloat(x + kindLabelW + 3*notePad),
			Y: svgFloat(startY - defaultRowHeight/4 + 14),
			Anchor: "start", Dominant: "auto",
			Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", mr.th.MessageText, mr.fontSize-1),
			Content: "[" + b.Label + "]",
		})
	}

	for i, brY := range branchYs {
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(brY),
			X2: svgFloat(x + w), Y2: svgFloat(brY),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5", mr.th.MessageStroke, defaultStrokeWidth),
		})
		if i < len(b.Branches) && b.Branches[i].Label != "" {
			elems = append(elems, &text{
				X: svgFloat(x + notePad), Y: svgFloat(brY + 14),
				Anchor: "start", Dominant: "auto",
				Style:   fmt.Sprintf("fill:%s;font-size:%.0fpx", mr.th.MessageText, mr.fontSize-1),
				Content: "[" + b.Branches[i].Label + "]",
			})
		}
	}

	return elems
}

func (mr *messageRenderer) handleLifeline(m diagram.Message) {
	switch m.Lifeline {
	case diagram.LifelineEffectActivate:
		mr.actStack[m.To] = append(mr.actStack[m.To], mr.curY)
	case diagram.LifelineEffectDeactivate:
		// Mermaid spec: `-` suffix deactivates the SOURCE, not target.
		stack := mr.actStack[m.From]
		if len(stack) > 0 {
			startY := stack[len(stack)-1]
			mr.actStack[m.From] = stack[:len(stack)-1]
			mr.actElems = append(mr.actElems, mr.activationRect(m.From, startY, mr.curY))
		}
	}
}

func (mr *messageRenderer) flushActivations() []any {
	// Sort unclosed activations by participant index for deterministic output.
	ids := make([]string, 0, len(mr.actStack))
	for id, stack := range mr.actStack {
		if len(stack) > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		return mr.pIndex[ids[i]] < mr.pIndex[ids[j]]
	})
	var elems []any
	for _, id := range ids {
		for _, startY := range mr.actStack[id] {
			elems = append(elems, mr.activationRect(id, startY, mr.curY))
		}
	}
	elems = append(elems, mr.actElems...)
	return elems
}

func (mr *messageRenderer) activationRect(id string, startY, endY float64) *rect {
	idx := mr.pIndex[id]
	x := mr.lay.participantX[idx]
	return &rect{
		X: svgFloat(x - defaultActivationW/2), Y: svgFloat(startY),
		Width: svgFloat(defaultActivationW), Height: svgFloat(endY - startY),
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
			mr.th.ParticipantFill, mr.th.ParticipantStroke, defaultStrokeWidth),
	}
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
