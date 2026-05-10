// Package c4 parses Mermaid C4 diagram syntax (Context, Container,
// Component, Dynamic, Deployment).
package c4

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

var headerVariants = map[string]diagram.C4Variant{
	"C4Context":    diagram.C4VariantContext,
	"C4Container":  diagram.C4VariantContainer,
	"C4Component":  diagram.C4VariantComponent,
	"C4Dynamic":    diagram.C4VariantDynamic,
	"C4Deployment": diagram.C4VariantDeployment,
}

// elementKeywords is ordered by keyword length (longest first) so
// that keyword dispatch is deterministic even though the `(` suffix
// already prevents prefix ambiguity (e.g. `Container(` vs
// `ContainerDb(`). Covers every documented C4 element keyword
// across Context / Container / Component / Deployment variants.
var elementKeywords = []struct {
	kw   string
	kind diagram.C4ElementKind
}{
	// 19+ char keywords first.
	{"ComponentQueue_Ext", diagram.C4ElementComponentQueueExt},
	{"ContainerQueue_Ext", diagram.C4ElementContainerQueueExt},
	{"ContainerDb_Ext", diagram.C4ElementContainerDBExt},
	{"SystemQueue_Ext", diagram.C4ElementSystemQueueExt},
	{"ComponentDb_Ext", diagram.C4ElementComponentDBExt},
	{"Component_Ext", diagram.C4ElementComponentExt},
	{"ContainerDb", diagram.C4ElementContainerDB},
	{"ContainerQueue", diagram.C4ElementContainerQueue},
	{"Container_Ext", diagram.C4ElementContainerExt},
	{"ComponentDb", diagram.C4ElementComponentDB},
	{"ComponentQueue", diagram.C4ElementComponentQueue},
	{"SystemDb_Ext", diagram.C4ElementSystemDBExt},
	{"SystemQueue", diagram.C4ElementSystemQueue},
	{"Deployment_Node", diagram.C4ElementDeploymentNode},
	{"Person_Ext", diagram.C4ElementPersonExt},
	{"System_Ext", diagram.C4ElementSystemExt},
	{"Component", diagram.C4ElementComponent},
	{"Container", diagram.C4ElementContainer},
	{"SystemDb", diagram.C4ElementSystemDB},
	// `Node`, `Node_L`, `Node_R` are aliases for Deployment_Node
	// per the spec — the underscore-suffixed variants pin the
	// node to a left/right column. We map all three to the same
	// kind for now; layout uses the Direction-style hints later.
	{"Node_L", diagram.C4ElementDeploymentNode},
	{"Node_R", diagram.C4ElementDeploymentNode},
	{"System", diagram.C4ElementSystem},
	{"Person", diagram.C4ElementPerson},
	{"Node", diagram.C4ElementDeploymentNode},
}

// c4Frame is one entry on the boundary scope stack. boundary is
// nil for the top-level (diagram) scope; otherwise it's the open
// boundary that newly-parsed elements should attach to.
//
// pendingBrace=true marks a frame whose `Boundary(...)` line did
// NOT carry an inline `{` — the next non-blank line must be a
// bare `{` to actually open the scope. Anything else is rejected
// so an element line can't silently land in the wrong scope.
type c4Frame struct {
	boundary     *diagram.C4Boundary
	pendingBrace bool
}

func Parse(r io.Reader) (*diagram.C4Diagram, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	d := &diagram.C4Diagram{}
	lineNum := 0
	headerSeen := false
	var accDescrLines []string
	inAccDescrBlock := false
	stack := []c4Frame{{}}

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			variant, ok := headerVariants[line]
			if !ok {
				return nil, fmt.Errorf("line %d: expected C4* header, got %q", lineNum, line)
			}
			d.Variant = variant
			headerSeen = true
			continue
		}
		if inAccDescrBlock {
			if line == "}" {
				d.AccDescr = strings.Join(accDescrLines, "\n")
				accDescrLines = accDescrLines[:0]
				inAccDescrBlock = false
				continue
			}
			accDescrLines = append(accDescrLines, line)
			continue
		}
		if line == "accDescr {" || line == "accDescr{" {
			inAccDescrBlock = true
			continue
		}
		top := &stack[len(stack)-1]
		// A bare `{` after a `Boundary(...)` whose brace lived on
		// a separate line activates that frame's scope.
		if top.pendingBrace {
			if line != "{" {
				return nil, fmt.Errorf("line %d: expected '{' to open Boundary %q, got %q", lineNum, top.boundary.ID, line)
			}
			top.pendingBrace = false
			continue
		}
		if line == "}" {
			// Only treat `}` as an error when we know it was
			// supposed to close an open Boundary. A stray `}` at
			// top level may belong to another brace-delimited
			// construct mmgo doesn't yet recognise (e.g. a
			// Deployment_Node block with `{ ... }`); silently
			// skipping is safer than aborting the whole parse.
			if len(stack) == 1 {
				continue
			}
			stack = stack[:len(stack)-1]
			continue
		}
		// The trailing `{` may live on the same line or as the
		// only content of the next line; pendingBrace handles
		// the latter.
		if kind, rest, ok := matchBoundaryKeyword(line); ok {
			args, opened, err := splitBoundaryHead(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			b, err := parseBoundary(kind, args)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			parent := top.boundary
			if parent == nil {
				d.Boundaries = append(d.Boundaries, b)
			} else {
				parent.Boundaries = append(parent.Boundaries, b)
			}
			stack = append(stack, c4Frame{boundary: b, pendingBrace: !opened})
			continue
		}
		if err := parseLine(d, line, top.boundary); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if inAccDescrBlock {
		return nil, fmt.Errorf("unterminated accDescr { ... } block")
	}
	if len(stack) > 1 {
		return nil, fmt.Errorf("unterminated Boundary block %q", stack[len(stack)-1].boundary.ID)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing C4 header")
	}
	return d, nil
}

func parseLine(d *diagram.C4Diagram, line string, scope *diagram.C4Boundary) error {
	if v, ok := parserutil.MatchKeywordValue(line, "title"); ok {
		d.Title = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
		d.AccTitle = v
		return nil
	}
	if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
		d.AccDescr = v
		return nil
	}
	switch line {
	case "LAYOUT_TOP_DOWN", "LAYOUT_TOP_DOWN()":
		d.Direction = diagram.C4LayoutTopDown
		return nil
	case "LAYOUT_LEFT_RIGHT", "LAYOUT_LEFT_RIGHT()":
		d.Direction = diagram.C4LayoutLeftRight
		return nil
	case "SHOW_LEGEND", "SHOW_LEGEND()":
		d.ShowLegend = true
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "UpdateElementStyle("); ok {
		parseUpdateElementStyle(d, strings.TrimSuffix(rest, ")"))
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "UpdateRelStyle("); ok {
		parseUpdateRelStyle(d, strings.TrimSuffix(rest, ")"))
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "UpdateLayoutConfig("); ok {
		parseUpdateLayoutConfig(d, strings.TrimSuffix(rest, ")"))
		return nil
	}
	if rest, ok := strings.CutPrefix(line, "RelIndex("); ok {
		if rel, ok := parseRelIndex(strings.TrimSuffix(rest, ")")); ok {
			d.Relations = append(d.Relations, rel)
		}
		return nil
	}
	if dir, args, ok := matchRelation(line); ok {
		rel, ok := parseRelation(args)
		if !ok {
			return nil
		}
		rel.Direction = dir
		d.Relations = append(d.Relations, rel)
		return nil
	}
	// Element: Person(id, "label"[, "description"])
	//          Container(id, "label", "technology"[, "description"])
	if kind, rest, ok := matchElementKeyword(line); ok {
		if elem, ok := parseElement(kind, rest); ok {
			idx := len(d.Elements)
			d.Elements = append(d.Elements, elem)
			if scope != nil {
				scope.Elements = append(scope.Elements, idx)
			}
		}
	}
	return nil
}

// relKeywords is ordered longest-first so `Rel_Right(` wins over
// `Rel_R(` and `Rel_Back(` wins over `Rel(`. Both the long-form
// (`Rel_Up`, `Rel_Down`, `Rel_Left`, `Rel_Right`) and short-form
// (`Rel_U`/`_D`/`_L`/`_R`) directions are recognised.
var relKeywords = []struct {
	kw  string
	dir diagram.C4RelDirection
}{
	{"Rel_Right", diagram.C4RelRight},
	{"Rel_Back", diagram.C4RelBack},
	{"Rel_Down", diagram.C4RelDown},
	{"Rel_Left", diagram.C4RelLeft},
	{"BiRel", diagram.C4RelBi},
	{"Rel_Up", diagram.C4RelUp},
	{"Rel_U", diagram.C4RelUp},
	{"Rel_D", diagram.C4RelDown},
	{"Rel_L", diagram.C4RelLeft},
	{"Rel_R", diagram.C4RelRight},
	{"Rel", diagram.C4RelDefault},
}

func matchRelation(line string) (diagram.C4RelDirection, string, bool) {
	for _, rk := range relKeywords {
		if rest, ok := strings.CutPrefix(line, rk.kw+"("); ok {
			return rk.dir, rest, true
		}
	}
	return 0, "", false
}

func matchElementKeyword(line string) (diagram.C4ElementKind, string, bool) {
	for _, ek := range elementKeywords {
		if rest, ok := strings.CutPrefix(line, ek.kw+"("); ok {
			return ek.kind, rest, true
		}
	}
	return 0, "", false
}

func parseRelation(rest string) (diagram.C4Relation, bool) {
	rest = strings.TrimSuffix(rest, ")")
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 3 {
		return diagram.C4Relation{}, false
	}
	rel := diagram.C4Relation{
		From:  pos[0],
		To:    pos[1],
		Label: parserutil.Unquote(pos[2]),
	}
	if len(pos) >= 4 {
		rel.Technology = parserutil.Unquote(pos[3])
	}
	if v, ok := named["techn"]; ok && v != "" {
		rel.Technology = v
	}
	rel.Tags = named["tags"]
	rel.Link = named["link"]
	rel.Sprite = named["sprite"]
	if v, ok := named["offsetX"]; ok {
		rel.OffsetX = parseFloatOrZero(v)
	}
	if v, ok := named["offsetY"]; ok {
		rel.OffsetY = parseFloatOrZero(v)
	}
	return rel, true
}

// parseFloatOrZero accepts the documented numeric forms ("12", "-5",
// "1.5") and returns 0 on anything malformed — Mermaid's equivalent
// silently no-ops the offset rather than rejecting the whole line.
func parseFloatOrZero(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func parseElement(kind diagram.C4ElementKind, rest string) (diagram.C4Element, bool) {
	rest = strings.TrimSuffix(rest, ")")
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 2 {
		return diagram.C4Element{}, false
	}
	elem := diagram.C4Element{
		ID:    pos[0],
		Kind:  kind,
		Label: parserutil.Unquote(pos[1]),
	}
	// Container / Component / their _Ext / DB / Queue variants put
	// technology at positional 2, description at 3. When `?techn=` /
	// `$descr=` consumes its slot, the positional cursor advances —
	// otherwise `Container(id, "L", ?techn="x", "descr")` would
	// silently drop the description.
	// nonEmpty: an empty named value (`$descr=""`) must not clobber
	// the positional fallback — it's the user signalling "no override"
	// rather than "set to empty".
	nonEmpty := func(key string) (string, bool) {
		v, ok := named[key]
		if !ok || v == "" {
			return "", false
		}
		return v, true
	}
	cursor := 2
	if takesTechnology(kind) {
		if v, ok := nonEmpty("techn"); ok {
			elem.Technology = v
		} else if cursor < len(pos) {
			elem.Technology = parserutil.Unquote(pos[cursor])
			cursor++
		}
	}
	if v, ok := nonEmpty("descr"); ok {
		elem.Description = v
	} else if cursor < len(pos) {
		elem.Description = parserutil.Unquote(pos[cursor])
	}
	elem.Tags = named["tags"]
	elem.Link = named["link"]
	elem.Sprite = named["sprite"]
	return elem, true
}

// parseUpdateElementStyle parses
// `UpdateElementStyle(kind, $bgColor=…, $fontColor=…, $borderColor=…)`
// into d.ElementStyles[kind]. Multiple calls for the same kind merge
// — empty fields don't clobber a prior non-empty.
func parseUpdateElementStyle(d *diagram.C4Diagram, rest string) {
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 1 || pos[0] == "" {
		return
	}
	if d.ElementStyles == nil {
		d.ElementStyles = make(map[string]diagram.C4ElementStyleOverride)
	}
	cur := d.ElementStyles[pos[0]]
	if v := named["bgColor"]; v != "" {
		cur.BgColor = v
	}
	if v := named["fontColor"]; v != "" {
		cur.FontColor = v
	}
	if v := named["borderColor"]; v != "" {
		cur.BorderColor = v
	}
	d.ElementStyles[pos[0]] = cur
}

// parseUpdateRelStyle parses
// `UpdateRelStyle(from, to, $textColor=…, $lineColor=…, $offsetX=…,
// $offsetY=…)` into d.RelStyles["from->to"].
func parseUpdateRelStyle(d *diagram.C4Diagram, rest string) {
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 2 {
		return
	}
	key := pos[0] + "->" + pos[1]
	if d.RelStyles == nil {
		d.RelStyles = make(map[string]diagram.C4RelStyleOverride)
	}
	cur := d.RelStyles[key]
	if v := named["textColor"]; v != "" {
		cur.TextColor = v
	}
	if v := named["lineColor"]; v != "" {
		cur.LineColor = v
	}
	if v, ok := named["offsetX"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cur.OffsetX = f
		}
	}
	if v, ok := named["offsetY"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cur.OffsetY = f
		}
	}
	d.RelStyles[key] = cur
}

// parseUpdateLayoutConfig parses
// `UpdateLayoutConfig($c4ShapeInRow=N, $c4BoundaryInRow=M)`. Both
// args are optional named ints; absent values stay zero so the
// renderer sees "no override".
func parseUpdateLayoutConfig(d *diagram.C4Diagram, rest string) {
	_, named := splitPositionalAndNamed(splitArgs(rest))
	if v, ok := named["c4ShapeInRow"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			d.LayoutConfig.ShapesInRow = n
		}
	}
	if v, ok := named["c4BoundaryInRow"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			d.LayoutConfig.BoundariesInRow = n
		}
	}
}

// parseRelIndex parses `RelIndex(N, from, to, "label", ?"techn",
// $named=...)`. The numeric first positional arg is the sequence
// index; the rest mirror parseRelation's surface so all the named-
// arg overrides flow through.
func parseRelIndex(rest string) (diagram.C4Relation, bool) {
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 4 {
		return diagram.C4Relation{}, false
	}
	idx, err := strconv.Atoi(strings.TrimSpace(pos[0]))
	if err != nil || idx < 1 {
		return diagram.C4Relation{}, false
	}
	rel := diagram.C4Relation{
		Index: idx,
		From:  pos[1],
		To:    pos[2],
		Label: parserutil.Unquote(pos[3]),
	}
	if len(pos) >= 5 {
		rel.Technology = parserutil.Unquote(pos[4])
	}
	if v, ok := named["techn"]; ok && v != "" {
		rel.Technology = v
	}
	rel.Tags = named["tags"]
	rel.Link = named["link"]
	rel.Sprite = named["sprite"]
	if v, ok := named["offsetX"]; ok {
		rel.OffsetX = parseFloatOrZero(v)
	}
	if v, ok := named["offsetY"]; ok {
		rel.OffsetY = parseFloatOrZero(v)
	}
	return rel, true
}

// takesTechnology reports whether the kind's call signature reserves
// a positional slot for technology. Mermaid's spec covers Container /
// Component plus all their _Ext / DB / Queue variants; everything
// else (Person, System, Boundary aliases, Deployment_Node) jumps
// straight from label to description.
func takesTechnology(kind diagram.C4ElementKind) bool {
	switch kind {
	case diagram.C4ElementContainer, diagram.C4ElementContainerExt,
		diagram.C4ElementContainerDB, diagram.C4ElementContainerDBExt,
		diagram.C4ElementContainerQueue, diagram.C4ElementContainerQueueExt,
		diagram.C4ElementComponent, diagram.C4ElementComponentExt,
		diagram.C4ElementComponentDB, diagram.C4ElementComponentDBExt,
		diagram.C4ElementComponentQueue, diagram.C4ElementComponentQueueExt:
		return true
	}
	return false
}

// splitArgs splits a parenthesized argument list on commas outside
// quotes and trims each field. Thin wrapper around the shared
// parserutil.SplitUnquotedCommas, which handles backslash escapes
// inside quoted spans.
func splitArgs(s string) []string {
	raw := parserutil.SplitUnquotedCommas(s)
	for i, v := range raw {
		raw[i] = strings.TrimSpace(v)
	}
	return raw
}

// splitPositionalAndNamed partitions a parsed arg list into
// positional values and named `$key=val` / `?key=val` pairs. Keys
// drop the sigil so callers can switch on a flat string.
func splitPositionalAndNamed(args []string) (positional []string, named map[string]string) {
	for _, a := range args {
		if k, v, ok := splitNamed(a); ok {
			if named == nil {
				named = make(map[string]string)
			}
			named[k] = v
			continue
		}
		positional = append(positional, a)
	}
	return positional, named
}

// splitNamed returns (key, value, true) when arg looks like
// `$key="val"` or `?key=val`. An empty assignment (`$key=`) is still
// recognised as named — leaking the literal `$key=""` token into
// positional slots would shift downstream positional indices and
// break semantic fields. Override sites guard against an empty
// value clobbering a positional value via a non-empty check there.
func splitNamed(arg string) (key, value string, ok bool) {
	rest, found := strings.CutPrefix(arg, "$")
	if !found {
		if rest, found = strings.CutPrefix(arg, "?"); !found {
			return "", "", false
		}
	}
	k, v, found := strings.Cut(rest, "=")
	if !found || k == "" {
		return "", "", false
	}
	return k, parserutil.Unquote(strings.TrimSpace(v)), true
}

// boundaryKeywords pairs each documented boundary keyword with
// its kind. Order matters only insofar as any keyword that is a
// prefix of another must come AFTER the longer one — here the
// only such pair is `Boundary` (a suffix of every other entry),
// which is listed last.
var boundaryKeywords = []struct {
	kw   string
	kind diagram.C4BoundaryKind
}{
	{"Enterprise_Boundary", diagram.C4BoundaryEnterprise},
	{"Container_Boundary", diagram.C4BoundaryContainer},
	{"System_Boundary", diagram.C4BoundarySystem},
	{"Boundary", diagram.C4BoundaryGeneric},
}

func matchBoundaryKeyword(line string) (diagram.C4BoundaryKind, string, bool) {
	for _, bk := range boundaryKeywords {
		if rest, ok := strings.CutPrefix(line, bk.kw+"("); ok {
			return bk.kind, rest, true
		}
	}
	return 0, "", false
}

// splitBoundaryHead extracts the parenthesized argument list from
// the rest of a `Boundary(args) [{]` line. Returns (args, opened,
// err) where opened reports whether a `{` was present on the same
// line. A missing `{` is fine — the next non-blank line may carry
// it standalone.
func splitBoundaryHead(rest string) (args string, opened bool, err error) {
	rparen := strings.LastIndex(rest, ")")
	if rparen < 0 {
		return "", false, fmt.Errorf("boundary: missing closing ')'")
	}
	args = rest[:rparen]
	tail := strings.TrimSpace(rest[rparen+1:])
	switch tail {
	case "":
		return args, false, nil
	case "{":
		return args, true, nil
	}
	return "", false, fmt.Errorf("boundary: unexpected trailing content after ')': %q", tail)
}

// parseBoundary turns a `Boundary(alias, "label", ?typeHint, …)` arg
// list into a fresh C4Boundary. Named-arg forms (`$tags=`, `$link=`,
// `$sprite=`) layer on top of the positional triple.
func parseBoundary(kind diagram.C4BoundaryKind, rest string) (*diagram.C4Boundary, error) {
	pos, named := splitPositionalAndNamed(splitArgs(rest))
	if len(pos) < 1 || pos[0] == "" {
		return nil, fmt.Errorf("boundary: requires a non-empty alias")
	}
	b := &diagram.C4Boundary{Kind: kind, ID: pos[0]}
	if len(pos) >= 2 {
		b.Label = parserutil.Unquote(pos[1])
	}
	if len(pos) >= 3 {
		b.TypeHint = parserutil.Unquote(pos[2])
	}
	b.Tags = named["tags"]
	b.Link = named["link"]
	b.Sprite = named["sprite"]
	return b, nil
}
