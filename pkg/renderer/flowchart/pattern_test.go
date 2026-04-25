package flowchart

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
)

// classifyBranch tags a Loop only when a back-edge inside the branch
// targets the branch's source (not just any node).
func TestClassifyBranch_Loop(t *testing.T) {
	src, target := "S", "A"
	inGroup := map[string]bool{"S": true, "A": true, "B": true, "C": true}
	convergence := map[string]bool{}
	backBySource := map[string]string{"C": "S"}
	pattern, backTo, _ := classifyBranch(src, target, inGroup, convergence, backBySource, nil)
	if pattern != PatternLoop {
		t.Errorf("Pattern = %v, want PatternLoop", pattern)
	}
	if backTo != "S" {
		t.Errorf("BackEdgeTo = %q, want S", backTo)
	}
}

// A back-edge inside the branch that points to an unrelated upstream
// node (not the source) must NOT be classified as a Loop — that's
// some other graph's loop, not ours.
func TestClassifyBranch_LoopRequiresBackToSource(t *testing.T) {
	src, target := "S", "A"
	inGroup := map[string]bool{"S": true, "A": true, "B": true}
	convergence := map[string]bool{}
	// B has a back-edge to "Other" — outside this branch entirely.
	backBySource := map[string]string{"B": "Other"}
	pattern, _, _ := classifyBranch(src, target, inGroup, convergence, backBySource, nil)
	if pattern == PatternLoop {
		t.Errorf("back-edge to unrelated node should NOT be PatternLoop")
	}
}

// classifyBranch identifies a condition when the branch's first hop
// IS the convergence node — i.e. multiple sibling branches merge here.
func TestClassifyBranch_Condition(t *testing.T) {
	src, target := "S", "M"
	inGroup := map[string]bool{"S": true}
	convergence := map[string]bool{"M": true}
	pattern, _, mergeID := classifyBranch(src, target, inGroup, convergence, nil, nil)
	if pattern != PatternCondition {
		t.Errorf("Pattern = %v, want PatternCondition", pattern)
	}
	if mergeID != "M" {
		t.Errorf("MergeNodeID = %q, want M", mergeID)
	}
}

// Generic branch (no back-edges, no convergence) gets PatternNone.
func TestClassifyBranch_Generic(t *testing.T) {
	src, target := "S", "A"
	inGroup := map[string]bool{"S": true, "A": true}
	convergence := map[string]bool{}
	pattern, _, _ := classifyBranch(src, target, inGroup, convergence, nil, nil)
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
