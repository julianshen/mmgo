package diagram

import "testing"

func TestFlowchartType(t *testing.T) {
	if (&FlowchartDiagram{}).Type() != Flowchart {
		t.Error("FlowchartDiagram.Type() != Flowchart")
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
	checkStringer(t, map[Direction]string{
		DirectionUnknown: "unknown",
		DirectionTB:      "TB",
		DirectionBT:      "BT",
		DirectionLR:      "LR",
		DirectionRL:      "RL",
	})
}

func TestDirectionStringOutOfRange(t *testing.T) {
	if Direction(127).String() != "unknown" {
		t.Error("positive out-of-range Direction should return 'unknown'")
	}
	// Exercise the i < 0 branch of enumString explicitly.
	if Direction(-1).String() != "unknown" {
		t.Error("negative out-of-range Direction should return 'unknown'")
	}
}

func TestNodeShapeString(t *testing.T) {
	// Pin exact string values — catches name-swap regressions in nodeShapeNames.
	checkStringer(t, map[NodeShape]string{
		NodeShapeUnknown:          "unknown",
		NodeShapeRectangle:        "rectangle",
		NodeShapeRoundedRectangle: "rounded-rectangle",
		NodeShapeStadium:          "stadium",
		NodeShapeSubroutine:       "subroutine",
		NodeShapeCylinder:         "cylinder",
		NodeShapeCircle:           "circle",
		NodeShapeAsymmetric:       "asymmetric",
		NodeShapeDiamond:          "diamond",
		NodeShapeHexagon:          "hexagon",
		NodeShapeParallelogram:    "parallelogram",
		NodeShapeParallelogramAlt: "parallelogram-alt",
		NodeShapeTrapezoid:        "trapezoid",
		NodeShapeTrapezoidAlt:     "trapezoid-alt",
		NodeShapeDoubleCircle:     "double-circle",
	})
}

func TestLineStyleString(t *testing.T) {
	checkStringer(t, map[LineStyle]string{
		LineStyleUnknown: "unknown",
		LineStyleSolid:   "solid",
		LineStyleDotted:  "dotted",
		LineStyleThick:   "thick",
	})
}

func TestArrowHeadString(t *testing.T) {
	checkStringer(t, map[ArrowHead]string{
		ArrowHeadUnknown: "unknown",
		ArrowHeadNone:    "none",
		ArrowHeadArrow:   "arrow",
		ArrowHeadOpen:    "open",
		ArrowHeadCross:   "cross",
		ArrowHeadCircle:  "circle",
	})
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
		Nodes: []Node{
			{ID: "a", Classes: []string{"highlight", "big"}},
		},
		Styles: []StyleDef{
			{NodeID: "a", CSS: "fill:#f9f"},
		},
		Classes: map[string]string{
			"highlight": "fill:#ff0,stroke:#000",
			"big":       "font-size:20px",
		},
	}
	if len(f.Nodes[0].Classes) != 2 {
		t.Errorf("expected 2 classes on node, got %d", len(f.Nodes[0].Classes))
	}
	if f.Styles[0].CSS != "fill:#f9f" {
		t.Error("style not preserved")
	}
	if f.Classes["highlight"] == "" {
		t.Error("class not preserved")
	}
}
