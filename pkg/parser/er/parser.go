package er

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.ERDiagram, error) {
	p := &parser{
		diagram:   &diagram.ERDiagram{},
		entityIdx: make(map[string]int),
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
			if line != "erDiagram" {
				return nil, fmt.Errorf("line %d: expected 'erDiagram' header, got %q", p.lineNum, line)
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
		return nil, fmt.Errorf("missing erDiagram header")
	}
	return p.diagram, nil
}

type parser struct {
	diagram   *diagram.ERDiagram
	entityIdx map[string]int
	scanner   *bufio.Scanner
	lineNum   int
}

func (p *parser) parseLine(line string) error {
	// Relationships first — their cardinality markers can contain `{`.
	if rel, ok := parseRelationship(line); ok {
		p.ensureEntity(rel.From)
		p.ensureEntity(rel.To)
		p.diagram.Relationships = append(p.diagram.Relationships, rel)
		return nil
	}
	if strings.HasSuffix(strings.TrimSpace(line), "{") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "{"))
		return p.parseEntityBody(name)
	}
	return nil
}

func (p *parser) parseEntityBody(name string) error {
	p.ensureEntity(name)
	idx := p.entityIdx[name]
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			return nil
		}
		attr := parseAttribute(line)
		p.diagram.Entities[idx].Attributes = append(p.diagram.Entities[idx].Attributes, attr)
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading entity %q: %w", name, err)
	}
	return fmt.Errorf("unclosed entity %q", name)
}

func (p *parser) ensureEntity(name string) {
	if _, ok := p.entityIdx[name]; ok {
		return
	}
	p.entityIdx[name] = len(p.diagram.Entities)
	p.diagram.Entities = append(p.diagram.Entities, diagram.EREntity{Name: name})
}

func parseAttribute(line string) diagram.ERAttribute {
	parts := strings.Fields(line)
	attr := diagram.ERAttribute{}
	if len(parts) >= 1 {
		attr.Type = parts[0]
	}
	if len(parts) >= 2 {
		attr.Name = parts[1]
	}
	if len(parts) >= 3 {
		switch strings.ToUpper(parts[2]) {
		case "PK":
			attr.Key = diagram.ERKeyPK
		case "FK":
			attr.Key = diagram.ERKeyFK
		case "UK":
			attr.Key = diagram.ERKeyUK
		}
		if attr.Key == diagram.ERKeyNone {
			attr.Comment = strings.Join(parts[2:], " ")
		}
	}
	if len(parts) >= 4 && attr.Key != diagram.ERKeyNone {
		attr.Comment = strings.Join(parts[3:], " ")
	}
	return attr
}

// Relationship arrows: cardinality markers around -- or ..
// Format: ENTITY1 <left-card>--<right-card> ENTITY2 : "label"
// Cardinality markers: || (exactly-one), |o/o| (zero-or-one),
// }|/|{ (one-or-more), }o/o{ (zero-or-more)
var cardPairs = []struct {
	left  string
	right string
	lCard diagram.ERCardinality
	rCard diagram.ERCardinality
}{
	{"||--||", "", diagram.ERCardExactlyOne, diagram.ERCardExactlyOne},
	{"||--o{", "", diagram.ERCardExactlyOne, diagram.ERCardZeroOrMore},
	{"||--|{", "", diagram.ERCardExactlyOne, diagram.ERCardOneOrMore},
	{"||--o|", "", diagram.ERCardExactlyOne, diagram.ERCardZeroOrOne},
	{"}o--||", "", diagram.ERCardZeroOrMore, diagram.ERCardExactlyOne},
	{"}|--||", "", diagram.ERCardOneOrMore, diagram.ERCardExactlyOne},
	{"o|--||", "", diagram.ERCardZeroOrOne, diagram.ERCardExactlyOne},
	{"}o--o{", "", diagram.ERCardZeroOrMore, diagram.ERCardZeroOrMore},
	{"}|--|{", "", diagram.ERCardOneOrMore, diagram.ERCardOneOrMore},
	{"||..||", "", diagram.ERCardExactlyOne, diagram.ERCardExactlyOne},
	{"||..o{", "", diagram.ERCardExactlyOne, diagram.ERCardZeroOrMore},
	{"||..|{", "", diagram.ERCardExactlyOne, diagram.ERCardOneOrMore},
	{"}o..||", "", diagram.ERCardZeroOrMore, diagram.ERCardExactlyOne},
	{"}|..||", "", diagram.ERCardOneOrMore, diagram.ERCardExactlyOne},
	{"}|..|{", "", diagram.ERCardOneOrMore, diagram.ERCardOneOrMore},
}

func parseRelationship(line string) (diagram.ERRelationship, bool) {
	for _, cp := range cardPairs {
		idx := strings.Index(line, cp.left)
		if idx < 0 {
			continue
		}
		from := strings.TrimSpace(line[:idx])
		rest := strings.TrimSpace(line[idx+len(cp.left):])
		to, label := splitRelLabel(rest)
		if from == "" || to == "" {
			continue
		}
		return diagram.ERRelationship{
			From: from, To: to,
			FromCard: cp.lCard, ToCard: cp.rCard,
			Label: label,
		}, true
	}
	return diagram.ERRelationship{}, false
}

func splitRelLabel(s string) (to, label string) {
	if idx := strings.Index(s, ":"); idx >= 0 {
		to = strings.TrimSpace(s[:idx])
		label = strings.Trim(strings.TrimSpace(s[idx+1:]), "\"")
		return to, label
	}
	return strings.TrimSpace(s), ""
}
