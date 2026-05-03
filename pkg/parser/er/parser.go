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
	if rest, ok := strings.CutPrefix(line, "direction "); ok {
		dir, err := parserutil.ParseDirection(strings.TrimSpace(rest))
		if err != nil {
			return err
		}
		p.diagram.Direction = dir
		return nil
	}
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

// parseAttribute reads an attribute line of the form
//
//	type name [key[, key…]] ["comment text"]
//
// `*name` is a Mermaid shorthand for marking the attribute as
// PRIMARY KEY; the asterisk is stripped and ERKeyPK is added to
// Keys. Comma-separated constraints (PK, FK, UK) land in Keys in
// source order. Duplicate keys are deduplicated so `*id PK, FK`
// yields [PK FK], not [PK PK FK]. A trailing quoted run is the
// comment; the surrounding double quotes are stripped.
func parseAttribute(line string) diagram.ERAttribute {
	attr := diagram.ERAttribute{}
	if i := strings.Index(line, `"`); i >= 0 {
		if j := strings.LastIndex(line, `"`); j > i {
			attr.Comment = line[i+1 : j]
			line = strings.TrimSpace(line[:i])
		}
	}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return attr
	}
	attr.Type = parts[0]
	if len(parts) >= 2 {
		name := parts[1]
		if strings.HasPrefix(name, "*") {
			name = strings.TrimPrefix(name, "*")
			attr.Keys = appendUniqueKey(attr.Keys, diagram.ERKeyPK)
		}
		attr.Name = name
	}
	if len(parts) >= 3 {
		raw := strings.Join(parts[2:], " ")
		for _, k := range strings.Split(raw, ",") {
			if key, ok := parseERKey(k); ok {
				attr.Keys = appendUniqueKey(attr.Keys, key)
			}
		}
	}
	if len(attr.Keys) > 0 {
		attr.Key = attr.Keys[0]
	}
	return attr
}

// parseERKey converts a textual key constraint (PK / FK / UK) into
// its enum value. Whitespace and case are tolerated; unknown tokens
// return ok=false so the caller can ignore them.
func parseERKey(s string) (diagram.ERAttributeKey, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "PK":
		return diagram.ERKeyPK, true
	case "FK":
		return diagram.ERKeyFK, true
	case "UK":
		return diagram.ERKeyUK, true
	}
	return diagram.ERKeyNone, false
}

// appendUniqueKey appends k to keys unless it's already present.
// Used to dedupe the `*name PK` case (asterisk + explicit PK both
// add ERKeyPK).
func appendUniqueKey(keys []diagram.ERAttributeKey, k diagram.ERAttributeKey) []diagram.ERAttributeKey {
	for _, existing := range keys {
		if existing == k {
			return keys
		}
	}
	return append(keys, k)
}

// parseRelationship recognises the cardinality arrow as
// `[leftGlyph][line][rightGlyph]` where each glyph is exactly two
// chars from the set {||, |o, o|, }|, |{, }o, o{} and the line is
// either `--` (identifying) or `..` (non-identifying). This covers
// the full 4×4×2 = 32 combinations without enumerating each pair.
func parseRelationship(line string) (diagram.ERRelationship, bool) {
	span, leftGlyph, rightGlyph, ok := findCardinalityArrow(line)
	if !ok {
		return diagram.ERRelationship{}, false
	}
	from := strings.TrimSpace(line[:span.start])
	rest := strings.TrimSpace(line[span.end:])
	to, label := splitRelLabel(rest)
	if from == "" || to == "" {
		return diagram.ERRelationship{}, false
	}
	return diagram.ERRelationship{
		From: from, To: to,
		FromCard: glyphToCard(leftGlyph),
		ToCard:   glyphToCard(rightGlyph),
		Label:    label,
	}, true
}

type arrowSpan struct {
	start, end int
}

// findCardinalityArrow scans for the leftmost 6-char cardinality
// arrow of the form `<2-char-glyph><2-char-line><2-char-glyph>`.
// All valid arrows are exactly 6 chars, so leftmost = unique match
// per relationship line.
func findCardinalityArrow(line string) (arrowSpan, string, string, bool) {
	for i := 0; i+6 <= len(line); i++ {
		left := line[i : i+2]
		mid := line[i+2 : i+4]
		right := line[i+4 : i+6]
		if !isLeftGlyph(left) || !isLine(mid) || !isRightGlyph(right) {
			continue
		}
		return arrowSpan{start: i, end: i + 6}, left, right, true
	}
	return arrowSpan{}, "", "", false
}

func isLine(s string) bool { return s == "--" || s == ".." }

// The `{`/`}` bracket's open side always faces the relationship
// line — so `}|--||` is valid but `|{--||` is not. That asymmetry
// is the only difference between the left and right glyph sets.
func isLeftGlyph(s string) bool {
	switch s {
	case "||", "|o", "o|", "}|", "}o":
		return true
	}
	return false
}

func isRightGlyph(s string) bool {
	switch s {
	case "||", "|o", "o|", "|{", "o{":
		return true
	}
	return false
}

// glyphToCard maps a 2-char cardinality glyph to its enum value.
// Side doesn't influence the mapping — the bracket-open-side rule
// above guarantees each glyph is unambiguous.
func glyphToCard(g string) diagram.ERCardinality {
	switch g {
	case "||":
		return diagram.ERCardExactlyOne
	case "|o", "o|":
		return diagram.ERCardZeroOrOne
	case "}o", "o{":
		return diagram.ERCardZeroOrMore
	case "}|", "|{":
		return diagram.ERCardOneOrMore
	}
	return diagram.ERCardUnknown
}

func splitRelLabel(s string) (to, label string) {
	if idx := strings.Index(s, ":"); idx >= 0 {
		to = strings.TrimSpace(s[:idx])
		label = strings.Trim(strings.TrimSpace(s[idx+1:]), "\"")
		return to, label
	}
	return strings.TrimSpace(s), ""
}
