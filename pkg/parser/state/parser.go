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
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	p := &parser{
		diagram: &diagram.StateDiagram{
			CSSClasses: make(map[string]string),
		},
	}
	// Optional `---\n…\n---\n` frontmatter at the top supplies a
	// diagram title (Mermaid's universal frontmatter convention).
	front, body := parserutil.SplitFrontmatter(src)
	if len(front) > 0 {
		if t := parserutil.FrontmatterValue(front, "title"); t != "" {
			p.diagram.Title = t
		}
	}
	p.scanner = bufio.NewScanner(strings.NewReader(string(body)))
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
		if err := p.parseLine(line, &p.diagram.States, ""); err != nil {
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

func (p *parser) parseLine(line string, target *[]diagram.StateDef, scope string) error {
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
		return p.parseStateDecl(strings.TrimSpace(rest), target, scope)
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
	if t, fromCSS, toCSS, ok := parseTransition(line); ok {
		fromState := upsertState(target, t.From)
		toState := upsertState(target, t.To)
		if fromState != nil && fromCSS != "" {
			fromState.CSSClasses = append(fromState.CSSClasses, fromCSS)
		}
		if toState != nil && toCSS != "" {
			toState.CSSClasses = append(toState.CSSClasses, toCSS)
		}
		t.Scope = scope
		p.diagram.Transitions = append(p.diagram.Transitions, t)
		return nil
	}
	if id, label, ok := parseStateDescription(line); ok {
		s := upsertState(target, id)
		if s != nil {
			s.Label = label
		}
		return nil
	}
	// Bare state identifier with no transition or description.
	if !strings.ContainsAny(line, " \t") {
		upsertState(target, line)
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

// parseClassBinding handles `class id1,id2 className`. Bindings to
// unknown state IDs are silently skipped (Mermaid's behaviour — the
// syntax-docs example uses `class end badBadEvent` where `end` is
// never declared as a real state). Lookup is recursive so a child of
// a composite can be addressed by its bare ID.
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
			continue
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
// any arrow-bearing transition. Mermaid accepts the colon with or
// without surrounding whitespace, e.g. `id : desc`, `id: desc`, or
// `id :desc` (per the syntax docs' composite-states example, which
// uses `NamedComposite: Another Composite`). The triple-colon `:::`
// is reserved for inline CSS class shorthand and never matches here.
func parseStateDescription(line string) (id, desc string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 1 {
		return "", "", false
	}
	// Skip the CSS class shorthand `id:::class` — that's handled
	// elsewhere and is not a description.
	if strings.HasPrefix(line[colon:], ":::") {
		return "", "", false
	}
	id = strings.TrimSpace(line[:colon])
	if id == "" || strings.ContainsAny(id, " \t") {
		return "", "", false
	}
	return id, strings.TrimSpace(line[colon+1:]), true
}

func (p *parser) parseStateDecl(rest string, target *[]diagram.StateDef, scope string) error {
	var label, id string
	var after string
	cssClass := ""

	if strings.HasPrefix(rest, "\"") {
		// Quoted label form: state "Label" or state "Label" as ID
		endQuote := strings.Index(rest[1:], "\"")
		if endQuote < 0 {
			return fmt.Errorf("unterminated quote in state declaration")
		}
		label = rest[1 : endQuote+1]
		after = strings.TrimSpace(rest[endQuote+2:])
		if idPart, ok := strings.CutPrefix(after, "as "); ok {
			idPart = strings.TrimSpace(idPart)
			if idPart == "" {
				return fmt.Errorf("state declaration: missing identifier after 'as'")
			}
			rawID, tail := splitStateIDAndTail(idPart)
			var ok bool
			id, cssClass, ok = parserutil.ExtractCSSClassShorthand(rawID)
			if !ok {
				return fmt.Errorf("state declaration: chained CSS shorthand not allowed")
			}
			after = tail
		} else {
			// No "as" keyword — use the quoted string as both ID and label.
			id = label
		}
	} else {
		rawID, tail := splitStateIDAndTail(rest)
		var ok bool
		id, cssClass, ok = parserutil.ExtractCSSClassShorthand(rawID)
		if !ok {
			return fmt.Errorf("state declaration: chained CSS shorthand not allowed")
		}
		after = tail
		label = id
	}

	// Handle CSS shorthand in `after` for quoted-no-as forms:
	// e.g. state "Label":::hot or state "Label":::hot {.
	if strings.HasPrefix(after, ":::") {
		raw := strings.TrimSpace(after[3:])
		if j := strings.IndexAny(raw, " \t{<"); j >= 0 {
			if c := strings.TrimSpace(raw[:j]); c != "" && cssClass == "" {
				cssClass = c
			}
			after = strings.TrimSpace(raw[j:])
		} else {
			if c := strings.TrimSpace(raw); c != "" && cssClass == "" {
				cssClass = c
			}
			after = ""
		}
	}

	if id == "" {
		return fmt.Errorf("state declaration: missing identifier")
	}

	s := upsertState(target, id)
	if s == nil {
		return fmt.Errorf("invalid state name %q", id)
	}
	// Only overwrite an existing label when an explicit one is given
	// (i.e. label differs from the bare id). This preserves labels set
	// by prior `id : text` or `state "Label" as id` declarations.
	if label != id {
		s.Label = label
	}
	if cssClass != "" {
		s.CSSClasses = append(s.CSSClasses, cssClass)
	}

	_ = scope // scope of the declaration itself is not currently needed; child transitions get s.ID as their scope via parseCompositeBody.
	after = strings.TrimSpace(after)
	if after == "{" || strings.HasPrefix(after, "{") {
		return p.parseCompositeBody(s)
	}
	if strings.HasPrefix(after, "<<") && strings.HasSuffix(after, ">>") {
		annotation := strings.Trim(after, "<>")
		s.Kind = parseStateKind(annotation)
		return nil
	}
	return nil
}

// splitStateIDAndTail splits a state declaration remainder into the
// identifier and everything that follows ({ or <<kind>>). CSS shorthand
// (:::css) is kept as part of the identifier; callers should strip it
// with parserutil.ExtractCSSClassShorthand afterwards.
func splitStateIDAndTail(s string) (id, tail string) {
	s = strings.TrimSpace(s)
	braceIdx := strings.IndexByte(s, '{')
	kindIdx := strings.Index(s, "<<")

	minIdx := -1
	if braceIdx >= 0 {
		minIdx = braceIdx
	}
	if kindIdx >= 0 && (minIdx < 0 || kindIdx < minIdx) {
		minIdx = kindIdx
	}

	if minIdx >= 0 {
		return strings.TrimSpace(s[:minIdx]), strings.TrimSpace(s[minIdx:])
	}
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", ""
	}
	return parts[0], strings.TrimSpace(s[len(parts[0]):])
}

// parseCompositeBody reads inner-state lines until the matching `}`.
// `--` lines split the body into parallel regions; the parent's
// Regions slice records each region's children, while Children
// continues to hold the concatenated union (so existing consumers
// that only walk Children keep working).
func (p *parser) parseCompositeBody(parent *diagram.StateDef) error {
	target := &parent.Children
	regionStart := 0
	hasSeparator := false
	for p.scanner.Scan() {
		p.lineNum++
		line := strings.TrimSpace(parserutil.StripComment(p.scanner.Text()))
		if line == "" {
			continue
		}
		if line == "}" {
			if hasSeparator {
				// Capture the trailing region (everything after the
				// last `--`), then we're done.
				region := append([]diagram.StateDef(nil), (*target)[regionStart:]...)
				parent.Regions = append(parent.Regions, region)
			}
			return nil
		}
		if line == "--" {
			region := append([]diagram.StateDef(nil), (*target)[regionStart:]...)
			parent.Regions = append(parent.Regions, region)
			regionStart = len(*target)
			hasSeparator = true
			continue
		}
		if err := p.parseLine(line, target, parent.ID); err != nil {
			return err
		}
	}
	if err := p.scanner.Err(); err != nil {
		return fmt.Errorf("reading composite state: %w", err)
	}
	return fmt.Errorf("unclosed composite state")
}

// parseTransition extracts a transition from a line, returning the
// transition, any CSS class shorthand attached to the from/to states,
// and whether the line is a transition at all.
func parseTransition(line string) (diagram.StateTransition, string, string, bool) {
	idx := strings.Index(line, "-->")
	if idx < 0 {
		return diagram.StateTransition{}, "", "", false
	}
	from := strings.TrimSpace(line[:idx])
	// Extract inline CSS shorthand from the LHS.
	fromCSS := ""
	if i := strings.LastIndex(from, ":::"); i >= 0 {
		fromCSS = strings.TrimSpace(from[i+3:])
		from = strings.TrimSpace(from[:i])
	}
	rest := strings.TrimSpace(line[idx+3:])
	// Extract inline CSS shorthand from the RHS before looking for
	// the label separator so `A:::hot --> B` is parsed correctly.
	to := rest
	label := ""
	toCSS := ""
	if cssIdx := strings.Index(rest, ":::"); cssIdx >= 0 {
		to = strings.TrimSpace(rest[:cssIdx])
		// Any trailing content after the CSS shorthand is the label
		// (e.g. `B:::hot : label` → to=B, toCSS=hot, label=label).
		rest = strings.TrimSpace(rest[cssIdx+3:])
		if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
			toCSS = strings.TrimSpace(rest[:colonIdx])
			label = parserutil.ExpandLineBreaks(strings.TrimSpace(rest[colonIdx+1:]))
		} else {
			toCSS = rest
		}
	} else if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
		to = strings.TrimSpace(rest[:colonIdx])
		label = parserutil.ExpandLineBreaks(strings.TrimSpace(rest[colonIdx+1:]))
	}
	if from == "" || to == "" {
		return diagram.StateTransition{}, "", "", false
	}
	return diagram.StateTransition{From: from, To: to, Label: label}, fromCSS, toCSS, true
}

func parseStateKind(annotation string) diagram.StateKind {
	switch strings.ToLower(annotation) {
	case "fork":
		return diagram.StateKindFork
	case "join":
		return diagram.StateKindJoin
	case "choice":
		return diagram.StateKindChoice
	case "history":
		return diagram.StateKindHistory
	case "deephistory", "deep_history":
		return diagram.StateKindDeepHistory
	default:
		return diagram.StateKindNormal
	}
}
