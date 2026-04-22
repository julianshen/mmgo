package mindmap

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

var shapePatterns = []struct {
	prefix, suffix string
	shape          diagram.MindmapNodeShape
}{
	{"((", "))", diagram.MindmapShapeCircle},
	{"{{", "}}", diagram.MindmapShapeHexagon},
	{"))", "((", diagram.MindmapShapeBang},
	{"(-", "-)", diagram.MindmapShapeCloud},
	{"(", ")", diagram.MindmapShapeRound},
	{"[", "]", diagram.MindmapShapeSquare},
	{")", "(", diagram.MindmapShapeCloud},
}

const (
	iconPrefix  = "::icon("
	classPrefix = ":::"
)

func Parse(r io.Reader) (*diagram.MindmapDiagram, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false

	type stackEntry struct {
		node  *diagram.MindmapNode
		level int
	}
	var stack []stackEntry
	var lastNode *diagram.MindmapNode

	d := &diagram.MindmapDiagram{}

	for scanner.Scan() {
		lineNum++
		raw := parserutil.StripComment(scanner.Text())
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if !headerSeen {
			if trimmed != "mindmap" {
				return nil, fmt.Errorf("line %d: expected 'mindmap' header, got %q", lineNum, trimmed)
			}
			headerSeen = true
			continue
		}

		if strings.HasPrefix(trimmed, iconPrefix) {
			if lastNode != nil {
				icon := parseIconDecoration(trimmed)
				if icon != "" {
					lastNode.Icon = icon
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, classPrefix) && len(trimmed) > len(classPrefix) {
			if lastNode != nil {
				cls := strings.TrimSpace(trimmed[len(classPrefix):])
				if cls != "" {
					lastNode.Class = cls
				}
			}
			continue
		}

		indent := parserutil.IndentWidth(raw)
		id, text, shape := parseNodeContent(trimmed)
		node := &diagram.MindmapNode{ID: id, Text: text, Shape: shape}
		lastNode = node

		if d.Root == nil {
			d.Root = node
			stack = []stackEntry{{node: node, level: indent}}
			continue
		}

		for len(stack) > 1 && stack[len(stack)-1].level >= indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1].node
		parent.Children = append(parent.Children, node)
		stack = append(stack, stackEntry{node: node, level: indent})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing mindmap header")
	}
	return d, nil
}

func parseIconDecoration(s string) string {
	inner := s[len(iconPrefix):]
	closeIdx := strings.Index(inner, ")")
	if closeIdx < 0 {
		return ""
	}
	return inner[:closeIdx]
}

func parseNodeContent(s string) (id, text string, shape diagram.MindmapNodeShape) {
	delimChars := "([){"
	firstDelim := -1
	for i, ch := range s {
		if strings.ContainsRune(delimChars, ch) {
			firstDelim = i
			break
		}
	}
	if firstDelim < 0 {
		return s, s, diagram.MindmapShapeDefault
	}

	potentialID := strings.TrimSpace(s[:firstDelim])
	rest := s[firstDelim:]

	inner, shape := parseShapeOnly(rest)
	if shape == diagram.MindmapShapeDefault {
		return s, s, diagram.MindmapShapeDefault
	}

	id = potentialID
	text = inner
	if id == "" {
		id = text
	}
	return id, text, shape
}

func parseShapeOnly(s string) (string, diagram.MindmapNodeShape) {
	for _, p := range shapePatterns {
		if !strings.HasPrefix(s, p.prefix) {
			continue
		}
		after := s[len(p.prefix):]
		closeIdx := strings.Index(after, p.suffix)
		if closeIdx < 0 {
			continue
		}
		inner := after[:closeIdx]
		if inner == "" {
			continue
		}
		remaining := after[closeIdx+len(p.suffix):]
		if strings.TrimSpace(remaining) != "" {
			continue
		}
		return inner, p.shape
	}
	return s, diagram.MindmapShapeDefault
}
