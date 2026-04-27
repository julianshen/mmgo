// Package sequence parses Mermaid sequenceDiagram syntax into a
// SequenceDiagram AST.
package sequence

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

// Parse reads a sequence diagram definition from r and returns the
// resulting SequenceDiagram. Errors include a 1-based line number
// pointing to the offending input.
func Parse(r io.Reader) (*diagram.SequenceDiagram, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	p := &parser{
		diagram:       &diagram.SequenceDiagram{},
		participantIx: make(map[string]int),
		destroyed:     make(map[string]bool),
	}
	frontmatter, body := parserutil.SplitFrontmatter(src)
	if title := parserutil.FrontmatterValue(frontmatter, "title"); title != "" {
		p.diagram.Title = title
	}
	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if _, ok := trimKeyword(line, "sequenceDiagram"); !ok {
				return nil, fmt.Errorf("line %d: expected 'sequenceDiagram' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing sequenceDiagram header")
	}
	if len(p.blockStack) > 0 {
		return nil, fmt.Errorf("unclosed %v block (missing 'end')", p.blockStack[len(p.blockStack)-1].block.Kind)
	}
	if p.accDescrBlock != nil {
		return nil, fmt.Errorf("unclosed accDescr block (missing '}')")
	}
	return p.diagram, nil
}

// participantIx maps participant ID to its position in
// diagram.Participants so that explicit declarations can upgrade an
// entry that was implicitly registered by a prior message.
type parser struct {
	diagram       *diagram.SequenceDiagram
	participantIx map[string]int
	blockStack    []*blockFrame
	boxFrame      *boxFrameState
	destroyed     map[string]bool
	// non-nil while inside an accDescr {…} block.
	accDescrBlock *strings.Builder
}

type boxFrameState struct {
	fill      string
	label     string
	memberIDs []string
}

type blockFrame struct {
	block        *diagram.Block
	activeBranch *diagram.Block
}

// trimKeyword returns the whitespace-trimmed remainder after kw when
// line starts with kw followed by end-of-string or whitespace. The
// word-boundary check prevents `sequenceDiagramX` from matching
// `sequenceDiagram`.
func trimKeyword(line, kw string) (string, bool) {
	if !strings.HasPrefix(line, kw) {
		return "", false
	}
	if len(line) == len(kw) {
		return "", true
	}
	c := line[len(kw)]
	if c != ' ' && c != '\t' {
		return "", false
	}
	return strings.TrimSpace(line[len(kw)+1:]), true
}

func (p *parser) parseLine(line string) error {
	// Inside an accDescr {…} block every line is description text — must
	// short-circuit before any other dispatch, otherwise lines containing
	// "note", "end", etc. would be mis-parsed as diagram statements.
	if p.accDescrBlock != nil {
		if line == "}" {
			p.diagram.AccDescr = p.accDescrBlock.String()
			p.accDescrBlock = nil
			return nil
		}
		if p.accDescrBlock.Len() > 0 {
			p.accDescrBlock.WriteByte('\n')
		}
		p.accDescrBlock.WriteString(line)
		return nil
	}
	if rest, ok := trimKeyword(line, "participant"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindParticipant)
	}
	if rest, ok := trimKeyword(line, "actor"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindActor)
	}
	if rest, ok := trimKeyword(line, "autonumber"); ok {
		return p.parseAutonumber(rest)
	}
	if rest, ok := trimKeyword(line, "Note"); ok {
		return p.parseNote(rest)
	}
	if rest, ok := trimKeyword(line, "note"); ok {
		return p.parseNote(rest)
	}
	if rest, ok := trimKeyword(line, "box"); ok {
		return p.openBox(rest)
	}
	if line == "end" {
		if p.boxFrame != nil {
			return p.closeBox()
		}
		return p.closeBlock()
	}
	if kind, label, ok := matchBlockOpen(line); ok {
		return p.openBlock(kind, label)
	}
	if parent, label, ok := matchBranchKeyword(line); ok {
		return p.addBranch(parent, label)
	}
	if rest, ok := trimKeyword(line, "create"); ok {
		return p.parseCreate(rest)
	}
	if rest, ok := trimKeyword(line, "destroy"); ok {
		return p.parseDestroy(rest)
	}
	if rest, ok := trimKeyword(line, "activate"); ok {
		return p.parseActivation(rest, true)
	}
	if rest, ok := trimKeyword(line, "deactivate"); ok {
		return p.parseActivation(rest, false)
	}
	if rest, ok := strings.CutPrefix(line, "title:"); ok {
		p.diagram.Title = strings.TrimSpace(rest)
		return nil
	}
	if rest, ok := trimKeyword(line, "title"); ok {
		p.diagram.Title = rest
		return nil
	}
	if parserutil.HasHeaderKeyword(line, "accTitle") {
		p.diagram.AccTitle = parserutil.TrimKeyword(line, "accTitle")
		return nil
	}
	if parserutil.HasHeaderKeyword(line, "accDescr") {
		rest := parserutil.TrimKeyword(line, "accDescr")
		if rest == "{" {
			p.accDescrBlock = &strings.Builder{}
			return nil
		}
		p.diagram.AccDescr = rest
		return nil
	}
	if m, ok := parseMessage(line); ok {
		p.ensureParticipant(m.From)
		p.ensureParticipant(m.To)
		p.appendItem(diagram.NewMessageItem(m))
		return nil
	}
	return fmt.Errorf("unrecognized statement: %q", line)
}

func (p *parser) parseActivation(rest string, activate bool) error {
	id := strings.TrimSpace(rest)
	if id == "" {
		kw := "activate"
		if !activate {
			kw = "deactivate"
		}
		return fmt.Errorf("%s requires a participant ID", kw)
	}
	p.ensureParticipant(id)
	p.appendItem(diagram.NewActivationItem(diagram.Activation{Participant: id, Activate: activate}))
	return nil
}

func (p *parser) parseAutonumber(rest string) error {
	an := diagram.AutoNumber{Enabled: true, Start: 1, Step: 1}
	if rest == "" {
		p.diagram.AutoNumber = an
		return nil
	}
	fields := strings.Fields(rest)
	if len(fields) > 2 {
		return fmt.Errorf("autonumber accepts at most 2 arguments (start step), got %d", len(fields))
	}
	start, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("autonumber start: %w", err)
	}
	if start <= 0 {
		return fmt.Errorf("autonumber start must be positive, got %d", start)
	}
	an.Start = start
	if len(fields) == 2 {
		step, err := strconv.Atoi(fields[1])
		if err != nil {
			return fmt.Errorf("autonumber step: %w", err)
		}
		if step <= 0 {
			return fmt.Errorf("autonumber step must be positive, got %d", step)
		}
		an.Step = step
	}
	p.diagram.AutoNumber = an
	return nil
}

func (p *parser) parseNote(rest string) error {
	var pos diagram.NotePosition
	var after string
	if r, ok := strings.CutPrefix(rest, "left of "); ok {
		pos, after = diagram.NotePositionLeft, r
	} else if r, ok := strings.CutPrefix(rest, "right of "); ok {
		pos, after = diagram.NotePositionRight, r
	} else if r, ok := strings.CutPrefix(rest, "over "); ok {
		pos, after = diagram.NotePositionOver, r
	} else {
		return fmt.Errorf("note position must be 'left of', 'right of', or 'over': %q", rest)
	}
	colon := strings.IndexByte(after, ':')
	if colon < 0 {
		return fmt.Errorf("note missing ':' before text: %q", rest)
	}
	who := strings.TrimSpace(after[:colon])
	text := strings.TrimSpace(after[colon+1:])
	parts := splitParticipantList(who)
	if pos != diagram.NotePositionOver && len(parts) != 1 {
		return fmt.Errorf("'left of'/'right of' note takes exactly one participant, got %d", len(parts))
	}
	if len(parts) == 0 || len(parts) > 2 {
		return fmt.Errorf("note expects 1 or 2 participants, got %d", len(parts))
	}
	for _, id := range parts {
		p.ensureParticipant(id)
	}
	p.appendItem(diagram.NewNoteItem(diagram.Note{
		Participants: parts,
		Text:         text,
		Position:     pos,
	}))
	return nil
}

func (p *parser) appendItem(item diagram.SequenceItem) {
	if len(p.blockStack) > 0 {
		top := p.blockStack[len(p.blockStack)-1]
		top.activeBranch.Items = append(top.activeBranch.Items, item)
	} else {
		p.diagram.Items = append(p.diagram.Items, item)
	}
}

var blockOpeners = []struct {
	kw   string
	kind diagram.BlockKind
}{
	{"alt", diagram.BlockKindAlt},
	{"opt", diagram.BlockKindOpt},
	{"loop", diagram.BlockKindLoop},
	{"par", diagram.BlockKindPar},
	{"critical", diagram.BlockKindCritical},
	{"break", diagram.BlockKindBreak},
	{"rect", diagram.BlockKindRect},
}

func matchBlockOpen(line string) (diagram.BlockKind, string, bool) {
	for _, bo := range blockOpeners {
		if rest, ok := trimKeyword(line, bo.kw); ok {
			return bo.kind, rest, true
		}
	}
	return 0, "", false
}

func parseColorValue(s string) (color, rest string) {
	if strings.HasPrefix(s, "rgba(") {
		closeIdx := strings.IndexByte(s, ')')
		if closeIdx < 0 {
			return "", s
		}
		return s[:closeIdx+1], strings.TrimSpace(s[closeIdx+1:])
	}
	if strings.HasPrefix(s, "rgb(") {
		closeIdx := strings.IndexByte(s, ')')
		if closeIdx < 0 {
			return "", s
		}
		return s[:closeIdx+1], strings.TrimSpace(s[closeIdx+1:])
	}
	if len(s) > 0 && s[0] == '#' {
		i := 1
		for i < len(s) && ((s[i] >= '0' && s[i] <= '9') || (s[i] >= 'a' && s[i] <= 'f') || (s[i] >= 'A' && s[i] <= 'F')) {
			i++
		}
		if i >= 4 && i <= 9 {
			return s[:i], strings.TrimSpace(s[i:])
		}
	}
	return "", s
}

func (p *parser) openBlock(kind diagram.BlockKind, label string) error {
	var fill string
	var hasAlpha bool
	if kind == diagram.BlockKindRect {
		color, rest := parseColorValue(label)
		fill = color
		label = rest
		hasAlpha = strings.HasPrefix(color, "rgba(")
	}
	b := &diagram.Block{Kind: kind, Label: label, Fill: fill, HasAlpha: hasAlpha}
	frame := &blockFrame{block: b, activeBranch: b}
	p.blockStack = append(p.blockStack, frame)
	return nil
}

func (p *parser) closeBlock() error {
	if len(p.blockStack) == 0 {
		return fmt.Errorf("'end' without a matching block opener")
	}
	top := p.blockStack[len(p.blockStack)-1]
	p.blockStack = p.blockStack[:len(p.blockStack)-1]
	p.appendItem(diagram.NewBlockItem(*top.block))
	return nil
}

func (p *parser) openBox(rest string) error {
	if p.boxFrame != nil {
		return fmt.Errorf("nested boxes are not allowed")
	}
	color, label := parseColorValue(rest)
	p.boxFrame = &boxFrameState{
		fill:  color,
		label: strings.TrimSpace(label),
	}
	return nil
}

func (p *parser) closeBox() error {
	bx := p.boxFrame
	p.boxFrame = nil
	boxIndex := len(p.diagram.Boxes)
	p.diagram.Boxes = append(p.diagram.Boxes, diagram.Box{
		Label:   bx.label,
		Fill:    bx.fill,
		Members: bx.memberIDs,
	})
	for _, id := range bx.memberIDs {
		if idx, ok := p.participantIx[id]; ok {
			p.diagram.Participants[idx].BoxIndex = boxIndex
		}
	}
	return nil
}

// Branch keywords: else → alt, and → par, option → critical.
var branchKeywords = []struct {
	kw     string
	parent diagram.BlockKind
}{
	{"else", diagram.BlockKindAlt},
	{"and", diagram.BlockKindPar},
	{"option", diagram.BlockKindCritical},
}

func matchBranchKeyword(line string) (diagram.BlockKind, string, bool) {
	for _, bk := range branchKeywords {
		if rest, ok := trimKeyword(line, bk.kw); ok {
			return bk.parent, rest, true
		}
	}
	return 0, "", false
}

func (p *parser) addBranch(expectedParent diagram.BlockKind, label string) error {
	if len(p.blockStack) == 0 {
		return fmt.Errorf("branch keyword outside of any block")
	}
	top := p.blockStack[len(p.blockStack)-1]
	if top.block.Kind != expectedParent {
		return fmt.Errorf("branch keyword not valid inside %v block", top.block.Kind)
	}
	branch := diagram.Block{Kind: top.block.Kind, Label: label}
	top.block.Branches = append(top.block.Branches, branch)
	top.activeBranch = &top.block.Branches[len(top.block.Branches)-1]
	return nil
}

func splitParticipantList(s string) []string {
	raw := strings.Split(s, ",")
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		if t := strings.TrimSpace(r); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (p *parser) registerParticipant(id, alias string, kind diagram.ParticipantKind, createdAt int, upgradeExisting bool) {
	if existing, ok := p.participantIx[id]; ok {
		if upgradeExisting {
			p.diagram.Participants[existing].Kind = kind
			if alias != "" {
				p.diagram.Participants[existing].Alias = alias
			}
		}
		if createdAt >= 0 {
			p.diagram.Participants[existing].CreatedAtItem = createdAt
		}
		if p.boxFrame != nil && p.diagram.Participants[existing].BoxIndex == -1 {
			p.diagram.Participants[existing].BoxIndex = len(p.diagram.Boxes)
			p.boxFrame.memberIDs = append(p.boxFrame.memberIDs, id)
		}
		return
	}
	boxIdx := -1
	if p.boxFrame != nil {
		boxIdx = len(p.diagram.Boxes)
		p.boxFrame.memberIDs = append(p.boxFrame.memberIDs, id)
	}
	p.participantIx[id] = len(p.diagram.Participants)
	p.diagram.Participants = append(p.diagram.Participants, diagram.Participant{
		ID: id, Alias: alias, Kind: kind, BoxIndex: boxIdx,
		CreatedAtItem: createdAt, DestroyedAtItem: -1,
	})
}

func (p *parser) parseParticipant(rest string, kind diagram.ParticipantKind) error {
	if rest == "" {
		return fmt.Errorf("participant declaration missing ID")
	}
	var id, alias string
	if idx := strings.Index(rest, " as "); idx >= 0 {
		id = strings.TrimSpace(rest[:idx])
		alias = strings.TrimSpace(rest[idx+len(" as "):])
	} else {
		id = rest
	}
	if id == "" {
		return fmt.Errorf("participant declaration missing ID")
	}
	p.registerParticipant(id, alias, kind, -1, true)
	return nil
}

func (p *parser) ensureParticipant(id string) {
	if _, ok := p.participantIx[id]; ok {
		return
	}
	p.registerParticipant(id, "", diagram.ParticipantKindParticipant, -1, false)
}

func (p *parser) currentItemCount() int {
	if len(p.blockStack) > 0 {
		top := p.blockStack[len(p.blockStack)-1]
		return len(top.activeBranch.Items)
	}
	return len(p.diagram.Items)
}

func (p *parser) parseCreate(rest string) error {
	var kind diagram.ParticipantKind
	var idRest string
	if r, ok := trimKeyword(rest, "participant"); ok {
		kind = diagram.ParticipantKindParticipant
		idRest = r
	} else if r, ok := trimKeyword(rest, "actor"); ok {
		kind = diagram.ParticipantKindActor
		idRest = r
	} else {
		return fmt.Errorf("create expects 'participant' or 'actor', got %q", rest)
	}

	var id, alias string
	if idx := strings.Index(idRest, " as "); idx >= 0 {
		id = strings.TrimSpace(idRest[:idx])
		alias = strings.TrimSpace(idRest[idx+len(" as "):])
	} else {
		id = strings.TrimSpace(idRest)
	}
	if id == "" {
		return fmt.Errorf("create participant missing ID")
	}

	p.registerParticipant(id, alias, kind, p.currentItemCount(), true)
	return nil
}

func (p *parser) parseDestroy(rest string) error {
	id := strings.TrimSpace(rest)
	if id == "" {
		return fmt.Errorf("destroy missing participant ID")
	}
	if _, ok := p.participantIx[id]; !ok {
		return fmt.Errorf("destroy: unknown participant %q", id)
	}
	if p.destroyed[id] {
		return fmt.Errorf("destroy: participant %q already destroyed", id)
	}
	itemIndex := p.currentItemCount()
	p.diagram.Participants[p.participantIx[id]].DestroyedAtItem = itemIndex
	p.destroyed[id] = true
	p.appendItem(diagram.NewDestroyItem(id))
	return nil
}

// arrowTokens lists the 8 supported message arrow literals with their
// ArrowType. Order matters: longer prefixes must precede their shorter
// prefixes so `-->>` is matched before `-->`, `--x` before `-x`, etc.
var arrowTokens = []struct {
	lit string
	typ diagram.ArrowType
}{
	{"<<-->>", diagram.ArrowTypeDashedBi},
	{"<<->>", diagram.ArrowTypeSolidBi},
	{"-->>", diagram.ArrowTypeDashed},
	{"-->", diagram.ArrowTypeDashedNoHead},
	{"--x", diagram.ArrowTypeDashedCross},
	{"--)", diagram.ArrowTypeDashedOpen},
	{"->>", diagram.ArrowTypeSolid},
	{"->", diagram.ArrowTypeSolidNoHead},
	{"-x", diagram.ArrowTypeSolidCross},
	{"-)", diagram.ArrowTypeSolidOpen},
}

// parseMessage finds the LEFTMOST arrow token in line and splits
// From/To around it. Leftmost-wins is correctness-critical when a
// message label contains an arrow-like substring — e.g.,
// `A->>B: send --> response` must split on the leading `->>`, not the
// `-->` inside the label. When two tokens start at the same position
// the longer one wins so `-->>` beats `-->`, `--x` beats `-x`, etc.
func parseMessage(line string) (diagram.Message, bool) {
	bestIdx := -1
	var best struct {
		lit string
		typ diagram.ArrowType
	}
	for _, tok := range arrowTokens {
		idx := strings.Index(line, tok.lit)
		if idx <= 0 {
			continue
		}
		if bestIdx == -1 || idx < bestIdx || (idx == bestIdx && len(tok.lit) > len(best.lit)) {
			bestIdx, best = idx, tok
		}
	}
	if bestIdx == -1 {
		return diagram.Message{}, false
	}
	from := strings.TrimSpace(line[:bestIdx])
	to, label := splitMessageLabel(line[bestIdx+len(best.lit):])
	to = strings.TrimSpace(to)
	lifeline := diagram.LifelineEffectNone
	if r, ok := strings.CutPrefix(to, "+"); ok {
		lifeline, to = diagram.LifelineEffectActivate, strings.TrimSpace(r)
	} else if r, ok := strings.CutPrefix(to, "-"); ok {
		lifeline, to = diagram.LifelineEffectDeactivate, strings.TrimSpace(r)
	}
	if from == "" || to == "" {
		return diagram.Message{}, false
	}
	return diagram.Message{
		From:      from,
		To:        to,
		Label:     label,
		ArrowType: best.typ,
		Lifeline:  lifeline,
	}, true
}

// splitMessageLabel splits on the FIRST colon so labels containing
// additional colons (e.g., "hello: world") are preserved intact.
func splitMessageLabel(rest string) (to, label string) {
	idx := strings.IndexByte(rest, ':')
	if idx < 0 {
		return rest, ""
	}
	return rest[:idx], strings.TrimSpace(rest[idx+1:])
}
