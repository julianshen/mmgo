package mindmap

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
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

		indent := parserutil.IndentWidth(raw)
		text, shape := parseNodeContent(trimmed)
		node := &diagram.MindmapNode{Text: text, Shape: shape}

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


func parseNodeContent(s string) (string, diagram.MindmapNodeShape) {
	if strings.HasPrefix(s, "((") && strings.HasSuffix(s, "))") {
		return s[2 : len(s)-2], diagram.MindmapShapeCloud
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		return s[2 : len(s)-2], diagram.MindmapShapeBang
	}
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return s[1 : len(s)-1], diagram.MindmapShapeRound
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return s[1 : len(s)-1], diagram.MindmapShapeSquare
	}
	return s, diagram.MindmapShapeDefault
}
