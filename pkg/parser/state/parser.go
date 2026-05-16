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
		if err := p.parseLine(line, &p.diagram.States, "", 0); err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
		}
	}
	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing stateDiagram header")
	}
	// resolveDuplicateStates may prune phantom children, so promote
	// runs against a freshly-walked tree to keep paths accurate.
	resolveDuplicateStates(walkStateTree(&p.diagram.States))
	promoteCrossScopeTransitions(p.diagram, walkStateTree(&p.diagram.States))
	return p.diagram, nil
}

// stateWalkEntry pairs a state with the slice it lives in and the
// path from root that reaches it.
type stateWalkEntry struct {
	state  *diagram.StateDef
	parent *[]diagram.StateDef
	path   []string // ancestor IDs from root (exclusive of state.ID)
}

// walkStateTree returns one entry per state in the tree. Each entry's
// path slice is independently allocated so callers may retain it.
func walkStateTree(top *[]diagram.StateDef) []stateWalkEntry {
	var out []stateWalkEntry
	var walk func(slice *[]diagram.StateDef, path []string)
	walk = func(slice *[]diagram.StateDef, path []string) {
		for i := range *slice {
			s := &(*slice)[i]
			// Defensive copy: path's backing array is reused across
			// sibling iterations, so a retained path slice would see
			// its tail mutated by the next sibling's recursion.
			myPath := append([]string(nil), path...)
			out = append(out, stateWalkEntry{state: s, parent: slice, path: myPath})
			if len(s.Children) > 0 {
				walk(&s.Children, append(path, s.ID))
			}
		}
	}
	walk(top, nil)
	return out
}

// resolveDuplicateStates merges entries that share the same ID across
// scopes down to a single canonical occurrence. Mermaid scopes state
// IDs globally — `Normal --> DH` inside `state Running { … }` refers
// to the same `DH` declared at root scope (or wherever else), not to
// a new child inside Running. The parser, being single-pass and
// upsert-driven, can create a phantom child for the forward reference
// before the real `state DH <<deepHistory>>` declaration is seen on a
// later line; this post-pass finds duplicates and keeps only the
// most-attributed entry. Phantom occurrences are dropped from their
// parent's Children slice.
func resolveDuplicateStates(walked []stateWalkEntry) {
	byID := make(map[string][]stateWalkEntry, len(walked))
	for _, e := range walked {
		byID[e.state.ID] = append(byID[e.state.ID], e)
	}
	// Collect the specific StateDef pointers to drop. Keying by
	// *StateDef (rather than by ID-per-parent) keeps the canonical
	// entry safe even if a future parser refactor produces two
	// same-ID children inside one parent slice — only the non-best
	// entries' addresses are flagged for removal.
	drop := make(map[*[]diagram.StateDef]map[*diagram.StateDef]struct{})
	for _, entries := range byID {
		if len(entries) < 2 {
			continue
		}
		bestIdx := 0
		for i := 1; i < len(entries); i++ {
			if moreCanonicalState(*entries[i].state, *entries[bestIdx].state) {
				bestIdx = i
			}
		}
		for i, e := range entries {
			if i == bestIdx {
				continue
			}
			if drop[e.parent] == nil {
				drop[e.parent] = make(map[*diagram.StateDef]struct{})
			}
			drop[e.parent][e.state] = struct{}{}
		}
	}
	for parent, dropped := range drop {
		filtered := (*parent)[:0]
		for i := range *parent {
			s := &(*parent)[i]
			if _, isDropped := dropped[s]; isDropped {
				continue
			}
			filtered = append(filtered, *s)
		}
		// Zero out the trailing slots so the GC can reclaim the
		// removed StateDefs' transitively-held data.
		for i := len(filtered); i < len(*parent); i++ {
			(*parent)[i] = diagram.StateDef{}
		}
		*parent = filtered
	}
}

// promoteCrossScopeTransitions rewrites edges whose endpoints live in
// different composite scopes so they lay out at the lowest common
// ancestor. After resolveDuplicateStates has removed phantom children
// for forward references, a transition written inside `state X { … }`
// may target a state that lives outside X — in that case the edge
// crosses scope boundaries and dagre would otherwise re-create a
// phantom inside X's sub-graph. Replacing the inner endpoint with the
// ancestor composite that's visible at the LCA preserves the
// topological relationship; the arrow then exits the composite's
// boundary as Mermaid renders it.
//
// `walked` must reflect the tree AFTER resolveDuplicateStates, so the
// path index built here points only at canonical state locations.
func promoteCrossScopeTransitions(d *diagram.StateDiagram, walked []stateWalkEntry) {
	if d == nil || len(d.Transitions) == 0 {
		return
	}
	pathByID := make(map[string][]string, len(walked))
	for _, e := range walked {
		pathByID[e.state.ID] = e.path
	}
	for i := range d.Transitions {
		t := &d.Transitions[i]
		// `[*]` endpoints are scope-local by construction (synthesised
		// in the scope where the transition was written).
		if t.From == "[*]" || t.To == "[*]" {
			continue
		}
		fromPath, fromOK := pathByID[t.From]
		toPath, toOK := pathByID[t.To]
		if !fromOK || !toOK {
			continue
		}
		lca := commonPrefix(fromPath, toPath)
		layoutScopeID := ""
		if len(lca) > 0 {
			layoutScopeID = lca[len(lca)-1]
		}
		// Replace any endpoint deeper than the LCA with the ancestor
		// composite at the LCA's child level.
		if len(fromPath) > len(lca) {
			t.From = fromPath[len(lca)]
		}
		if len(toPath) > len(lca) {
			t.To = toPath[len(lca)]
		}
		// RegionIdx is meaningless above its original scope.
		if t.Scope != layoutScopeID {
			t.RegionIdx = 0
		}
		t.Scope = layoutScopeID
	}
}

// commonPrefix returns the longest path shared by a and b from the
// root downward.
func commonPrefix(a, b []string) []string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

// moreCanonicalState reports whether candidate is a more-attributed
// view of the same logical state than current. Attributes are scored
// with bit-disjoint weights so a higher-priority attribute always
// outvotes any combination of lower-priority ones: Kind > Children >
// Regions > description Label > CSSClasses. Ties resolve to `current`
// so the first occurrence wins, which keeps parsing order observable
// for diagnostics.
func moreCanonicalState(candidate, current diagram.StateDef) bool {
	score := func(s diagram.StateDef) int {
		n := 0
		if s.Kind != diagram.StateKindNormal {
			n += 16
		}
		if len(s.Children) > 0 {
			n += 8
		}
		if len(s.Regions) > 0 {
			n += 4
		}
		if s.Label != "" && s.Label != s.ID {
			n += 2
		}
		if len(s.CSSClasses) > 0 {
			n += 1
		}
		return n
	}
	return score(candidate) > score(current)
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

func (p *parser) parseLine(line string, target *[]diagram.StateDef, scope string, regionIdx int) error {
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
		return p.parseStateDecl(strings.TrimSpace(rest), target, scope, regionIdx)
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
		// Upsert both endpoints and attach CSS in two passes — a
		// single-pass `upsertState(from) … upsertState(to) … mutate`
		// is unsafe because the second append may reallocate the
		// backing array, invalidating the first pointer.
		if fromState := upsertState(target, t.From); fromState != nil && fromCSS != "" {
			fromState.CSSClasses = append(fromState.CSSClasses, fromCSS)
		}
		if toState := upsertState(target, t.To); toState != nil && toCSS != "" {
			toState.CSSClasses = append(toState.CSSClasses, toCSS)
		}
		t.Scope = scope
		t.RegionIdx = regionIdx
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

func (p *parser) parseStateDecl(rest string, target *[]diagram.StateDef, scope string, regionIdx int) error {
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

	_, _ = scope, regionIdx // declaration scope/region aren't needed here; child transitions get their own scope via parseCompositeBody.
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
	regionIdx := 0
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
			regionIdx++
			hasSeparator = true
			continue
		}
		if err := p.parseLine(line, target, parent.ID, regionIdx); err != nil {
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
	fromCSS := ""
	if i := strings.LastIndex(from, ":::"); i >= 0 {
		fromCSS = strings.TrimSpace(from[i+3:])
		from = strings.TrimSpace(from[:i])
	}
	rest := strings.TrimSpace(line[idx+3:])
	// Split label off first using the canonical " : " separator
	// (a bare `:` with both sides padded). This keeps any `:::` that
	// happens to be inside the label (e.g. `A --> B : use ::: op`)
	// from being misread as endpoint CSS shorthand.
	to := rest
	label := ""
	if li := labelSeparatorIndex(rest); li >= 0 {
		to = strings.TrimSpace(rest[:li])
		label = parserutil.ExpandLineBreaks(strings.TrimSpace(rest[li+1:]))
	}
	toCSS := ""
	if i := strings.LastIndex(to, ":::"); i >= 0 {
		toCSS = strings.TrimSpace(to[i+3:])
		to = strings.TrimSpace(to[:i])
	}
	if from == "" || to == "" {
		return diagram.StateTransition{}, "", "", false
	}
	return diagram.StateTransition{From: from, To: to, Label: label}, fromCSS, toCSS, true
}

// labelSeparatorIndex returns the byte index of the first `:` that
// separates the transition endpoint from its label, or -1 if there
// is none. The colon must not be part of a `:::` CSS-class shorthand
// (handled separately on the endpoint token). Mermaid accepts the
// label colon with or without surrounding whitespace.
func labelSeparatorIndex(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != ':' {
			continue
		}
		// Skip `:::` (the colon is part of CSS shorthand, not a
		// label separator).
		if i+1 < len(s) && s[i+1] == ':' {
			i += 2 // jump past the next two colons of `:::`
			continue
		}
		if i > 0 && s[i-1] == ':' {
			continue
		}
		return i
	}
	return -1
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
