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
// PRIMARY KEY; the asterisk is stripped from Name and ERKeyPK is
// added to Keys. Comma-separated key constraints (PK, FK, UK)
// land in Keys in source order; Key is set to Keys[0] for back-
// compat with single-key consumers. A trailing quoted run is the
// comment; surrounding double quotes are stripped.
func parseAttribute(line string) diagram.ERAttribute {
	attr := diagram.ERAttribute{}
	// Pull off a quoted trailing comment first; this lets the rest
	// of the line use simple whitespace splitting without worrying
	// about embedded spaces inside the comment.
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
		// `*name` shorthand for PK.
		if strings.HasPrefix(name, "*") {
			name = strings.TrimPrefix(name, "*")
			attr.Keys = append(attr.Keys, diagram.ERKeyPK)
		}
		attr.Name = name
	}
	// Remaining tokens are key constraints, possibly comma-separated
	// (`PK, FK`). Join them to handle the comma-with-space form, then
	// re-split on commas.
	if len(parts) >= 3 {
		raw := strings.Join(parts[2:], " ")
		for _, k := range strings.Split(raw, ",") {
			switch strings.ToUpper(strings.TrimSpace(k)) {
			case "PK":
				attr.Keys = append(attr.Keys, diagram.ERKeyPK)
			case "FK":
				attr.Keys = append(attr.Keys, diagram.ERKeyFK)
			case "UK":
				attr.Keys = append(attr.Keys, diagram.ERKeyUK)
			}
		}
	}
	if len(attr.Keys) > 0 {
		attr.Key = attr.Keys[0]
	}
	return attr
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
		FromCard: glyphToCard(leftGlyph, true),
		ToCard:   glyphToCard(rightGlyph, false),
		Label:    label,
	}, true
}

type arrowSpan struct {
	start, end int
}

// findCardinalityArrow scans for the longest substring of the form
// `<2-char-glyph><2-char-line><2-char-glyph>` (length 6) anywhere in
// the line, where each glyph and the line conform to ER syntax.
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

// Left-side glyphs (mostly mirror the right side; keeping symmetric
// helpers makes the call sites read clearly).
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
// `leftSide` flips the interpretation of asymmetric glyphs (e.g.,
// `}o` is zero-or-more on the left, `o{` is the same shape on the
// right).
func glyphToCard(g string, leftSide bool) diagram.ERCardinality {
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
	_ = leftSide
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
