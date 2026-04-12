package acyclic

import (
	"slices"
	"sort"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// --- Helpers ---

// edgeRecord is a direction+label tuple for comparing edge sets across
// reversal operations (EdgeID.ID values change through reversal).
type edgeRecord struct {
	From  string
	To    string
	Label string
}

func (e edgeRecord) Key() string { return e.From + "->" + e.To + ":" + e.Label }

func collectEdges(g *graph.Graph) []edgeRecord {
	var out []edgeRecord
	for _, eid := range g.Edges() {
		attrs, _ := g.EdgeAttrs(eid)
		out = append(out, edgeRecord{From: eid.From, To: eid.To, Label: attrs.Label})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

func buildLinear(edges ...[2]string) *graph.Graph {
	g := graph.New()
	for _, e := range edges {
		g.SetEdge(e[0], e[1], graph.EdgeAttrs{})
	}
	return g
}

// nonSelfLoopTopoSort returns a topological sort after temporarily removing
// self-loops. Used to verify that Run produces an acyclic graph even when
// self-loops are present (since self-loops always defeat TopologicalSort).
func nonSelfLoopTopoSort(g *graph.Graph) ([]string, error) {
	h := g.Copy()
	for _, eid := range h.Edges() {
		if eid.From == eid.To {
			h.RemoveEdge(eid)
		}
	}
	return h.TopologicalSort()
}

// --- Run: trivial cases ---

func TestRunEmptyGraph(t *testing.T) {
	g := graph.New()
	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("expected no reversals, got %d", len(reversed))
	}
}

func TestRunSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("expected no reversals, got %d", len(reversed))
	}
}

func TestRunLinearChain(t *testing.T) {
	g := buildLinear([2]string{"a", "b"}, [2]string{"b", "c"}, [2]string{"c", "d"})
	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("expected no reversals for acyclic graph, got %d", len(reversed))
	}
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph should be acyclic: %v", err)
	}
}

func TestRunAlreadyAcyclic(t *testing.T) {
	// Diamond: a -> b -> d; a -> c -> d
	g := buildLinear(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("already-acyclic graph should not be modified, got %d reversals", len(reversed))
	}
}

// --- Run: cycle cases ---

func TestRunSimpleCycle(t *testing.T) {
	// a -> b, b -> a
	g := buildLinear([2]string{"a", "b"}, [2]string{"b", "a"})
	reversed := Run(g)
	if len(reversed) != 1 {
		t.Errorf("expected 1 reversed edge, got %d", len(reversed))
	}
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic after Run: %v", err)
	}
}

func TestRunTriangleCycle(t *testing.T) {
	// a -> b -> c -> a
	g := buildLinear([2]string{"a", "b"}, [2]string{"b", "c"}, [2]string{"c", "a"})
	reversed := Run(g)
	if len(reversed) != 1 {
		t.Errorf("expected 1 reversed edge, got %d", len(reversed))
	}
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic after Run: %v", err)
	}
}

func TestRunMultipleCycles(t *testing.T) {
	// Two overlapping cycles:
	//   a -> b -> c -> a   (cycle 1)
	//   c -> d -> e -> c   (cycle 2)
	g := buildLinear(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "a"},
		[2]string{"c", "d"},
		[2]string{"d", "e"},
		[2]string{"e", "c"},
	)
	Run(g)
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic after Run: %v", err)
	}
}

func TestRunLongCycle(t *testing.T) {
	// a -> b -> c -> d -> e -> a
	g := buildLinear(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "d"},
		[2]string{"d", "e"},
		[2]string{"e", "a"},
	)
	reversed := Run(g)
	if len(reversed) != 1 {
		t.Errorf("expected 1 reversed edge for a 5-node cycle, got %d", len(reversed))
	}
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic after Run: %v", err)
	}
}

// --- Run: self-loops ---

func TestRunSelfLoopPreservedAlone(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("self-loop should not be reversed, got %d reversals", len(reversed))
	}
	if !g.HasEdge("a", "a") {
		t.Error("self-loop should still exist")
	}
}

func TestRunSelfLoopWithOtherEdges(t *testing.T) {
	// a has a self-loop plus two real edges; no cycles among real edges.
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	g.SetEdge("b", "c", graph.EdgeAttrs{})

	reversed := Run(g)
	if len(reversed) != 0 {
		t.Errorf("no real cycles, expected 0 reversals, got %d", len(reversed))
	}
	if !g.HasEdge("a", "a") {
		t.Error("self-loop should be preserved")
	}
	if !g.HasEdge("a", "b") || !g.HasEdge("b", "c") {
		t.Error("real edges should be preserved")
	}
}

func TestRunSelfLoopWithBackEdge(t *testing.T) {
	// a has a self-loop; there's also a real back edge b -> a
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	g.SetEdge("b", "a", graph.EdgeAttrs{})

	Run(g)

	if !g.HasEdge("a", "a") {
		t.Error("self-loop should be preserved")
	}
	// After reversal, we should have no b->a (it's been reversed) and
	// two a->b edges (the original plus the reversed one).
	if len(g.EdgesBetween("b", "a")) != 0 {
		t.Error("back edge b->a should be reversed")
	}
	if len(g.EdgesBetween("a", "b")) != 2 {
		t.Errorf("expected 2 a->b edges after reversal, got %d", len(g.EdgesBetween("a", "b")))
	}
	// Non-self-loop part should be acyclic.
	if _, err := nonSelfLoopTopoSort(g); err != nil {
		t.Errorf("non-self-loop graph should be acyclic: %v", err)
	}
}

// --- Run: determinism ---

func TestRunDeterministic(t *testing.T) {
	// Two identical triangles should produce the same result.
	build := func() *graph.Graph {
		return buildLinear(
			[2]string{"a", "b"},
			[2]string{"b", "c"},
			[2]string{"c", "a"},
		)
	}
	g1 := build()
	g2 := build()

	Run(g1)
	Run(g2)

	e1 := collectEdges(g1)
	e2 := collectEdges(g2)
	if len(e1) != len(e2) {
		t.Fatalf("edge count mismatch: %d vs %d", len(e1), len(e2))
	}
	for i := range e1 {
		if e1[i] != e2[i] {
			t.Errorf("determinism broken at index %d: %v vs %v", i, e1[i], e2[i])
		}
	}
}

// --- Run: disconnected components ---

func TestRunDisconnectedComponents(t *testing.T) {
	// Two separate components: one acyclic, one with a cycle.
	g := buildLinear(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"x", "y"},
		[2]string{"y", "x"}, // cycle in component 2
	)
	Run(g)
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic after Run: %v", err)
	}
}

// --- Undo ---

func TestUndoRestoresDirections(t *testing.T) {
	g := buildLinear(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "a"},
	)
	// Add distinguishing labels so we can verify edge identity.
	for _, eid := range g.Edges() {
		g.SetEdgeAttrs(eid, graph.EdgeAttrs{Label: eid.From + eid.To})
	}

	orig := collectEdges(g)

	reversed := Run(g)
	if len(reversed) == 0 {
		t.Fatal("expected at least one reversal on a cycle")
	}
	Undo(g, reversed)

	after := collectEdges(g)
	if !slices.Equal(orig, after) {
		t.Errorf("Undo should restore original edges\norig:  %v\nafter: %v", orig, after)
	}
}

func TestUndoEmpty(t *testing.T) {
	g := buildLinear([2]string{"a", "b"}, [2]string{"b", "c"})
	Undo(g, nil) // no reversals to undo — should be no-op
	if g.EdgeCount() != 2 {
		t.Errorf("Undo with nil should not modify graph, got %d edges", g.EdgeCount())
	}
}

func TestRunThenUndoLinearChainIsIdentity(t *testing.T) {
	g := buildLinear([2]string{"a", "b"}, [2]string{"b", "c"})
	orig := collectEdges(g)

	reversed := Run(g)
	Undo(g, reversed)

	after := collectEdges(g)
	if !slices.Equal(orig, after) {
		t.Errorf("Run+Undo should be identity for acyclic graphs\norig:  %v\nafter: %v", orig, after)
	}
}

// --- Multi-edges ---

func TestRunPreservesMultiEdges(t *testing.T) {
	// Two parallel edges a -> b, plus a back edge b -> a.
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{Label: "1"})
	g.SetEdge("a", "b", graph.EdgeAttrs{Label: "2"})
	g.SetEdge("b", "a", graph.EdgeAttrs{Label: "3"})

	Run(g)

	// After reversal we should still have 3 edges total.
	if g.EdgeCount() != 3 {
		t.Errorf("expected 3 edges, got %d", g.EdgeCount())
	}
	// Back edge should be reversed, giving three a->b edges.
	if len(g.EdgesBetween("a", "b")) != 3 {
		t.Errorf("expected 3 a->b edges, got %d", len(g.EdgesBetween("a", "b")))
	}
}
