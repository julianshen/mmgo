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
		diagram: &diagram.ClassDiagram{
			CSSClasses: make(map[string]string),
		},
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
	if v, ok := parserutil.MatchKeywordValue(line, "title"); ok {
		p.diagram.Title = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
		p.diagram.AccTitle = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
		p.diagram.AccDescr = v
		return nil
	}
	if strings.HasPrefix(line, "classDef ") {
		return p.parseClassDef(line)
	}
	if strings.HasPrefix(line, "style ") {
		return p.parseStyleRule(line)
	}
	if strings.HasPrefix(line, "cssClass ") {
		return p.parseCSSClassBinding(line)
	}
	if strings.HasPrefix(line, "click ") {
		return p.parseClick(line)
	}
	if strings.HasPrefix(line, "link ") {
		return p.parseLinkOrCallback(line, false)
	}
	if strings.HasPrefix(line, "callback ") {
		return p.parseLinkOrCallback(line, true)
	}
	if rest, ok := cutNamespaceKeyword(line); ok {
		return p.parseNamespace(rest)
	}
	if rest, ok := strings.CutPrefix(line, "class "); ok {
		rest = strings.TrimSpace(rest)
		hdr, hasBody, err := parseClassHeader(rest)
		if err != nil {
			return err
		}
		if err := p.declareClass(hdr.id, hdr.label, hdr.generic); err != nil {
			return err
		}
		idx := p.classIdx[hdr.id]
		if hdr.annotation != diagram.AnnotationNone {
			p.diagram.Classes[idx].Annotation = hdr.annotation
		}
		if hdr.cssClass != "" {
			p.diagram.Classes[idx].CSSClasses = append(p.diagram.Classes[idx].CSSClasses, hdr.cssClass)
		}
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
	if strings.HasPrefix(line, "note ") || line == "note" {
		return p.parseNote(line)
	}
	if id, ann, ok := parseBareAnnotation(line); ok {
		p.ensureClass(id)
		p.diagram.Classes[p.classIdx[id]].Annotation = ann
		return nil
	}
	if id, memberLine, ok := parseSingleLineMember(line); ok {
		p.ensureClass(id)
		idx := p.classIdx[id]
		p.diagram.Classes[idx].Members = append(p.diagram.Classes[idx].Members, parseMember(memberLine))
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

// parseNote handles `note "text"` and `note for ClassName "text"`.
// `\n` inside the quoted body becomes a real newline so renderers
// can split on it directly.
func (p *parser) parseNote(line string) error {
	rest := strings.TrimSpace(strings.TrimPrefix(line, "note"))
	target := ""
	if r, ok := strings.CutPrefix(rest, "for "); ok {
		// `note for ClassName "text"` — split on the first quote.
		q := strings.IndexByte(r, '"')
		if q < 0 {
			return fmt.Errorf("note %q: missing quoted text", line)
		}
		target = strings.TrimSpace(r[:q])
		if target == "" {
			return fmt.Errorf("note %q: missing target class", line)
		}
		// Mermaid's grammar requires a bare identifier here. Reject
		// quoted or whitespace-containing targets so the text doesn't
		// silently absorb part of the body.
		if strings.ContainsAny(target, "\"' \t") {
			return fmt.Errorf("note %q: target class must be a bare identifier, got %q", line, target)
		}
		rest = r[q:]
		p.ensureClass(target)
	}
	text := parserutil.Unquote(rest)
	if text == rest {
		return fmt.Errorf("note %q: text must be quoted", line)
	}
	p.diagram.Notes = append(p.diagram.Notes, diagram.ClassNote{
		Text: parserutil.ExpandLineBreaks(text),
		For:  target,
	})
	return nil
}

// parseBareAnnotation matches `ClassName <<Annotation>>` and returns
// (id, annotation, true) on success. An unrecognised annotation
// keyword is treated as AnnotationNone — the caller decides whether
// to apply it (we still consume the line so it doesn't fall through
// to the relation parser).
func parseBareAnnotation(line string) (string, diagram.ClassAnnotation, bool) {
	open := strings.Index(line, "<<")
	close := strings.Index(line, ">>")
	if open < 0 || close < open {
		return "", diagram.AnnotationNone, false
	}
	id := strings.TrimSpace(line[:open])
	if id == "" {
		return "", diagram.AnnotationNone, false
	}
	// Reject if the prefix isn't a bare identifier — a relation line
	// like `A <-- B <<...>>` (nonsense, but parseable) should fall
	// through to other matchers.
	if strings.ContainsAny(id, " \t") {
		return "", diagram.AnnotationNone, false
	}
	ann := parseAnnotation(strings.TrimSpace(line[open+2 : close]))
	return id, ann, true
}

func (p *parser) parseClassDef(line string) error {
	name, css, err := parserutil.ParseClassDefLine(line[len("classDef "):])
	if err != nil {
		return err
	}
	p.diagram.CSSClasses[name] = css
	return nil
}

// parseStyleRule stores `style ID CSS` as an inline override on the
// named class. Multiple style lines for the same class accumulate in
// source order; the renderer applies them after classDef references.
//
// `style ID …` references an actual class by ID, so we ensure the
// class exists. classDef, by contrast, defines a free-standing CSS
// class name and never auto-registers a diagram class.
func (p *parser) parseStyleRule(line string) error {
	rest := line[len("style "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("style requires an ID and CSS")
	}
	id := strings.TrimSpace(parts[0])
	p.ensureClass(id)
	p.diagram.Styles = append(p.diagram.Styles, diagram.ClassStyleDef{
		ClassID: id,
		CSS:     parserutil.NormalizeCSS(strings.TrimSpace(parts[1])),
	})
	return nil
}

// parseCSSClassBinding parses `cssClass "id1,id2" className`. The
// quoted-id-list form is the canonical Mermaid syntax; we accept the
// unquoted form too since both appear in the wild.
func (p *parser) parseCSSClassBinding(line string) error {
	rest := strings.TrimSpace(line[len("cssClass "):])
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("cssClass requires id list and class name")
	}
	idList := parserutil.Unquote(strings.TrimSpace(parts[0]))
	cssName := strings.TrimSpace(parts[1])
	for _, id := range strings.Split(idList, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		idx := p.ensureClass(id)
		p.diagram.Classes[idx].CSSClasses = append(p.diagram.Classes[idx].CSSClasses, cssName)
	}
	return nil
}

// cutNamespaceKeyword recognises the `namespace` keyword followed by
// a space OR tab and returns the trailing argument. Both whitespace
// forms are common in real diagrams; using strings.CutPrefix with a
// single-space literal would let `namespace\t…` slip past.
func cutNamespaceKeyword(line string) (string, bool) {
	const kw = "namespace"
	if !strings.HasPrefix(line, kw) || len(line) <= len(kw) {
		return "", false
	}
	switch line[len(kw)] {
	case ' ', '\t':
		return strings.TrimSpace(line[len(kw)+1:]), true
	}
	return "", false
}

// parseNamespace consumes a `namespace NAME { … }` block. Inner
// content is dispatched back to parseLine so the same syntax (class
// declarations, relations, members) is accepted as at top level.
// Classes declared inside still register flat in p.diagram.Classes,
// so cross-namespace relations resolve normally; their IDs are also
// recorded on a ClassNamespace entry for the renderer.
//
// Nested `namespace` blocks are rejected — Mermaid doesn't support
// them and the recursive case would surprise.
func (p *parser) parseNamespace(rest string) error {
	rest = strings.TrimSpace(rest)
	rest = strings.TrimSuffix(rest, "{")
	name := strings.TrimSpace(rest)
	if name == "" {
		return fmt.Errorf("namespace requires a name")
	}
	ns := diagram.ClassNamespace{Name: name}
	classesBefore := len(p.diagram.Classes)
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			// Anything appended to p.diagram.Classes while we were
			// inside the block belongs to this namespace, including
			// classes auto-registered by relations (`A --> B` inside
			// a namespace puts both A and B in the namespace).
			for i := classesBefore; i < len(p.diagram.Classes); i++ {
				ns.ClassIDs = append(ns.ClassIDs, p.diagram.Classes[i].ID)
			}
			p.diagram.Namespaces = append(p.diagram.Namespaces, ns)
			return nil
		}
		if _, isNamespace := cutNamespaceKeyword(line); isNamespace {
			return fmt.Errorf("namespace %q: nested namespaces are not supported", name)
		}
		if err := p.parseLine(line); err != nil {
			return fmt.Errorf("namespace %q: %w", name, err)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading namespace %q: %w", name, err)
	}
	return fmt.Errorf("unclosed namespace %q", name)
}

// parseClick handles `click ID call func(args)` (Callback) and
// `click ID href "url" "tooltip" "target"` (URL). The two forms
// share a keyword and are disambiguated by the `call` / `href`
// subkeyword. Bare `click ID "url" …` is treated as href for
// compatibility with mermaid-cli's loose matching.
func (p *parser) parseClick(line string) error {
	rest := strings.TrimSpace(line[len("click "):])
	classID, afterID, err := splitClickHead(rest, "click")
	if err != nil {
		return err
	}
	if err := p.requireClass(classID, "click"); err != nil {
		return err
	}
	cd := diagram.ClassClickDef{ClassID: classID}
	switch {
	case afterID == "call" || strings.HasPrefix(afterID, "call "):
		callback := strings.TrimSpace(strings.TrimPrefix(afterID, "call"))
		if callback == "" {
			return fmt.Errorf("click %s: missing callback after `call`", classID)
		}
		cd.Callback = callback
	case afterID == "href" || strings.HasPrefix(afterID, "href "):
		argSrc := strings.TrimSpace(strings.TrimPrefix(afterID, "href"))
		if err := fillClickURLArgs(&cd, argSrc); err != nil {
			return fmt.Errorf("click %s: %w", classID, err)
		}
	default:
		if err := fillClickURLArgs(&cd, afterID); err != nil {
			return fmt.Errorf("click %s: %w", classID, err)
		}
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

// parseLinkOrCallback handles the `link` / `callback` aliases.
// `link ID "url" "tooltip"` ⇔ `click ID href "url" "tooltip"`.
// `callback ID "func" "tooltip"` ⇔ `click ID call func` plus tooltip.
// isCallback distinguishes the two without restringing the keyword.
func (p *parser) parseLinkOrCallback(line string, isCallback bool) error {
	kw := "link"
	if isCallback {
		kw = "callback"
	}
	rest := strings.TrimSpace(line[len(kw)+1:])
	classID, argSrc, err := splitClickHead(rest, kw)
	if err != nil {
		return err
	}
	if err := p.requireClass(classID, kw); err != nil {
		return err
	}
	cd := diagram.ClassClickDef{ClassID: classID}
	if isCallback {
		parts, perr := parserutil.SplitClickArgs(argSrc, 3)
		if perr != nil {
			return fmt.Errorf("%s %s: %w", kw, classID, perr)
		}
		if len(parts) == 0 || parts[0] == "" {
			return fmt.Errorf("%s %s: missing callback", kw, classID)
		}
		cd.Callback = parts[0]
		if len(parts) >= 2 {
			cd.Tooltip = parts[1]
		}
		if len(parts) >= 3 {
			cd.Target = parts[2]
		}
	} else if err := fillClickURLArgs(&cd, argSrc); err != nil {
		return fmt.Errorf("%s %s: %w", kw, classID, err)
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

// splitClickHead splits "ClassID rest…" into id + rest. Single
// SplitClickArgs scan covers both the length check and the id
// extraction, replacing the prior strings.Fields-then-rescan pattern.
func splitClickHead(rest, kw string) (id, after string, err error) {
	parts, perr := parserutil.SplitClickArgs(rest, 2)
	if perr != nil {
		return "", "", fmt.Errorf("%s: %w", kw, perr)
	}
	if len(parts) < 2 {
		return "", "", fmt.Errorf("%s requires class id and target", kw)
	}
	id = parts[0]
	return id, strings.TrimSpace(rest[len(id):]), nil
}

// requireClass surfaces an error when a click/link/callback names a
// class that hasn't been declared. Auto-registering would silently
// accept typos (e.g. `click Fooo …` would create a phantom Fooo).
// Mirrors flowchart's strict lookup behavior.
func (p *parser) requireClass(id, kw string) error {
	if _, ok := p.classIdx[id]; !ok {
		return fmt.Errorf("%s references undefined class %q", kw, id)
	}
	return nil
}

func fillClickURLArgs(cd *diagram.ClassClickDef, src string) error {
	parts, err := parserutil.SplitClickArgs(src, 3)
	if err != nil {
		return err
	}
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("missing URL")
	}
	cd.URL = parts[0]
	if len(parts) >= 2 {
		cd.Tooltip = parts[1]
	}
	if len(parts) >= 3 {
		cd.Target = parts[2]
	}
	return nil
}

// parseSingleLineMember matches `ClassName : memberText` and returns
// the class id and the member text (without the colon). The colon
// must be surrounded by whitespace so a field type containing `:`
// inside a class body — `template: String` — doesn't false-match.
func parseSingleLineMember(line string) (string, string, bool) {
	idx := strings.Index(line, " : ")
	if idx < 0 {
		return "", "", false
	}
	id := strings.TrimSpace(line[:idx])
	if id == "" || strings.ContainsAny(id, " \t") {
		return "", "", false
	}
	return id, strings.TrimSpace(line[idx+3:]), true
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

// declareClass registers a class with explicit metadata. Each class
// may carry one explicit label and one generic across the source —
// a second declaration with a different label or generic is a
// conflict and surfaces as an error rather than silently shadowing.
// Re-declaring a class with the same metadata (or with no metadata,
// e.g. a bare `class Foo` after `class Foo["L"]`) is a no-op.
func (p *parser) declareClass(id, label, generic string) error {
	idx := p.ensureClass(id)
	c := &p.diagram.Classes[idx]
	if label != "" {
		// c.Label == c.ID is the auto-default ensureClass sets; treat
		// it as "no explicit label yet" so an attached `class Foo["L"]`
		// after a relation-only mention is welcome, not a conflict.
		if c.Label != id && c.Label != label {
			return fmt.Errorf("class %q already declared with label %q; cannot reassign to %q", id, c.Label, label)
		}
		c.Label = label
	}
	if generic != "" {
		if c.Generic != "" && c.Generic != generic {
			return fmt.Errorf("class %q already declared with generic %q; cannot reassign to %q", id, c.Generic, generic)
		}
		c.Generic = generic
	}
	return nil
}

// classHeader is the parsed result of `class NAME[...]~...~ <<Ann>>:::cssClass`.
type classHeader struct {
	id         string
	label      string // from `["..."]`
	generic    string // from `~...~`
	annotation diagram.ClassAnnotation
	cssClass   string // from `:::name` shorthand
}

// parseClassHeader splits `Foo["My Label"]~T~` (or any subset) into
// id / label / generic and reports whether a `{` follows. Body content
// is left for parseClassBody to consume from the scanner.
//
// Malformed `[…]` (no closing bracket / no quoted content) and an
// unmatched `~` (no closing tilde) surface as errors — silently
// dropping them would leave bracket / tilde junk inside the parsed ID
// and trigger mysterious lookup failures downstream.
func parseClassHeader(rest string) (classHeader, bool, error) {
	hasBody := false
	if i := strings.IndexByte(rest, '{'); i >= 0 {
		rest = strings.TrimSpace(rest[:i])
		hasBody = true
	}
	// `:::cssClass` shorthand attaches a named CSS class. Stripped
	// before the rest of header parsing so the class name doesn't
	// confuse the label/generic/annotation matchers.
	id, cssClass, ok := parserutil.ExtractCSSClassShorthand(rest)
	if !ok {
		return classHeader{}, false, fmt.Errorf("class header: only one `:::` cssClass shorthand is allowed")
	}
	rest = id
	// Inline annotation: `class Foo <<Interface>>`. Strip before
	// label/generic so the brackets/tildes don't trip on `<<` chars.
	var annotation diagram.ClassAnnotation
	if i := strings.Index(rest, "<<"); i >= 0 {
		j := strings.Index(rest, ">>")
		if j <= i {
			return classHeader{}, false, fmt.Errorf("class header %q: unmatched `<<`", rest)
		}
		annotation = parseAnnotation(strings.TrimSpace(rest[i+2 : j]))
		rest = strings.TrimSpace(rest[:i] + rest[j+2:])
		// Mermaid only supports one annotation per class; a second
		// `<<...>>` would be silently swallowed into the ID.
		if strings.Contains(rest, "<<") {
			return classHeader{}, false, fmt.Errorf("class header %q: only one annotation is allowed", rest)
		}
	}
	var label string
	if i := strings.IndexByte(rest, '['); i >= 0 {
		j := strings.LastIndexByte(rest, ']')
		if j <= i {
			return classHeader{}, false, fmt.Errorf("class header %q: unclosed `[`", rest)
		}
		inside := strings.TrimSpace(rest[i+1 : j])
		unq := parserutil.Unquote(inside)
		if unq == inside {
			return classHeader{}, false, fmt.Errorf("class header %q: bracketed label must be quoted", rest)
		}
		label = unq
		rest = strings.TrimSpace(rest[:i])
	}
	var generic string
	if i := strings.IndexByte(rest, '~'); i >= 0 {
		// Use the LAST `~` so nested generics like `Wrapper~List~int~~`
		// give Generic="List~int~" rather than "List".
		j := strings.LastIndexByte(rest, '~')
		if j <= i {
			return classHeader{}, false, fmt.Errorf("class header %q: unmatched `~`", rest)
		}
		generic = rest[i+1 : j]
		rest = strings.TrimSpace(rest[:i])
	}
	return classHeader{id: rest, label: label, generic: generic, annotation: annotation, cssClass: cssClass}, hasBody, nil
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

// extractMemberModifiers strips trailing `$` (static) / `*` (abstract)
// markers from a member text. Mermaid's grammar attaches them at the
// end of a token — after the name (`pi$ double`), after the type
// (`name double$`), or after a method's `)` (`log()$ void`). We strip
// only at token boundaries (the *last* char of a whitespace-delimited
// token), so a `$` or `*` appearing inside an identifier or type name
// is preserved verbatim.
func extractMemberModifiers(s string) (cleaned string, isStatic, isAbstract bool) {
	tokens := strings.Fields(s)
	out := tokens[:0]
	for _, tok := range tokens {
		for len(tok) > 0 {
			switch tok[len(tok)-1] {
			case '$':
				isStatic = true
				tok = tok[:len(tok)-1]
			case '*':
				isAbstract = true
				tok = tok[:len(tok)-1]
			default:
				goto done
			}
		}
	done:
		if tok != "" {
			out = append(out, tok)
		}
	}
	return strings.Join(out, " "), isStatic, isAbstract
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
	glyphLollipop                  // `()` — provided-interface lollipop
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
	// `<|` and `()` are two chars; check them before the single-char glyphs.
	if lineStart >= 2 && line[lineStart-2] == '<' && line[lineStart-1] == '|' {
		return glyphTriangle, lineStart - 2
	}
	if lineStart >= 2 && line[lineStart-2] == '(' && line[lineStart-1] == ')' {
		return glyphLollipop, lineStart - 2
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
	if lineEnd+1 < len(line) && line[lineEnd] == '(' && line[lineEnd+1] == ')' {
		return glyphLollipop, lineEnd + 2
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
		case glyphLollipop:
			// Lollipop on both ends would mean two-way provided
			// interface — not part of Mermaid's grammar.
			return 0, 0, false
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
	case glyphLollipop:
		// Mermaid only documents the solid-line lollipop (`()--`).
		// Reject the dashed form rather than guess at semantics.
		if dashed {
			return 0, false
		}
		return diagram.RelationTypeLollipop, true
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
