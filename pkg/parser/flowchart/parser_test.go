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

// --- Inline edge labels (I1) ---

func TestParseInlineEdgeLabels(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		label     string
		lineStyle diagram.LineStyle
		arrowHead diagram.ArrowHead
	}{
		{"solid arrow", "graph LR\n    A -- yes --> B", "yes", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
		{"solid no-head", "graph LR\n    A -- no --- B", "no", diagram.LineStyleSolid, diagram.ArrowHeadNone},
		{"thick arrow", "graph LR\n    A == go ==> B", "go", diagram.LineStyleThick, diagram.ArrowHeadArrow},
		{"thick no-head", "graph LR\n    A == stop === B", "stop", diagram.LineStyleThick, diagram.ArrowHeadNone},
		{"multi-word label", "graph LR\n    A -- hello world --> B", "hello world", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
		{"long-dash terminator", "graph LR\n    A -- foo ----> B", "foo", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, tc.input)
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			e := f.Edges[0]
			if e.From != "A" || e.To != "B" {
				t.Errorf("edge = %s→%s", e.From, e.To)
			}
			if e.Label != tc.label {
				t.Errorf("label = %q, want %q", e.Label, tc.label)
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

// --- Arrow inside a label (I2) ---

func TestParseArrowInsideLabel(t *testing.T) {
	// `-->` inside a node label must not be confused with an edge.
	f := mustParse(t, "graph LR\n    A[contains --> text] --> B")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	if f.Edges[0].From != "A" || f.Edges[0].To != "B" {
		t.Errorf("edge = %s→%s", f.Edges[0].From, f.Edges[0].To)
	}
	a := mustNode(t, f, "A")
	if a.Label != "contains --> text" {
		t.Errorf("A.Label = %q", a.Label)
	}
}

func TestParseArrowInsideQuotedLabel(t *testing.T) {
	// Arrow-like text inside a double-quoted region is also skipped.
	f := mustParse(t, `graph LR`+"\n"+`    A["has --> inside"] --> B`)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
}

// --- Inline label rejection paths ---

func TestParseInlineLabelRejections(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string // substring
	}{
		// Opener `--` not followed by whitespace: not an inline label,
		// falls through to "unrecognized shape".
		{"no space after opener", "graph LR\n    A --text --> B", "shape"},
		// Empty label between opener and terminator: the `--` is not
		// recognized as an inline opener (no label text), so findArrow
		// picks the later `-->` and the `A --` leftover fails as an
		// unrecognized shape.
		{"empty inline label", "graph LR\n    A --  --> B", "shape"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tc.input))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

// --- Long-dash arrow variants (I3) ---

func TestParseLongArrows(t *testing.T) {
	cases := []struct {
		name      string
		arrow     string
		lineStyle diagram.LineStyle
		arrowHead diagram.ArrowHead
	}{
		{"solid 3-dash", "--->", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
		{"solid 4-dash", "---->", diagram.LineStyleSolid, diagram.ArrowHeadArrow},
		{"solid 5-dash no-head", "-----", diagram.LineStyleSolid, diagram.ArrowHeadNone},
		{"thick 3-eq", "===>", diagram.LineStyleThick, diagram.ArrowHeadArrow},
		{"thick 4-eq no-head", "====", diagram.LineStyleThick, diagram.ArrowHeadNone},
		{"dotted 2-dot", "-..->", diagram.LineStyleDotted, diagram.ArrowHeadArrow},
		{"dotted 3-dot no-head", "-...-", diagram.LineStyleDotted, diagram.ArrowHeadNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, "graph LR\n    A "+tc.arrow+" B")
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			e := f.Edges[0]
			if e.From != "A" || e.To != "B" {
				t.Errorf("edge = %s→%s", e.From, e.To)
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

// --- Chained edges ---

func TestParseChainedEdges(t *testing.T) {
	f := mustParse(t, "graph LR\n    A --> B --> C")
	if len(f.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(f.Nodes))
	}
	if len(f.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(f.Edges))
	}
	if f.Edges[0].From != "A" || f.Edges[0].To != "B" {
		t.Errorf("edge[0] = %s→%s, want A→B", f.Edges[0].From, f.Edges[0].To)
	}
	if f.Edges[1].From != "B" || f.Edges[1].To != "C" {
		t.Errorf("edge[1] = %s→%s, want B→C", f.Edges[1].From, f.Edges[1].To)
	}
}

func TestParseChainedEdgesWithShapes(t *testing.T) {
	f := mustParse(t, "graph LR\n    A[Start] --> B{Check} --> C((End))")
	if len(f.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(f.Edges))
	}
	if mustNode(t, f, "B").Shape != diagram.NodeShapeDiamond {
		t.Errorf("B should be diamond")
	}
	if mustNode(t, f, "C").Shape != diagram.NodeShapeCircle {
		t.Errorf("C should be circle")
	}
}

func TestParseChainedEdgesMixedStyles(t *testing.T) {
	f := mustParse(t, "graph LR\n    A --> B -.-> C ==> D")
	if len(f.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(f.Edges))
	}
	if f.Edges[0].LineStyle != diagram.LineStyleSolid {
		t.Errorf("edge[0] style = %v, want solid", f.Edges[0].LineStyle)
	}
	if f.Edges[1].LineStyle != diagram.LineStyleDotted {
		t.Errorf("edge[1] style = %v, want dotted", f.Edges[1].LineStyle)
	}
	if f.Edges[2].LineStyle != diagram.LineStyleThick {
		t.Errorf("edge[2] style = %v, want thick", f.Edges[2].LineStyle)
	}
}

// --- Comments inside labels ---

func TestParseCommentNotStrippedInsideLabel(t *testing.T) {
	cases := []struct {
		name  string
		input string
		check func(t *testing.T, f *diagram.FlowchartDiagram)
	}{
		{
			"percent literal no space",
			"graph LR\n    A[100%%]",
			func(t *testing.T, f *diagram.FlowchartDiagram) {
				if got := mustNode(t, f, "A").Label; got != "100%%" {
					t.Errorf("label = %q", got)
				}
			},
		},
		{
			"percent with space inside rectangle",
			"graph LR\n    A[foo %% bar]",
			func(t *testing.T, f *diagram.FlowchartDiagram) {
				if got := mustNode(t, f, "A").Label; got != "foo %% bar" {
					t.Errorf("label = %q", got)
				}
			},
		},
		{
			"percent inside pipe edge label",
			"graph LR\n    A -->|foo %% bar| B",
			func(t *testing.T, f *diagram.FlowchartDiagram) {
				if len(f.Edges) != 1 || f.Edges[0].Label != "foo %% bar" {
					t.Errorf("edge label = %q", f.Edges[0].Label)
				}
			},
		},
		{
			"percent inside quoted label",
			`graph LR` + "\n" + `    A["foo %% bar"] --> B`,
			func(t *testing.T, f *diagram.FlowchartDiagram) {
				if got := mustNode(t, f, "A").Label; got != "foo %% bar" {
					t.Errorf("label = %q", got)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, tc.input)
			tc.check(t, f)
		})
	}
}

// --- Hyphens in node IDs ---

func TestParseHyphenInID(t *testing.T) {
	cases := []struct {
		name  string
		input string
		from  string
		to    string
	}{
		{"spaced arrow", "graph LR\n    node-1 --> node-2", "node-1", "node-2"},
		{"tight arrow", "graph LR\n    node-1-->node-2", "node-1", "node-2"},
		{"with shapes", "graph LR\n    first-step[Start] --> last-step[End]", "first-step", "last-step"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, tc.input)
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			e := f.Edges[0]
			if e.From != tc.from || e.To != tc.to {
				t.Errorf("edge = %s→%s, want %s→%s", e.From, e.To, tc.from, tc.to)
			}
		})
	}
}

// --- Whitespace between ID and shape ---

func TestParseSpaceBetweenIDAndShape(t *testing.T) {
	f := mustParse(t, "graph LR\n    A [Label] --> B (Rounded)")
	a := mustNode(t, f, "A")
	if a.Label != "Label" || a.Shape != diagram.NodeShapeRectangle {
		t.Errorf("A = %+v", a)
	}
	b := mustNode(t, f, "B")
	if b.Label != "Rounded" || b.Shape != diagram.NodeShapeRoundedRectangle {
		t.Errorf("B = %+v", b)
	}
}

// --- Tight spacing around arrows ---

func TestParseTightArrowSpacing(t *testing.T) {
	// No space around the arrow should still parse.
	f := mustParse(t, "graph LR\n    A-->B")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	if f.Edges[0].From != "A" || f.Edges[0].To != "B" {
		t.Errorf("edge = %s→%s", f.Edges[0].From, f.Edges[0].To)
	}
}

func TestParseTightArrowWithShapes(t *testing.T) {
	f := mustParse(t, "graph LR\n    A[x]-->B[y]")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	if mustNode(t, f, "A").Label != "x" {
		t.Errorf("A.Label = %q", mustNode(t, f, "A").Label)
	}
	if mustNode(t, f, "B").Label != "y" {
		t.Errorf("B.Label = %q", mustNode(t, f, "B").Label)
	}
}

// --- Header-only and blank-leading-lines ---

func TestParseHeaderOnly(t *testing.T) {
	f := mustParse(t, "graph LR")
	if len(f.Nodes) != 0 || len(f.Edges) != 0 {
		t.Errorf("expected empty diagram, got %d nodes, %d edges", len(f.Nodes), len(f.Edges))
	}
	if f.Direction != diagram.DirectionLR {
		t.Errorf("direction = %v, want LR", f.Direction)
	}
}

func TestParseBlankLinesBeforeHeader(t *testing.T) {
	f := mustParse(t, "\n\n\ngraph LR\n    A --> B")
	if f.Direction != diagram.DirectionLR {
		t.Errorf("direction = %v", f.Direction)
	}
	if len(f.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(f.Edges))
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
		{"extra tokens after direction", "graph LR foo", "extra tokens"},
		{"graph word boundary", "grapha LR", "expected 'graph' or 'flowchart'"},
		{"flowchart word boundary", "flowchartfoo TB", "expected 'graph' or 'flowchart'"},
		{"unclosed bracket", "graph LR\n    A[unclosed", "unclosed"},
		{"unclosed pipe label", "graph LR\n    A -->|Yes B", "unclosed edge label"},
		{"non-header first line", "A --> B", "expected 'graph' or 'flowchart'"},
		// Unterminated inline label (missing closing `-->`/`---`).
		{"unterminated inline label", "graph LR\n    A -- text", "unterminated inline edge label"},
		// Unicode IDs — pinned as a clearer error until supported.
		{"non-ASCII id leading", "graph LR\n    日本 --> B", "non-ASCII"},
		{"non-ASCII id after ASCII", "graph LR\n    A日 --> B", "non-ASCII"},
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

// =======================================================================
// Phase 1 — New feature tests
// =======================================================================

// --- Invisible edges (~~~) ---

func TestParseInvisibleEdge(t *testing.T) {
	f := mustParse(t, "graph LR\n    A ~~~ B")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.From != "A" || e.To != "B" {
		t.Errorf("edge = %s→%s, want A→B", e.From, e.To)
	}
	if e.LineStyle != diagram.LineStyleInvisible {
		t.Errorf("line style = %v, want invisible", e.LineStyle)
	}
	if e.ArrowHead != diagram.ArrowHeadNone {
		t.Errorf("arrow head = %v, want none", e.ArrowHead)
	}
}

func TestParseInvisibleEdgeLongTildes(t *testing.T) {
	f := mustParse(t, "graph LR\n    A ~~~~~ B")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	if f.Edges[0].LineStyle != diagram.LineStyleInvisible {
		t.Errorf("line style = %v, want invisible", f.Edges[0].LineStyle)
	}
}

func TestParseInvisibleEdgeTooShort(t *testing.T) {
	_, err := Parse(strings.NewReader("graph LR\n    A ~~ B"))
	if err == nil {
		t.Fatal("expected error for ~~ (too short for invisible)")
	}
}

// --- Circle/Cross arrow endpoints ---

func TestParseCircleArrowHead(t *testing.T) {
	cases := []struct {
		name  string
		arrow string
		head  diagram.ArrowHead
	}{
		{"solid-circle", "-->o", diagram.ArrowHeadCircle},
		{"solid-cross", "-->x", diagram.ArrowHeadCross},
		{"dotted-circle", "-.->o", diagram.ArrowHeadCircle},
		{"dotted-cross", "-.->x", diagram.ArrowHeadCross},
		{"thick-circle", "==>o", diagram.ArrowHeadCircle},
		{"thick-cross", "==>x", diagram.ArrowHeadCross},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, "graph LR\n    A "+tc.arrow+" B")
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			if f.Edges[0].ArrowHead != tc.head {
				t.Errorf("arrow head = %v, want %v", f.Edges[0].ArrowHead, tc.head)
			}
		})
	}
}

func TestParseCircleArrowHeadNoTip(t *testing.T) {
	f := mustParse(t, "graph LR\n    A ---o B")
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.ArrowHead != diagram.ArrowHeadCircle {
		t.Errorf("arrow head = %v, want circle", e.ArrowHead)
	}
}

// --- Bidirectional arrows ---

func TestParseBidirectionalArrows(t *testing.T) {
	cases := []struct {
		name      string
		arrow     string
		head      diagram.ArrowHead
		tail      diagram.ArrowHead
		lineStyle diagram.LineStyle
	}{
		{"solid", "<-->", diagram.ArrowHeadArrow, diagram.ArrowHeadArrow, diagram.LineStyleSolid},
		{"solid-no-head", "<---", diagram.ArrowHeadNone, diagram.ArrowHeadArrow, diagram.LineStyleSolid},
		{"thick", "<==>", diagram.ArrowHeadArrow, diagram.ArrowHeadArrow, diagram.LineStyleThick},
		{"circle-circle", "o--o", diagram.ArrowHeadCircle, diagram.ArrowHeadCircle, diagram.LineStyleSolid},
		{"cross-cross", "x--x", diagram.ArrowHeadCross, diagram.ArrowHeadCross, diagram.LineStyleSolid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := mustParse(t, "graph LR\n    A "+tc.arrow+" B")
			if len(f.Edges) != 1 {
				t.Fatalf("expected 1 edge, got %d", len(f.Edges))
			}
			e := f.Edges[0]
			if e.ArrowHead != tc.head {
				t.Errorf("arrow head = %v, want %v", e.ArrowHead, tc.head)
			}
			if e.ArrowTail != tc.tail {
				t.Errorf("arrow tail = %v, want %v", e.ArrowTail, tc.tail)
			}
			if e.LineStyle != tc.lineStyle {
				t.Errorf("line style = %v, want %v", e.LineStyle, tc.lineStyle)
			}
		})
	}
}

// --- Subgraphs ---

func TestParseSubgraph(t *testing.T) {
	input := `graph LR
    subgraph sg1
        A --> B
    end`
	f := mustParse(t, input)
	if len(f.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(f.Subgraphs))
	}
	sg := f.Subgraphs[0]
	if sg.ID != "sg1" {
		t.Errorf("subgraph ID = %q, want %q", sg.ID, "sg1")
	}
	if sg.Label != "sg1" {
		t.Errorf("subgraph label = %q, want %q", sg.Label, "sg1")
	}
	if len(sg.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes in subgraph, got %d", len(sg.Nodes))
	}
	if len(sg.Edges) != 1 {
		t.Errorf("expected 1 edge in subgraph, got %d", len(sg.Edges))
	}
}

func TestParseSubgraphWithLabel(t *testing.T) {
	input := `graph LR
    subgraph sg1 My Group
        A --> B
    end`
	f := mustParse(t, input)
	if len(f.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(f.Subgraphs))
	}
	sg := f.Subgraphs[0]
	if sg.ID != "sg1" {
		t.Errorf("subgraph ID = %q, want %q", sg.ID, "sg1")
	}
	if sg.Label != "My Group" {
		t.Errorf("subgraph label = %q, want %q", sg.Label, "My Group")
	}
}

func TestParseSubgraphWithQuotedLabel(t *testing.T) {
	input := `graph LR
    subgraph sg1 "My Group"
        A --> B
    end`
	f := mustParse(t, input)
	sg := f.Subgraphs[0]
	if sg.Label != "My Group" {
		t.Errorf("subgraph label = %q, want %q", sg.Label, "My Group")
	}
}

func TestParseNestedSubgraph(t *testing.T) {
	input := `graph LR
    subgraph outer
        A --> B
        subgraph inner
            C --> D
        end
    end`
	f := mustParse(t, input)
	if len(f.Subgraphs) != 1 {
		t.Fatalf("expected 1 top-level subgraph, got %d", len(f.Subgraphs))
	}
	outer := f.Subgraphs[0]
	if len(outer.Children) != 1 {
		t.Fatalf("expected 1 nested subgraph, got %d", len(outer.Children))
	}
	inner := outer.Children[0]
	if inner.ID != "inner" {
		t.Errorf("inner ID = %q, want %q", inner.ID, "inner")
	}
	if len(inner.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes in inner, got %d", len(inner.Nodes))
	}
}

func TestParseEndWithoutSubgraph(t *testing.T) {
	_, err := Parse(strings.NewReader("graph LR\n    end"))
	if err == nil {
		t.Fatal("expected error for 'end' without subgraph")
	}
	if !strings.Contains(err.Error(), "unexpected 'end'") {
		t.Errorf("error = %v, want 'unexpected end'", err)
	}
}

// --- Direction inside subgraph ---

func TestParseDirectionInSubgraph(t *testing.T) {
	input := `graph LR
    subgraph sg1
        direction TB
        A --> B
    end`
	f := mustParse(t, input)
	sg := f.Subgraphs[0]
	if sg.Direction != diagram.DirectionTB {
		t.Errorf("subgraph direction = %v, want TB", sg.Direction)
	}
}

// --- style directive ---

func TestParseStyleDirective(t *testing.T) {
	input := `graph LR
    A --> B
    style A fill:#f9f,stroke:#333`
	f := mustParse(t, input)
	if len(f.Styles) != 1 {
		t.Fatalf("expected 1 style, got %d", len(f.Styles))
	}
	s := f.Styles[0]
	if s.NodeID != "A" {
		t.Errorf("style node ID = %q, want %q", s.NodeID, "A")
	}
	if s.CSS != "fill:#f9f,stroke:#333" {
		t.Errorf("style CSS = %q", s.CSS)
	}
}

// --- classDef directive ---

func TestParseClassDefDirective(t *testing.T) {
	input := `graph LR
    A --> B
    classDef red fill:#f00,stroke:#333`
	f := mustParse(t, input)
	if len(f.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(f.Classes))
	}
	css, ok := f.Classes["red"]
	if !ok {
		t.Fatal("class 'red' not found")
	}
	if css != "fill:#f00,stroke:#333" {
		t.Errorf("class CSS = %q", css)
	}
}

// --- class directive ---

func TestParseClassDirective(t *testing.T) {
	input := `graph LR
    A[Start] --> B[End]
    class A,B red`
	f := mustParse(t, input)
	a := mustNode(t, f, "A")
	found := false
	for _, c := range a.Classes {
		if c == "red" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("A.Classes = %v, want 'red' included", a.Classes)
	}
	b := mustNode(t, f, "B")
	found = false
	for _, c := range b.Classes {
		if c == "red" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("B.Classes = %v, want 'red' included", b.Classes)
	}
}

// --- linkStyle directive ---

func TestParseLinkStyleDirective(t *testing.T) {
	input := `graph LR
    A --> B
    A --> C
    linkStyle 0,1 stroke:red`
	f := mustParse(t, input)
	if len(f.LinkStyles) != 2 {
		t.Fatalf("expected 2 linkStyles, got %d", len(f.LinkStyles))
	}
	if f.LinkStyles[0] != "stroke:red" {
		t.Errorf("linkStyle[0] = %q", f.LinkStyles[0])
	}
	if f.LinkStyles[1] != "stroke:red" {
		t.Errorf("linkStyle[1] = %q", f.LinkStyles[1])
	}
}

// --- Ampersand branching ---

func TestParseAmpersandBranching(t *testing.T) {
	input := "graph LR\n    A --> B & C"
	f := mustParse(t, input)
	if len(f.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(f.Edges))
	}
	e0 := f.Edges[0]
	if e0.From != "A" || e0.To != "B" {
		t.Errorf("edge[0] = %s→%s, want A→B", e0.From, e0.To)
	}
	e1 := f.Edges[1]
	if e1.From != "A" || e1.To != "C" {
		t.Errorf("edge[1] = %s→%s, want A→C", e1.From, e1.To)
	}
}

func TestParseAmpersandThreeWay(t *testing.T) {
	input := "graph LR\n    A --> B & C & D"
	f := mustParse(t, input)
	if len(f.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(f.Edges))
	}
	for _, e := range f.Edges {
		if e.From != "A" {
			t.Errorf("edge from = %q, want A", e.From)
		}
	}
}

func TestParseAmpersandWithShapes(t *testing.T) {
	input := "graph LR\n    A[Start] --> B[Mid] & C[End]"
	f := mustParse(t, input)
	if len(f.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(f.Edges))
	}
	if mustNode(t, f, "B").Label != "Mid" {
		t.Errorf("B.Label = %q", mustNode(t, f, "B").Label)
	}
	if mustNode(t, f, "C").Label != "End" {
		t.Errorf("C.Label = %q", mustNode(t, f, "C").Label)
	}
}

// --- Edge IDs ---

func TestParseEdgeID(t *testing.T) {
	input := "graph LR\n    A e1@--> B"
	f := mustParse(t, input)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.ID != "e1" {
		t.Errorf("edge ID = %q, want %q", e.ID, "e1")
	}
	if e.From != "A" || e.To != "B" {
		t.Errorf("edge = %s→%s, want A→B", e.From, e.To)
	}
}

func TestParseEdgeIDWithLabel(t *testing.T) {
	input := "graph LR\n    A e2@-->|yes| B"
	f := mustParse(t, input)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.ID != "e2" {
		t.Errorf("edge ID = %q, want %q", e.ID, "e2")
	}
	if e.Label != "yes" {
		t.Errorf("edge label = %q, want %q", e.Label, "yes")
	}
}

// --- Dotted inline labels ---

func TestParseDottedInlineLabel(t *testing.T) {
	input := "graph LR\n    A -. text .-> B"
	f := mustParse(t, input)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.Label != "text" {
		t.Errorf("label = %q, want %q", e.Label, "text")
	}
	if e.LineStyle != diagram.LineStyleDotted {
		t.Errorf("line style = %v, want dotted", e.LineStyle)
	}
	if e.ArrowHead != diagram.ArrowHeadArrow {
		t.Errorf("arrow head = %v, want arrow", e.ArrowHead)
	}
}

func TestParseDottedInlineLabelNoHead(t *testing.T) {
	input := "graph LR\n    A -. text .- B"
	f := mustParse(t, input)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	e := f.Edges[0]
	if e.Label != "text" {
		t.Errorf("label = %q, want %q", e.Label, "text")
	}
	if e.LineStyle != diagram.LineStyleDotted {
		t.Errorf("line style = %v, want dotted", e.LineStyle)
	}
	if e.ArrowHead != diagram.ArrowHeadNone {
		t.Errorf("arrow head = %v, want none", e.ArrowHead)
	}
}

// --- Quote stripping ---

func TestParseQuoteStripping(t *testing.T) {
	f := mustParse(t, `graph LR`+"\n"+`    A["Hello World"] --> B["Goodbye"]`)
	a := mustNode(t, f, "A")
	if a.Label != "Hello World" {
		t.Errorf("A.Label = %q, want %q", a.Label, "Hello World")
	}
	b := mustNode(t, f, "B")
	if b.Label != "Goodbye" {
		t.Errorf("B.Label = %q, want %q", b.Label, "Goodbye")
	}
}

// --- ::: class operator ---

func TestParseInlineClassOperator(t *testing.T) {
	input := "graph LR\n    A[Start]:::red --> B"
	f := mustParse(t, input)
	a := mustNode(t, f, "A")
	found := false
	for _, c := range a.Classes {
		if c == "red" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("A.Classes = %v, want 'red' included", a.Classes)
	}
	if a.Label != "Start" {
		t.Errorf("A.Label = %q, want %q", a.Label, "Start")
	}
}

func TestParseInlineClassOperatorBareNode(t *testing.T) {
	input := "graph LR\n    A:::red --> B"
	f := mustParse(t, input)
	a := mustNode(t, f, "A")
	found := false
	for _, c := range a.Classes {
		if c == "red" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("A.Classes = %v, want 'red' included", a.Classes)
	}
}

func TestParseInlineClassMultiple(t *testing.T) {
	input := "graph LR\n    A[Start]:::red:::blue --> B"
	f := mustParse(t, input)
	a := mustNode(t, f, "A")
	if len(a.Classes) < 2 {
		t.Errorf("A.Classes = %v, want at least 2", a.Classes)
	}
	hasRed, hasBlue := false, false
	for _, c := range a.Classes {
		if c == "red" {
			hasRed = true
		}
		if c == "blue" {
			hasBlue = true
		}
	}
	if !hasRed || !hasBlue {
		t.Errorf("A.Classes = %v, want red and blue", a.Classes)
	}
}

// --- Pipe label quote stripping ---

func TestParsePipeLabelQuoteStripping(t *testing.T) {
	f := mustParse(t, `graph LR`+"\n"+`    A -->|"Hello World"| B`)
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(f.Edges))
	}
	if f.Edges[0].Label != "Hello World" {
		t.Errorf("label = %q, want %q", f.Edges[0].Label, "Hello World")
	}
}

// --- Subgraph with edges crossing boundaries ---

func TestParseSubgraphWithEdgeToOutside(t *testing.T) {
	input := `graph LR
    subgraph sg1
        A --> B
    end
    B --> C`
	f := mustParse(t, input)
	if len(f.Subgraphs) != 1 {
		t.Fatalf("expected 1 subgraph, got %d", len(f.Subgraphs))
	}
	if len(f.Edges) != 1 {
		t.Fatalf("expected 1 top-level edge, got %d", len(f.Edges))
	}
	if f.Edges[0].From != "B" || f.Edges[0].To != "C" {
		t.Errorf("top edge = %s→%s, want B→C", f.Edges[0].From, f.Edges[0].To)
	}
}

// --- AllNodes / AllEdges helpers ---

func TestAllNodesAllEdges(t *testing.T) {
	input := `graph LR
    A --> B
    subgraph sg1
        C --> D
    end`
	f := mustParse(t, input)
	allNodes := f.AllNodes()
	if len(allNodes) < 4 {
		t.Errorf("AllNodes = %d, want at least 4", len(allNodes))
	}
	allEdges := f.AllEdges()
	if len(allEdges) != 2 {
		t.Errorf("AllEdges = %d, want 2", len(allEdges))
	}
}
