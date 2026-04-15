// Package sequence parses Mermaid sequenceDiagram syntax into a
// SequenceDiagram AST.
package sequence

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// Parse reads a sequence diagram definition from r and returns the
// resulting SequenceDiagram. Errors include a 1-based line number
// pointing to the offending input.
func Parse(r io.Reader) (*diagram.SequenceDiagram, error) {
	p := &parser{
		diagram:       &diagram.SequenceDiagram{},
		participantIx: make(map[string]int),
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(stripComment(scanner.Text()))
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
	return p.diagram, nil
}

// participantIx maps participant ID to its position in
// diagram.Participants so that explicit declarations can upgrade an
// entry that was implicitly registered by a prior message.
type parser struct {
	diagram       *diagram.SequenceDiagram
	participantIx map[string]int
}

// stripComment removes a `%%` to-end-of-line comment. The `%%` only
// starts a comment when at the start of the line or preceded by
// whitespace, so tokens like "50%%" stay intact.
func stripComment(line string) string {
	for i := 0; i+1 < len(line); i++ {
		if line[i] != '%' || line[i+1] != '%' {
			continue
		}
		if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
			return line[:i]
		}
	}
	return line
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
	if rest, ok := trimKeyword(line, "participant"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindParticipant)
	}
	if rest, ok := trimKeyword(line, "actor"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindActor)
	}
	if _, ok := trimKeyword(line, "autonumber"); ok {
		p.diagram.AutoNumber = true
		return nil
	}
	if m, ok := parseMessage(line); ok {
		p.ensureParticipant(m.From)
		p.ensureParticipant(m.To)
		p.diagram.Items = append(p.diagram.Items, diagram.NewMessageItem(m))
		return nil
	}
	return fmt.Errorf("unrecognized statement: %q", line)
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
	if existing, ok := p.participantIx[id]; ok {
		p.diagram.Participants[existing].Kind = kind
		if alias != "" {
			p.diagram.Participants[existing].Alias = alias
		}
		return nil
	}
	p.participantIx[id] = len(p.diagram.Participants)
	p.diagram.Participants = append(p.diagram.Participants, diagram.Participant{
		ID: id, Alias: alias, Kind: kind,
	})
	return nil
}

func (p *parser) ensureParticipant(id string) {
	if _, ok := p.participantIx[id]; ok {
		return
	}
	p.participantIx[id] = len(p.diagram.Participants)
	p.diagram.Participants = append(p.diagram.Participants, diagram.Participant{
		ID: id, Kind: diagram.ParticipantKindParticipant,
	})
}

// arrowTokens lists the 8 supported message arrow literals with their
// ArrowType. Order matters: longer prefixes must precede their shorter
// prefixes so `-->>` is matched before `-->`, `--x` before `-x`, etc.
var arrowTokens = []struct {
	lit string
	typ diagram.ArrowType
}{
	{"-->>", diagram.ArrowTypeDashed},
	{"-->", diagram.ArrowTypeDashedNoHead},
	{"--x", diagram.ArrowTypeDashedCross},
	{"--)", diagram.ArrowTypeDashedOpen},
	{"->>", diagram.ArrowTypeSolid},
	{"->", diagram.ArrowTypeSolidNoHead},
	{"-x", diagram.ArrowTypeSolidCross},
	{"-)", diagram.ArrowTypeSolidOpen},
}

func parseMessage(line string) (diagram.Message, bool) {
	for _, tok := range arrowTokens {
		idx := strings.Index(line, tok.lit)
		if idx <= 0 {
			continue
		}
		from := strings.TrimSpace(line[:idx])
		to, label := splitMessageLabel(line[idx+len(tok.lit):])
		to = strings.TrimSpace(to)
		if from == "" || to == "" {
			return diagram.Message{}, false
		}
		return diagram.Message{
			From:      from,
			To:        to,
			Label:     label,
			ArrowType: tok.typ,
		}, true
	}
	return diagram.Message{}, false
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
