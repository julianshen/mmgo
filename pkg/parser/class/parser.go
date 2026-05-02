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
		hdr, hasBody := parseClassHeader(rest)
		p.declareClass(hdr.id, hdr.label, hdr.generic)
		if hasBody {
			return p.parseClassBody(hdr.id)
		}
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "direction "); ok {
		dir, err := parserutil.ParseDirection(strings.TrimSpace(rest))
		if err != nil {
			return err
		}
		p.diagram.Direction = dir
		return nil
	}
	rel, hasArrow, err := parseRelation(line)
	if err != nil {
		return err
	}
	if hasArrow {
		p.ensureClass(rel.From)
		p.ensureClass(rel.To)
		p.diagram.Relations = append(p.diagram.Relations, rel)
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

// ensureClass registers `id` if not already present. Idempotent; used
// when a relation references a class before any explicit declaration.
func (p *parser) ensureClass(id string) int {
	if idx, ok := p.classIdx[id]; ok {
		return idx
	}
	p.classIdx[id] = len(p.diagram.Classes)
	p.diagram.Classes = append(p.diagram.Classes, diagram.ClassDef{ID: id, Label: id})
	return p.classIdx[id]
}

// declareClass registers a class with explicit metadata. Non-empty
// label and generic override what ensureClass set, so a relation
// auto-registering a class first and a later `class Foo["…"]~T~`
// declaration still wins.
func (p *parser) declareClass(id, label, generic string) {
	idx := p.ensureClass(id)
	if label != "" {
		p.diagram.Classes[idx].Label = label
	}
	if generic != "" {
		p.diagram.Classes[idx].Generic = generic
	}
}

// classHeader is the parsed result of `class NAME[...]~...~`.
type classHeader struct {
	id      string
	label   string // from `["..."]`
	generic string // from `~...~`
}

// parseClassHeader splits `Foo["My Label"]~T~` (or any subset) into
// id / label / generic and reports whether a `{` follows. Body content
// is left for parseClassBody to consume from the scanner.
func parseClassHeader(rest string) (classHeader, bool) {
	hasBody := false
	if i := strings.IndexByte(rest, '{'); i >= 0 {
		rest = strings.TrimSpace(rest[:i])
		hasBody = true
	}
	var label string
	if i := strings.IndexByte(rest, '['); i >= 0 {
		if j := strings.LastIndexByte(rest, ']'); j > i {
			inside := rest[i+1 : j]
			if len(inside) >= 2 && inside[0] == '"' && inside[len(inside)-1] == '"' {
				label = inside[1 : len(inside)-1]
				rest = strings.TrimSpace(rest[:i])
			}
		}
	}
	var generic string
	if i := strings.IndexByte(rest, '~'); i >= 0 {
		// Use the LAST `~` so nested generics like `Wrapper~List~int~~`
		// give Generic="List~int~" rather than "List".
		if j := strings.LastIndexByte(rest, '~'); j > i {
			generic = rest[i+1 : j]
			rest = strings.TrimSpace(rest[:i])
		}
	}
	return classHeader{id: rest, label: label, generic: generic}, hasBody
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
			tail = strings.TrimPrefix(tail, ":")
			tail, m.IsStatic, m.IsAbstract = extractMemberModifiers(tail)
			m.ReturnType = strings.TrimSpace(tail)
		} else {
			m.Name = strings.TrimSpace(line)
		}
	} else {
		// Preserve fields verbatim. Both `String name` (Java/C#) and
		// `name: String` (TypeScript) are valid mermaid; splitting on
		// whitespace inverts the former, splitting on `:` mangles the
		// latter (`-template: String` → `-String : template:`).
		var stripped string
		stripped, m.IsStatic, m.IsAbstract = extractMemberModifiers(line)
		m.Name = strings.TrimSpace(stripped)
	}
	return m
}

// extractMemberModifiers strips trailing-or-embedded `$` (static) and
// `*` (abstract) markers from a member text. The Mermaid grammar puts
// the marker either right after the member's name (`pi$ double`) or
// after its type (`name double$`); both reduce to "remove the rune".
// Identifier text never contains either character, so we drop all
// occurrences and report which kind we saw.
func extractMemberModifiers(s string) (cleaned string, isStatic, isAbstract bool) {
	if strings.ContainsRune(s, '$') {
		isStatic = true
		s = strings.ReplaceAll(s, "$", "")
	}
	if strings.ContainsRune(s, '*') {
		isAbstract = true
		s = strings.ReplaceAll(s, "*", "")
	}
	// Collapse the double-space that "name$ double" → "name  double"
	// produces after marker removal.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s, isStatic, isAbstract
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

// parseRelation returns (relation, hasArrow, error). Once the line is
// recognised as an arrow (tokenizeArrow matched), a downstream failure
// — unsupported glyph pair, missing endpoint — surfaces as an error
// rather than a silent drop, since the user clearly intended to write
// a relation.
func parseRelation(line string) (diagram.ClassRelation, bool, error) {
	m, ok := tokenizeArrow(line)
	if !ok {
		return diagram.ClassRelation{}, false, nil
	}
	rt, dir, ok := classifyArrow(m)
	if !ok {
		return diagram.ClassRelation{}, false, fmt.Errorf("unsupported relation arrow in %q", line)
	}

	leftRaw := strings.TrimSpace(line[:m.startIdx])
	rightRaw := strings.TrimSpace(line[m.endIdx:])

	from, fromCard := extractCardinality(leftRaw)
	to, label, toCard := extractRightSide(rightRaw)
	if from == "" || to == "" {
		return diagram.ClassRelation{}, false, fmt.Errorf("relation %q is missing an endpoint", line)
	}

	return diagram.ClassRelation{
		From:            from,
		To:              to,
		RelationType:    rt,
		Label:           label,
		FromCardinality: fromCard,
		ToCardinality:   toCard,
		Direction:       dir,
	}, true, nil
}

// tokenizeArrow finds the relation arrow inside a line by locating the
// line core (a contiguous run of `--` or `..`) and walking outward to
// pick up any glyph characters bracketing it. Glyphs are restricted to
// the chars `< > | * o` so they can't be confused with class names.
//
// We deliberately do not anchor to whitespace — `Animal<|--Dog` (no
// spaces, as some users write) tokenizes the same as `Animal <|-- Dog`.
//
// Cardinality literals like "0..*" contain arrow-shaped chars; we track
// whether we're inside a `"…"` run as we scan and skip arrow-detection
// on those positions.
func tokenizeArrow(line string) (arrowMatch, bool) {
	bestLen := 0
	var best arrowMatch
	inString := false
	for i := 0; i < len(line)-1; i++ {
		if line[i] == '"' {
			inString = !inString
			continue
		}
		if inString {
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

// classifyArrow maps a glyph pair + line style to a RelationType and
// RelationDirection. Unsupported glyph combinations (e.g., a triangle
// on one end and a diamond on the other) are rejected with ok=false.
func classifyArrow(m arrowMatch) (rt diagram.RelationType, dir diagram.RelationDirection, ok bool) {
	// Both ends carry a glyph → bidirectional. Glyphs must agree on kind.
	if m.left != glyphNone && m.right != glyphNone {
		if m.left != m.right {
			return 0, 0, false
		}
		switch m.left {
		case glyphTriangle:
			if m.dashed {
				return diagram.RelationTypeRealization, diagram.RelationBidirectional, true
			}
			return diagram.RelationTypeInheritance, diagram.RelationBidirectional, true
		case glyphFilledDiamond:
			return diagram.RelationTypeComposition, diagram.RelationBidirectional, true
		case glyphHollowDiamond:
			return diagram.RelationTypeAggregation, diagram.RelationBidirectional, true
		case glyphArrowhead:
			if m.dashed {
				return diagram.RelationTypeDependency, diagram.RelationBidirectional, true
			}
			return diagram.RelationTypeAssociation, diagram.RelationBidirectional, true
		}
		return 0, 0, false
	}

	// Single-end glyph: forward is whichever side matches Mermaid's
	// canonical literal. The canonical side is not consistent across
	// types — `<|--` (inheritance) puts the triangle on the LEFT, but
	// `..|>` (realization) puts it on the RIGHT. canonicalRightSide
	// encodes that small table.
	if m.left != glyphNone {
		rt, ok = glyphToRelation(m.left, m.dashed)
		if canonicalRightSide(m.left, m.dashed) {
			return rt, diagram.RelationReverse, ok
		}
		return rt, diagram.RelationForward, ok
	}
	if m.right != glyphNone {
		rt, ok = glyphToRelation(m.right, m.dashed)
		if canonicalRightSide(m.right, m.dashed) {
			return rt, diagram.RelationForward, ok
		}
		return rt, diagram.RelationReverse, ok
	}
	// No glyph at either end: plain link / dashed link.
	if m.dashed {
		return diagram.RelationTypeDashedLink, diagram.RelationForward, true
	}
	return diagram.RelationTypeLink, diagram.RelationForward, true
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
