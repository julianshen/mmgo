package diagram

import "testing"

func TestFlowchartImplementsDiagram(t *testing.T) {
	var d Diagram = &FlowchartDiagram{}
	if d.Type() != Flowchart {
		t.Errorf("expected Type() = Flowchart, got %v", d.Type())
	}
}

func TestFlowchartConstruction(t *testing.T) {
	f := &FlowchartDiagram{
		Direction: DirectionLR,
		Nodes: []Node{
			{ID: "a", Label: "Start", Shape: NodeShapeRectangle},
			{ID: "b", Label: "End", Shape: NodeShapeRoundedRectangle},
		},
		Edges: []Edge{
			{From: "a", To: "b", Label: "go", LineStyle: LineStyleSolid, ArrowHead: ArrowHeadArrow},
		},
	}
	if len(f.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(f.Nodes))
	}
	if len(f.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(f.Edges))
	}
}

func TestDirectionString(t *testing.T) {
	cases := map[Direction]string{
		DirectionUnknown: "unknown",
		DirectionTB:      "TB",
		DirectionBT:      "BT",
		DirectionLR:      "LR",
		DirectionRL:      "RL",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("Direction(%d).String() = %q, want %q", d, got, want)
		}
	}
}

func TestDirectionStringInvalid(t *testing.T) {
	d := Direction(999)
	if got := d.String(); got == "" {
		t.Error("invalid Direction should return non-empty string")
	}
}

func TestNodeShapeString(t *testing.T) {
	// Verify every defined NodeShape has a non-empty String() and they're unique.
	shapes := []NodeShape{
		NodeShapeUnknown,
		NodeShapeRectangle,
		NodeShapeRoundedRectangle,
		NodeShapeStadium,
		NodeShapeSubroutine,
		NodeShapeCylinder,
		NodeShapeCircle,
		NodeShapeAsymmetric,
		NodeShapeDiamond,
		NodeShapeHexagon,
		NodeShapeParallelogram,
		NodeShapeParallelogramAlt,
		NodeShapeTrapezoid,
		NodeShapeTrapezoidAlt,
		NodeShapeDoubleCircle,
	}
	seen := make(map[string]bool)
	for _, s := range shapes {
		str := s.String()
		if str == "" {
			t.Errorf("NodeShape(%d) has empty String()", s)
		}
		if seen[str] {
			t.Errorf("duplicate String() value: %q", str)
		}
		seen[str] = true
	}
}

func TestLineStyleString(t *testing.T) {
	cases := map[LineStyle]string{
		LineStyleUnknown: "unknown",
		LineStyleSolid:   "solid",
		LineStyleDotted:  "dotted",
		LineStyleThick:   "thick",
	}
	for ls, want := range cases {
		if got := ls.String(); got != want {
			t.Errorf("LineStyle(%d).String() = %q, want %q", ls, got, want)
		}
	}
}

func TestArrowHeadString(t *testing.T) {
	cases := map[ArrowHead]string{
		ArrowHeadUnknown: "unknown",
		ArrowHeadNone:    "none",
		ArrowHeadArrow:   "arrow",
		ArrowHeadOpen:    "open",
		ArrowHeadCross:   "cross",
		ArrowHeadCircle:  "circle",
	}
	for ah, want := range cases {
		if got := ah.String(); got != want {
			t.Errorf("ArrowHead(%d).String() = %q, want %q", ah, got, want)
		}
	}
}

func TestSubgraphConstruction(t *testing.T) {
	sg := Subgraph{
		ID:    "sg1",
		Label: "Group 1",
		Nodes: []Node{{ID: "a"}, {ID: "b"}},
		Children: []Subgraph{
			{ID: "sg2", Label: "Nested"},
		},
	}
	if len(sg.Nodes) != 2 {
		t.Errorf("expected 2 nodes in subgraph, got %d", len(sg.Nodes))
	}
	if len(sg.Children) != 1 {
		t.Errorf("expected 1 child subgraph, got %d", len(sg.Children))
	}
}

func TestFlowchartStylesAndClasses(t *testing.T) {
	f := &FlowchartDiagram{
		Styles: []StyleDef{
			{NodeID: "a", CSS: "fill:#f9f"},
		},
		Classes: map[string]string{
			"highlight": "fill:#ff0,stroke:#000",
		},
	}
	if f.Styles[0].CSS != "fill:#f9f" {
		t.Error("style not preserved")
	}
	if f.Classes["highlight"] == "" {
		t.Error("class not preserved")
	}
}
