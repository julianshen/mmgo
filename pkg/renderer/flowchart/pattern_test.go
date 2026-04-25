package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// classifyBranch identifies a loop when any back-edge originates
// inside the branch group (member-or-source).
func TestClassifyBranch_Loop(t *testing.T) {
	target := "A"
	inGroup := map[string]bool{"S": true, "A": true, "B": true, "C": true}
	convergence := map[string]bool{}
	backBySource := map[string]string{"C": "S"}
	pattern, backTo, _ := classifyBranch(target, inGroup, convergence, backBySource, nil)
	if pattern != PatternLoop {
		t.Errorf("Pattern = %v, want PatternLoop", pattern)
	}
	if backTo != "S" {
		t.Errorf("BackEdgeTo = %q, want S", backTo)
	}
}

// classifyBranch identifies a condition when the branch's first hop
// IS the convergence node — i.e. multiple sibling branches merge here.
func TestClassifyBranch_Condition(t *testing.T) {
	target := "M"
	inGroup := map[string]bool{"S": true}
	convergence := map[string]bool{"M": true}
	pattern, _, mergeID := classifyBranch(target, inGroup, convergence, nil, nil)
	if pattern != PatternCondition {
		t.Errorf("Pattern = %v, want PatternCondition", pattern)
	}
	if mergeID != "M" {
		t.Errorf("MergeNodeID = %q, want M", mergeID)
	}
}

// Generic branch (no back-edges, no convergence) gets PatternNone.
func TestClassifyBranch_Generic(t *testing.T) {
	target := "A"
	inGroup := map[string]bool{"S": true, "A": true}
	convergence := map[string]bool{}
	pattern, _, _ := classifyBranch(target, inGroup, convergence, nil, nil)
	if pattern != PatternNone {
		t.Errorf("Pattern = %v, want PatternNone", pattern)
	}
}

// paletteFor routes loop branches to the warm palette and condition
// branches to the cool palette.
func TestPaletteFor(t *testing.T) {
	loop := paletteFor(BranchGroup{Pattern: PatternLoop})
	cond := paletteFor(BranchGroup{Pattern: PatternCondition})
	gen := paletteFor(BranchGroup{Pattern: PatternNone})
	if loop.Fill != loopPalette[0].Fill {
		t.Errorf("loop palette mismatch: got %s, want %s", loop.Fill, loopPalette[0].Fill)
	}
	if cond.Fill != conditionPalette[0].Fill {
		t.Errorf("condition palette mismatch: got %s, want %s", cond.Fill, conditionPalette[0].Fill)
	}
	if gen.Fill != branchColorPalette[0].Fill {
		t.Errorf("generic palette mismatch: got %s, want %s", gen.Fill, branchColorPalette[0].Fill)
	}
}

// DetectBranches populates EdgeFromTo with every (From, To) pair fully
// inside each branch group.
func TestDetectBranches_PopulatesEdgeFromTo(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "D", To: "A"}, {From: "D", To: "B"}, {From: "D", To: "C"},
			{From: "A", To: "L"},
		},
	}
	l := &layout.Result{Nodes: map[string]layout.NodeLayout{
		"D": {X: 0, Y: 0, Width: 60, Height: 40},
		"A": {X: 0, Y: 0, Width: 60, Height: 40},
		"B": {X: 0, Y: 0, Width: 60, Height: 40},
		"C": {X: 0, Y: 0, Width: 60, Height: 40},
		"L": {X: 0, Y: 0, Width: 60, Height: 40},
	}}
	groups := DetectBranches(d, l)
	// Find A's group (it should contain edge A→L).
	var aGroup *BranchGroup
	for i := range groups {
		for _, n := range groups[i].NodeIDs {
			if n == "A" {
				aGroup = &groups[i]
				break
			}
		}
	}
	if aGroup == nil {
		t.Fatal("no group containing A found")
	}
	found := false
	for _, ft := range aGroup.EdgeFromTo {
		if ft[0] == "A" && ft[1] == "L" {
			found = true
		}
	}
	if !found {
		t.Errorf("A's group EdgeFromTo missing A→L: got %+v", aGroup.EdgeFromTo)
	}
}
