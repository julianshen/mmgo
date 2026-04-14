package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

// --- Helpers ---

func mustParse(t *testing.T, input string) *diagram.FlowchartDiagram {
	t.Helper()
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v\ninput:\n%s", err, input)
	}
	return d
}

func findNode(f *diagram.FlowchartDiagram, id string) (diagram.Node, bool) {
	for _, n := range f.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return diagram.Node{}, false
}

func mustNode(t *testing.T, f *diagram.FlowchartDiagram, id string) diagram.Node {
	t.Helper()
	n, ok := findNode(f, id)
	if !ok {
		t.Fatalf("node %q not found; have %d nodes", id, len(f.Nodes))
	}
	return n
}

// --- Direction header ---

// TestParseFlowchartKeyword exercises the `flowchart` keyword path
// (TestParseAllDirections below covers `graph` for every direction).
func TestParseFlowchartKeyword(t *testing.T) {
	f := mustParse(t, "flowchart TB")
	if f.Direction != diagram.DirectionTB {
		t.Errorf("expected TB, got %v", f.Direction)
	}
}

func TestParseAllDirections(t *testing.T) {
	cases := map[string]diagram.Direction{
		"TB": diagram.DirectionTB,
		"TD": diagram.DirectionTB, // TD is an alias for TB
		"BT": diagram.DirectionBT,
		"LR": diagram.DirectionLR,
		"RL": diagram.DirectionRL,
	}
	for kw, want := range cases {
		t.Run(kw, func(t *testing.T) {
			f := mustParse(t, "graph "+kw)
			if f.Direction != want {
				t.Errorf("%s → got %v, want %v", kw, f.Direction, want)
			}
		})
	}
}

func TestParseDefaultDirection(t *testing.T) {
	// No direction keyword — defaults to TB per Mermaid.
	f := mustParse(t, "graph")
	if f.Direction != diagram.DirectionTB {
		t.Errorf("expected TB default, got %v", f.Direction)
	}
}

// --- Node shapes ---

func TestParseNodeShapes(t *testing.T) {
	cases := []struct {
		name  string
		def   string
		shape diagram.NodeShape
		label string
	}{
		{"rectangle", "A[Label]", diagram.NodeShapeRectangle, "Label"},
		{"rounded", "A(Label)", diagram.NodeShapeRoundedRectangle, "Label"},
		{"stadium", "A([Label])", diagram.NodeShapeStadium, "Label"},
		{"subroutine", "A[[Label]]", diagram.NodeShapeSubroutine, "Label"},
		{"cylinder", "A[(Label)]", diagram.NodeShapeCylinder, "Label"},
		{"circle", "A((Label))", diagram.NodeShapeCircle, "Label"},
		{"asymmetric", "A>Label]", diagram.NodeShapeAsymmetric, "Label"},
		{"diamond", "A{Label}", diagram.NodeShapeDiamond, "Label"},
		{"hexagon", "A{{Label}}", diagram.NodeShapeHexagon, "Label"},
		{"parallelogram", "A[/Label/]", diagram.NodeShapeParallelogram, "Label"},
		{"parallelogram-alt", `A[\Label\]`, diagram.NodeShapeParallelogramAlt, "Label"},
		{"trapezoid", `A[/Label\]`, diagram.NodeShapeTrapezoid, "Label"},
		{"trapezoid-alt", `A[\Label/]`, diagram.NodeShapeTrapezoidAlt, "Label"},
		{"double-circle", "A(((Label)))", diagram.NodeShapeDoubleCircle, "Label"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, "graph LR\n    "+tc.def)
			n := mustNode(t, f, "A")
			if n.Shape != tc.shape {
				t.Errorf("shape = %v, want %v", n.Shape, tc.shape)
			}
			if n.Label != tc.label {
				t.Errorf("label = %q, want %q", n.Label, tc.label)
			}
		})
	}
}

func TestParseNodeWithSpacesInLabel(t *testing.T) {
	f := mustParse(t, "graph LR\n    A[Hello World]")
	n := mustNode(t, f, "A")
	if n.Label != "Hello World" {
		t.Errorf("label = %q, want %q", n.Label, "Hello World")
	}
}

func TestParseBareNodeInEdge(t *testing.T) {
	// First defines A with a shape, then references it bare in another edge.
	f := mustParse(t, "graph LR\n    A[Start] --> B\n    B --> C")
	if len(f.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(f.Nodes))
	}
}

// --- Edges ---

func TestParseEdgeTypes(t *testing.T) {
	cases := []struct {
		name      string
		arrow     string
		lineStyle diagram.LineStyle
		arrowHead diagram.ArrowHead
	}{
		{"solid-arrow", "-->", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
		{"solid-none", "---", diagram.LineStyleSolid, diagram.ArrowHeadNone},
		{"dotted-arrow", "-.->", diagram.LineStyleDotted, diagram.ArrowHeadArrow},
		{"dotted-none", "-.-", diagram.LineStyleDotted, diagram.ArrowHeadNone},
		{"thick-arrow", "==>", diagram.LineStyleThick, diagram.ArrowHeadArrow},
		{"thick-none", "===", diagram.LineStyleThick, diagram.ArrowHeadNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, "graph LR\n    A "+tc.arrow+" B")
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			e := f.Edges[0]
			if e.From != "A" || e.To != "B" {
				t.Errorf("edge = %s→%s, want A→B", e.From, e.To)
			}
			if e.LineStyle != tc.lineStyle {
				t.Errorf("line style = %v, want %v", e.LineStyle, tc.lineStyle)
			}
			if e.ArrowHead != tc.arrowHead {
				t.Errorf("arrow head = %v, want %v", e.ArrowHead, tc.arrowHead)
			}
		})
	}
}

func TestParseEdgeWithPipeLabel(t *testing.T) {
	f := mustParse(t, "graph LR\n    A -->|Yes| B")
	e := f.Edges[0]
	if e.Label != "Yes" {
		t.Errorf("label = %q, want %q", e.Label, "Yes")
	}
	if e.From != "A" || e.To != "B" {
		t.Errorf("edge = %s→%s, want A→B", e.From, e.To)
	}
}

func TestParseEdgeLabelWithSpaces(t *testing.T) {
	f := mustParse(t, "graph LR\n    A -->|hello world| B")
	e := f.Edges[0]
	if e.Label != "hello world" {
		t.Errorf("label = %q", e.Label)
	}
}

// --- Combined node+edge on one line ---

func TestParseNodesAndEdgeOnOneLine(t *testing.T) {
	f := mustParse(t, "graph LR\n    A[Start] --> B[End]")
	if len(f.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(f.Nodes))
	}
	a := mustNode(t, f, "A")
	if a.Label != "Start" {
		t.Errorf("A.Label = %q", a.Label)
	}
	b := mustNode(t, f, "B")
	if b.Label != "End" {
		t.Errorf("B.Label = %q", b.Label)
	}
	if len(f.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(f.Edges))
	}
}

// --- Multi-line ---

func TestParseMultiLine(t *testing.T) {
	input := `graph LR
    A[Start] --> B{Check}
    B -->|Yes| C[Ok]
    B -->|No| D[Fail]
    C --> E((End))
    D --> E`

	f := mustParse(t, input)
	if len(f.Nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d: %+v", len(f.Nodes), f.Nodes)
	}
	if len(f.Edges) != 5 {
		t.Errorf("expected 5 edges, got %d", len(f.Edges))
	}

	b := mustNode(t, f, "B")
	if b.Shape != diagram.NodeShapeDiamond {
		t.Errorf("B should be diamond, got %v", b.Shape)
	}
	e := mustNode(t, f, "E")
	if e.Shape != diagram.NodeShapeCircle {
		t.Errorf("E should be circle, got %v", e.Shape)
	}
}

// --- Comments ---

func TestParseCommentLine(t *testing.T) {
	input := `graph LR
    %% this is a comment
    A --> B`

	f := mustParse(t, input)
	if len(f.Nodes) != 2 || len(f.Edges) != 1 {
		t.Errorf("expected 2 nodes and 1 edge, got %d/%d", len(f.Nodes), len(f.Edges))
	}
}

func TestParseCommentAtEndOfLine(t *testing.T) {
	input := `graph LR
    A --> B %% inline comment`

	f := mustParse(t, input)
	if len(f.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(f.Edges))
	}
}

// --- Whitespace / empty lines ---

func TestParseEmptyLines(t *testing.T) {
	input := `graph LR

    A --> B

    B --> C

`
	f := mustParse(t, input)
	if len(f.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(f.Edges))
	}
}

// --- Node deduplication ---

func TestParseNodeDeduplication(t *testing.T) {
	// A is defined twice: bare reference then with label. The parser
	// should fill in the shape/label once and not duplicate the node.
	input := `graph LR
    A --> B
    A[Start] --> C`

	f := mustParse(t, input)
	if len(f.Nodes) != 3 {
		t.Errorf("expected 3 nodes (A, B, C), got %d", len(f.Nodes))
	}
	a := mustNode(t, f, "A")
	if a.Label != "Start" {
		t.Errorf("A.Label should be 'Start' (filled in), got %q", a.Label)
	}
}

// --- Error cases ---

func TestParseErrorReportsLineNumber(t *testing.T) {
	input := `graph LR
    A[Start] --> B
    ???`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "line 3:") {
		t.Errorf("error should mention 'line 3:': %v", err)
	}
}

func TestParseErrorCases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string // substring expected in error
	}{
		{"unknown direction", "graph XY", "unknown direction"},
		{"unclosed bracket", "graph LR\n    A[unclosed", "unclosed"},
		{"unclosed pipe label", "graph LR\n    A -->|Yes B", "unclosed edge label"},
		{"non-header first line", "A --> B", "expected 'graph' or 'flowchart'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.input))
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

// --- Determinism ---

func TestParseDeterministic(t *testing.T) {
	input := `graph LR
    A[Start] --> B{Check}
    B -->|Yes| C
    B -->|No| D`

	f1 := mustParse(t, input)
	f2 := mustParse(t, input)

	if len(f1.Nodes) != len(f2.Nodes) || len(f1.Edges) != len(f2.Edges) {
		t.Fatal("parse results differ in size")
	}
	// Node has a slice field (Classes) so we compare field-by-field.
	for i := range f1.Nodes {
		a, b := f1.Nodes[i], f2.Nodes[i]
		if a.ID != b.ID || a.Label != b.Label || a.Shape != b.Shape {
			t.Errorf("node %d differs: %+v vs %+v", i, f1.Nodes[i], f2.Nodes[i])
		}
	}
	for i := range f1.Edges {
		if f1.Edges[i] != f2.Edges[i] {
			t.Errorf("edge %d differs: %+v vs %+v", i, f1.Edges[i], f2.Edges[i])
		}
	}
}

// --- Parser output satisfies Diagram interface ---

func TestParserReturnsDiagramInterface(t *testing.T) {
	f := mustParse(t, "graph LR\n    A --> B")
	var d diagram.Diagram = f
	if d.Type() != diagram.Flowchart {
		t.Errorf("Type() = %v, want Flowchart", d.Type())
	}
}
