// Package flowchart parses Mermaid flowchart/graph syntax into a
// FlowchartDiagram AST.
package flowchart

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
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
	return p.diagram, nil
}

type parser struct {
	diagram       *diagram.FlowchartDiagram
	nodeIndex     map[string]int
	subgraphStack []*diagram.Subgraph
}

func (p *parser) currentSubgraph() *diagram.Subgraph {
	if len(p.subgraphStack) == 0 {
		return nil
	}
	return p.subgraphStack[len(p.subgraphStack)-1]
}

func (p *parser) addNode(id string, shape diagram.NodeShape, label string, classes []string) {
	if id == "" {
		return
	}
	sg := p.currentSubgraph()
	if sg != nil {
		for i := range sg.Nodes {
			if sg.Nodes[i].ID == id {
				if sg.Nodes[i].Shape == diagram.NodeShapeUnknown && shape != diagram.NodeShapeUnknown {
					sg.Nodes[i].Shape = shape
				}
				if sg.Nodes[i].Label == "" && label != "" {
					sg.Nodes[i].Label = label
				}
				if len(classes) > 0 {
					sg.Nodes[i].Classes = append(sg.Nodes[i].Classes, classes...)
				}
				return
			}
		}
		sg.Nodes = append(sg.Nodes, diagram.Node{ID: id, Label: label, Shape: shape, Classes: classes})
		return
	}
	if idx, ok := p.nodeIndex[id]; ok {
		existing := &p.diagram.Nodes[idx]
		if existing.Shape == diagram.NodeShapeUnknown && shape != diagram.NodeShapeUnknown {
			existing.Shape = shape
		}
		if existing.Label == "" && label != "" {
			existing.Label = label
		}
		if len(classes) > 0 {
			existing.Classes = append(existing.Classes, classes...)
		}
		return
	}
	p.nodeIndex[id] = len(p.diagram.Nodes)
	p.diagram.Nodes = append(p.diagram.Nodes, diagram.Node{ID: id, Label: label, Shape: shape, Classes: classes})
}

func (p *parser) addEdge(e diagram.Edge) {
	sg := p.currentSubgraph()
	if sg != nil {
		sg.Edges = append(sg.Edges, e)
	} else {
		p.diagram.Edges = append(p.diagram.Edges, e)
	}
}

func (p *parser) ensureNode(id string) {
	if id == "" {
		return
	}
	sg := p.currentSubgraph()
	if sg != nil {
		for _, n := range sg.Nodes {
			if n.ID == id {
				return
			}
		}
		sg.Nodes = append(sg.Nodes, diagram.Node{ID: id})
		return
	}
	if _, ok := p.nodeIndex[id]; ok {
		return
	}
	p.nodeIndex[id] = len(p.diagram.Nodes)
	p.diagram.Nodes = append(p.diagram.Nodes, diagram.Node{ID: id})
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

	label = stripQuotes(label)

	sg := &diagram.Subgraph{
		ID:    id,
		Label: label,
	}

	if p.currentSubgraph() != nil {
		p.currentSubgraph().Children = append(p.currentSubgraph().Children, *sg)
		p.subgraphStack = append(p.subgraphStack, &p.currentSubgraph().Children[len(p.currentSubgraph().Children)-1])
	} else {
		p.diagram.Subgraphs = append(p.diagram.Subgraphs, *sg)
		p.subgraphStack = append(p.subgraphStack, &p.diagram.Subgraphs[len(p.diagram.Subgraphs)-1])
	}
	p.ensureNode(id)
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
		CSS:    parts[1],
	})
	return nil
}

func (p *parser) parseClassDef(line string) error {
	rest := line[len("classDef "):]
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return fmt.Errorf("classDef requires name and CSS")
	}
	p.diagram.Classes[parts[0]] = parts[1]
	return nil
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
	sg := p.currentSubgraph()
	if sg != nil {
		for i := range sg.Nodes {
			if sg.Nodes[i].ID == nodeID {
				sg.Nodes[i].Classes = append(sg.Nodes[i].Classes, className)
				return
			}
		}
	}
	if idx, ok := p.nodeIndex[nodeID]; ok {
		p.diagram.Nodes[idx].Classes = append(p.diagram.Nodes[idx].Classes, className)
	}
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
		p.diagram.LinkStyles[n] = css
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
		_ = leftID
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
		label = stripQuotes(inner)
		return id, sp.shape, label, cls, nil
	}

	if openMatched != "" {
		return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("unclosed %q in %q", openMatched, s)
	}
	return "", diagram.NodeShapeUnknown, "", nil, fmt.Errorf("unrecognized shape in %q", s)
}

func stripInlineClass(s string) (rest string, classes []string) {
	for {
		idx := strings.LastIndex(s, ":::")
		if idx < 0 {
			return s, classes
		}
		cls := strings.TrimSpace(s[idx+3:])
		if cls != "" {
			classes = append([]string{cls}, classes...)
		}
		s = s[:idx]
	}
}

func stripQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
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

func matchBidirectionalAt(line string, i int) (arrowMatch, bool) {
	if i+1 >= len(line) {
		return arrowMatch{}, false
	}
	rest := line[i+1:]

	if len(rest) >= 3 && rest[0] == 'o' && rest[1] == '-' {
		m, ok := matchDashAt(line, i+2, '-', diagram.LineStyleSolid)
		if ok {
			m.arrowTail = diagram.ArrowHeadCircle
			m.start = i
			if m.arrowHead == diagram.ArrowHeadNone {
				m.arrowHead = diagram.ArrowHeadCircle
			}
			return m, true
		}
	}
	if len(rest) >= 3 && rest[0] == 'x' && rest[1] == '-' {
		m, ok := matchDashAt(line, i+2, '-', diagram.LineStyleSolid)
		if ok {
			m.arrowTail = diagram.ArrowHeadCross
			m.start = i
			if m.arrowHead == diagram.ArrowHeadNone {
				m.arrowHead = diagram.ArrowHeadCross
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
			if line[j] == 'o' {
				head = diagram.ArrowHeadCircle
			} else {
				head = diagram.ArrowHeadCross
			}
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
		head := diagram.ArrowHeadCross
		if line[j] == 'o' {
			head = diagram.ArrowHeadCircle
		}
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
		if line[j] == 'o' {
			head = diagram.ArrowHeadCircle
		} else {
			head = diagram.ArrowHeadCross
		}
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
				label:     stripQuotes(label),
			}, true
		}
		head := diagram.ArrowHeadNone
		end := closeStart
		if closeStart < len(line) && (line[closeStart] == 'o' || line[closeStart] == 'x') {
			if line[closeStart] == 'o' {
				head = diagram.ArrowHeadCircle
			} else {
				head = diagram.ArrowHeadCross
			}
			end = closeStart + 1
		}
		return arrowMatch{
			start:     openerStart,
			end:       end,
			lineStyle: diagram.LineStyleDotted,
			arrowHead: head,
			label:     stripQuotes(label),
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
				label:     stripQuotes(label),
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
				if line[closeEnd] == 'o' {
					head = diagram.ArrowHeadCircle
				} else {
					head = diagram.ArrowHeadCross
				}
				closeEnd++
			}
			return arrowMatch{
				start:     openerStart,
				end:       closeEnd,
				lineStyle: diagram.LineStyleDotted,
				arrowHead: head,
				label:     stripQuotes(label),
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
				label:     stripQuotes(label),
			}, true
		}
		if count >= 3 {
			return arrowMatch{
				start:     openerStart,
				end:       m,
				lineStyle: style,
				arrowHead: diagram.ArrowHeadNone,
				label:     stripQuotes(label),
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
	m.label = stripQuotes(rest[:closeIdx])
	m.end += consumed + 1 + closeIdx + 1
}
