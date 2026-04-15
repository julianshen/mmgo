// Package sequence parses Mermaid sequenceDiagram syntax into a
// SequenceDiagram AST.
//
// Slice-A scope (this PR):
//
//	sequenceDiagram
//	    participant Alice
//	    participant B as Bob
//	    actor C as Carol
//	    autonumber
//	    A->>B: hello
//	    B-->>A: hi back
//	    %% line and trailing comments stripped
//
// Supported message arrows: ->>, ->, -->>, -->, -x, --x, -), --).
// Participants referenced in messages without an explicit declaration
// are auto-registered in first-seen order, matching Mermaid's behavior.
//
// Out of scope here, landing in follow-up PRs:
//   - Activation markers (+/- suffix) and notes
//   - Structural blocks: alt/else, opt, loop, par, critical, break, rect
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
			if !matchKeyword(line, "sequenceDiagram") {
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

// parser holds mutable state during line-by-line parsing. participantIx
// maps participant ID to its position in diagram.Participants so that
// explicit declarations can upgrade an entry that was implicitly
// registered by a prior message.
type parser struct {
	diagram       *diagram.SequenceDiagram
	participantIx map[string]int
}

// stripComment removes a `%%` to-end-of-line comment. For slice A we
// don't have bracketed constructs to worry about, so a simple scan is
// sufficient — `%%` only starts a comment when at the start of the
// line or preceded by whitespace, so tokens like "50%%" stay intact.
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

// matchKeyword reports whether line is the keyword kw, followed by
// either end-of-string or whitespace. Word boundary prevents
// `sequenceDiagramX` from matching `sequenceDiagram`.
func matchKeyword(line, kw string) bool {
	if !strings.HasPrefix(line, kw) {
		return false
	}
	if len(line) == len(kw) {
		return true
	}
	c := line[len(kw)]
	return c == ' ' || c == '\t'
}

func (p *parser) parseLine(line string) error {
	if rest, ok := trimKeyword(line, "participant"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindParticipant)
	}
	if rest, ok := trimKeyword(line, "actor"); ok {
		return p.parseParticipant(rest, diagram.ParticipantKindActor)
	}
	if matchKeyword(line, "autonumber") {
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

// trimKeyword returns the whitespace-trimmed remainder after kw when
// line starts with `kw` + whitespace, and false otherwise.
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

// parseParticipant handles `participant ID` and `participant ID as Alias`.
// Empty ID is a syntax error. Re-declaring an existing participant
// upgrades its kind/alias in place rather than appending a duplicate.
func (p *parser) parseParticipant(rest string, kind diagram.ParticipantKind) error {
	if rest == "" {
		return fmt.Errorf("participant declaration missing ID")
	}
	var id, alias string
	if idx := findAsKeyword(rest); idx >= 0 {
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

// findAsKeyword locates ` as ` as a standalone keyword (surrounded by
// whitespace) so an ID like `class` containing "as" as a substring
// isn't wrongly split.
func findAsKeyword(s string) int {
	for i := 0; i+3 < len(s); i++ {
		if (s[i] == ' ' || s[i] == '\t') && s[i+1] == 'a' && s[i+2] == 's' &&
			(s[i+3] == ' ' || s[i+3] == '\t') {
			return i
		}
	}
	return -1
}

// ensureParticipant auto-registers an ID referenced by a message when
// it hasn't been explicitly declared yet. Auto-registered participants
// default to the ParticipantKindParticipant (box) style.
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

// parseMessage tries to parse `From<arrow>To[: label]`. Returns ok=false
// when no supported arrow token is found — the caller then reports an
// "unrecognized statement" error. Participant IDs are trimmed of
// whitespace on both sides of the arrow.
func parseMessage(line string) (diagram.Message, bool) {
	for _, tok := range arrowTokens {
		idx := strings.Index(line, tok.lit)
		if idx <= 0 {
			continue
		}
		from := strings.TrimSpace(line[:idx])
		rest := line[idx+len(tok.lit):]
		to, label := splitMessageLabel(rest)
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

// splitMessageLabel separates the target from an optional `: label`
// suffix. Only the FIRST colon is the separator so labels may contain
// additional colons (e.g., "hello: world"). Label whitespace is
// trimmed.
func splitMessageLabel(rest string) (to, label string) {
	idx := strings.IndexByte(rest, ':')
	if idx < 0 {
		return rest, ""
	}
	return rest[:idx], strings.TrimSpace(rest[idx+1:])
}
