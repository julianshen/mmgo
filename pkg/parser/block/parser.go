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

	// Each frame is the parser state for one nesting level: which
	// `Items` slice new content lands in, and which `Columns` int
	// `columns N` directives target. The bottom frame is the
	// diagram itself; deeper frames are open `block:ID ... end`
	// groups.
	type frame struct {
		items *[]diagram.BlockItem
		group *diagram.BlockGroup
		cols  *int
	}
	stack := []frame{{items: &d.Items, cols: &d.Columns}}
	current := func() frame { return stack[len(stack)-1] }
	push := func(f frame) { stack = append(stack, f) }
	pop := func() { stack = stack[:len(stack)-1] }

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
				*current().cols = n
			}
			continue
		}
		if line == "end" {
			if len(stack) == 1 {
				return nil, fmt.Errorf("line %d: 'end' with no matching 'block:' to close", lineNum)
			}
			pop()
			continue
		}
		if rest, ok := strings.CutPrefix(line, "block:"); ok {
			grp, err := parseGroupHead(rest)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			*current().items = append(*current().items, diagram.BlockItem{
				Kind: diagram.BlockItemGroup, Group: grp,
			})
			push(frame{items: &grp.Items, group: grp, cols: &grp.Columns})
			continue
		}
		if from, to, label, style, head, tail, ok := matchArrow(line); ok {
			ensureNode(d, nodeIdx, from, "", diagram.BlockShapeRect, 0)
			ensureNode(d, nodeIdx, to, "", diagram.BlockShapeRect, 0)
			d.Edges = append(d.Edges, diagram.BlockEdge{
				From: from, To: to, Label: label,
				LineStyle: style, ArrowHead: head, ArrowTail: tail,
			})
			continue
		}
		// Token line: each whitespace-separated token becomes one
		// item in the current scope. `space` / `space:N` is a
		// reserved name that emits a spacer instead of a node.
		for _, tok := range tokenize(line) {
			head, width := splitWidthSuffix(tok)
			if head == "space" {
				cols := width
				if cols <= 0 {
					cols = 1
				}
				*current().items = append(*current().items, diagram.BlockItem{
					Kind: diagram.BlockItemSpace, Cols: cols,
				})
				continue
			}
			id, label, shape := parseNodeToken(head)
			if id == "" {
				continue
			}
			ensureNode(d, nodeIdx, id, label, shape, width)
			*current().items = append(*current().items, diagram.BlockItem{
				Kind: diagram.BlockItemNodeRef, NodeID: id,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing block-beta header")
	}
	if len(stack) != 1 {
		return nil, fmt.Errorf("missing 'end' for block: %q", stack[len(stack)-1].group.ID)
	}
	return d, nil
}

// parseGroupHead parses the right-hand side of `block:` — i.e.
// `ID`, `ID:N`, `ID["label"]`, or `ID["label"]:N` — into a fresh
// BlockGroup. The group's `Items` are filled by subsequent lines
// until a matching `end`.
func parseGroupHead(s string) (*diagram.BlockGroup, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("block: requires an identifier")
	}
	head, width := splitWidthSuffix(s)
	id, label, _ := parseNodeToken(head)
	if id == "" {
		return nil, fmt.Errorf("block: requires an identifier, got %q", s)
	}
	g := &diagram.BlockGroup{ID: id, Width: width}
	if label != "" && label != id {
		g.Label = label
	}
	return g, nil
}

// splitWidthSuffix peels an optional trailing `:N` width
// modifier off a token. The colon must come after the structural
// closing bracket (so a bracket-enclosed label that itself
// contains `:` is left alone). Returns (tok, 0) when no width
// suffix is present.
func splitWidthSuffix(tok string) (head string, width int) {
	searchFrom := 0
	if idx := strings.LastIndexAny(tok, "])}"); idx >= 0 {
		searchFrom = idx + 1
	}
	colonAt := strings.LastIndex(tok[searchFrom:], ":")
	if colonAt < 0 {
		return tok, 0
	}
	abs := searchFrom + colonAt
	n, err := strconv.Atoi(tok[abs+1:])
	if err != nil || n <= 0 {
		return tok, 0
	}
	return tok[:abs], n
}

func ensureNode(d *diagram.BlockDiagram, idx map[string]int, id, label string, shape diagram.BlockShape, width int) {
	if label == "" {
		label = id
	}
	if existing, ok := idx[id]; ok {
		// Upgrade shape/label/width on redeclaration with explicit
		// content; otherwise keep the prior values so a later
		// reference doesn't blank a previously-set label.
		if label != id {
			d.Nodes[existing].Label = label
		}
		if shape != diagram.BlockShapeRect {
			d.Nodes[existing].Shape = shape
		}
		if width > 0 {
			d.Nodes[existing].Width = width
		}
		return
	}
	idx[id] = len(d.Nodes)
	d.Nodes = append(d.Nodes, diagram.BlockNode{ID: id, Label: label, Shape: shape, Width: width})
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
//	id[/Label/]    → parallelogram
//	id[\Label\]    → parallelogram-alt
//	id[/Label\]    → trapezoid
//	id[\Label/]    → trapezoid-alt
//	id[[Label]]    → subroutine
//	id[Label]      → rect
//	id{{Label}}    → hexagon
//	id{Label}      → diamond
//	id(Label)      → round
//	id>Label]      → asymmetric
func parseNodeToken(tok string) (id, label string, shape diagram.BlockShape) {
	i := strings.IndexAny(tok, "[({>")
	if i <= 0 {
		return tok, "", diagram.BlockShapeRect
	}
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
	case strings.HasPrefix(rest, "[/") && strings.HasSuffix(rest, "/]"):
		return id, extractBetween(rest, "[/", "/]"), diagram.BlockShapeParallelogram
	case strings.HasPrefix(rest, "[\\") && strings.HasSuffix(rest, "\\]"):
		return id, extractBetween(rest, "[\\", "\\]"), diagram.BlockShapeParallelogramAlt
	case strings.HasPrefix(rest, "[/") && strings.HasSuffix(rest, "\\]"):
		return id, extractBetween(rest, "[/", "\\]"), diagram.BlockShapeTrapezoid
	case strings.HasPrefix(rest, "[\\") && strings.HasSuffix(rest, "/]"):
		return id, extractBetween(rest, "[\\", "/]"), diagram.BlockShapeTrapezoidAlt
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
	case strings.HasPrefix(rest, ">") && strings.HasSuffix(rest, "]"):
		return id, extractBetween(rest, ">", "]"), diagram.BlockShapeAsymmetric
	}
	return tok, "", diagram.BlockShapeRect
}

func extractBetween(s, open, close string) string {
	s = strings.TrimPrefix(s, open)
	s = strings.TrimSuffix(s, close)
	s = strings.TrimSpace(s)
	return strings.Trim(s, "\"")
}

// arrowSpecs maps each spec-defined arrow tail to (LineStyle,
// ArrowHead, ArrowTail). Order matters: longer tails are tried
// first so `<-->` beats `-->` and `-.->` beats `-->`. `~~~` and
// `<-->` only work as full tokens (no `-- text -->` separator).
var arrowSpecs = []struct {
	tail  string
	style diagram.LineStyle
	head  diagram.ArrowHead
	bidi  bool
	// sep is the text-form left separator (`-- text TAIL`). Empty
	// disables inline text labels for this arrow.
	sep string
}{
	{"<-->", diagram.LineStyleSolid, diagram.ArrowHeadArrow, true, ""},
	{"-.->", diagram.LineStyleDotted, diagram.ArrowHeadArrow, false, "-."},
	{"==>", diagram.LineStyleThick, diagram.ArrowHeadArrow, false, "=="},
	{"~~~", diagram.LineStyleInvisible, diagram.ArrowHeadNone, false, ""},
	{"--x", diagram.LineStyleSolid, diagram.ArrowHeadCross, false, "--"},
	{"--o", diagram.LineStyleSolid, diagram.ArrowHeadCircle, false, "--"},
	{"-->", diagram.LineStyleSolid, diagram.ArrowHeadArrow, false, "--"},
	{"---", diagram.LineStyleSolid, diagram.ArrowHeadNone, false, "--"},
}

// matchArrow recognises the full block-beta edge lexicon. Returns
// the source and target node ids, an optional label (`|x|` pipe
// form or `-- text -->` inline form), the line style, and the head
// markers. The non-spec `: label` trailing form is no longer
// accepted — colons inside a label must come from the pipe form.
func matchArrow(line string) (from, to, label string, style diagram.LineStyle, head, tail diagram.ArrowHead, ok bool) {
	for _, spec := range arrowSpecs {
		idx := findArrowOutsideBrackets(line, spec.tail)
		if idx < 0 {
			continue
		}
		fromRaw := strings.TrimSpace(line[:idx])
		rest := strings.TrimSpace(line[idx+len(spec.tail):])

		// `-- text -->` inline label form: split fromRaw on the
		// matching separator. The right segment becomes the label.
		if spec.sep != "" {
			if sepIdx := findArrowOutsideBrackets(fromRaw, " "+spec.sep+" "); sepIdx >= 0 {
				labelText := strings.TrimSpace(fromRaw[sepIdx+len(spec.sep)+2:])
				fromRaw = strings.TrimSpace(fromRaw[:sepIdx])
				label = labelText
			}
		}

		// `|label|` pipe-style label still wins over inline text
		// when both are present (mermaid behaviour).
		if strings.HasPrefix(rest, "|") {
			if end := strings.Index(rest[1:], "|"); end >= 0 {
				label = strings.TrimSpace(rest[1 : end+1])
				rest = strings.TrimSpace(rest[end+2:])
			}
		}

		from = firstID(fromRaw)
		to = firstID(rest)
		if from == "" || to == "" {
			return "", "", "", 0, 0, 0, false
		}
		style = spec.style
		head = spec.head
		tail = diagram.ArrowHeadNone
		if spec.bidi {
			tail = diagram.ArrowHeadArrow
		}
		return from, to, label, style, head, tail, true
	}
	return "", "", "", 0, 0, 0, false
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
		case '>':
			// `>` opens an asymmetric shape that closes at `]`.
			// tokenize is only called for token lines (arrow lines
			// dispatch earlier), so a stray `>` only appears here
			// as the shape prefix.
			if cur.Len() > 0 {
				closers = append(closers, ']')
			}
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
