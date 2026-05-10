package sequence

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	defaultActivationW = 10.0
	selfLoopW          = 50.0
	selfLoopH          = 40.0
)

type messageRenderer struct {
	lay                   seqLayout
	th                    Theme
	fontSize              float64
	curY                  float64
	msgNum                int
	autoNum               diagram.AutoNumber
	actStack              map[string][]float64
	actElems              []any
	participants          []diagram.Participant
	created               map[string]bool
	createdAtIdx          map[int]int
	createY               map[string]float64
	destroyY              map[string]float64
	autoNumStyles         *autoNumStyles
	msgTextStyle          string
	msgTextSmallStyle     string
	msgTextSmallBoldStyle string
	msgLineStyleSolid     string
	msgLineStyleDashed    string
}

type autoNumStyles struct {
	circle string
	text   string
}

func newMessageRenderer(d *diagram.SequenceDiagram, lay seqLayout, th Theme, fontSize float64) *messageRenderer {
	createdAtIdx := make(map[int]int)
	for i, p := range d.Participants {
		if p.CreatedAtItem >= 0 {
			createdAtIdx[p.CreatedAtItem] = i
		}
	}
	mr := &messageRenderer{
		lay:                   lay,
		th:                    th,
		fontSize:              fontSize,
		curY:                  lay.bodyStartY + defaultRowHeight/2,
		msgNum:                d.AutoNumber.Start - d.AutoNumber.Step,
		autoNum:               d.AutoNumber,
		actStack:              make(map[string][]float64),
		participants:          d.Participants,
		created:               make(map[string]bool),
		createdAtIdx:          createdAtIdx,
		createY:               make(map[string]float64),
		destroyY:              make(map[string]float64),
		msgTextStyle:          fmt.Sprintf("fill:%s;font-size:%.0fpx", th.MessageText, fontSize),
		msgTextSmallStyle:     fmt.Sprintf("fill:%s;font-size:%.0fpx", th.MessageText, fontSize-1),
		msgTextSmallBoldStyle: fmt.Sprintf("fill:%s;font-size:%.0fpx;font-weight:bold", th.MessageText, fontSize-1),
		msgLineStyleSolid:     fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", th.MessageStroke, defaultStrokeWidth),
		msgLineStyleDashed:    fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none;stroke-dasharray:5,5", th.MessageStroke, defaultStrokeWidth),
	}
	if d.AutoNumber.Enabled {
		mr.autoNumStyles = &autoNumStyles{
			circle: fmt.Sprintf("fill:%s;stroke:none", th.MessageStroke),
			text:   fmt.Sprintf("fill:#fff;font-size:%.0fpx;font-weight:bold", fontSize-2),
		}
	}
	return mr
}

func (mr *messageRenderer) renderItems(items []diagram.SequenceItem, isTopLevel bool, depth int) []any {
	var elems []any
	for i, item := range items {
		if isTopLevel {
			if pi, ok := mr.createdAtIdx[i]; ok {
				p := mr.participants[pi]
				if !mr.created[p.ID] {
					mr.created[p.ID] = true
					mr.createY[p.ID] = mr.curY
					x := mr.lay.participantX[pi]
					elems = append(elems, drawParticipant(p.Kind, x, mr.curY-defaultRowHeight/2+2, mr.lay.participantW[pi], p.Label(), mr.th, mr.fontSize)...)
				}
			}
		}
		switch {
		case item.Destroy != nil:
			mr.destroyY[*item.Destroy] = mr.curY
			elems = append(elems, mr.renderDestroy(*item.Destroy)...)
			mr.curY += defaultRowHeight
		case item.Activation != nil:
			mr.handleStandaloneActivation(*item.Activation)
		case item.Message != nil:
			elems = append(elems, mr.renderMessage(*item.Message)...)
			// Multiline labels render above the message line; reserve
			// extra vertical space so the next row's label doesn't
			// collide with the current label's top lines. Self-messages
			// also need a tall-enough row so the arc (which extends
			// selfLoopH below the anchor y) clears the next row's label.
			advance := defaultRowHeight + extraLinesHeight(item.Message.Label, mr.fontSize)
			if item.Message.From == item.Message.To {
				if delta := selfLoopRowExtra(mr.fontSize); delta > 0 {
					advance += delta
				}
			}
			mr.curY += advance
		case item.Note != nil:
			elems = append(elems, mr.renderNote(*item.Note)...)
			mr.curY += defaultRowHeight + extraLinesHeight(item.Note.Text, mr.fontSize)
		case item.Block != nil:
			elems = append(elems, mr.renderBlock(*item.Block, depth)...)
		}
	}
	return elems
}

func (mr *messageRenderer) renderMessage(m diagram.Message) []any {
	fromIdx, fromOK := mr.lay.participantIx[m.From]
	toIdx, toOK := mr.lay.participantIx[m.To]
	if !fromOK || !toOK {
		return nil
	}

	mr.handleLifeline(m)

	fromX := mr.lay.participantX[fromIdx]
	toX := mr.lay.participantX[toIdx]
	y := mr.curY

	if mr.created[m.To] && mr.createY[m.To] == y {
		halfW := mr.lay.participantW[toIdx] / 2
		if toX > fromX {
			toX -= halfW
		} else {
			toX += halfW
		}
	}

	mr.msgNum += mr.autoNum.Step
	var elems []any

	// When an autonumber badge sits at the source endpoint of a
	// straight (cross-participant) message, start the arrow line at
	// the badge's edge (one radius along the message direction) so
	// the line doesn't visibly pierce the circle — mmdc parity.
	// Self-message loops anchor both ends on the lifeline so the
	// offset would break the geometry; left as-is.
	lineFromX := fromX
	if mr.autoNum.Enabled && fromIdx != toIdx {
		if toX > fromX {
			lineFromX = fromX + autoNumberRadius
		} else {
			lineFromX = fromX - autoNumberRadius
		}
	}

	if fromIdx == toIdx {
		elems = append(elems, mr.renderSelfMessage(fromX, y, m)...)
	} else {
		elems = append(elems, mr.renderStraightMessage(lineFromX, toX, y, m)...)
	}

	if mr.autoNum.Enabled {
		elems = mr.appendAutoNumberBadge(elems, fromX, y, mr.msgNum)
	}

	return elems
}

const autoNumberRadius = 10.0

// White-on-stroke regardless of theme so the digit stays legible —
// using Theme.Background would invert under dark themes and put
// white text on a white circle.
func (mr *messageRenderer) appendAutoNumberBadge(elems []any, srcX, y float64, n int) []any {
	return append(elems,
		&circle{
			CX: svgFloat(srcX), CY: svgFloat(y),
			R:     svgFloat(autoNumberRadius),
			Style: mr.autoNumStyles.circle,
		},
		&text{
			X: svgFloat(srcX), Y: svgFloat(y),
			Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
			Style:   mr.autoNumStyles.text,
			Content: fmt.Sprintf("%d", n),
		},
	)
}

func (mr *messageRenderer) renderStraightMessage(fromX, toX, y float64, m diagram.Message) []any {
	style := mr.messageLineStyle(m.ArrowType)
	mid := (fromX + toX) / 2

	var elems []any
	l := &line{
		X1: svgFloat(fromX), Y1: svgFloat(y),
		X2: svgFloat(toX), Y2: svgFloat(y),
		Style: style,
	}
	if ref := m.ArrowType.MarkerRef(); ref != "" {
		l.MarkerEnd = ref
	}
	elems = append(elems, l)
	if m.ArrowType.IsBidirectional() {
		// The PNG rasterizer (tdewolff/canvas) does not reliably render both
		// marker-start and marker-end on the same line. Emit inline polygon
		// arrowheads at each endpoint so both heads always appear.
		dir := 1.0
		if toX < fromX {
			dir = -1.0
		}
		elems = append(elems, bidirArrowhead(toX, y, dir, mr.th.MessageStroke))
		elems = append(elems, bidirArrowhead(fromX, y, -dir, mr.th.MessageStroke))
	}

	if m.Label != "" {
		elems = append(elems, multilineTextAbove(m.Label, mid, y-6, "middle", mr.msgTextStyle, mr.fontSize)...)
	}
	return elems
}

func (mr *messageRenderer) renderSelfMessage(x, y float64, m diagram.Message) []any {
	style := mr.messageLineStyle(m.ArrowType)
	// Symmetric control points keep the curve's end tangent
	// horizontal so the auto-oriented arrowhead points cleanly back
	// at the lifeline.
	p := &path{
		D: fmt.Sprintf("M%.2f,%.2f C%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			x, y,
			x+selfLoopW, y,
			x+selfLoopW, y+selfLoopH,
			x, y+selfLoopH),
		Style: style,
	}
	if ref := m.ArrowType.MarkerRef(); ref != "" {
		p.MarkerEnd = ref
	}

	var elems []any
	elems = append(elems, p)
	if m.Label != "" {
		// Label sits centered above the arc (mmdc parity). The arc
		// bulges selfLoopW to the right of the lifeline, so the
		// horizontal anchor is the arc midpoint; the vertical anchor
		// is just above the arc's top, matching the y-6 offset used
		// for non-self messages.
		elems = append(elems, multilineTextAbove(m.Label, x+selfLoopW/2, y-6, "middle", mr.msgTextStyle, mr.fontSize)...)
	}
	return elems
}

const (
	noteW       = 120.0
	noteH       = 30.0
	notePad     = 8.0
	noteOffset  = 10.0
	blockPad    = 15.0
	blockIndent = 8.0
)

func (mr *messageRenderer) renderNote(n diagram.Note) []any {
	if len(n.Participants) == 0 {
		return nil
	}
	idx0, ok := mr.lay.participantIx[n.Participants[0]]
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
			idx1, ok2 := mr.lay.participantIx[n.Participants[1]]
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
	out := []any{
		&rect{
			X: svgFloat(rx), Y: svgFloat(y - noteH/2),
			Width: svgFloat(w), Height: svgFloat(noteH),
			RX: 3, RY: 3,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f", mr.th.NoteFill, mr.th.MessageStroke, defaultStrokeWidth),
		},
	}
	out = append(out, multilineText(n.Text, cx, y, "middle", "central", mr.msgTextStyle, mr.fontSize)...)
	return out
}

func (mr *messageRenderer) renderBlock(b diagram.Block, depth int) []any {
	startY := mr.curY

	// Full row of header so the b.Label inside the kind tab clears
	// the first message's above-line label. Rect blocks have no tab
	// or label, so a half-row gap is enough.
	blockHeaderGap := defaultRowHeight
	if b.Kind == diagram.BlockKindRect {
		blockHeaderGap = defaultRowHeight / 2
	}
	mr.curY += blockHeaderGap

	content := mr.renderItems(b.Items, false, depth+1)

	var branchYs []float64
	for _, br := range b.Branches {
		branchYs = append(branchYs, mr.curY)
		// Full row matches the slot countBlockRows reserves so the
		// bracket label clears the next message's above-line label.
		mr.curY += defaultRowHeight
		content = append(content, mr.renderItems(br.Items, false, depth+1)...)
	}
	blockFooterGap := defaultRowHeight / 2
	mr.curY += blockFooterGap
	endY := mr.curY

	indent := float64(depth) * blockIndent
	x := blockPad + indent
	if len(mr.lay.participantX) > 0 {
		x = mr.lay.participantX[0] - defaultParticipantGap/3 + indent
	}
	w := mr.lay.width - 2*x
	if w < 0 {
		// Degenerate layout (lay.width < 2*blockPad). Fall back to
		// the un-indented base so we still emit a non-negative
		// width; visual nesting is lost but the SVG stays valid.
		x = blockPad
		w = mr.lay.width - 2*x
	}

	// Non-rect blocks render with a dashed border for mmdc parity.
	// Rect blocks carry only a fill (no border) since the colored
	// region is the entire visual cue — a dark stroke would read as
	// a flow-control region instead.
	blockStyle := fmt.Sprintf("fill:none;stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5", mr.th.ParticipantStroke, defaultStrokeWidth)
	if b.Kind == diagram.BlockKindRect && b.Fill != "" {
		blockStyle = fillStyleWithOpacity(b.Fill, "none", b.HasAlpha, defaultBlockFillOpacity)
	}

	rectY := startY - defaultRowHeight/4
	rectH := endY - startY + defaultRowHeight/4
	if b.Kind == diagram.BlockKindRect {
		// Span the full row band so the first message's above-line
		// label still falls inside the colored region. rectY starts
		// at startY (= header-gap above the first message line),
		// rectH stops half a row below the last message line.
		rectY = startY
		rectH = math.Max(0, endY-startY-blockFooterGap-defaultRowHeight/2)
	}

	// Rect blocks emit the fill *before* their content so messages
	// draw on top of the colored band. Other block kinds have
	// fill:none and can emit in either order.
	bodyRect := &rect{
		X: svgFloat(x), Y: svgFloat(rectY),
		Width: svgFloat(w), Height: svgFloat(rectH),
		RX: 3, RY: 3,
		Style: blockStyle,
	}
	var elems []any
	if b.Kind == diagram.BlockKindRect {
		elems = append(elems, bodyRect)
		elems = append(elems, content...)
	} else {
		elems = append(elems, content...)
		elems = append(elems, bodyRect)
	}

	if b.Kind != diagram.BlockKindRect {
		kindLabel := b.Kind.String()
		kindLabelW := textmeasure.EstimateWidth(kindLabel, mr.fontSize)
		elems = append(elems, &rect{
			X: svgFloat(x), Y: svgFloat(startY - defaultRowHeight/4),
			Width:  svgFloat(kindLabelW + 2*notePad),
			Height: svgFloat(20),
			RX:     6, RY: 6,
			Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
				mr.th.ParticipantFill, mr.th.ParticipantStroke, defaultStrokeWidth),
		})
		elems = append(elems, &text{
			X: svgFloat(x + notePad), Y: svgFloat(startY - defaultRowHeight/4 + 14),
			Anchor: svgutil.AnchorStart, Dominant: svgutil.BaselineAuto,
			Style:   mr.msgTextSmallBoldStyle,
			Content: kindLabel,
		})

		if b.Label != "" {
			elems = append(elems, &text{
				X:      svgFloat(x + w/2),
				Y:      svgFloat(startY - defaultRowHeight/4 + 14),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineAuto,
				Style:   mr.msgTextSmallStyle,
				Content: "[" + b.Label + "]",
			})
		}
	}

	for i, brY := range branchYs {
		elems = append(elems, &line{
			X1: svgFloat(x), Y1: svgFloat(brY),
			X2: svgFloat(x + w), Y2: svgFloat(brY),
			Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;stroke-dasharray:5,5", mr.th.ParticipantStroke, defaultStrokeWidth),
		})
		if i < len(b.Branches) && b.Branches[i].Label != "" {
			// Roughly equidistant between the divider above (at brY)
			// and the next message's above-line label (at brY+44):
			// central baseline at brY+15 gives ~8 px breathing room
			// on each side without crashing into either neighbour.
			elems = append(elems, &text{
				X: svgFloat(x + w/2), Y: svgFloat(brY + 15),
				Anchor: svgutil.AnchorMiddle, Dominant: svgutil.BaselineCentral,
				Style:   mr.msgTextSmallStyle,
				Content: "[" + b.Branches[i].Label + "]",
			})
		}
	}

	return elems
}

func (mr *messageRenderer) handleStandaloneActivation(a diagram.Activation) {
	if a.Activate {
		mr.actStack[a.Participant] = append(mr.actStack[a.Participant], mr.curY)
		return
	}
	stack := mr.actStack[a.Participant]
	if len(stack) == 0 {
		return
	}
	depth := len(stack) - 1
	startY := stack[depth]
	mr.actStack[a.Participant] = stack[:depth]
	mr.actElems = append(mr.actElems, mr.activationRect(a.Participant, startY, mr.curY, depth))
}

func (mr *messageRenderer) handleLifeline(m diagram.Message) {
	switch m.Lifeline {
	case diagram.LifelineEffectActivate:
		mr.actStack[m.To] = append(mr.actStack[m.To], mr.curY)
	case diagram.LifelineEffectDeactivate:
		stack := mr.actStack[m.From]
		if len(stack) > 0 {
			depth := len(stack) - 1
			startY := stack[depth]
			mr.actStack[m.From] = stack[:depth]
			mr.actElems = append(mr.actElems, mr.activationRect(m.From, startY, mr.curY, depth))
		}
	}
}

func (mr *messageRenderer) renderDestroy(id string) []any {
	idx, ok := mr.lay.participantIx[id]
	if !ok {
		return nil
	}
	x := mr.lay.participantX[idx]
	y := mr.curY
	half := 6.0
	style := fmt.Sprintf("stroke:%s;stroke-width:%.1f", mr.th.MessageStroke, defaultStrokeWidth*2)
	return []any{
		&line{X1: svgFloat(x - half), Y1: svgFloat(y - half), X2: svgFloat(x + half), Y2: svgFloat(y + half), Style: style},
		&line{X1: svgFloat(x - half), Y1: svgFloat(y + half), X2: svgFloat(x + half), Y2: svgFloat(y - half), Style: style},
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
		return mr.lay.participantIx[ids[i]] < mr.lay.participantIx[ids[j]]
	})
	var elems []any
	for _, id := range ids {
		for depth, startY := range mr.actStack[id] {
			elems = append(elems, mr.activationRect(id, startY, mr.curY, depth))
		}
	}
	elems = append(elems, mr.actElems...)
	return elems
}

func (mr *messageRenderer) activationRect(id string, startY, endY float64, depth int) *rect {
	idx := mr.lay.participantIx[id]
	x := mr.lay.participantX[idx]
	// Stacked activations shift right by half-bar-width per nesting
	// level so inner bars stay visible alongside their parents.
	offset := float64(depth) * defaultActivationW / 2
	return &rect{
		X: svgFloat(x - defaultActivationW/2 + offset), Y: svgFloat(startY),
		Width: svgFloat(defaultActivationW), Height: svgFloat(endY - startY),
		// Neutral gray so bars don't visually compete with the
		// purple lifeline / participant theme — mmdc parity.
		Style: fmt.Sprintf("fill:%s;stroke:%s;stroke-width:%.1f",
			"#F4F4F4", "#666", defaultStrokeWidth),
	}
}

func (mr *messageRenderer) messageLineStyle(at diagram.ArrowType) string {
	if at.IsDashed() {
		return mr.msgLineStyleDashed
	}
	return mr.msgLineStyleSolid
}

// brTokenRe matches Mermaid's <br>, <br/>, <br /> (any case, any
// whitespace before the slash) so multi-line message labels split
// correctly regardless of how the user spells the tag.
var brTokenRe = regexp.MustCompile(`(?i)<br\s*/?>`)

// splitLabelLines splits s on Mermaid's <br> family of tokens. Returns
// the original string as a single-element slice when no break tokens
// are present.
func splitLabelLines(s string) []string {
	if strings.IndexByte(s, '<') < 0 {
		return []string{s}
	}
	return brTokenRe.Split(s, -1)
}

// labelLineHeight is the per-line vertical advance for stacked labels.
// Owned here so callers don't need to recompute it when positioning the
// stack relative to other elements (see multilineTextAbove).
func labelLineHeight(fontSize float64) float64 { return fontSize + 2 }

// selfLoopRowExtra returns the additional row advance (beyond defaultRowHeight)
// required when the current item is a self-message, so the arc — which extends
// selfLoopH below the message anchor y — clears the next row's label-above-arrow
// (drawn at next_y - 6 with one line of label height above the baseline).
//
// Returns 0 when defaultRowHeight already accommodates everything.
func selfLoopRowExtra(fontSize float64) float64 {
	const labelGap = 6.0
	needed := selfLoopH + labelLineHeight(fontSize) + labelGap
	if needed > defaultRowHeight {
		return needed - defaultRowHeight
	}
	return 0
}

// multilineText returns one or more text elements forming a vertically
// stacked label centered on (cx, cy). Used wherever a Mermaid label may
// contain `<br/>` line breaks.
func multilineText(content string, cx, cy float64, anchor, dominant, style string, fontSize float64) []any {
	lines := splitLabelLines(content)
	if len(lines) <= 1 {
		return []any{&text{
			X: svgFloat(cx), Y: svgFloat(cy),
			Anchor: anchor, Dominant: dominant,
			Style: style, Content: content,
		}}
	}
	lineH := labelLineHeight(fontSize)
	totalH := lineH * float64(len(lines)-1)
	startY := cy - totalH/2
	out := make([]any, 0, len(lines))
	for i, ln := range lines {
		out = append(out, &text{
			X: svgFloat(cx), Y: svgFloat(startY + float64(i)*lineH),
			Anchor: anchor, Dominant: dominant,
			Style: style, Content: ln,
		})
	}
	return out
}

// multilineTextAbove is like multilineText but positions the *bottom*
// line at anchorY and grows upward. Use when the label sits above a
// reference y like an arrow.
func multilineTextAbove(content string, cx, anchorY float64, anchor, style string, fontSize float64) []any {
	lines := splitLabelLines(content)
	cy := anchorY - float64(len(lines)-1)*labelLineHeight(fontSize)/2
	return multilineText(content, cx, cy, anchor, "auto", style, fontSize)
}

func arrowMarkerID(at diagram.ArrowType) string {
	return fmt.Sprintf("seq-arrow-%s", at.String())
}

// bidirArrowhead returns a filled triangle pointing in the +dir direction
// (dir is +1 for right-pointing, -1 for left-pointing). The tip sits at
// (tipX, tipY); the base is 8px back along the line.
func bidirArrowhead(tipX, tipY, dir float64, fill string) *polygon {
	const length = 8.0
	const halfWidth = 4.0
	baseX := tipX - dir*length
	return &polygon{
		Points: fmt.Sprintf("%.2f,%.2f %.2f,%.2f %.2f,%.2f",
			tipX, tipY,
			baseX, tipY-halfWidth,
			baseX, tipY+halfWidth),
		Style: fmt.Sprintf("fill:%s", fill),
	}
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
			// `-x` / `--x` arrows render as an ✕ at the destination end
			// (mmdc parity). Two crossed strokes inside the 10×10 viewBox.
			ID: arrowMarkerID(diagram.ArrowTypeSolidCross), ViewBox: "0 0 10 10",
			RefX: 5, RefY: 5, Width: 10, Height: 10, Orient: "auto",
			Children: []any{&path{D: "M2,2 L8,8 M2,8 L8,2", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
		},
		{
			ID: arrowMarkerID(diagram.ArrowTypeDashedCross), ViewBox: "0 0 10 10",
			RefX: 5, RefY: 5, Width: 10, Height: 10, Orient: "auto",
			Children: []any{&path{D: "M2,2 L8,8 M2,8 L8,2", Style: fmt.Sprintf("stroke:%s;stroke-width:%.1f;fill:none", stroke, sw)}},
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
