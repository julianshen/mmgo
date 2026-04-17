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
		diagram:  &diagram.StateDiagram{},
		stateIdx: make(map[string]int),
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
		if err := p.parseLine(line); err != nil {
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
	diagram  *diagram.StateDiagram
	stateIdx map[string]int
	scanner  *bufio.Scanner
	lineNum  int
}

func (p *parser) parseLine(line string) error {
	if rest, ok := strings.CutPrefix(line, "state "); ok {
		return p.parseStateDecl(strings.TrimSpace(rest))
	}
	if t, ok := parseTransition(line); ok {
		p.ensureState(t.From)
		p.ensureState(t.To)
		p.diagram.Transitions = append(p.diagram.Transitions, t)
		return nil
	}
	return nil
}

func (p *parser) parseStateDecl(rest string) error {
	if strings.HasPrefix(rest, "\"") {
		return p.parseAliasDecl(rest)
	}
	if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
		name := strings.TrimSpace(rest[:braceIdx])
		return p.parseCompositeBody(name)
	}
	parts := strings.Fields(rest)
	if len(parts) >= 2 && strings.HasPrefix(parts[1], "<<") {
		id := parts[0]
		annotation := strings.Trim(parts[1], "<>")
		kind := parseStateKind(annotation)
		p.ensureState(id)
		idx := p.stateIdx[id]
		p.diagram.States[idx].Kind = kind
		return nil
	}
	if len(parts) >= 1 {
		p.ensureState(parts[0])
	}
	return nil
}

func (p *parser) parseAliasDecl(rest string) error {
	endQuote := strings.Index(rest[1:], "\"")
	if endQuote < 0 {
		return fmt.Errorf("unterminated quote in state declaration")
	}
	label := rest[1 : endQuote+1]
	after := strings.TrimSpace(rest[endQuote+2:])
	if id, ok := strings.CutPrefix(after, "as "); ok {
		id = strings.TrimSpace(id)
		p.ensureState(id)
		idx := p.stateIdx[id]
		p.diagram.States[idx].Label = label
	}
	return nil
}

func (p *parser) parseCompositeBody(name string) error {
	p.ensureState(name)
	idx := p.stateIdx[name]
	childStates := make(map[string]int)
	addChild := func(id string) {
		if id == "[*]" {
			return
		}
		if _, exists := childStates[id]; exists {
			return
		}
		childStates[id] = len(p.diagram.States[idx].Children)
		p.diagram.States[idx].Children = append(p.diagram.States[idx].Children,
			diagram.StateDef{ID: id, Label: id})
	}
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			return nil
		}
		if rest, ok := strings.CutPrefix(line, "state "); ok {
			rest = strings.TrimSpace(rest)
			if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
				childName := strings.TrimSpace(rest[:braceIdx])
				if err := p.parseCompositeBody(childName); err != nil {
					return err
				}
				addChild(childName)
				continue
			}
			parts := strings.Fields(rest)
			if len(parts) >= 1 {
				addChild(parts[0])
			}
			continue
		}
		if t, ok := parseTransition(line); ok {
			addChild(t.From)
			addChild(t.To)
			p.diagram.Transitions = append(p.diagram.Transitions, t)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading composite state %q: %w", name, err)
	}
	return fmt.Errorf("unclosed composite state %q", name)
}

func (p *parser) ensureState(id string) {
	if id == "[*]" {
		return
	}
	if _, ok := p.stateIdx[id]; ok {
		return
	}
	p.stateIdx[id] = len(p.diagram.States)
	p.diagram.States = append(p.diagram.States, diagram.StateDef{ID: id, Label: id})
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
