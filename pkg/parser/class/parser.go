package class

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.ClassDiagram, error) {
	p := &parser{
		diagram:  &diagram.ClassDiagram{},
		classIdx: make(map[string]int),
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
			if line != "classDiagram" {
				return nil, fmt.Errorf("line %d: expected 'classDiagram' header, got %q", p.lineNum, line)
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
		return nil, fmt.Errorf("missing classDiagram header")
	}
	return p.diagram, nil
}

type parser struct {
	diagram  *diagram.ClassDiagram
	classIdx map[string]int
	scanner  *bufio.Scanner
	lineNum  int
}

func (p *parser) parseLine(line string) error {
	if rest, ok := strings.CutPrefix(line, "class "); ok {
		rest = strings.TrimSpace(rest)
		if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
			name := strings.TrimSpace(rest[:braceIdx])
			return p.parseClassBody(name)
		}
		p.ensureClass(rest)
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "direction "); ok {
		switch strings.TrimSpace(rest) {
		case "TB":
			p.diagram.Direction = diagram.DirectionTB
		case "BT":
			p.diagram.Direction = diagram.DirectionBT
		case "LR":
			p.diagram.Direction = diagram.DirectionLR
		case "RL":
			p.diagram.Direction = diagram.DirectionRL
		}
		return nil
	}
	if rel, ok := parseRelation(line); ok {
		p.ensureClass(rel.From)
		p.ensureClass(rel.To)
		p.diagram.Relations = append(p.diagram.Relations, rel)
		return nil
	}
	return nil
}

func (p *parser) parseClassBody(name string) error {
	p.ensureClass(name)
	idx := p.classIdx[name]
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			return nil
		}
		if strings.HasPrefix(line, "<<") && strings.HasSuffix(line, ">>") {
			ann := strings.TrimPrefix(strings.TrimSuffix(line, ">>"), "<<")
			p.diagram.Classes[idx].Annotation = parseAnnotation(ann)
			continue
		}
		p.diagram.Classes[idx].Members = append(p.diagram.Classes[idx].Members, parseMember(line))
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading class body for %q: %w", name, err)
	}
	return fmt.Errorf("unclosed class body for %q", name)
}

func (p *parser) ensureClass(id string) {
	if _, ok := p.classIdx[id]; ok {
		return
	}
	p.classIdx[id] = len(p.diagram.Classes)
	p.diagram.Classes = append(p.diagram.Classes, diagram.ClassDef{ID: id, Label: id})
}

func parseMember(line string) diagram.ClassMember {
	m := diagram.ClassMember{}
	if len(line) > 0 {
		switch line[0] {
		case '+':
			m.Visibility = diagram.VisibilityPublic
			line = line[1:]
		case '-':
			m.Visibility = diagram.VisibilityPrivate
			line = line[1:]
		case '#':
			m.Visibility = diagram.VisibilityProtected
			line = line[1:]
		case '~':
			m.Visibility = diagram.VisibilityPackage
			line = line[1:]
		}
	}
	if idx := strings.Index(line, "("); idx >= 0 {
		m.IsMethod = true
		// Match the closing `)` by depth so args containing grouped
		// expressions like `execute(callback (x, y))` aren't truncated
		// at the first inner `)`.
		if closeIdx := matchCloseParen(line, idx); closeIdx >= 0 {
			m.Name = strings.TrimSpace(line[:idx])
			m.Args = strings.TrimSpace(line[idx+1 : closeIdx])
			// Allow either `foo() bar` or `foo(): bar`; mermaid accepts both.
			tail := strings.TrimSpace(line[closeIdx+1:])
			m.ReturnType = strings.TrimSpace(strings.TrimPrefix(tail, ":"))
		} else {
			m.Name = strings.TrimSpace(line)
		}
	} else {
		// Preserve fields verbatim. Both `String name` (Java/C#) and
		// `name: String` (TypeScript) are valid mermaid; splitting on
		// whitespace inverts the former, splitting on `:` mangles the
		// latter (`-template: String` → `-String : template:`).
		m.Name = strings.TrimSpace(line)
	}
	return m
}

// matchCloseParen returns the index of the `)` that pairs with the `(`
// at openIdx, or -1 if unbalanced. Tracks nesting depth.
func matchCloseParen(line string, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(line); i++ {
		switch line[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func parseAnnotation(s string) diagram.ClassAnnotation {
	switch strings.ToLower(s) {
	case "interface":
		return diagram.AnnotationInterface
	case "abstract":
		return diagram.AnnotationAbstract
	case "service":
		return diagram.AnnotationService
	case "enum":
		return diagram.AnnotationEnum
	default:
		return diagram.AnnotationNone
	}
}

// arrowGlyph tags one end of a relation arrow. The same glyph can render
// differently depending on direction (e.g., the inheritance triangle is
// written `<|` on the left and `|>` on the right), but the *kind* is the
// same — that's what arrowGlyph captures.
type arrowGlyph int8

const (
	glyphNone           arrowGlyph = iota
	glyphTriangle                  // `<|` or `|>` — inheritance/realization head
	glyphFilledDiamond             // `*` — composition
	glyphHollowDiamond             // `o` — aggregation
	glyphArrowhead                 // `<` or `>` — association/dependency head
)

// arrowMatch is the structured result of tokenizing a relation arrow.
type arrowMatch struct {
	startIdx int // start of the arrow span in the source line
	endIdx   int // index just past the last arrow character
	left     arrowGlyph
	right    arrowGlyph
	dashed   bool // true for `..` line, false for `--`
}

func parseRelation(line string) (diagram.ClassRelation, bool) {
	m, ok := tokenizeArrow(line)
	if !ok {
		return diagram.ClassRelation{}, false
	}
	rt, reverse, bidir, ok := classifyArrow(m)
	if !ok {
		return diagram.ClassRelation{}, false
	}

	leftRaw := strings.TrimSpace(line[:m.startIdx])
	rightRaw := strings.TrimSpace(line[m.endIdx:])

	from, fromCard := extractCardinality(leftRaw)
	to, label, toCard := extractRightSide(rightRaw)
	if from == "" || to == "" {
		return diagram.ClassRelation{}, false
	}

	return diagram.ClassRelation{
		From:            from,
		To:              to,
		RelationType:    rt,
		Label:           label,
		FromCardinality: fromCard,
		ToCardinality:   toCard,
		Reverse:         reverse,
		Bidirectional:   bidir,
	}, true
}

// tokenizeArrow finds the relation arrow inside a line by locating the
// line core (a contiguous run of `--` or `..`) and walking outward to
// pick up any glyph characters bracketing it. Glyphs are restricted to
// the chars `< > | * o` so they can't be confused with class names.
//
// We deliberately do not anchor to whitespace — `Animal<|--Dog` (no
// spaces, as some users write) tokenizes the same as `Animal <|-- Dog`.
func tokenizeArrow(line string) (arrowMatch, bool) {
	// Compute "in-string" mask first so cardinality literals like
	// "0..*" don't get tokenized as a dashed arrow.
	inStr := make([]bool, len(line))
	open := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			open = !open
			continue
		}
		inStr[i] = open
	}

	bestLen := 0
	var best arrowMatch
	for i := 0; i < len(line)-1; i++ {
		if inStr[i] || inStr[i+1] {
			continue
		}
		c := line[i]
		if c != '-' && c != '.' {
			continue
		}
		if line[i+1] != c {
			continue
		}
		// `i..i+1` is a candidate line core. Extend in case of `---`
		// (we still treat the line as solid; only its first 2 chars
		// matter for meaning).
		j := i + 2
		for j < len(line) && line[j] == c {
			j++
		}
		left, lstart := scanLeftGlyph(line, i)
		right, rend := scanRightGlyph(line, j)
		span := rend - lstart
		if span > bestLen {
			bestLen = span
			best = arrowMatch{
				startIdx: lstart,
				endIdx:   rend,
				left:     left,
				right:    right,
				dashed:   c == '.',
			}
		}
	}
	if bestLen == 0 {
		return arrowMatch{}, false
	}
	return best, true
}

// scanLeftGlyph reads up to two glyph characters immediately preceding
// the line core and returns the glyph kind plus the new start index.
func scanLeftGlyph(line string, lineStart int) (arrowGlyph, int) {
	if lineStart == 0 {
		return glyphNone, lineStart
	}
	// `<|` is two chars; check it before the single-char glyphs.
	if lineStart >= 2 && line[lineStart-2] == '<' && line[lineStart-1] == '|' {
		return glyphTriangle, lineStart - 2
	}
	switch line[lineStart-1] {
	case '*':
		return glyphFilledDiamond, lineStart - 1
	case 'o':
		// Disambiguate against an identifier ending in `o` like `Foo--Bar`:
		// require either start-of-line or a non-identifier char before it.
		if lineStart-1 == 0 || !isIdentChar(line[lineStart-2]) {
			return glyphHollowDiamond, lineStart - 1
		}
	case '<':
		return glyphArrowhead, lineStart - 1
	}
	return glyphNone, lineStart
}

// scanRightGlyph reads up to two glyph characters immediately following
// the line core and returns the glyph kind plus the new end index.
func scanRightGlyph(line string, lineEnd int) (arrowGlyph, int) {
	if lineEnd >= len(line) {
		return glyphNone, lineEnd
	}
	if lineEnd+1 < len(line) && line[lineEnd] == '|' && line[lineEnd+1] == '>' {
		return glyphTriangle, lineEnd + 2
	}
	switch line[lineEnd] {
	case '*':
		return glyphFilledDiamond, lineEnd + 1
	case 'o':
		if lineEnd+1 == len(line) || !isIdentChar(line[lineEnd+1]) {
			return glyphHollowDiamond, lineEnd + 1
		}
	case '>':
		return glyphArrowhead, lineEnd + 1
	}
	return glyphNone, lineEnd
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_'
}

// classifyArrow maps a glyph pair + line style to a RelationType plus
// the Reverse / Bidirectional flags. Unsupported glyph combinations
// (e.g., a triangle on one end and a diamond on the other) are rejected.
func classifyArrow(m arrowMatch) (rt diagram.RelationType, reverse, bidir bool, ok bool) {
	// Both ends carry a glyph → bidirectional. Glyphs must agree on kind.
	if m.left != glyphNone && m.right != glyphNone {
		if m.left != m.right {
			return 0, false, false, false
		}
		switch m.left {
		case glyphTriangle:
			if m.dashed {
				return diagram.RelationTypeRealization, false, true, true
			}
			return diagram.RelationTypeInheritance, false, true, true
		case glyphFilledDiamond:
			return diagram.RelationTypeComposition, false, true, true
		case glyphHollowDiamond:
			return diagram.RelationTypeAggregation, false, true, true
		case glyphArrowhead:
			if m.dashed {
				return diagram.RelationTypeDependency, false, true, true
			}
			return diagram.RelationTypeAssociation, false, true, true
		}
		return 0, false, false, false
	}

	// Single-end glyph: forward is whichever side matches Mermaid's
	// canonical literal for that (glyph, line) pair. Notably, the
	// canonical side is not consistent across types — `<|--` (inheritance)
	// has the triangle on the LEFT, but `..|>` (realization) has it on
	// the RIGHT. canonicalRightSide encodes that table.
	if m.left != glyphNone {
		rt, ok = glyphToRelation(m.left, m.dashed)
		return rt, canonicalRightSide(m.left, m.dashed), false, ok
	}
	if m.right != glyphNone {
		rt, ok = glyphToRelation(m.right, m.dashed)
		return rt, !canonicalRightSide(m.right, m.dashed), false, ok
	}
	// No glyph at either end: plain link / dashed link.
	if m.dashed {
		return diagram.RelationTypeDashedLink, false, false, true
	}
	return diagram.RelationTypeLink, false, false, true
}

// canonicalRightSide returns true when Mermaid's canonical literal for
// the given (glyph, line) pair places the glyph on the right end. It's
// a small lookup table — the only "right canonical" cases are the
// arrowhead heads (`-->`, `..>`) and realization (`..|>`).
func canonicalRightSide(g arrowGlyph, dashed bool) bool {
	switch g {
	case glyphArrowhead:
		return true
	case glyphTriangle:
		return dashed // realization: `..|>`
	}
	return false
}

func glyphToRelation(g arrowGlyph, dashed bool) (diagram.RelationType, bool) {
	switch g {
	case glyphTriangle:
		if dashed {
			return diagram.RelationTypeRealization, true
		}
		return diagram.RelationTypeInheritance, true
	case glyphFilledDiamond:
		return diagram.RelationTypeComposition, true
	case glyphHollowDiamond:
		return diagram.RelationTypeAggregation, true
	case glyphArrowhead:
		if dashed {
			return diagram.RelationTypeDependency, true
		}
		return diagram.RelationTypeAssociation, true
	}
	return 0, false
}

func extractCardinality(s string) (id, cardinality string) {
	if idx := strings.Index(s, "\""); idx >= 0 {
		endIdx := strings.Index(s[idx+1:], "\"")
		if endIdx >= 0 {
			cardinality = s[idx+1 : idx+1+endIdx]
			id = strings.TrimSpace(s[:idx])
			return id, cardinality
		}
	}
	return s, ""
}

func extractRightSide(s string) (id, label, cardinality string) {
	if idx := strings.Index(s, ":"); idx >= 0 {
		label = strings.TrimSpace(s[idx+1:])
		s = strings.TrimSpace(s[:idx])
	}
	if idx := strings.Index(s, "\""); idx >= 0 {
		endIdx := strings.Index(s[idx+1:], "\"")
		if endIdx >= 0 {
			cardinality = s[idx+1 : idx+1+endIdx]
			id = strings.TrimSpace(s[idx+1+endIdx+1:])
			if id == "" {
				id = strings.TrimSpace(s[:idx])
			}
			return id, label, cardinality
		}
	}
	return s, label, ""
}
