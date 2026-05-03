package state

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.StateDiagram, error) {
	p := &parser{
		diagram: &diagram.StateDiagram{},
	}
	p.scanner = bufio.NewScanner(r)
	p.scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	headerSeen := false
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if line != "stateDiagram-v2" && line != "stateDiagram" {
				return nil, fmt.Errorf("line %d: expected 'stateDiagram-v2' header, got %q", p.lineNum, line)
			}
			headerSeen = true
			continue
		}
		if err := p.parseLine(line, &p.diagram.States); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing stateDiagram header")
	}
	return p.diagram, nil
}

type parser struct {
	diagram *diagram.StateDiagram
	scanner *bufio.Scanner
	lineNum int
}

// upsertState returns a pointer to the state with the given id in
// target, creating it if absent. `[*]` pseudo-states return nil.
func upsertState(target *[]diagram.StateDef, id string) *diagram.StateDef {
	if id == "[*]" {
		return nil
	}
	for i := range *target {
		if (*target)[i].ID == id {
			return &(*target)[i]
		}
	}
	*target = append(*target, diagram.StateDef{ID: id, Label: id})
	return &(*target)[len(*target)-1]
}

func (p *parser) parseLine(line string, target *[]diagram.StateDef) error {
	if rest, ok := strings.CutPrefix(line, "state "); ok {
		return p.parseStateDecl(strings.TrimSpace(rest), target)
	}
	if strings.HasPrefix(line, "note ") {
		return p.parseNote(line, target)
	}
	if rest, ok := strings.CutPrefix(line, "direction "); ok {
		dir, err := parserutil.ParseDirection(strings.TrimSpace(rest))
		if err != nil {
			return err
		}
		p.diagram.Direction = dir
		return nil
	}
	if t, ok := parseTransition(line); ok {
		upsertState(target, t.From)
		upsertState(target, t.To)
		p.diagram.Transitions = append(p.diagram.Transitions, t)
		return nil
	}
	if id, desc, ok := parseStateDescription(line); ok {
		s := upsertState(target, id)
		if s != nil {
			s.Description = desc
		}
		return nil
	}
	return nil
}

// parseNote handles the four note forms Mermaid v2 supports:
//
//   - `note left of S : text`     (single-line)
//   - `note right of S : text`    (single-line)
//   - `note left of S\n…\nend note`  (block form)
//   - `note right of S\n…\nend note` (block form)
//
// The single-line form's text inherits `\n` → real-newline expansion;
// the block form joins its body lines with real newlines.
func (p *parser) parseNote(line string, target *[]diagram.StateDef) error {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "note"))
	var side diagram.NoteSide
	switch {
	case strings.HasPrefix(rest, "left of "):
		side = diagram.NoteSideLeft
		rest = rest[len("left of "):]
	case strings.HasPrefix(rest, "right of "):
		side = diagram.NoteSideRight
		rest = rest[len("right of "):]
	default:
		return fmt.Errorf("note must be `left of` or `right of`")
	}
	stateID := rest
	text := ""
	if i := strings.Index(rest, " : "); i >= 0 {
		// Single-line form.
		stateID = strings.TrimSpace(rest[:i])
		text = parserutil.ExpandLineBreaks(strings.TrimSpace(rest[i+3:]))
	} else {
		// Block form: scan until `end note`.
		stateID = strings.TrimSpace(stateID)
		body, err := p.scanBlockNote()
		if err != nil {
			return err
		}
		text = body
	}
	if stateID == "" {
		return fmt.Errorf("note: missing target state id")
	}
	upsertState(target, stateID)
	p.diagram.Notes = append(p.diagram.Notes, diagram.StateNote{
		Text: text, Side: side, Target: stateID,
	})
	return nil
}

// scanBlockNote reads body lines until it sees `end note` (trimmed)
// and returns them joined by real newlines. Blank lines and comments
// inside the block are preserved as separators / dropped, matching
// the rest of the parser's whitespace policy.
func (p *parser) scanBlockNote() (string, error) {
	var lines []string
	for p.scanner.Scan() {
		p.lineNum++
		raw := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if raw == "" {
			continue
		}
		if raw == "end note" {
			return strings.Join(lines, "\n"), nil
		}
		lines = append(lines, raw)
	}
	if err := p.scanner.Err(); err != nil {
		return "", fmt.Errorf("reading note body: %w", err)
	}
	return "", fmt.Errorf("unclosed note block")
}

// parseStateDescription matches `id : description text` outside of
// any arrow-bearing transition. Mermaid's grammar requires whitespace
// around the colon, so we only accept that form — same convention as
// the class parser uses for single-line members. A bare `id:text`
// is rejected (it's typically a typo or a misparsed transition).
func parseStateDescription(line string) (id, desc string, ok bool) {
	colon := strings.Index(line, " : ")
	if colon < 0 {
		return "", "", false
	}
	id = strings.TrimSpace(line[:colon])
	if id == "" || strings.ContainsAny(id, " \t") {
		return "", "", false
	}
	return id, strings.TrimSpace(line[colon+3:]), true
}

func (p *parser) parseStateDecl(rest string, target *[]diagram.StateDef) error {
	if strings.HasPrefix(rest, "\"") {
		return p.parseAliasDecl(rest, target)
	}
	if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
		name := strings.TrimSpace(rest[:braceIdx])
		s := upsertState(target, name)
		if s == nil {
			return fmt.Errorf("invalid composite state name %q", name)
		}
		return p.parseCompositeBody(&s.Children)
	}
	parts := strings.Fields(rest)
	if len(parts) >= 2 && strings.HasPrefix(parts[1], "<<") && strings.HasSuffix(parts[1], ">>") {
		id := parts[0]
		annotation := strings.Trim(parts[1], "<>")
		s := upsertState(target, id)
		if s != nil {
			s.Kind = parseStateKind(annotation)
		}
		return nil
	}
	if len(parts) >= 1 {
		upsertState(target, parts[0])
	}
	return nil
}

func (p *parser) parseAliasDecl(rest string, target *[]diagram.StateDef) error {
	endQuote := strings.Index(rest[1:], "\"")
	if endQuote < 0 {
		return fmt.Errorf("unterminated quote in state declaration")
	}
	label := rest[1 : endQuote+1]
	after := strings.TrimSpace(rest[endQuote+2:])
	if id, ok := strings.CutPrefix(after, "as "); ok {
		id = strings.TrimSpace(id)
		s := upsertState(target, id)
		if s != nil {
			s.Label = label
		}
	}
	return nil
}

func (p *parser) parseCompositeBody(target *[]diagram.StateDef) error {
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			return nil
		}
		if err := p.parseLine(line, target); err != nil {
			return err
		}
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading composite state: %w", err)
	}
	return fmt.Errorf("unclosed composite state")
}

func parseTransition(line string) (diagram.StateTransition, bool) {
	idx := strings.Index(line, "-->")
	if idx < 0 {
		return diagram.StateTransition{}, false
	}
	from := strings.TrimSpace(line[:idx])
	rest := strings.TrimSpace(line[idx+3:])
	to := rest
	label := ""
	if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
		to = strings.TrimSpace(rest[:colonIdx])
		// Mermaid uses literal `\n` as a line-break in transition labels.
		label = parserutil.ExpandLineBreaks(strings.TrimSpace(rest[colonIdx+1:]))
	}
	if from == "" || to == "" {
		return diagram.StateTransition{}, false
	}
	return diagram.StateTransition{From: from, To: to, Label: label}, true
}

func parseStateKind(annotation string) diagram.StateKind {
	switch strings.ToLower(annotation) {
	case "fork":
		return diagram.StateKindFork
	case "join":
		return diagram.StateKindJoin
	case "choice":
		return diagram.StateKindChoice
	default:
		return diagram.StateKindNormal
	}
}
