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

var elementKeywords = map[string]diagram.C4ElementKind{
	"Person":       diagram.C4ElementPerson,
	"Person_Ext":   diagram.C4ElementPersonExt,
	"System":       diagram.C4ElementSystem,
	"System_Ext":   diagram.C4ElementSystemExt,
	"SystemDb":     diagram.C4ElementSystemDB,
	"SystemDb_Ext": diagram.C4ElementSystemDB,
	"Container":    diagram.C4ElementContainer,
	"ContainerDb":  diagram.C4ElementContainerDB,
	"Component":    diagram.C4ElementComponent,
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
	if rest, ok := strings.CutPrefix(line, "title "); ok {
		d.Title = strings.TrimSpace(rest)
		return nil
	}
	// Relation: Rel(from, to, "label"[, "technology"])
	// Directional variants: Rel_U, Rel_D, Rel_L, Rel_R, Rel_Back
	if isRelation(line) {
		rel, ok := parseRelation(line)
		if !ok {
			return nil
		}
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

func isRelation(line string) bool {
	return strings.HasPrefix(line, "Rel(") ||
		strings.HasPrefix(line, "Rel_U(") ||
		strings.HasPrefix(line, "Rel_D(") ||
		strings.HasPrefix(line, "Rel_L(") ||
		strings.HasPrefix(line, "Rel_R(") ||
		strings.HasPrefix(line, "Rel_Back(") ||
		strings.HasPrefix(line, "BiRel(")
}

func matchElementKeyword(line string) (diagram.C4ElementKind, string, bool) {
	for kw, kind := range elementKeywords {
		if rest, ok := strings.CutPrefix(line, kw+"("); ok {
			return kind, rest, true
		}
	}
	return 0, "", false
}

func parseRelation(line string) (diagram.C4Relation, bool) {
	openIdx := strings.Index(line, "(")
	if openIdx < 0 {
		return diagram.C4Relation{}, false
	}
	inner := line[openIdx+1:]
	inner = strings.TrimSuffix(inner, ")")
	args := splitArgs(inner)
	if len(args) < 3 {
		return diagram.C4Relation{}, false
	}
	rel := diagram.C4Relation{
		From:  args[0],
		To:    args[1],
		Label: unquote(args[2]),
	}
	if len(args) >= 4 {
		rel.Technology = unquote(args[3])
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
		Label: unquote(args[1]),
	}
	if kind == diagram.C4ElementContainer || kind == diagram.C4ElementContainerDB ||
		kind == diagram.C4ElementComponent {
		if len(args) >= 3 {
			elem.Technology = unquote(args[2])
		}
		if len(args) >= 4 {
			elem.Description = unquote(args[3])
		}
	} else if len(args) >= 3 {
		elem.Description = unquote(args[2])
	}
	return elem, true
}

// splitArgs splits a parenthesized argument list on commas, respecting
// double-quoted strings.
func splitArgs(s string) []string {
	var args []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			cur.WriteByte(c)
			continue
		}
		if c == ',' && !inQuote {
			args = append(args, strings.TrimSpace(cur.String()))
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		args = append(args, strings.TrimSpace(cur.String()))
	}
	return args
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
