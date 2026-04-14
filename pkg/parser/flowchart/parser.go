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
//   - solid:  -->, ---
//   - dotted: -.->, -.-
//   - thick:  ==>, ===
//
// Edge labels use the pipe form: A -->|label| B
//
// TODO(features): subgraphs (nested `subgraph` ... `end` blocks),
// style/classDef/class directives, init directives (%%{init: ...}%%),
// additional arrow variants (-x, -o, longer forms), and inline edge
// labels (-- label -->). These are planned for a follow-up PR once
// the renderer lands and we can eyeball parser output end-to-end.
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
			if err := detectInlineEdgeLabel(trimmed); err != nil {
				return err
			}
			id, shape, label, err := parseNodeDef(trimmed)
			if err != nil {
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
			if detErr := detectInlineEdgeLabel(leftText); detErr != nil {
				return fmt.Errorf("left side: %w", detErr)
			}
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

// detectInlineEdgeLabel returns a helpful error when the user writes an
// inline-label edge (`A -- text --> B`), which is a common Mermaid form
// that the MVP parser does not yet support. Without this, the failure
// surfaces as a cryptic "unrecognized shape" or "invalid node ID".
// TODO(features): support inline edge labels; until then, keep this
// detection in sync with the tokens that would signal the intent.
func detectInlineEdgeLabel(segment string) error {
	if strings.Contains(segment, " -- ") || strings.Contains(segment, " == ") {
		return fmt.Errorf("inline edge labels (`A -- text --> B`) are not yet supported; use the pipe form `A -->|text| B`")
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

// arrowMatch describes one arrow operator found in a line.
type arrowMatch struct {
	start, end    int               // byte offsets of the arrow token within the line
	lineStyle     diagram.LineStyle // solid/dotted/thick
	arrowHead     diagram.ArrowHead // arrow/none
	label         string            // label between pipes, or ""
	labelUnclosed bool              // saw opening `|` with no closing `|`
}

// arrowToken is a static definition of an arrow operator we recognize.
type arrowToken struct {
	token     string
	lineStyle diagram.LineStyle
	arrowHead diagram.ArrowHead
}

// arrowTokens lists the recognized arrow operators. Order is
// load-bearing for two reasons:
//  1. Prefix-conflict avoidance: `-.->` must appear before plain-solid
//     arrows so that a dotted-arrow line isn't scanned as `-->` at a
//     later index.
//  2. Tie-breaking: findArrow uses strict `<` when updating `best`, so
//     the FIRST entry wins if two tokens match at the same index. In
//     practice this matters for dotted-vs-solid prefixes like `-.-`.
var arrowTokens = []arrowToken{
	// Dotted variants (must come before plain solid so `-.->` isn't
	// mistaken for `-->`).
	{"-.->", diagram.LineStyleDotted, diagram.ArrowHeadArrow},
	{"-.-", diagram.LineStyleDotted, diagram.ArrowHeadNone},
	// Thick variants.
	{"==>", diagram.LineStyleThick, diagram.ArrowHeadArrow},
	{"===", diagram.LineStyleThick, diagram.ArrowHeadNone},
	// Solid variants.
	{"-->", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
	{"---", diagram.LineStyleSolid, diagram.ArrowHeadNone},
}

// findArrow scans line for the leftmost arrow operator. Returns nil
// if none is found. If the arrow is followed by `|label|`, the label
// is captured and the match's end index points past the closing pipe.
func findArrow(line string) *arrowMatch {
	best := -1
	var bestTok arrowToken
	for _, at := range arrowTokens {
		if i := strings.Index(line, at.token); i >= 0 {
			if best < 0 || i < best {
				best = i
				bestTok = at
			}
		}
	}
	if best < 0 {
		return nil
	}

	start := best
	end := best + len(bestTok.token)

	// Check for an edge label: `|text|` immediately (optional space)
	// after the arrow.
	label := ""
	unclosed := false
	trailing := strings.TrimLeft(line[end:], " \t")
	if strings.HasPrefix(trailing, "|") {
		consumed := len(line[end:]) - len(trailing) // spaces skipped
		rest := trailing[1:]                        // drop opening pipe
		if closeIdx := strings.Index(rest, "|"); closeIdx >= 0 {
			label = rest[:closeIdx]
			end += consumed + 1 + closeIdx + 1 // spaces + "|" + inner + "|"
		} else {
			unclosed = true
		}
	}

	return &arrowMatch{
		start:         start,
		end:           end,
		lineStyle:     bestTok.lineStyle,
		arrowHead:     bestTok.arrowHead,
		label:         label,
		labelUnclosed: unclosed,
	}
}
