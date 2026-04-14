// Package flowchart parses Mermaid flowchart/graph syntax into a
// FlowchartDiagram AST.
//
// Supported syntax (MVP scope):
//
//	graph LR            // or: flowchart TB, graph TD, etc.
//	    A[Rectangle] --> B(Rounded)
//	    B --> C{Diamond}
//	    C -->|Yes| D((Circle))
//	    %% comments are stripped to end of line
//
// Supported node shapes: rectangle [], rounded-rectangle (), stadium ([]),
// subroutine [[]], cylinder [()], circle (()), asymmetric >], diamond {},
// hexagon {{}}, parallelogram [//], parallelogram-alt [\\],
// trapezoid [/\], trapezoid-alt [\/], double-circle ((())).
//
// Supported edges:
//   - solid:  -->, ---, and long-dash variants (--->, ---->, ---- ...)
//   - dotted: -.->, -.-, and extended-dot variants (-..->, -...-  ...)
//   - thick:  ==>, ===, and long variants (===>, ====>, ==== ...)
//
// Edge labels:
//   - pipe form:   A -->|label| B
//   - inline form: A -- label --> B     (solid family)
//                  A == label ==> B     (thick family)
//
// Chained edges (`A --> B --> C`) emit one edge per segment in order.
// Arrow detection skips bracketed regions and double-quoted strings,
// so labels like `A[--> not an arrow] --> B` parse correctly.
//
// TODO(features): subgraphs (nested `subgraph` ... `end` blocks),
// style/classDef/class directives, init directives (%%{init: ...}%%),
// additional arrow endpoints (-x, -o), dotted inline labels
// (`-. label .->`), and Unicode node IDs. These are planned for a
// follow-up PR once the renderer lands.
package flowchart

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// Parse reads a flowchart/graph definition from r and returns the
// resulting FlowchartDiagram. Errors include a 1-based line number
// pointing to the offending input.
func Parse(r io.Reader) (*diagram.FlowchartDiagram, error) {
	p := &parser{
		nodeIndex: make(map[string]int),
	}
	scanner := bufio.NewScanner(r)
	// Bump the line buffer past the 64 KB default so generated diagrams
	// with long inline labels (HTML, rich text) don't hit ErrTooLong.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNum := 0
	headerSeen := false
	for scanner.Scan() {
		lineNum++
		raw := stripComment(scanner.Text())
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		if !headerSeen {
			if err := p.parseHeader(line); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
			headerSeen = true
			continue
		}

		if err := p.parseLine(line); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	if !headerSeen {
		return nil, fmt.Errorf("missing graph/flowchart header")
	}
	return p.diagram, nil
}

// parser holds mutable state during line-by-line parsing.
type parser struct {
	diagram *diagram.FlowchartDiagram
	// nodeIndex maps node ID to its position in diagram.Nodes so we
	// can merge shape/label info from multiple definitions of the
	// same node without duplicating it.
	nodeIndex map[string]int
}

// stripComment removes the "%%" to-end-of-line comment from a raw line.
// `%%` is only treated as a comment when it appears at the start of the
// line or is preceded by whitespace; this keeps `%%` inside a node
// label like `A[100%%]` intact.
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

// parseHeader recognizes the "graph <DIR>" / "flowchart <DIR>" header
// and initializes the diagram. DIR defaults to TB if omitted. The
// keyword match requires a word boundary so `grapha LR` is rejected.
func (p *parser) parseHeader(line string) error {
	rest, ok := matchKeyword(line, "flowchart")
	if !ok {
		rest, ok = matchKeyword(line, "graph")
	}
	if !ok {
		return fmt.Errorf("expected 'graph' or 'flowchart', got %q", line)
	}

	dir, err := parseDirection(rest)
	if err != nil {
		return err
	}
	p.diagram = &diagram.FlowchartDiagram{Direction: dir}
	return nil
}

// matchKeyword reports whether line starts with kw followed by either
// end-of-string or whitespace, and returns the trimmed remainder. This
// prevents matching `grapha` / `flowchartfoo` as the header keyword.
func matchKeyword(line, kw string) (rest string, ok bool) {
	if !strings.HasPrefix(line, kw) {
		return "", false
	}
	if len(line) == len(kw) {
		return "", true
	}
	c := line[len(kw)]
	if c != ' ' && c != '\t' {
		return "", false
	}
	return strings.TrimSpace(line[len(kw):]), true
}

// parseDirection converts a Mermaid direction keyword to a Direction.
// An empty string defaults to TB, matching Mermaid's default. Extra
// tokens after a valid direction (e.g. `LR foo`) are reported as a
// separate error from "unknown direction".
func parseDirection(s string) (diagram.Direction, error) {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return diagram.DirectionUnknown, fmt.Errorf("extra tokens after direction %q", s)
	}
	switch s {
	case "", "TB", "TD":
		return diagram.DirectionTB, nil
	case "BT":
		return diagram.DirectionBT, nil
	case "LR":
		return diagram.DirectionLR, nil
	case "RL":
		return diagram.DirectionRL, nil
	default:
		return diagram.DirectionUnknown, fmt.Errorf("unknown direction %q", s)
	}
}

// parseLine dispatches a non-header, non-comment, non-empty line. It
// walks through chained edges left-to-right: `A --> B --> C` produces
// A→B and B→C edges in order.
func (p *parser) parseLine(line string) error {
	for {
		arrow := findArrow(line)
		if arrow == nil {
			// No arrow: final segment is a standalone node (or nothing).
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				return nil
			}
			id, shape, label, err := parseNodeDef(trimmed)
			if err != nil {
				if detErr := diagnoseMalformedArrow(trimmed); detErr != nil {
					return detErr
				}
				return err
			}
			p.upsertNode(id, shape, label)
			return nil
		}
		if arrow.labelUnclosed {
			return fmt.Errorf("unclosed edge label: missing %q", "|")
		}
		leftText := strings.TrimSpace(line[:arrow.start])

		// The right side extends to the next arrow (chained edges) or EOL.
		rightStart := arrow.end
		nextArrow := findArrow(line[rightStart:])
		rightEnd := len(line)
		if nextArrow != nil {
			rightEnd = rightStart + nextArrow.start
		}
		rightText := strings.TrimSpace(line[rightStart:rightEnd])

		leftID, leftShape, leftLabel, err := parseNodeDef(leftText)
		if err != nil {
			return fmt.Errorf("left side: %w", err)
		}
		rightID, rightShape, rightLabel, err := parseNodeDef(rightText)
		if err != nil {
			return fmt.Errorf("right side: %w", err)
		}

		p.upsertNode(leftID, leftShape, leftLabel)
		p.upsertNode(rightID, rightShape, rightLabel)
		p.diagram.Edges = append(p.diagram.Edges, diagram.Edge{
			From:      leftID,
			To:        rightID,
			Label:     arrow.label,
			LineStyle: arrow.lineStyle,
			ArrowHead: arrow.arrowHead,
		})

		if nextArrow == nil {
			return nil
		}
		// Advance: the right node becomes the next left. The remainder
		// of the line starts at nextArrow, so rebuild with rightText as
		// the new left-hand segment.
		line = rightText + " " + line[rightEnd:]
	}
}

// diagnoseMalformedArrow returns a helpful error for common malformed
// arrow forms that parseNodeDef's "unrecognized shape" wouldn't
// pinpoint. The main case is an inline edge label without a closing
// terminator: `A -- text` (missing `-->` / `---`).
func diagnoseMalformedArrow(segment string) error {
	if strings.Contains(segment, " -- ") || strings.Contains(segment, " == ") {
		return fmt.Errorf("unterminated inline edge label: expected `-->` / `---` / `==>` / `===` terminator")
	}
	return nil
}

// upsertNode adds a node or merges shape/label info into an existing
// entry. A bare reference (shape=Unknown, label="") never overwrites a
// previously defined shape/label.
func (p *parser) upsertNode(id string, shape diagram.NodeShape, label string) {
	if id == "" {
		return
	}
	if idx, ok := p.nodeIndex[id]; ok {
		// Merge: fill in any fields the existing entry is missing.
		existing := &p.diagram.Nodes[idx]
		if existing.Shape == diagram.NodeShapeUnknown && shape != diagram.NodeShapeUnknown {
			existing.Shape = shape
		}
		if existing.Label == "" && label != "" {
			existing.Label = label
		}
		return
	}
	p.nodeIndex[id] = len(p.diagram.Nodes)
	p.diagram.Nodes = append(p.diagram.Nodes, diagram.Node{
		ID:    id,
		Label: label,
		Shape: shape,
	})
}

// shapePattern is a bracketed node-shape definition — an opening
// delimiter, a closing delimiter, and the corresponding NodeShape.
type shapePattern struct {
	open, close string
	shape       diagram.NodeShape
}

// shapePatterns lists all supported node shapes in length-descending
// order so that more specific patterns (e.g., `[[`) are tried before
// less specific ones (e.g., `[`). Order is load-bearing: rearranging
// breaks disambiguation.
var shapePatterns = []shapePattern{
	// 3-char openings first.
	{"(((", ")))", diagram.NodeShapeDoubleCircle},
	// 2-char openings.
	{"((", "))", diagram.NodeShapeCircle},
	{"([", "])", diagram.NodeShapeStadium},
	{"[[", "]]", diagram.NodeShapeSubroutine},
	{"[(", ")]", diagram.NodeShapeCylinder},
	{"{{", "}}", diagram.NodeShapeHexagon},
	{"[/", "/]", diagram.NodeShapeParallelogram},
	{`[\`, `\]`, diagram.NodeShapeParallelogramAlt},
	{"[/", `\]`, diagram.NodeShapeTrapezoid},
	{`[\`, "/]", diagram.NodeShapeTrapezoidAlt},
	// 1-char openings last.
	{">", "]", diagram.NodeShapeAsymmetric},
	{"(", ")", diagram.NodeShapeRoundedRectangle},
	{"[", "]", diagram.NodeShapeRectangle},
	{"{", "}", diagram.NodeShapeDiamond},
}

// parseNodeDef reads a token like `A[Label]`, `B((Circle))`, or a bare
// `C` and returns the node ID, shape, and label. A bare reference has
// shape NodeShapeUnknown and empty label.
//
// Returns an error for malformed input (e.g., empty ID, unclosed
// bracket).
func parseNodeDef(s string) (id string, shape diagram.NodeShape, label string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", diagram.NodeShapeUnknown, "", fmt.Errorf("empty node definition")
	}

	// Read the ID: a run of word characters (letters, digits, underscore).
	i := 0
	for i < len(s) && isIDChar(s[i]) {
		i++
	}
	if i == 0 {
		if s[0] >= 0x80 {
			return "", diagram.NodeShapeUnknown, "", fmt.Errorf("non-ASCII node IDs are not yet supported (got %q)", s)
		}
		return "", diagram.NodeShapeUnknown, "", fmt.Errorf("invalid node ID in %q", s)
	}
	id = s[:i]
	rest := s[i:]

	if rest == "" {
		// Bare reference.
		return id, diagram.NodeShapeUnknown, "", nil
	}

	// Try each shape pattern in order. Both the open and close tokens
	// must match exactly. Track whether any pattern matched the opening
	// delimiter so we can distinguish "unclosed bracket" from an
	// entirely unrecognized shape.
	openMatched := ""
	for _, sp := range shapePatterns {
		if !strings.HasPrefix(rest, sp.open) {
			continue
		}
		if openMatched == "" {
			openMatched = sp.open
		}
		if !strings.HasSuffix(rest, sp.close) {
			continue
		}
		inner := rest[len(sp.open) : len(rest)-len(sp.close)]
		return id, sp.shape, inner, nil
	}

	if openMatched != "" {
		return "", diagram.NodeShapeUnknown, "", fmt.Errorf("unclosed %q in %q", openMatched, s)
	}
	return "", diagram.NodeShapeUnknown, "", fmt.Errorf("unrecognized shape in %q", s)
}

// isIDChar reports whether c is a valid character in a Mermaid node ID.
// ASCII-only: node IDs are restricted to [A-Za-z0-9_]. Unicode IDs are
// not yet supported.
func isIDChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

// arrowMatch describes one arrow operator found in a line. If the
// arrow carries an inline label (`A -- text --> B`) the label is
// captured during scanning and label/start/end span the whole
// opener+text+terminator. Pipe-form labels (`A -->|text| B`) are
// attached after scanning by attachPipeLabel.
type arrowMatch struct {
	start, end    int               // byte offsets of the arrow span within the line
	lineStyle     diagram.LineStyle // solid/dotted/thick
	arrowHead     diagram.ArrowHead // arrow/none
	label         string            // inline or pipe-form label, or ""
	labelUnclosed bool              // saw opening `|` with no closing `|`
}

// findArrow returns the leftmost arrow in line, or nil. Arrow detection
// skips content inside bracketed regions (`[...]`, `(...)`, `{...}`)
// and double-quoted strings so labels like `A[--> not an arrow] --> B`
// work correctly. On success, a pipe-form `|label|` immediately after
// the arrow is attached to the match (unless an inline label was
// already captured inside the arrow span).
func findArrow(line string) *arrowMatch {
	depth := 0
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if inQuote {
			if c == '"' {
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
			continue
		case '[', '(', '{':
			depth++
			continue
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth > 0 {
			continue
		}
		if m, ok := matchArrowAt(line, i); ok {
			if m.label == "" {
				attachPipeLabel(&m, line)
			}
			return &m
		}
	}
	return nil
}

// matchArrowAt tries to recognize an arrow token starting at line[i].
// Dispatches on the first byte to the appropriate family.
func matchArrowAt(line string, i int) (arrowMatch, bool) {
	switch line[i] {
	case '-':
		if i+1 < len(line) && line[i+1] == '.' {
			return matchDottedAt(line, i)
		}
		return matchDashAt(line, i, '-', diagram.LineStyleSolid)
	case '=':
		return matchDashAt(line, i, '=', diagram.LineStyleThick)
	}
	return arrowMatch{}, false
}

// matchDashAt handles the solid (`-`) and thick (`=`) arrow families,
// including long-dash variants (`--->`, `---->`, `====>`, `-----`, ...)
// and the `--`/`==` inline-label opener. dash is the repeated char.
//
// Grammar:
//
//	arrow-with-head := dash{2,} ">"
//	arrow-no-head   := dash{3,}
//	inline-label    := dash dash whitespace text whitespace terminator
//
// Two dashes/equals with no `>` is reserved for the inline-label opener
// (otherwise `A -- text --> B` would scan as an unterminated `--`).
func matchDashAt(line string, i int, dash byte, style diagram.LineStyle) (arrowMatch, bool) {
	j := i + 1
	for j < len(line) && line[j] == dash {
		j++
	}
	count := j - i
	if count < 2 {
		return arrowMatch{}, false
	}
	if j < len(line) && line[j] == '>' {
		return arrowMatch{
			start:     i,
			end:       j + 1,
			lineStyle: style,
			arrowHead: diagram.ArrowHeadArrow,
		}, true
	}
	if count >= 3 {
		return arrowMatch{
			start:     i,
			end:       j,
			lineStyle: style,
			arrowHead: diagram.ArrowHeadNone,
		}, true
	}
	return matchInlineLabelAt(line, i, j, dash, style)
}

// matchDottedAt handles `-.->` / `-..->` / ... (with-head) and
// `-.-` / `-..-` / ... (no-head). Entry precondition: line[i]='-' and
// line[i+1]='.'. Extra dots extend the arrow's visual length.
func matchDottedAt(line string, i int) (arrowMatch, bool) {
	j := i + 1 // pointing at the first '.'
	for j < len(line) && line[j] == '.' {
		j++
	}
	if j >= len(line) || line[j] != '-' {
		return arrowMatch{}, false
	}
	j++
	if j < len(line) && line[j] == '>' {
		return arrowMatch{
			start:     i,
			end:       j + 1,
			lineStyle: diagram.LineStyleDotted,
			arrowHead: diagram.ArrowHeadArrow,
		}, true
	}
	return arrowMatch{
		start:     i,
		end:       j,
		lineStyle: diagram.LineStyleDotted,
		arrowHead: diagram.ArrowHeadNone,
	}, true
}

// matchInlineLabelAt handles `A -- text -->B` and the thick analog
// `A == text ==>B`. openerEnd points at the byte after the opening
// `--`/`==`. The terminator is the next run of the same dash char
// followed by optional `>`; the label is the whitespace-trimmed text
// between. Mermaid requires whitespace between the opener and label.
//
// Labels containing the dash char itself (e.g. `-- a--b --`) are not
// supported — the first subsequent run wins.
func matchInlineLabelAt(line string, openerStart, openerEnd int, dash byte, style diagram.LineStyle) (arrowMatch, bool) {
	if openerEnd >= len(line) || (line[openerEnd] != ' ' && line[openerEnd] != '\t') {
		return arrowMatch{}, false
	}
	k := openerEnd + 1
	for k < len(line) {
		if line[k] != dash {
			k++
			continue
		}
		m := k + 1
		for m < len(line) && line[m] == dash {
			m++
		}
		count := m - k
		if count < 2 {
			k = m
			continue
		}
		label := strings.TrimSpace(line[openerEnd:k])
		if label == "" {
			return arrowMatch{}, false
		}
		if m < len(line) && line[m] == '>' {
			return arrowMatch{
				start:     openerStart,
				end:       m + 1,
				lineStyle: style,
				arrowHead: diagram.ArrowHeadArrow,
				label:     label,
			}, true
		}
		if count >= 3 {
			return arrowMatch{
				start:     openerStart,
				end:       m,
				lineStyle: style,
				arrowHead: diagram.ArrowHeadNone,
				label:     label,
			}, true
		}
		k = m
	}
	return arrowMatch{}, false
}

// attachPipeLabel captures a pipe-form edge label `|text|` that
// immediately follows the arrow (after optional whitespace), advancing
// m.end past the closing pipe. If the opening pipe has no closing pipe
// on the line, labelUnclosed is set so parseLine can raise a clear
// error. No-op when m.label was already captured inline.
func attachPipeLabel(m *arrowMatch, line string) {
	trailing := strings.TrimLeft(line[m.end:], " \t")
	if !strings.HasPrefix(trailing, "|") {
		return
	}
	consumed := len(line[m.end:]) - len(trailing)
	rest := trailing[1:]
	closeIdx := strings.Index(rest, "|")
	if closeIdx < 0 {
		m.labelUnclosed = true
		return
	}
	m.label = rest[:closeIdx]
	m.end += consumed + 1 + closeIdx + 1
}
