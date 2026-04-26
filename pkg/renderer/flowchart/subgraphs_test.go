package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestRenderSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{
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

	elems, _ := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
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
	bb, ok := subgraphBBox(nodes, layoutNodes)
	if !ok {
		t.Fatal("subgraphBBox ok=false")
	}
	wantMinX := 100.0 - 60.0/2
	if bb.MinX != wantMinX {
		t.Errorf("MinX = %f, want %f", bb.MinX, wantMinX)
	}
	wantMaxX := 300.0 + 60.0/2
	if bb.MaxX != wantMaxX {
		t.Errorf("MaxX = %f, want %f", bb.MaxX, wantMaxX)
	}
	wantMinY := 100.0 - 40.0/2
	if bb.MinY != wantMinY {
		t.Errorf("MinY = %f, want %f", bb.MinY, wantMinY)
	}
	wantMaxY := 200.0 + 40.0/2
	if bb.MaxY != wantMaxY {
		t.Errorf("MaxY = %f, want %f", bb.MaxY, wantMaxY)
	}
}

func TestSubgraphBBoxEmpty(t *testing.T) {
	_, ok := subgraphBBox(nil, map[string]layout.NodeLayout{})
	if ok {
		t.Error("expected ok=false for empty input")
	}
	_, ok = subgraphBBox([]diagram.Node{{ID: "missing"}}, map[string]layout.NodeLayout{})
	if ok {
		t.Error("expected ok=false when all nodes missing from layout")
	}
}

func TestRenderEmptySubgraphProducesNoNaN(t *testing.T) {
	// A subgraph whose nodes are all missing from the layout (or that
	// contains no nodes at all) must NOT emit a `<rect>` with NaN/Inf
	// coordinates. Regression for the ±Inf bbox bug.
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{
			{ID: "empty", Label: "Empty", Nodes: []diagram.Node{{ID: "ghost"}}},
		},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{}, Width: 100, Height: 100}
	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	if strings.Contains(string(out), "NaN") || strings.Contains(string(out), "Inf") {
		t.Errorf("output contains NaN or Inf:\n%s", out)
	}
}

func TestRenderSubgraphContents(t *testing.T) {
	// A subgraph-contained node MUST appear in the rendered SVG. Per
	// the AST contract, nodes inside a subgraph are stored only in
	// Subgraph.Nodes — top-level renderNodes won't see them.
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{
			{
				ID: "sg1", Label: "Group",
				Nodes: []diagram.Node{
					{ID: "A", Label: "Inside", Shape: diagram.NodeShapeRectangle},
				},
				Edges: []diagram.Edge{
					{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow, Label: "scoped"},
				},
			},
		},
		Nodes: []diagram.Node{
			{ID: "B", Label: "Outside", Shape: diagram.NodeShapeRectangle},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Inside", Width: 80, Height: 40})
	g.SetNode("B", graph.NodeAttrs{Label: "Outside", Width: 80, Height: 40})
	g.SetEdge("A", "B", graph.EdgeAttrs{Label: "scoped"})
	l := layout.Layout(g, layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Inside<") {
		t.Errorf("subgraph-contained node label 'Inside' missing from SVG:\n%s", raw)
	}
	if !strings.Contains(raw, ">Outside<") {
		t.Errorf("top-level node 'Outside' missing")
	}
	if !strings.Contains(raw, ">scoped<") {
		t.Errorf("subgraph-scoped edge label 'scoped' missing")
	}
}

func TestNestedSubgraph(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{
			{
				ID: "outer", Label: "Outer",
				Nodes: []diagram.Node{{ID: "A"}},
				Children: []*diagram.Subgraph{
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
	elems, _ := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
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

func TestNestedSubgraphWidthProgression(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{
			{
				ID: "outer", Label: "Outer",
				Nodes: []diagram.Node{{ID: "A"}},
				Children: []*diagram.Subgraph{
					{
						ID: "mid", Label: "Mid",
						Nodes: []diagram.Node{{ID: "B"}},
						Children: []*diagram.Subgraph{
							{ID: "inner", Label: "Inner", Nodes: []diagram.Node{{ID: "C"}}},
						},
					},
				},
			},
		},
	}
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 100, Y: 100, Width: 60, Height: 40},
			"B": {X: 100, Y: 200, Width: 60, Height: 40},
			"C": {X: 100, Y: 300, Width: 60, Height: 40},
		},
		Width: 200, Height: 400,
	}

	elems, bb := renderSubgraphs(d, l, 10, DefaultTheme(), 16)
	if bb.W == 0 {
		t.Fatal("expected non-zero subgraph bounding box")
	}

	outerGroup := elems[0].(*Group)
	outerRect := findRectWithID(outerGroup, "outer")
	midRect := findRectWithID(outerGroup, "mid")
	innerRect := findRectWithID(outerGroup, "inner")

	if outerRect == nil || midRect == nil || innerRect == nil {
		t.Fatal("missing subgraph rects")
	}

	if outerRect.Width <= midRect.Width {
		t.Errorf("outer width %.2f should be > mid width %.2f", outerRect.Width, midRect.Width)
	}
	if midRect.Width <= innerRect.Width {
		t.Errorf("mid width %.2f should be > inner width %.2f", midRect.Width, innerRect.Width)
	}
}

func findRectWithID(g *Group, id string) *Rect {
	if g.ID == id {
		for _, c := range g.Children {
			if r, ok := c.(*Rect); ok {
				return r
			}
		}
	}
	for _, c := range g.Children {
		if child, ok := c.(*Group); ok {
			if r := findRectWithID(child, id); r != nil {
				return r
			}
		}
	}
	return nil
}
