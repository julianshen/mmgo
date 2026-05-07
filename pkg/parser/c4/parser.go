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

func Parse(r io.Reader) (*diagram.C4Diagram, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	d := &diagram.C4Diagram{}
	lineNum := 0
	headerSeen := false

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
		if err := parseLine(d, line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing C4 header")
	}
	return d, nil
}

func parseLine(d *diagram.C4Diagram, line string) error {
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
			d.Elements = append(d.Elements, elem)
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

