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
	{"!", "!", diagram.MindmapShapeBang}, // historical bang form
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

	d := &diagram.MindmapDiagram{
		CSSClasses: make(map[string]string),
	}

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

		if v, ok := parserutil.MatchKeywordValue(trimmed, "accTitle"); ok {
			d.AccTitle = v
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(trimmed, "accDescr"); ok {
			d.AccDescr = v
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "classDef "); ok {
			name, css, err := parserutil.ParseClassDefLine(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			d.CSSClasses[name] = css
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, "style "); ok {
			parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
			if len(parts) < 2 {
				return nil, fmt.Errorf("line %d: style requires an ID and CSS", lineNum)
			}
			d.Styles = append(d.Styles, diagram.MindmapStyleDef{
				NodeID: strings.TrimSpace(parts[0]),
				CSS:    parserutil.NormalizeCSS(strings.TrimSpace(parts[1])),
			})
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
				rest := strings.TrimSpace(trimmed[len(classPrefix):])
				if rest != "" {
					// `:::a b c` attaches every space-separated
					// class name in document order so authors can
					// stack overrides.
					lastNode.CSSClasses = append(lastNode.CSSClasses, strings.Fields(rest)...)
				}
			}
			continue
		}

		indent := parserutil.IndentWidth(raw)
		id, text, shape := parseNodeContent(trimmed)
		text = expandLabel(text)
		node := &diagram.MindmapNode{ID: id, Text: text, Shape: shape}
		lastNode = node

		if d.Root == nil {
			d.Root = node
			stack = []stackEntry{{node: node, level: indent}}
			continue
		}

		// A second top-level (indent <= the root's indent) line is a
		// second root. Mermaid mindmaps can have only one root; flag
		// the violation rather than silently dropping the node.
		if indent <= stack[0].level {
			return nil, fmt.Errorf("line %d: mindmap supports a single root; %q would create a second one", lineNum, trimmed)
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

// expandLabel strips surrounding backticks (the renderer handles
// markdown inside) and converts the literal `\n` escape into a real
// newline for multi-line labels.
func expandLabel(s string) string {
	s = parserutil.ExpandLineBreaks(s)
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		s = s[1 : len(s)-1]
	}
	return s
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
	// `!` enables the historical bang form `!text!`; the canonical
	// `))text((` form is also routed via this scan because it
	// starts with `)`.
	delimChars := "([){!"
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

// parseQuotedContent extracts a double-quoted description from s.
// Quoted descriptions let authors embed shape delimiter characters
// (e.g. [], (), {}) inside node text without triggering shape parsing.
//
// The Mermaid grammar supports two forms:
//
//	"description"       → NSTR
//	"`description`"     → NSTR2 (backtick-wrapped inside quotes)
//
// The caller should ensure s starts with a double-quote character.
// ok is false when the closing delimiter is not found.
func parseQuotedContent(s string) (content string, consumed int, ok bool) {
	// NSTR2: "`...`" — content cannot contain backticks or quotes.
	// The closing sequence is exactly `\".
	if len(s) >= 2 && s[1] == '`' {
		for i := 2; i < len(s); i++ {
			switch s[i] {
			case '"':
				return "", 0, false // bare quote inside NSTR2
			case '`':
				if i+1 < len(s) && s[i+1] == '"' {
					return s[2:i], i + 2, true
				}
				return "", 0, false // bare backtick not followed by quote
			}
		}
		return "", 0, false // no closing found
	}

	// NSTR: "..." — content cannot contain quotes.
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], i + 1, true
		}
	}
	return "", 0, false // no closing found
}

func parseShapeOnly(s string) (string, diagram.MindmapNodeShape) {
	for _, p := range shapePatterns {
		if !strings.HasPrefix(s, p.prefix) {
			continue
		}
		after := s[len(p.prefix):]

		// Quoted descriptions must be resolved before naive suffix
		// matching so delimiter characters inside quotes are not
		// misinterpreted as shape boundaries.
		if strings.HasPrefix(after, `"`) {
			content, consumed, ok := parseQuotedContent(after)
			if ok && content != "" && after[consumed:] == p.suffix {
				return content, p.shape
			}
		}

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
