package class

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func Parse(r io.Reader) (*diagram.ClassDiagram, error) {
	p := &parser{
		diagram:  &diagram.ClassDiagram{},
		classIdx: make(map[string]int),
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if line != "classDiagram" {
				return nil, fmt.Errorf("line %d: expected 'classDiagram' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if err := p.parseLine(line, scanner, &lineNum); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
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
}

func (p *parser) parseLine(line string, scanner *bufio.Scanner, lineNum *int) error {
	if rest, ok := strings.CutPrefix(line, "class "); ok {
		rest = strings.TrimSpace(rest)
		if braceIdx := strings.IndexByte(rest, '{'); braceIdx >= 0 {
			name := strings.TrimSpace(rest[:braceIdx])
			return p.parseClassBody(name, scanner, lineNum)
		}
		p.ensureClass(rest)
		return nil
	}
	if rel, ok := parseRelation(line); ok {
		p.ensureClass(rel.From)
		p.ensureClass(rel.To)
		p.diagram.Relations = append(p.diagram.Relations, rel)
		return nil
	}
	return nil
}

func (p *parser) parseClassBody(name string, scanner *bufio.Scanner, lineNum *int) error {
	p.ensureClass(name)
	idx := p.classIdx[name]
	for scanner.Scan() {
		*lineNum++
		line := strings.TrimSpace(stripComment(scanner.Text()))
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
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading class body for %q: %w", name, err)
	}
	return fmt.Errorf("unclosed class body for %q", name)
}

func (p *parser) ensureClass(id string) {
	if _, ok := p.classIdx[id]; ok {
		return
	}
	p.classIdx[id] = len(p.diagram.Classes)
	p.diagram.Classes = append(p.diagram.Classes, diagram.ClassDef{ID: id, Label: id})
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
		// Extract return type after closing paren
		if closeIdx := strings.Index(line, ")"); closeIdx >= 0 {
			m.Name = strings.TrimSpace(line[:idx])
			if closeIdx+1 < len(line) {
				m.ReturnType = strings.TrimSpace(line[closeIdx+1:])
			}
		} else {
			m.Name = strings.TrimSpace(line)
		}
	} else {
		parts := strings.Fields(line)
		switch len(parts) {
		case 0:
			// empty
		case 1:
			m.Name = parts[0]
		default:
			m.ReturnType = parts[0]
			m.Name = parts[1]
		}
	}
	return m
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

// Relation arrows ordered longest-first to avoid prefix ambiguity.
var relationArrows = []struct {
	lit string
	typ diagram.RelationType
}{
	{"<|--", diagram.RelationTypeInheritance},
	{"..|>", diagram.RelationTypeRealization},
	{"*--", diagram.RelationTypeComposition},
	{"o--", diagram.RelationTypeAggregation},
	{"..>", diagram.RelationTypeDependency},
	{"-->", diagram.RelationTypeAssociation},
	{"--", diagram.RelationTypeLink},
	{"..", diagram.RelationTypeDashedLink},
}

func parseRelation(line string) (diagram.ClassRelation, bool) {
	for _, arr := range relationArrows {
		idx := strings.Index(line, arr.lit)
		if idx < 0 {
			continue
		}
		leftRaw := strings.TrimSpace(line[:idx])
		rightRaw := strings.TrimSpace(line[idx+len(arr.lit):])

		from, fromCard := extractCardinality(leftRaw)
		to, label, toCard := extractRightSide(rightRaw)

		if from == "" || to == "" {
			continue
		}

		return diagram.ClassRelation{
			From:            from,
			To:              to,
			RelationType:    arr.typ,
			Label:           label,
			FromCardinality: fromCard,
			ToCardinality:   toCard,
		}, true
	}
	return diagram.ClassRelation{}, false
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

func stripComment(line string) string {
	for i := 0; i+1 < len(line); i++ {
		if line[i] != '%' || line[i+1] != '%' {
			continue
		}
		if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
			return line[:i]
		}
	}
	return line
}
