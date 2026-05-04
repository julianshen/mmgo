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
		diagram: &diagram.ERDiagram{
			CSSClasses: make(map[string]string),
		},
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
	// Keyword-prefixed lines first (Mermaid convention: keywords
	// take precedence over bare-name matchers).
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
	if rest, ok := strings.CutPrefix(line, "direction "); ok {
		dir, err := parserutil.ParseDirection(strings.TrimSpace(rest))
		if err != nil {
			return err
		}
		p.diagram.Direction = dir
		return nil
	}
	if strings.HasPrefix(line, "classDef ") {
		return p.parseClassDef(line)
	}
	if strings.HasPrefix(line, "style ") {
		return p.parseStyleRule(line)
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
	if strings.HasPrefix(line, "class ") {
		return p.parseClassBinding(line[len("class "):])
	}
	// Relationships first — their cardinality markers can contain `{`.
	if rel, ok := parseRelationship(line); ok {
		fromID, fromCSS, fromOK := parserutil.ExtractCSSClassShorthand(rel.From)
		toID, toCSS, toOK := parserutil.ExtractCSSClassShorthand(rel.To)
		if !fromOK || !toOK {
			return fmt.Errorf("relationship: only one `:::` cssClass shorthand is allowed per entity reference")
		}
		if fromID == "" || toID == "" {
			return fmt.Errorf("relationship: empty entity id (a `:::class` reference needs a name)")
		}
		rel.From, rel.To = fromID, toID
		fromIdx := p.ensureEntityIdx(fromID)
		toIdx := p.ensureEntityIdx(toID)
		if fromCSS != "" {
			p.diagram.Entities[fromIdx].CSSClasses = append(p.diagram.Entities[fromIdx].CSSClasses, fromCSS)
		}
		if toCSS != "" {
			p.diagram.Entities[toIdx].CSSClasses = append(p.diagram.Entities[toIdx].CSSClasses, toCSS)
		}
		p.diagram.Relationships = append(p.diagram.Relationships, rel)
		return nil
	}
	if strings.HasSuffix(strings.TrimSpace(line), "{") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "{"))
		head, cssClass, ok := parserutil.ExtractCSSClassShorthand(name)
		if !ok {
			return fmt.Errorf("entity %q: only one `:::` cssClass shorthand is allowed", name)
		}
		id, label, err := extractEntityAlias(head)
		if err != nil {
			return err
		}
		if id == "" {
			return fmt.Errorf("entity declaration is missing a name")
		}
		idx := p.ensureEntityIdx(id)
		if label != "" {
			p.diagram.Entities[idx].Label = label
		}
		if cssClass != "" {
			p.diagram.Entities[idx].CSSClasses = append(p.diagram.Entities[idx].CSSClasses, cssClass)
		}
		return p.parseEntityBody(id)
	}
	// Bare entity name on its own line (with optional alias and
	// `:::cssClass`).
	if id, label, cssClass, ok, err := parseBareEntity(line); err != nil {
		return err
	} else if ok {
		idx := p.ensureEntityIdx(id)
		if label != "" {
			p.diagram.Entities[idx].Label = label
		}
		if cssClass != "" {
			p.diagram.Entities[idx].CSSClasses = append(p.diagram.Entities[idx].CSSClasses, cssClass)
		}
		return nil
	}
	return nil
}

// parseBareEntity matches a single-token entity declaration with an
// optional `["Display Label"]` alias and `:::cssClass` shorthand.
// The alias's quoted text may itself contain whitespace; only the
// ID and bracket+shorthand structure must be a single token.
//
// Multi-token lines (where there's whitespace OUTSIDE any
// `["..."]` alias) are not bare entities — those came from other
// keyword paths or are unrecognised lines that earlier matchers
// already handled.
func parseBareEntity(line string) (id, label, cssClass string, ok bool, err error) {
	// Replace anything inside `["..."]` with placeholder X's so the
	// embedded whitespace doesn't disqualify the line.
	stripped := stripBracketContents(line)
	if strings.ContainsAny(stripped, " \t") {
		return "", "", "", false, nil
	}
	rest, cssClass, valid := parserutil.ExtractCSSClassShorthand(line)
	if !valid {
		return "", "", "", false, fmt.Errorf("entity %q: only one `:::` cssClass shorthand is allowed", line)
	}
	id, label, err = extractEntityAlias(rest)
	if err != nil {
		return "", "", "", false, err
	}
	if id == "" {
		return "", "", "", false, nil
	}
	return id, label, cssClass, true, nil
}

// stripBracketContents replaces every byte inside `[...]` with `_`
// so a multi-word alias like `ORDER["Customer Order"]` looks like
// a single token to whitespace checks. Bracketless input is
// returned unchanged.
func stripBracketContents(s string) string {
	open := strings.IndexByte(s, '[')
	close := strings.LastIndexByte(s, ']')
	if open < 0 || close <= open {
		return s
	}
	b := []byte(s)
	for i := open + 1; i < close; i++ {
		b[i] = '_'
	}
	return string(b)
}

// extractEntityAlias splits `EntityID["Display Label"]` into the
// bare ID and the alias text. Without brackets, returns the input
// as-is and label="". Malformed `EntityID["unclosed` is an error.
func extractEntityAlias(s string) (id, label string, err error) {
	open := strings.IndexByte(s, '[')
	if open < 0 {
		return s, "", nil
	}
	closeIdx := strings.LastIndexByte(s, ']')
	if closeIdx <= open {
		return "", "", fmt.Errorf("entity alias %q: unclosed `[`", s)
	}
	inside := strings.TrimSpace(s[open+1 : closeIdx])
	unq := parserutil.Unquote(inside)
	if unq == inside {
		return "", "", fmt.Errorf("entity alias %q: bracketed label must be quoted", s)
	}
	return strings.TrimSpace(s[:open]), unq, nil
}

func (p *parser) parseClassDef(line string) error {
	name, css, err := parserutil.ParseClassDefLine(line[len("classDef "):])
	if err != nil {
		return err
	}
	p.diagram.CSSClasses[name] = css
	return nil
}

func (p *parser) parseStyleRule(line string) error {
	rest := line[len("style "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("style requires an ID and CSS")
	}
	p.diagram.Styles = append(p.diagram.Styles, diagram.ERStyleDef{
		EntityID: strings.TrimSpace(parts[0]),
		CSS:      parserutil.NormalizeCSS(strings.TrimSpace(parts[1])),
	})
	return nil
}

func (p *parser) parseClassBinding(rest string) error {
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("class requires entity ids and class name")
	}
	cssName := strings.TrimSpace(parts[1])
	for _, id := range strings.Split(parts[0], ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		idx, ok := p.entityIdx[id]
		if !ok {
			return fmt.Errorf("class binding references undefined entity %q", id)
		}
		p.diagram.Entities[idx].CSSClasses = append(p.diagram.Entities[idx].CSSClasses, cssName)
	}
	return nil
}

// parseClick handles `click ID call func()` and `click ID href "url"
// "tooltip" "target"`. Bare `click ID "url" …` falls through as href.
func (p *parser) parseClick(line string) error {
	rest := strings.TrimSpace(line[len("click "):])
	id, afterID, err := splitClickHead(rest, "click")
	if err != nil {
		return err
	}
	if err := p.requireEntity(id, "click"); err != nil {
		return err
	}
	cd := diagram.ERClickDef{EntityID: id}
	switch {
	case afterID == "call" || strings.HasPrefix(afterID, "call "):
		callback := strings.TrimSpace(strings.TrimPrefix(afterID, "call"))
		if callback == "" {
			return fmt.Errorf("click %s: missing callback after `call`", id)
		}
		cd.Callback = callback
	case afterID == "href" || strings.HasPrefix(afterID, "href "):
		argSrc := strings.TrimSpace(strings.TrimPrefix(afterID, "href"))
		if err := fillClickURLArgs(&cd, argSrc); err != nil {
			return fmt.Errorf("click %s: %w", id, err)
		}
	default:
		if err := fillClickURLArgs(&cd, afterID); err != nil {
			return fmt.Errorf("click %s: %w", id, err)
		}
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

func (p *parser) parseLinkOrCallback(line string, isCallback bool) error {
	kw := "link"
	if isCallback {
		kw = "callback"
	}
	rest := strings.TrimSpace(line[len(kw)+1:])
	id, argSrc, err := splitClickHead(rest, kw)
	if err != nil {
		return err
	}
	if err := p.requireEntity(id, kw); err != nil {
		return err
	}
	cd := diagram.ERClickDef{EntityID: id}
	if isCallback {
		parts, perr := parserutil.SplitClickArgs(argSrc, 3)
		if perr != nil {
			return fmt.Errorf("%s %s: %w", kw, id, perr)
		}
		if len(parts) == 0 || parts[0] == "" {
			return fmt.Errorf("%s %s: missing callback", kw, id)
		}
		cd.Callback = parts[0]
		if len(parts) >= 2 {
			cd.Tooltip = parts[1]
		}
		if len(parts) >= 3 {
			cd.Target = parts[2]
		}
	} else if err := fillClickURLArgs(&cd, argSrc); err != nil {
		return fmt.Errorf("%s %s: %w", kw, id, err)
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

func splitClickHead(rest, kw string) (id, after string, err error) {
	parts, perr := parserutil.SplitClickArgs(rest, 2)
	if perr != nil {
		return "", "", fmt.Errorf("%s: %w", kw, perr)
	}
	if len(parts) < 2 {
		return "", "", fmt.Errorf("%s requires entity id and target", kw)
	}
	id = parts[0]
	return id, strings.TrimSpace(rest[len(id):]), nil
}

func (p *parser) requireEntity(id, kw string) error {
	if _, ok := p.entityIdx[id]; !ok {
		return fmt.Errorf("%s references undefined entity %q", kw, id)
	}
	return nil
}

func fillClickURLArgs(cd *diagram.ERClickDef, src string) error {
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
	p.ensureEntityIdx(name)
}

func (p *parser) ensureEntityIdx(name string) int {
	if idx, ok := p.entityIdx[name]; ok {
		return idx
	}
	p.entityIdx[name] = len(p.diagram.Entities)
	p.diagram.Entities = append(p.diagram.Entities, diagram.EREntity{Name: name})
	return p.entityIdx[name]
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
