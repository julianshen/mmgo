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
	if t, ok := parseTransition(line); ok {
		upsertState(target, t.From)
		upsertState(target, t.To)
		p.diagram.Transitions = append(p.diagram.Transitions, t)
		return nil
	}
	return nil
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
		label = strings.TrimSpace(rest[colonIdx+1:])
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
