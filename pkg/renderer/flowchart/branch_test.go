package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// nodeLayoutAt is a tiny fixture builder that places a fixed-size node
// at the given center for convergence/subgraph tests that don't need
// real layout output.
func nodeLayoutAt(cx, cy float64) layout.NodeLayout {
	return layout.NodeLayout{X: cx, Y: cy, Width: 60, Height: 40}
}

func TestDetectBranches_SimpleFork(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "D"}, {ID: "A"}, {ID: "B"}, {ID: "C"}},
		Edges: []diagram.Edge{
			{From: "D", To: "A"},
			{From: "D", To: "B"},
			{From: "D", To: "C"},
		},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"D": nodeLayoutAt(100, 50), "A": nodeLayoutAt(50, 150),
		"B": nodeLayoutAt(100, 150), "C": nodeLayoutAt(150, 150),
	}}
	groups := DetectBranches(d, l)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups for a 3-outlet branch, got %d", len(groups))
	}
	// Each group has the source "D" and exactly one leaf.
	leaves := map[string]bool{}
	for _, g := range groups {
		if g.SourceNodeID != "D" {
			t.Errorf("SourceNodeID = %q, want D", g.SourceNodeID)
		}
		if len(g.NodeIDs) != 1 {
			t.Errorf("expected 1 leaf per branch, got %d", len(g.NodeIDs))
		}
		for _, n := range g.NodeIDs {
			leaves[n] = true
		}
	}
	for _, want := range []string{"A", "B", "C"} {
		if !leaves[want] {
			t.Errorf("leaf %q missing from branches", want)
		}
	}
}

// Convergence point Z is reachable from 2 branches → it must appear
// in NO group (regression guard: a naïve implementation would put Z
// into every branch).
func TestDetectBranches_ConvergencePointExcluded(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "D", To: "A"}, {From: "D", To: "B"}, {From: "D", To: "C"},
			{From: "A", To: "Z"}, {From: "B", To: "Z"},
		},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"D": nodeLayoutAt(0, 0), "A": nodeLayoutAt(0, 0),
		"B": nodeLayoutAt(0, 0), "C": nodeLayoutAt(0, 0), "Z": nodeLayoutAt(0, 0),
	}}
	groups := DetectBranches(d, l)
	for _, g := range groups {
		for _, n := range g.NodeIDs {
			if n == "Z" {
				t.Errorf("group %+v should NOT contain convergence point Z", g)
			}
		}
	}
}

// A branch whose members all sit in one user-defined subgraph must be
// suppressed: the subgraph's own styling provides the visual grouping.
func TestDetectBranches_SubgraphSuppression(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Subgraphs: []*diagram.Subgraph{{
			ID: "sg",
			Nodes: []diagram.Node{
				{ID: "D"}, {ID: "A"}, {ID: "B"}, {ID: "C"},
			},
			Edges: []diagram.Edge{
				{From: "D", To: "A"}, {From: "D", To: "B"}, {From: "D", To: "C"},
			},
		}},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"D": nodeLayoutAt(0, 0), "A": nodeLayoutAt(0, 0),
		"B": nodeLayoutAt(0, 0), "C": nodeLayoutAt(0, 0),
	}}
	groups := DetectBranches(d, l)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups (all members share one subgraph), got %d", len(groups))
	}
}

// Nested branch: A forks into B,C,D and B itself forks into E,F,G. B
// and its subtree must be excluded from A's groups, and B must start
// its own groups.
func TestDetectBranches_Nested(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B"}, {From: "A", To: "C"}, {From: "A", To: "D"},
			{From: "B", To: "E"}, {From: "B", To: "F"}, {From: "B", To: "G"},
		},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"A": nodeLayoutAt(0, 0), "B": nodeLayoutAt(0, 0), "C": nodeLayoutAt(0, 0),
		"D": nodeLayoutAt(0, 0), "E": nodeLayoutAt(0, 0), "F": nodeLayoutAt(0, 0),
		"G": nodeLayoutAt(0, 0),
	}}
	groups := DetectBranches(d, l)
	sources := map[string]int{}
	for _, g := range groups {
		sources[g.SourceNodeID]++
		for _, n := range g.NodeIDs {
			if g.SourceNodeID == "A" && (n == "E" || n == "F" || n == "G") {
				t.Errorf("A's group leaked into B's subtree via %q", n)
			}
			if g.SourceNodeID == "A" && n == "B" {
				t.Errorf("A's group must not include downstream branch node B")
			}
		}
	}
	if sources["A"] == 0 {
		t.Error("A (outer branch) produced no groups")
	}
	if sources["B"] == 0 {
		t.Error("B (nested branch) produced no groups")
	}
}

// renderBranchRegions produces one rect per group with the palette fill
// in its style string.
func TestRenderBranchRegions_EmitsRectPerGroup(t *testing.T) {
	groups := []BranchGroup{
		{SourceNodeID: "D", NodeIDs: []string{"A"}, ColorIndex: 0},
		{SourceNodeID: "D", NodeIDs: []string{"B"}, ColorIndex: 1},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"A": nodeLayoutAt(10, 10), "B": nodeLayoutAt(100, 10),
	}}
	elems := renderBranchRegions(groups, l, 0)
	if len(elems) != 2 {
		t.Fatalf("expected 2 region rects, got %d", len(elems))
	}
	// Spot-check: each rect carries a palette fill from branchColorPalette.
	for i, elem := range elems {
		r, ok := elem.(*Rect)
		if !ok {
			t.Fatalf("elem %d is %T, want *Rect", i, elem)
		}
		want := branchColorPalette[i%len(branchColorPalette)].Fill
		if !strings.Contains(r.Style, "fill:"+want) {
			t.Errorf("rect %d style %q missing palette fill %s", i, r.Style, want)
		}
	}
}

// Empty / nil inputs produce nil — no rects, no panics.
func TestRenderBranchRegions_EmptyInputs(t *testing.T) {
	if elems := renderBranchRegions(nil, &layout.Result{}, 0); elems != nil {
		t.Errorf("nil groups should produce nil output, got %v", elems)
	}
	if elems := renderBranchRegions([]BranchGroup{{NodeIDs: []string{"absent"}}}, &layout.Result{Nodes: map[string]layout.NodeLayout{}}, 0); len(elems) != 0 {
		t.Errorf("missing layout nodes should skip the group, got %d rects", len(elems))
	}
}
