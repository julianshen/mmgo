// Package block parses Mermaid block-beta diagram syntax.
package block

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.BlockDiagram, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	d := &diagram.BlockDiagram{}
	lineNum := 0
	headerSeen := false
	nodeIdx := make(map[string]int)

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(parserutil.StripComment(scanner.Text()))
		if line == "" {
			continue
		}
		if !headerSeen {
			if line != "block-beta" {
				return nil, fmt.Errorf("line %d: expected 'block-beta' header, got %q", lineNum, line)
			}
			headerSeen = true
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accTitle"); ok {
			d.AccTitle = v
			continue
		}
		if v, ok := parserutil.MatchKeywordValue(line, "accDescr"); ok {
			d.AccDescr = v
			continue
		}
		if rest, ok := strings.CutPrefix(line, "columns "); ok {
			n, err := strconv.Atoi(strings.TrimSpace(rest))
			if err == nil && n > 0 {
				d.Columns = n
			}
			continue
		}
		parseLine(d, line, nodeIdx)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing block-beta header")
	}
	return d, nil
}

func parseLine(d *diagram.BlockDiagram, line string, nodeIdx map[string]int) {
	// Arrow line first so node tokens with shapes don't get misidentified.
	if from, to, label, ok := matchArrow(line); ok {
		ensureNode(d, nodeIdx, from, "", diagram.BlockShapeRect)
		ensureNode(d, nodeIdx, to, "", diagram.BlockShapeRect)
		d.Edges = append(d.Edges, diagram.BlockEdge{From: from, To: to, Label: label})
		return
	}
	// Space-separated block tokens on a single line.
	for _, tok := range tokenize(line) {
		id, label, shape := parseNodeToken(tok)
		if id != "" {
			ensureNode(d, nodeIdx, id, label, shape)
		}
	}
}

func ensureNode(d *diagram.BlockDiagram, idx map[string]int, id, label string, shape diagram.BlockShape) {
	if label == "" {
		label = id
	}
	if existing, ok := idx[id]; ok {
		// Upgrade shape/label on redeclaration with explicit content.
		if label != id {
			d.Nodes[existing].Label = label
		}
		if shape != diagram.BlockShapeRect {
			d.Nodes[existing].Shape = shape
		}
		return
	}
	idx[id] = len(d.Nodes)
	d.Nodes = append(d.Nodes, diagram.BlockNode{ID: id, Label: label, Shape: shape})
}

// parseNodeToken accepts shape-bracketed forms.
//
// Shape lexicon, in delimiter precedence order so longer pairs
// match before their shorter prefixes:
//
//	id(((Label)))  → double-circle
//	id((Label))    → circle
//	id([Label])    → stadium
//	id[(Label)]    → cylinder
//	id[[Label]]    → subroutine
//	id[Label]      → rect
//	id{{Label}}    → hexagon
//	id{Label}      → diamond
//	id(Label)      → round
func parseNodeToken(tok string) (id, label string, shape diagram.BlockShape) {
	if i := strings.IndexAny(tok, "[({"); i > 0 {
		id = tok[:i]
		rest := tok[i:]
		switch {
		case strings.HasPrefix(rest, "((("):
			return id, extractBetween(rest, "(((", ")))"), diagram.BlockShapeDoubleCircle
		case strings.HasPrefix(rest, "(("):
			return id, extractBetween(rest, "((", "))"), diagram.BlockShapeCircle
		case strings.HasPrefix(rest, "(["):
			return id, extractBetween(rest, "([", "])"), diagram.BlockShapeStadium
		case strings.HasPrefix(rest, "[("):
			return id, extractBetween(rest, "[(", ")]"), diagram.BlockShapeCylinder
		case strings.HasPrefix(rest, "[["):
			return id, extractBetween(rest, "[[", "]]"), diagram.BlockShapeSubroutine
		case strings.HasPrefix(rest, "["):
			return id, extractBetween(rest, "[", "]"), diagram.BlockShapeRect
		case strings.HasPrefix(rest, "{{"):
			return id, extractBetween(rest, "{{", "}}"), diagram.BlockShapeHexagon
		case strings.HasPrefix(rest, "("):
			return id, extractBetween(rest, "(", ")"), diagram.BlockShapeRound
		case strings.HasPrefix(rest, "{"):
			return id, extractBetween(rest, "{", "}"), diagram.BlockShapeDiamond
		}
	}
	return tok, "", diagram.BlockShapeRect
}

func extractBetween(s, open, close string) string {
	s = strings.TrimPrefix(s, open)
	s = strings.TrimSuffix(s, close)
	s = strings.TrimSpace(s)
	return strings.Trim(s, "\"")
}

func matchArrow(line string) (from, to, label string, ok bool) {
	// Find the arrow token outside brackets so labels like
	// "a[x --> y]" aren't misread as an arrow.
	idx := findArrowOutsideBrackets(line, "-->")
	if idx < 0 {
		idx = findArrowOutsideBrackets(line, "---")
		if idx < 0 {
			return "", "", "", false
		}
	}
	fromRaw := strings.TrimSpace(line[:idx])
	rest := strings.TrimSpace(line[idx+3:])

	// `|label|` pipe-style label
	if strings.HasPrefix(rest, "|") {
		end := strings.Index(rest[1:], "|")
		if end >= 0 {
			label = strings.TrimSpace(rest[1 : end+1])
			rest = strings.TrimSpace(rest[end+2:])
		}
	}
	if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
		label = strings.TrimSpace(rest[colonIdx+1:])
		rest = strings.TrimSpace(rest[:colonIdx])
	}
	from = firstID(fromRaw)
	to = firstID(rest)
	if from == "" || to == "" {
		return "", "", "", false
	}
	return from, to, label, true
}

func findArrowOutsideBrackets(line, token string) int {
	depth := 0
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case '[', '(', '{':
			depth++
			continue
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 && i+len(token) <= len(line) && line[i:i+len(token)] == token {
			return i
		}
	}
	return -1
}

func firstID(s string) string {
	id, _, _ := parseNodeToken(s)
	return id
}

// tokenize splits on whitespace but keeps bracket-enclosed regions intact.
func tokenize(line string) []string {
	var out []string
	var cur strings.Builder
	var closers []byte
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		if len(closers) > 0 {
			cur.WriteByte(c)
			if c == closers[len(closers)-1] {
				closers = closers[:len(closers)-1]
			}
			continue
		}
		switch c {
		case '[':
			closers = append(closers, ']')
			cur.WriteByte(c)
		case '(':
			closers = append(closers, ')')
			cur.WriteByte(c)
		case '{':
			closers = append(closers, '}')
			cur.WriteByte(c)
		case ' ', '\t':
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	return out
}
