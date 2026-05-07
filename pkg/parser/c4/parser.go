// Package c4 parses Mermaid C4 diagram syntax (Context, Container,
// Component, Dynamic, Deployment).
package c4

import (
	"bufio"
	"fmt"
	"io"
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
			if len(stack) == 1 {
				return nil, fmt.Errorf("line %d: unmatched '}' (no open Boundary)", lineNum)
			}
			stack = stack[:len(stack)-1]
			continue
		}
		// Boundary opener: `Boundary(...) {`, `System_Boundary(...) {`, etc.
		// The trailing `{` may live on the same line or be the only
		// content of the next line.
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
	args := splitArgs(rest)
	if len(args) < 3 {
		return diagram.C4Relation{}, false
	}
	rel := diagram.C4Relation{
		From:  args[0],
		To:    args[1],
		Label: parserutil.Unquote(args[2]),
	}
	if len(args) >= 4 {
		rel.Technology = parserutil.Unquote(args[3])
	}
	return rel, true
}

func parseElement(kind diagram.C4ElementKind, rest string) (diagram.C4Element, bool) {
	rest = strings.TrimSuffix(rest, ")")
	args := splitArgs(rest)
	if len(args) < 2 {
		return diagram.C4Element{}, false
	}
	elem := diagram.C4Element{
		ID:    args[0],
		Kind:  kind,
		Label: parserutil.Unquote(args[1]),
	}
	if kind == diagram.C4ElementContainer || kind == diagram.C4ElementContainerDB ||
		kind == diagram.C4ElementComponent {
		if len(args) >= 3 {
			elem.Technology = parserutil.Unquote(args[2])
		}
		if len(args) >= 4 {
			elem.Description = parserutil.Unquote(args[3])
		}
	} else if len(args) >= 3 {
		elem.Description = parserutil.Unquote(args[2])
	}
	return elem, true
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


// boundaryKeywords pairs each documented boundary keyword with
// its kind. Ordered longest-first so `Container_Boundary(` wins
// over `Boundary(`.
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
	close := strings.LastIndex(rest, ")")
	if close < 0 {
		return "", false, fmt.Errorf("Boundary: missing closing ')'")
	}
	args = rest[:close]
	tail := strings.TrimSpace(rest[close+1:])
	switch tail {
	case "":
		return args, false, nil
	case "{":
		return args, true, nil
	}
	return "", false, fmt.Errorf("Boundary: unexpected trailing content after ')': %q", tail)
}

// parseBoundary turns a `Boundary(alias, "label", ?type, ?tags, $link)`
// argument list into a fresh C4Boundary. Today only the positional
// alias / label / optional type are recognised; named-arg `$tags=` /
// `$link=` parsing lands in Phase 3.
func parseBoundary(kind diagram.C4BoundaryKind, rest string) (*diagram.C4Boundary, error) {
	args := splitArgs(rest)
	if len(args) < 1 {
		return nil, fmt.Errorf("Boundary: requires at least an alias")
	}
	b := &diagram.C4Boundary{Kind: kind, ID: args[0]}
	if len(args) >= 2 {
		b.Label = parserutil.Unquote(args[1])
	}
	if len(args) >= 3 {
		b.Type = parserutil.Unquote(args[2])
	}
	return b, nil
}
