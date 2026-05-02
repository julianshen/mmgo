// Package flowchart parses Mermaid flowchart/graph syntax into a
// FlowchartDiagram AST.
package flowchart

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

func Parse(r io.Reader) (*diagram.FlowchartDiagram, error) {
	p := &parser{
		nodeIndex: make(map[string]int),
	}
	scanner := bufio.NewScanner(r)
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
	for _, className := range p.pendingApplyAll {
		p.applyClassToAll(className)
	}
	return p.diagram, nil
}

type parser struct {
	diagram         *diagram.FlowchartDiagram
	nodeIndex       map[string]int
	subgraphStack   []*diagram.Subgraph
	pendingApplyAll []string
}

func (p *parser) currentSubgraph() *diagram.Subgraph {
	if len(p.subgraphStack) == 0 {
		return nil
	}
	return p.subgraphStack[len(p.subgraphStack)-1]
}

func (p *parser) findNode(id string) *diagram.Node {
	for si := len(p.subgraphStack) - 1; si >= 0; si-- {
		for i := range p.subgraphStack[si].Nodes {
			if p.subgraphStack[si].Nodes[i].ID == id {
				return &p.subgraphStack[si].Nodes[i]
			}
		}
	}
	for si := range p.diagram.Subgraphs {
		if n := findNodeInSubgraph(p.diagram.Subgraphs[si], id); n != nil {
			return n
		}
	}
	if idx, ok := p.nodeIndex[id]; ok {
		return &p.diagram.Nodes[idx]
	}
	return nil
}

func findNodeInSubgraph(sg *diagram.Subgraph, id string) *diagram.Node {
	for i := range sg.Nodes {
		if sg.Nodes[i].ID == id {
			return &sg.Nodes[i]
		}
	}
	for _, child := range sg.Children {
		if n := findNodeInSubgraph(child, id); n != nil {
			return n
		}
	}
	return nil
}

func (p *parser) addNode(id string, shape diagram.NodeShape, label string, classes []string) {
	if id == "" {
		return
	}
	if existing := p.findNode(id); existing != nil {
		if existing.Shape == diagram.NodeShapeUnknown && shape != diagram.NodeShapeUnknown {
			existing.Shape = shape
		}
		if existing.Label == "" && label != "" {
			existing.Label = label
		}
		if len(classes) > 0 {
			existing.Classes = append(existing.Classes, classes...)
		}
		// Mermaid reassigns a node's subgraph membership to the
		// innermost subgraph that references it: when `B --> C`
		// appears inside `subgraph two` and B was previously
		// declared in `subgraph one`, B moves into `two`. Without
		// this, nested-subgraph diagrams render the parent's title
		// over the misplaced child node.
		cur := p.currentSubgraph()
		if cur != nil && !subgraphDirectlyOwns(cur, id) {
			node := *existing
			p.detachNode(id)
			cur.Nodes = append(cur.Nodes, node)
		}
		return
	}
	sg := p.currentSubgraph()
	if sg != nil {
		sg.Nodes = append(sg.Nodes, diagram.Node{ID: id, Label: label, Shape: shape, Classes: classes})
		return
	}
	p.nodeIndex[id] = len(p.diagram.Nodes)
	p.diagram.Nodes = append(p.diagram.Nodes, diagram.Node{ID: id, Label: label, Shape: shape, Classes: classes})
}

// subgraphDirectlyOwns reports whether sg has node id in its own Nodes
// slice (not including descendants).
func subgraphDirectlyOwns(sg *diagram.Subgraph, id string) bool {
	for _, n := range sg.Nodes {
		if n.ID == id {
			return true
		}
	}
	return false
}

// detachNode removes node id from wherever it currently lives —
// the root nodes slice, the parser's open subgraph stack, or any
// subgraph already attached to the diagram. nodeIndex is rebuilt so
// indices remain valid after removal from the root slice.
func (p *parser) detachNode(id string) {
	for si := range p.subgraphStack {
		if removeNodeFromSubgraph(p.subgraphStack[si], id) {
			return
		}
	}
	for si := range p.diagram.Subgraphs {
		if removeNodeFromSubgraph(p.diagram.Subgraphs[si], id) {
			return
		}
	}
	if idx, ok := p.nodeIndex[id]; ok {
		last := len(p.diagram.Nodes) - 1
		if idx != last {
			moved := p.diagram.Nodes[last]
			p.diagram.Nodes[idx] = moved
			p.nodeIndex[moved.ID] = idx
		}
		p.diagram.Nodes = p.diagram.Nodes[:last]
		delete(p.nodeIndex, id)
	}
}

func removeNodeFromSubgraph(sg *diagram.Subgraph, id string) bool {
	for i, n := range sg.Nodes {
		if n.ID == id {
			sg.Nodes = append(sg.Nodes[:i], sg.Nodes[i+1:]...)
			return true
		}
	}
	for _, child := range sg.Children {
		if removeNodeFromSubgraph(child, id) {
			return true
		}
	}
	return false
}

func (p *parser) addEdge(e diagram.Edge) {
	sg := p.currentSubgraph()
	if sg != nil {
		sg.Edges = append(sg.Edges, e)
	} else {
		p.diagram.Edges = append(p.diagram.Edges, e)
	}
}

func stripComment(line string) string {
	depth := 0
	inQuote := false
	inPipe := false
	for i := 0; i+1 < len(line); i++ {
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
		case '|':
			inPipe = !inPipe
			continue
		}
		if depth > 0 || inPipe {
			continue
		}
		if c != '%' || line[i+1] != '%' {
			continue
		}
		if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
			return line[:i]
		}
	}
	return line
}

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
	p.diagram = &diagram.FlowchartDiagram{
		Direction:  dir,
		Classes:    make(map[string]string),
		LinkStyles: make(map[int]string),
	}
	return nil
}

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

func parseDirection(s string) (diagram.Direction, error) {
	return parserutil.ParseDirection(s)
}

func (p *parser) parseLine(line string) error {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "subgraph") {
		return p.parseSubgraph(trimmed)
	}
	if trimmed == "end" {
		return p.parseEnd()
	}
	if strings.HasPrefix(trimmed, "style ") {
		return p.parseStyle(trimmed)
	}
	if strings.HasPrefix(trimmed, "classDef ") {
		return p.parseClassDef(trimmed)
	}
	if strings.HasPrefix(trimmed, "class ") {
		return p.parseClass(trimmed)
	}
	if strings.HasPrefix(trimmed, "linkStyle ") {
		return p.parseLinkStyle(trimmed)
	}
	if strings.HasPrefix(trimmed, "direction ") {
		return p.parseDirectionInSubgraph(trimmed)
	}
	if strings.HasPrefix(trimmed, "click ") {
		return p.parseClick(trimmed)
	}
	if strings.HasPrefix(trimmed, "title ") || strings.HasPrefix(trimmed, "title:") {
		p.diagram.Title = parserutil.TrimKeyword(trimmed, "title")
		return nil
	}
	if strings.HasPrefix(trimmed, "accTitle") {
		rest := trimmed[len("accTitle"):]
		if rest == "" || rest[0] == ':' || rest[0] == ' ' {
			p.diagram.AccTitle = parserutil.TrimKeyword(trimmed, "accTitle")
			return nil
		}
	}
	if strings.HasPrefix(trimmed, "accDescr") {
		rest := trimmed[len("accDescr"):]
		if rest == "" || rest[0] == ':' || rest[0] == ' ' {
			p.diagram.AccDescr = parserutil.TrimKeyword(trimmed, "accDescr")
			return nil
		}
	}

	return p.parseEdgeLine(line)
}

func (p *parser) parseSubgraph(line string) error {
	rest := strings.TrimSpace(line[len("subgraph"):])
	if rest == "" {
		return fmt.Errorf("subgraph requires an ID")
	}

	id, label := rest, rest
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) == 2 {
		id = parts[0]
		label = parts[1]
		label = strings.TrimSpace(label)
	}

	label = processLabel(label)
	if len(label) >= 2 && label[0] == '[' && label[len(label)-1] == ']' {
		label = label[1 : len(label)-1]
	}

	sg := &diagram.Subgraph{
		ID:    id,
		Label: label,
	}

	if p.currentSubgraph() != nil {
		p.currentSubgraph().Children = append(p.currentSubgraph().Children, sg)
	} else {
		p.diagram.Subgraphs = append(p.diagram.Subgraphs, sg)
	}
	p.subgraphStack = append(p.subgraphStack, sg)
	return nil
}

func (p *parser) parseEnd() error {
	if len(p.subgraphStack) == 0 {
		return fmt.Errorf("unexpected 'end' without subgraph")
	}
	p.subgraphStack = p.subgraphStack[:len(p.subgraphStack)-1]
	return nil
}

func (p *parser) parseStyle(line string) error {
	rest := line[len("style "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("style requires node ID and CSS")
	}
	p.diagram.Styles = append(p.diagram.Styles, diagram.StyleDef{
		NodeID: parts[0],
		CSS:    normalizeCSS(parts[1]),
	})
	return nil
}

// normalizeCSS converts Mermaid's comma-separated declaration syntax
// (`fill:#fff,stroke:#000`) into the semicolon-separated form CSS
// actually accepts. Without this, browsers/canvas parsers see a
// malformed value at the first comma and silently fall back to the
// default fill (typically black), producing the "all nodes black"
// regression on `style` and `classDef` rules.
func normalizeCSS(css string) string {
	if strings.IndexByte(css, ',') < 0 {
		return css
	}
	var sb strings.Builder
	sb.Grow(len(css))
	depth := 0
	inQuote := false
	for i := 0; i < len(css); i++ {
		c := css[i]
		switch {
		case c == '"' || c == '\'':
			inQuote = !inQuote
		case c == '(' && !inQuote:
			depth++
		case c == ')' && !inQuote:
			if depth > 0 {
				depth--
			}
		}
		if c == ',' && depth == 0 && !inQuote {
			sb.WriteByte(';')
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}

func (p *parser) parseClassDef(line string) error {
	rest := line[len("classDef "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("classDef requires name and CSS")
	}
	name := parts[0]
	css := parts[1]
	p.diagram.Classes[name] = normalizeCSS(css)
	cssFields := strings.Fields(css)
	if len(cssFields) >= 2 && cssFields[len(cssFields)-1] == "@@" {
		p.pendingApplyAll = append(p.pendingApplyAll, name)
	}
	return nil
}

func (p *parser) applyClassToAll(className string) {
	for i := range p.diagram.Nodes {
		p.diagram.Nodes[i].Classes = append(p.diagram.Nodes[i].Classes, className)
	}
	for _, sg := range p.diagram.Subgraphs {
		p.applyClassToAllInSubgraph(sg, className)
	}
}

func (p *parser) applyClassToAllInSubgraph(sg *diagram.Subgraph, className string) {
	for i := range sg.Nodes {
		sg.Nodes[i].Classes = append(sg.Nodes[i].Classes, className)
	}
	for _, child := range sg.Children {
		p.applyClassToAllInSubgraph(child, className)
	}
}

func (p *parser) parseClass(line string) error {
	rest := line[len("class "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("class requires node IDs and class name")
	}
	nodeIDs := strings.Split(parts[0], ",")
	className := parts[1]
	for _, nid := range nodeIDs {
		nid = strings.TrimSpace(nid)
		if nid == "" {
			continue
		}
		p.addNodeClass(nid, className)
	}
	return nil
}

func (p *parser) addNodeClass(nodeID, className string) {
	if n := p.findNode(nodeID); n != nil {
		n.Classes = append(n.Classes, className)
	}
}

func (p *parser) parseClick(line string) error {
	rest := line[len("click "):]
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return fmt.Errorf("click requires node ID and URL or callback")
	}
	nodeID := fields[0]
	cd := diagram.ClickDef{NodeID: nodeID}
	afterNode := strings.TrimSpace(rest[len(nodeID):])
	argSrc := afterNode
	switch {
	case strings.HasPrefix(afterNode, "call "):
		cd.Callback = strings.TrimSpace(afterNode[5:])
	case afterNode == "href" || strings.HasPrefix(afterNode, "href "):
		argSrc = strings.TrimSpace(afterNode[len("href"):])
		fallthrough
	default:
		parts := parseClickArgs(argSrc)
		if len(parts) == 0 {
			return fmt.Errorf("click requires a URL for node %q", nodeID)
		}
		cd.URL = parts[0]
		if len(parts) >= 2 {
			cd.Tooltip = parts[1]
		}
		if len(parts) >= 3 {
			cd.Target = parts[2]
		}
	}
	p.diagram.Clicks = append(p.diagram.Clicks, cd)
	return nil
}

func parseClickArgs(s string) []string {
	var parts []string
	i := 0
	for i < len(s) && len(parts) < 3 {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		if s[i] == '"' {
			i++
			end := strings.IndexByte(s[i:], '"')
			if end < 0 {
				parts = append(parts, s[i:])
				break
			}
			parts = append(parts, s[i:i+end])
			i = i + end + 1
		} else {
			end := strings.IndexAny(s[i:], " \t")
			if end < 0 {
				parts = append(parts, s[i:])
				break
			}
			parts = append(parts, s[i:i+end])
			i = i + end
		}
	}
	return parts
}

func (p *parser) parseLinkStyle(line string) error {
	rest := line[len("linkStyle "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("linkStyle requires index and CSS")
	}
	indices := strings.Split(parts[0], ",")
	css := parts[1]
	for _, idxStr := range indices {
		idxStr = strings.TrimSpace(idxStr)
		n, err := strconv.Atoi(idxStr)
		if err != nil {
			return fmt.Errorf("invalid linkStyle index %q", idxStr)
		}
		p.diagram.LinkStyles[n] = normalizeCSS(css)
	}
	return nil
}

func (p *parser) parseDirectionInSubgraph(line string) error {
	rest := strings.TrimSpace(line[len("direction "):])
	dir, err := parseDirection(rest)
	if err != nil {
		return err
	}
	sg := p.currentSubgraph()
	if sg != nil {
		sg.Direction = dir
	}
	return nil
}

func (p *parser) parseEdgeLine(line string) error {
	for {
		arrow := findArrow(line)
		if arrow == nil {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				return nil
			}
			id, shape, label, classes, err := parseNodeDef(trimmed)
			if err != nil {
				if detErr := diagnoseMalformedArrow(trimmed); detErr != nil {
					return detErr
				}
				return err
			}
			p.addNode(id, shape, label, classes)
			return nil
		}
		if arrow.labelUnclosed {
			return fmt.Errorf("unclosed edge label: missing %q", "|")
		}
		leftText := strings.TrimSpace(line[:arrow.start])
		leftText, arrow.edgeID = extractEdgeID(leftText)

		rightStart := arrow.end
		nextArrow := findArrow(line[rightStart:])
		rightEnd := len(line)
		if nextArrow != nil {
			rightEnd = rightStart + nextArrow.start
		}
		rightSegment := strings.TrimSpace(line[rightStart:rightEnd])

		leftID, leftShape, leftLabel, leftClasses, err := parseNodeDef(leftText)
		if err != nil {
			return fmt.Errorf("left side: %w", err)
		}
		p.addNode(leftID, leftShape, leftLabel, leftClasses)

		rightNodes := parseAmpersandNodes(rightSegment)
		for _, rn := range rightNodes {
			p.addNode(rn.id, rn.shape, rn.label, rn.classes)
			p.addEdge(diagram.Edge{
				From:      leftID,
				To:        rn.id,
				Label:     arrow.label,
				ID:        arrow.edgeID,
				LineStyle: arrow.lineStyle,
				ArrowHead: arrow.arrowHead,
				ArrowTail: arrow.arrowTail,
			})
		}

		if nextArrow == nil {
			return nil
		}
		lastRight := rightNodes[len(rightNodes)-1]
		line = lastRight.raw + " " + line[rightEnd:]
	}
}

type ampersandNode struct {
	id      string
	shape   diagram.NodeShape
	label   string
	classes []string
	raw     string
}

func parseAmpersandNodes(segment string) []ampersandNode {
	parts := splitOnAmpersand(segment)
	var nodes []ampersandNode
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, shape, label, classes, err := parseNodeDef(part)
		if err != nil {
			id = part
			shape = diagram.NodeShapeUnknown
		}
		nodes = append(nodes, ampersandNode{id: id, shape: shape, label: label, classes: classes, raw: part})
	}
	if len(nodes) == 0 {
		nodes = append(nodes, ampersandNode{})
	}
	return nodes
}

func splitOnAmpersand(s string) []string {
	depth := 0
	inQuote := false
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == '"' {
				inQuote = false
			}
			continue
		}
		switch c {
		case '"':
			inQuote = true
		case '[', '(', '{':
			depth++
		case ']', ')', '}':
			if depth > 0 {
				depth--
			}
		case '&':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func diagnoseMalformedArrow(segment string) error {
	if strings.Contains(segment, " -- ") || strings.Contains(segment, " == ") {
		return fmt.Errorf("unterminated inline edge label: expected `-->` / `---` / `==>` / `===` terminator")
	}
	return nil
}

type shapePattern struct {
	open, close string
	shape       diagram.NodeShape
}

var shapePatterns = []shapePattern{
	{"(((", ")))", diagram.NodeShapeDoubleCircle},
	{"((", "))", diagram.NodeShapeCircle},
	{"([", "])", diagram.NodeShapeStadium},
	{"[[", "]]", diagram.NodeShapeSubroutine},
	{"[(", ")]", diagram.NodeShapeCylinder},
	{"{{", "}}", diagram.NodeShapeHexagon},
	{"[/", "/]", diagram.NodeShapeParallelogram},
	{`[\`, `\]`, diagram.NodeShapeParallelogramAlt},
	{"[/", `\]`, diagram.NodeShapeTrapezoid},
	{`[\`, "/]", diagram.NodeShapeTrapezoidAlt},
	{">", "]", diagram.NodeShapeAsymmetric},
	{"(", ")", diagram.NodeShapeRoundedRectangle},
	{"[", "]", diagram.NodeShapeRectangle},
	{"{", "}", diagram.NodeShapeDiamond},
}

func extractEdgeID(s string) (rest, edgeID string) {
	atIdx := strings.LastIndex(s, "@")
	if atIdx < 0 {
		return s, ""
	}
	candidate := s[atIdx+1:]
	if candidate != "" {
		return s, ""
	}
	before := s[:atIdx]
	start := len(before)
	for start > 0 && isIDChar(before[start-1]) {
		start--
	}
	if start == len(before) {
		return s, ""
	}
	edgeID = before[start:]
	rest = strings.TrimRight(before[:start], " \t")
	return rest, edgeID
}

func parseNodeDef(s string) (id string, shape diagram.NodeShape, label string, classes []string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("empty node definition")
	}

	i := 0
	for i < len(s) {
		c := s[i]
		if isIDChar(c) {
			i++
			continue
		}
		if c == '-' && i+1 < len(s) && s[i+1] != '-' && s[i+1] != '>' && i > 0 {
			i++
			continue
		}
		break
	}
	if i == 0 {
		if s[0] >= 0x80 {
			return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("non-ASCII node IDs are not yet supported (got %q)", s)
		}
		return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("invalid node ID in %q", s)
	}
	id = s[:i]
	if i < len(s) && s[i] >= 0x80 {
		return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("non-ASCII node IDs are not yet supported (got %q)", s)
	}
	rest := strings.TrimLeft(s[i:], " \t")

	if rest == "" {
		return id, diagram.NodeShapeUnknown, "", nil, nil
	}

	rest, cls := stripInlineClass(rest)

	if rest == "" {
		return id, diagram.NodeShapeUnknown, "", cls, nil
	}

	// Extended `@{ shape: ..., label: "..." }` syntax may appear
	// either standalone (`A@{shape:diamond}`) or after a traditional
	// delimiter (`A["text"]@{shape:diamond}`). Detect a leading or
	// trailing `@{...}` block and let it take precedence over the
	// traditional shape; a label inside `@{}` overrides the
	// delimiter-supplied one.
	annoShape, annoLabel, annoHasLabel, annoRest, annoErr := stripShapeAnnotation(rest)
	if annoErr != nil {
		return "", diagram.NodeShapeUnknown, "", nil, annoErr
	}
	rest = annoRest

	if rest == "" {
		// `@{}` consumed the whole remainder — no traditional shape
		// to merge with. Require a shape value (a bare `@{}` with
		// neither `shape:` nor `label:` would yield NodeShapeUnknown,
		// which we promote to Rectangle as the canonical default).
		shape := annoShape
		if shape == diagram.NodeShapeUnknown {
			shape = diagram.NodeShapeRectangle
		}
		return id, shape, annoLabel, cls, nil
	}

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
		label = processLabel(inner)
		shape := sp.shape
		if annoShape != diagram.NodeShapeUnknown {
			shape = annoShape
		}
		if annoHasLabel {
			label = annoLabel
		}
		return id, shape, label, cls, nil
	}

	if openMatched != "" {
		return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("unclosed %q in %q", openMatched, s)
	}
	return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("unrecognized shape in %q", s)
}

// stripShapeAnnotation removes a single `@{...}` block from rest if
// present (anywhere — leading, middle, or trailing) and returns the
// resolved shape, the label override (if any), the rest with the block
// removed, and an error for malformed annotations. Returns
// NodeShapeUnknown / annoHasLabel=false when no `@{` is found.
//
// The search must skip `@{` occurrences inside quoted labels and
// inside traditional bracket pairs — `A["text @{ literal }"]` is
// valid Mermaid where the `@{` is literal label content, not an
// annotation. findShapeAnnotation walks the string tracking quote
// and bracket state and returns the first `@{` at depth 0 only.
func stripShapeAnnotation(rest string) (shape diagram.NodeShape, label string, hasLabel bool, remaining string, err error) {
	idx := findShapeAnnotation(rest)
	if idx < 0 {
		return diagram.NodeShapeUnknown, "", false, rest, nil
	}
	annoShape, annoLabel, labelSet, consumed, ok, err := parseShapeAnnotation(rest[idx:])
	if err != nil {
		return diagram.NodeShapeUnknown, "", false, rest, err
	}
	if !ok {
		return diagram.NodeShapeUnknown, "", false, rest, nil
	}
	remaining = strings.TrimSpace(rest[:idx] + rest[idx+consumed:])
	return annoShape, annoLabel, labelSet, remaining, nil
}

// findShapeAnnotation returns the index of the first `@{` that sits
// outside any quoted span or traditional bracket pair, or -1 if no
// such position exists. The opening delimiters tracked are `"`, `'`,
// `[`, `(` (matching shapePatterns); `{` itself is *not* tracked
// because inside a single-character `{...}` shape the `@{` would
// also be literal — but the design says `@{}` is always at the node
// suffix, never inside a `{...}` shape.
func findShapeAnnotation(s string) int {
	depth := 0
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '[', '(':
			depth++
		case ']', ')':
			if depth > 0 {
				depth--
			}
		case '@':
			if depth == 0 && i+1 < len(s) && s[i+1] == '{' {
				return i
			}
		}
	}
	return -1
}

// TODO(extended-shapes): this scan also doesn't respect quoted-label
// boundaries — `A["a:::b"]` mistakenly splits "b]" off as a class.
// Fixing it requires the same depth-aware walk that
// findShapeAnnotation does. Pre-existing hazard, not introduced by
// the @{} work; tracked separately.
func stripInlineClass(s string) (rest string, classes []string) {
	for {
		idx := strings.LastIndex(s, ":::")
		if idx < 0 {
			return s, classes
		}
		// Class names extend from `:::` up to the next `@{` (extended-
		// shape annotation) or end-of-string. Without the boundary,
		// `A:::cls@{shape:diamond}` would parse the entire suffix
		// `cls@{shape:diamond}` as a single class name.
		clsRaw := s[idx+3:]
		var trailing string
		if at := strings.Index(clsRaw, "@{"); at >= 0 {
			trailing = clsRaw[at:]
			clsRaw = clsRaw[:at]
		}
		cls := strings.TrimSpace(clsRaw)
		if cls != "" {
			classes = append([]string{cls}, classes...)
		}
		s = strings.TrimRight(s[:idx], " \t") + trailing
	}
}

var entityRe = regexp.MustCompile(`#([a-zA-Z]+|\d+);?`)

func decodeEntities(s string) string {
	if strings.IndexByte(s, '#') >= 0 {
		s = entityRe.ReplaceAllStringFunc(s, func(match string) string {
			name := strings.TrimSuffix(strings.TrimPrefix(match, "#"), ";")
			switch name {
			case "quot":
				return `"`
			case "amp":
				return "&"
			case "lt":
				return "<"
			case "gt":
				return ">"
			case "apos":
				return "'"
			case "nbsp":
				return "\u00a0"
			}
			if n, err := strconv.Atoi(name); err == nil {
				return string(rune(n))
			}
			return match
		})
	}
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br>", "\n")
	return s
}

func processLabel(s string) string {
	return decodeEntities(parserutil.Unquote(s))
}

func isIDChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '_'
}

type arrowMatch struct {
	start, end    int
	lineStyle     diagram.LineStyle
	arrowHead     diagram.ArrowHead
	arrowTail     diagram.ArrowHead
	label         string
	labelUnclosed bool
	edgeID        string
}

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

func matchArrowAt(line string, i int) (arrowMatch, bool) {
	if i >= len(line) {
		return arrowMatch{}, false
	}
	switch line[i] {
	case '-':
		if i+1 < len(line) && line[i+1] == '.' {
			return matchDottedAt(line, i)
		}
		m, ok := matchDashAt(line, i, '-', diagram.LineStyleSolid)
		if ok {
			if i >= 2 && (line[i-1] == 'o' || line[i-1] == 'x') && (line[i-2] == ' ' || line[i-2] == '\t') {
				if line[i-1] == 'o' {
					m.arrowTail = diagram.ArrowHeadCircle
				} else {
					m.arrowTail = diagram.ArrowHeadCross
				}
				m.start = i - 1
			}
			return m, true
		}
		return arrowMatch{}, false
	case '=':
		return matchDashAt(line, i, '=', diagram.LineStyleThick)
	case '~':
		return matchTildeAt(line, i)
	case '<':
		return matchBidirectionalAt(line, i)
	}
	return arrowMatch{}, false
}

func matchTildeAt(line string, i int) (arrowMatch, bool) {
	j := i
	for j < len(line) && line[j] == '~' {
		j++
	}
	if j-i < 3 {
		return arrowMatch{}, false
	}
	return arrowMatch{
		start:     i,
		end:       j,
		lineStyle: diagram.LineStyleInvisible,
		arrowHead: diagram.ArrowHeadNone,
	}, true
}

func circleOrCross(c byte) diagram.ArrowHead {
	if c == 'o' {
		return diagram.ArrowHeadCircle
	}
	return diagram.ArrowHeadCross
}

func matchBidirectionalAt(line string, i int) (arrowMatch, bool) {
	if i+1 >= len(line) {
		return arrowMatch{}, false
	}
	rest := line[i+1:]

	if len(rest) >= 3 && (rest[0] == 'o' || rest[0] == 'x') && rest[1] == '-' {
		tail := circleOrCross(rest[0])
		m, ok := matchDashAt(line, i+2, '-', diagram.LineStyleSolid)
		if ok {
			m.arrowTail = tail
			m.start = i
			if m.arrowHead == diagram.ArrowHeadNone {
				m.arrowHead = tail
			}
			return m, true
		}
	}

	if len(rest) >= 1 && (rest[0] == '-' || rest[0] == '=') {
		dash := rest[0]
		style := diagram.LineStyleSolid
		if dash == '=' {
			style = diagram.LineStyleThick
		}
		m, ok := matchDashAt(line, i+1, dash, style)
		if ok {
			m.arrowTail = diagram.ArrowHeadArrow
			m.start = i
			return m, true
		}
	}
	return arrowMatch{}, false
}

func resolveArrowHead(line string, j int, defaultHead diagram.ArrowHead) (diagram.ArrowHead, int) {
	if j < len(line) {
		switch line[j] {
		case 'o':
			return diagram.ArrowHeadCircle, j + 1
		case 'x':
			return diagram.ArrowHeadCross, j + 1
		}
	}
	return defaultHead, j
}

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
		head, end := resolveArrowHead(line, j+1, diagram.ArrowHeadArrow)
		return arrowMatch{
			start:     i,
			end:       end,
			lineStyle: style,
			arrowHead: head,
		}, true
	}
	if count >= 3 {
		head := diagram.ArrowHeadNone
		if j < len(line) && (line[j] == 'o' || line[j] == 'x') {
			head = circleOrCross(line[j])
			j++
		}
		return arrowMatch{
			start:     i,
			end:       j,
			lineStyle: style,
			arrowHead: head,
		}, true
	}
	if j < len(line) && (line[j] == 'o' || line[j] == 'x') {
		head := circleOrCross(line[j])
		return arrowMatch{
			start:     i,
			end:       j + 1,
			lineStyle: style,
			arrowHead: head,
		}, true
	}
	return matchInlineLabelAt(line, i, j, dash, style)
}

func matchDottedAt(line string, i int) (arrowMatch, bool) {
	j := i + 1
	for j < len(line) && line[j] == '.' {
		j++
	}
	if j < len(line) && line[j] == '-' {
		return matchDottedAfterDots(line, i, j)
	}
	if j < len(line) && (line[j] == ' ' || line[j] == '\t') {
		if m, ok := matchDottedInlineLabelOpen(line, i, j); ok {
			return m, true
		}
	}
	return arrowMatch{}, false
}

func matchDottedAfterDots(line string, i, j int) (arrowMatch, bool) {
	j++
	if j < len(line) && line[j] == '>' {
		head, end := resolveArrowHead(line, j+1, diagram.ArrowHeadArrow)
		return arrowMatch{
			start:     i,
			end:       end,
			lineStyle: diagram.LineStyleDotted,
			arrowHead: head,
		}, true
	}
	head := diagram.ArrowHeadNone
	end := j
	if j < len(line) && (line[j] == 'o' || line[j] == 'x') {
		head = circleOrCross(line[j])
		end = j + 1
	}
	if head == diagram.ArrowHeadNone {
		if m, ok := matchDottedInlineLabelAt(line, i, j); ok {
			return m, true
		}
	}
	return arrowMatch{
		start:     i,
		end:       end,
		lineStyle: diagram.LineStyleDotted,
		arrowHead: head,
	}, true
}

func matchDottedInlineLabelOpen(line string, openerStart, afterDot int) (arrowMatch, bool) {
	if afterDot >= len(line) || (line[afterDot] != ' ' && line[afterDot] != '\t') {
		return arrowMatch{}, false
	}
	labelStart := afterDot + 1
	k := labelStart
	for k < len(line) {
		if line[k] != '.' {
			k++
			continue
		}
		if k == 0 || line[k-1] != ' ' && line[k-1] != '\t' && line[k-1] != '-' {
			k++
			continue
		}
		closeStart := k
		dotCount := 0
		for closeStart < len(line) && line[closeStart] == '.' {
			dotCount++
			closeStart++
		}
		if closeStart >= len(line) || line[closeStart] != '-' {
			k = closeStart
			continue
		}
		closeStart++
		label := strings.TrimSpace(line[labelStart:k])
		if label == "" {
			k = closeStart
			continue
		}
		if closeStart < len(line) && line[closeStart] == '>' {
			head, end := resolveArrowHead(line, closeStart+1, diagram.ArrowHeadArrow)
			return arrowMatch{
				start:     openerStart,
				end:       end,
				lineStyle: diagram.LineStyleDotted,
				arrowHead: head,
				label:     processLabel(label),
			}, true
		}
		head := diagram.ArrowHeadNone
		end := closeStart
		if closeStart < len(line) && (line[closeStart] == 'o' || line[closeStart] == 'x') {
			head = circleOrCross(line[closeStart])
			end = closeStart + 1
		}
		return arrowMatch{
			start:     openerStart,
			end:       end,
			lineStyle: diagram.LineStyleDotted,
			arrowHead: head,
			label:     processLabel(label),
		}, true
	}
	return arrowMatch{}, false
}

func matchDottedInlineLabelAt(line string, openerStart, afterFirstDash int) (arrowMatch, bool) {
	j := afterFirstDash
	if j >= len(line) || (line[j] != ' ' && line[j] != '\t') {
		return arrowMatch{}, false
	}
	for j < len(line) {
		if line[j] != '.' {
			j++
			continue
		}
		if j+1 >= len(line) || line[j-1] != '-' {
			j++
			continue
		}
		closeEnd := j + 1
		if closeEnd < len(line) && line[closeEnd] == '>' {
			head, end := resolveArrowHead(line, closeEnd+1, diagram.ArrowHeadArrow)
			label := strings.TrimSpace(line[afterFirstDash : j-1])
			if label == "" {
				return arrowMatch{}, false
			}
			return arrowMatch{
				start:     openerStart,
				end:       end,
				lineStyle: diagram.LineStyleDotted,
				arrowHead: head,
				label:     processLabel(label),
			}, true
		}
		dotCount := 0
		for closeEnd < len(line) && line[closeEnd] == '.' {
			dotCount++
			closeEnd++
		}
		if dotCount >= 2 {
			label := strings.TrimSpace(line[afterFirstDash : j-1])
			if label == "" {
				return arrowMatch{}, false
			}
			head := diagram.ArrowHeadNone
			if closeEnd < len(line) && (line[closeEnd] == 'o' || line[closeEnd] == 'x') {
				head = circleOrCross(line[closeEnd])
				closeEnd++
			}
			return arrowMatch{
				start:     openerStart,
				end:       closeEnd,
				lineStyle: diagram.LineStyleDotted,
				arrowHead: head,
				label:     processLabel(label),
			}, true
		}
		j++
	}
	return arrowMatch{}, false
}

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
			head, end := resolveArrowHead(line, m+1, diagram.ArrowHeadArrow)
			return arrowMatch{
				start:     openerStart,
				end:       end,
				lineStyle: style,
				arrowHead: head,
				label:     processLabel(label),
			}, true
		}
		if count >= 3 {
			return arrowMatch{
				start:     openerStart,
				end:       m,
				lineStyle: style,
				arrowHead: diagram.ArrowHeadNone,
				label:     processLabel(label),
			}, true
		}
		k = m
	}
	return arrowMatch{}, false
}

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
	m.label = processLabel(rest[:closeIdx])
	m.end += consumed + 1 + closeIdx + 1
}
