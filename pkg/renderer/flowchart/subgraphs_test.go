package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

func TestRenderSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []diagram.Subgraph{
			{
				ID:    "sg1",
				Label: "My Group",
				Nodes: []diagram.Node{
					{ID: "A", Label: "Node A"},
					{ID: "B", Label: "Node B"},
				},
			},
		},
	}
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 50, Y: 50, Width: 80, Height: 40},
			"B": {X: 200, Y: 50, Width: 80, Height: 40},
		},
		Width: 280, Height: 100,
	}

	elems := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
	if len(elems) == 0 {
		t.Fatal("expected subgraph elements")
	}
	group, ok := elems[0].(*Group)
	if !ok {
		t.Fatalf("expected *Group, got %T", elems[0])
	}
	if group.ID != "sg1" {
		t.Errorf("group ID = %q, want %q", group.ID, "sg1")
	}

	hasRect, hasText := false, false
	for _, child := range group.Children {
		if _, ok := child.(*Rect); ok {
			hasRect = true
		}
		if txt, ok := child.(*Text); ok && txt.Content == "My Group" {
			hasText = true
		}
	}
	if !hasRect {
		t.Error("subgraph should contain a background rect")
	}
	if !hasText {
		t.Error("subgraph should contain title 'My Group'")
	}
}

func TestSubgraphBBox(t *testing.T) {
	nodes := []diagram.Node{{ID: "A"}, {ID: "B"}}
	layoutNodes := map[string]layout.NodeLayout{
		"A": {X: 100, Y: 100, Width: 60, Height: 40},
		"B": {X: 300, Y: 200, Width: 60, Height: 40},
	}
	bb := subgraphBBox(nodes, layoutNodes)
	wantMinX := 100.0 - 60.0/2
	if bb.MinX != wantMinX {
		t.Errorf("MinX = %f, want %f", bb.MinX, wantMinX)
	}
}

func TestNestedSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []diagram.Subgraph{
			{
				ID: "outer", Label: "Outer",
				Nodes: []diagram.Node{{ID: "A"}},
				Children: []diagram.Subgraph{
					{ID: "inner", Label: "Inner", Nodes: []diagram.Node{{ID: "B"}}},
				},
			},
		},
	}
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 100, Y: 100, Width: 60, Height: 40},
			"B": {X: 100, Y: 200, Width: 60, Height: 40},
		},
		Width: 200, Height: 300,
	}
	elems := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
	outer, ok := elems[0].(*Group)
	if !ok {
		t.Fatalf("expected *Group, got %T", elems[0])
	}
	hasInner := false
	for _, child := range outer.Children {
		if g, ok := child.(*Group); ok && g.ID == "inner" {
			hasInner = true
		}
	}
	if !hasInner {
		t.Error("outer should contain inner group")
	}
}
