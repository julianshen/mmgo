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

func buildGraph(edges ...[2]string) *graph.Graph {
	g := graph.New()
	for _, e := range edges {
		g.SetEdge(e[0], e[1], graph.EdgeAttrs{})
	}
	return g
}

func assertAcyclic(t *testing.T, g *graph.Graph) {
	t.Helper()
	if _, err := g.TopologicalSort(); err != nil {
		t.Errorf("graph not acyclic: %v", err)
	}
}

// nonSelfLoopTopoSort returns a topological sort after temporarily removing
// self-loops, for asserting acyclicity in the presence of self-loops.
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
	if reversed := Run(g); len(reversed) != 0 {
		t.Errorf("expected no reversals, got %d", len(reversed))
	}
}

func TestRunSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	if reversed := Run(g); len(reversed) != 0 {
		t.Errorf("expected no reversals, got %d", len(reversed))
	}
}

// --- Run: table-driven acyclic/cyclic cases ---

func TestRunCases(t *testing.T) {
	tests := []struct {
		name         string
		edges        [][2]string
		wantReversed int // -1 means "don't check count, only verify acyclic output"
	}{
		{
			name:         "linear chain",
			edges:        [][2]string{{"a", "b"}, {"b", "c"}, {"c", "d"}},
			wantReversed: 0,
		},
		{
			name:         "diamond (already acyclic)",
			edges:        [][2]string{{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"}},
			wantReversed: 0,
		},
		{
			name:         "simple 2-cycle",
			edges:        [][2]string{{"a", "b"}, {"b", "a"}},
			wantReversed: 1,
		},
		{
			name:         "triangle",
			edges:        [][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}},
			wantReversed: 1,
		},
		{
			name:         "5-node cycle",
			edges:        [][2]string{{"a", "b"}, {"b", "c"}, {"c", "d"}, {"d", "e"}, {"e", "a"}},
			wantReversed: 1,
		},
		{
			name: "two overlapping cycles",
			edges: [][2]string{
				{"a", "b"}, {"b", "c"}, {"c", "a"},
				{"c", "d"}, {"d", "e"}, {"e", "c"},
			},
			wantReversed: -1,
		},
		{
			name: "disconnected (one cyclic, one acyclic)",
			edges: [][2]string{
				{"a", "b"}, {"b", "c"},
				{"x", "y"}, {"y", "x"},
			},
			wantReversed: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := buildGraph(tc.edges...)
			reversed := Run(g)
			if tc.wantReversed >= 0 && len(reversed) != tc.wantReversed {
				t.Errorf("reversals: got %d, want %d", len(reversed), tc.wantReversed)
			}
			assertAcyclic(t, g)
		})
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
	if len(g.EdgesBetween("b", "a")) != 0 {
		t.Error("back edge b->a should be reversed")
	}
	if len(g.EdgesBetween("a", "b")) != 2 {
		t.Errorf("expected 2 a->b edges after reversal, got %d", len(g.EdgesBetween("a", "b")))
	}
	if _, err := nonSelfLoopTopoSort(g); err != nil {
		t.Errorf("non-self-loop graph should be acyclic: %v", err)
	}
}

// --- Run: determinism ---

func TestRunDeterministic(t *testing.T) {
	build := func() *graph.Graph {
		return buildGraph([2]string{"a", "b"}, [2]string{"b", "c"}, [2]string{"c", "a"})
	}
	g1, g2 := build(), build()
	Run(g1)
	Run(g2)

	e1, e2 := collectEdges(g1), collectEdges(g2)
	if !slices.Equal(e1, e2) {
		t.Errorf("determinism broken\nrun1: %v\nrun2: %v", e1, e2)
	}
}

// --- Undo ---

func TestUndoRestoresDirections(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"}, [2]string{"c", "a"})
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
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	Undo(g, nil)
	if g.EdgeCount() != 2 {
		t.Errorf("Undo with nil should not modify graph, got %d edges", g.EdgeCount())
	}
}

func TestRunThenUndoLinearChainIsIdentity(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	orig := collectEdges(g)

	reversed := Run(g)
	Undo(g, reversed)

	after := collectEdges(g)
	if !slices.Equal(orig, after) {
		t.Errorf("Run+Undo should be identity for acyclic graphs\norig:  %v\nafter: %v", orig, after)
	}
}

func TestUndoPanicsOnMissingEdge(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Undo should panic when edge no longer exists in graph")
		}
	}()

	g := graph.New()
	Undo(g, []graph.EdgeID{{From: "a", To: "b", ID: 999}})
}

// --- Multi-edges ---

func TestRunPreservesMultiEdges(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{Label: "1"})
	g.SetEdge("a", "b", graph.EdgeAttrs{Label: "2"})
	g.SetEdge("b", "a", graph.EdgeAttrs{Label: "3"})

	Run(g)

	if g.EdgeCount() != 3 {
		t.Errorf("expected 3 edges, got %d", g.EdgeCount())
	}
	if len(g.EdgesBetween("a", "b")) != 3 {
		t.Errorf("expected 3 a->b edges, got %d", len(g.EdgesBetween("a", "b")))
	}
}
