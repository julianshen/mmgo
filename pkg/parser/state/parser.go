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
		diagram: &diagram.StateDiagram{
			CSSClasses: make(map[string]string),
		},
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
	// Keyword-prefixed lines win over bare-id matchers (Mermaid
	// convention): `title : foo` is the diagram title, never a
	// state-with-description for a state named "title".
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
	if strings.HasPrefix(line, "click ") {
		return p.parseClick(line)
	}
	if strings.HasPrefix(line, "link ") {
		return p.parseLinkOrCallback(line, false)
	}
	if strings.HasPrefix(line, "callback ") {
		return p.parseLinkOrCallback(line, true)
	}
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
	// `class IDs name` binds a previously-defined classDef to states.
	// Must be checked AFTER `state ` (the `state ` prefix takes
	// precedence over the bare `class ` prefix on lines like
	// `state foo` — the keywords don't actually overlap, but we
	// keep the order explicit to mirror Mermaid's parser).
	if rest, ok := strings.CutPrefix(line, "class "); ok {
		return p.parseClassBinding(rest, target)
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
	id := strings.TrimSpace(parts[0])
	p.diagram.Styles = append(p.diagram.Styles, diagram.StateStyleDef{
		StateID: id,
		CSS:     parserutil.NormalizeCSS(strings.TrimSpace(parts[1])),
	})
	return nil
}

// parseClassBinding handles `class id1,id2 className`. State IDs
// must already exist; an unknown ID errors rather than silently
// creating a phantom state (a bare typo like `class Foo bar` when
// `Foo` was meant should not produce an undeclared shadow). Lookup
// is recursive so `class Foo bar` works when Foo lives inside a
// composite.
func (p *parser) parseClassBinding(rest string, target *[]diagram.StateDef) error {
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("class requires state ids and class name")
	}
	cssName := strings.TrimSpace(parts[1])
	for _, id := range strings.Split(parts[0], ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		s := findStateByID(target, id)
		if s == nil {
			s = findStateRecursive(p.diagram.States, id)
		}
		if s == nil {
			return fmt.Errorf("class binding references undefined state %q", id)
		}
		s.CSSClasses = append(s.CSSClasses, cssName)
	}
	return nil
}

func findStateByID(states *[]diagram.StateDef, id string) *diagram.StateDef {
	for i := range *states {
		if (*states)[i].ID == id {
			return &(*states)[i]
		}
	}
	return nil
}

func findStateRecursive(states []diagram.StateDef, id string) *diagram.StateDef {
	for i := range states {
		if states[i].ID == id {
			return &states[i]
		}
		if found := findStateRecursive(states[i].Children, id); found != nil {
			return found
		}
	}
	return nil
}

// parseClick handles `click ID call func()` and `click ID href "url"
// "tooltip" "target"`. Bare `click ID "url" …` is treated as href
// for compatibility with mermaid-cli.
func (p *parser) parseClick(line string) error {
	rest := strings.TrimSpace(line[len("click "):])
	stateID, afterID, err := splitClickHead(rest, "click")
	if err != nil {
		return err
	}
	if err := p.requireState(stateID, "click"); err != nil {
		return err
	}
	cd := diagram.StateClickDef{StateID: stateID}
	switch {
	case afterID == "call" || strings.HasPrefix(afterID, "call "):
		callback := strings.TrimSpace(strings.TrimPrefix(afterID, "call"))
		if callback == "" {
			return fmt.Errorf("click %s: missing callback after `call`", stateID)
		}
		cd.Callback = callback
	case afterID == "href" || strings.HasPrefix(afterID, "href "):
		argSrc := strings.TrimSpace(strings.TrimPrefix(afterID, "href"))
		if err := fillClickURLArgs(&cd, argSrc); err != nil {
			return fmt.Errorf("click %s: %w", stateID, err)
		}
	default:
		if err := fillClickURLArgs(&cd, afterID); err != nil {
			return fmt.Errorf("click %s: %w", stateID, err)
		}
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

// parseLinkOrCallback handles the `link` / `callback` aliases.
func (p *parser) parseLinkOrCallback(line string, isCallback bool) error {
	kw := "link"
	if isCallback {
		kw = "callback"
	}
	rest := strings.TrimSpace(line[len(kw)+1:])
	stateID, argSrc, err := splitClickHead(rest, kw)
	if err != nil {
		return err
	}
	if err := p.requireState(stateID, kw); err != nil {
		return err
	}
	cd := diagram.StateClickDef{StateID: stateID}
	if isCallback {
		parts, perr := parserutil.SplitClickArgs(argSrc, 3)
		if perr != nil {
			return fmt.Errorf("%s %s: %w", kw, stateID, perr)
		}
		if len(parts) == 0 || parts[0] == "" {
			return fmt.Errorf("%s %s: missing callback", kw, stateID)
		}
		cd.Callback = parts[0]
		if len(parts) >= 2 {
			cd.Tooltip = parts[1]
		}
		if len(parts) >= 3 {
			cd.Target = parts[2]
		}
	} else if err := fillClickURLArgs(&cd, argSrc); err != nil {
		return fmt.Errorf("%s %s: %w", kw, stateID, err)
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
		return "", "", fmt.Errorf("%s requires state id and target", kw)
	}
	id = parts[0]
	return id, strings.TrimSpace(rest[len(id):]), nil
}

// requireState reports an error when a click/link/callback names a
// state that hasn't been declared anywhere in the diagram. Lookup is
// global — Mermaid state-diagram clicks aren't composite-scoped, so
// `click Foo` works whether Foo is at the top level or inside a
// composite.
func (p *parser) requireState(id, kw string) error {
	if findStateRecursive(p.diagram.States, id) == nil {
		return fmt.Errorf("%s references undefined state %q", kw, id)
	}
	return nil
}

func fillClickURLArgs(cd *diagram.StateClickDef, src string) error {
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
		return fmt.Errorf("note must use `left of <state>` or `right of <state>`; got %q", line)
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

// scanBlockNote reads body lines until it sees `end note` and joins
// them with real newlines. Blank lines are preserved as paragraph
// separators (Mermaid renders them as visible gaps); comment-only
// lines are dropped via StripComment.
func (p *parser) scanBlockNote() (string, error) {
	var lines []string
	for p.scanner.Scan() {
		p.lineNum++
		raw := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
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
	// `:::cssClass` shorthand attaches a named CSS class to the
	// state. The marker sits at the end of the identifier; strip
	// it before any further parsing.
	cssClass := ""
	if i := strings.Index(rest, ":::"); i >= 0 {
		cssClass = strings.TrimSpace(rest[i+3:])
		// Stop at the first whitespace / brace so `state Foo:::hot {`
		// peels the class name correctly.
		if j := strings.IndexAny(cssClass, " \t{"); j >= 0 {
			cssClass = strings.TrimSpace(cssClass[:j])
		}
		// Reattach what came after to drive the rest of parsing.
		tail := rest[i+3+len(cssClass):]
		rest = strings.TrimSpace(rest[:i] + " " + strings.TrimSpace(tail))
	}
	if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
		name := strings.TrimSpace(rest[:braceIdx])
		s := upsertState(target, name)
		if s == nil {
			return fmt.Errorf("invalid composite state name %q", name)
		}
		if cssClass != "" {
			s.CSSClasses = append(s.CSSClasses, cssClass)
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
			if cssClass != "" {
				s.CSSClasses = append(s.CSSClasses, cssClass)
			}
		}
		return nil
	}
	if len(parts) >= 1 {
		s := upsertState(target, parts[0])
		if s != nil && cssClass != "" {
			s.CSSClasses = append(s.CSSClasses, cssClass)
		}
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
