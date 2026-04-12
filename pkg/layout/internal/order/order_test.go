package order

import (
	"slices"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// --- Helpers ---

func buildGraph(edges ...[2]string) *graph.Graph {
	g := graph.New()
	for _, e := range edges {
		g.SetEdge(e[0], e[1], graph.EdgeAttrs{})
	}
	return g
}

// ranksFromMap creates a rank map for the given node→rank pairs.
func ranksFromMap(pairs map[string]int) map[string]int {
	return pairs
}

// collectNodes returns the sorted set of all nodes across all ranks in order.
func collectNodes(order Order) []string {
	var all []string
	for _, nodes := range order {
		all = append(all, nodes...)
	}
	slices.Sort(all)
	return all
}

// --- Trivial cases ---

func TestRunEmptyGraph(t *testing.T) {
	g := graph.New()
	order := Run(g, map[string]int{})
	if len(order) != 0 {
		t.Errorf("expected empty order, got %v", order)
	}
}

func TestRunSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	order := Run(g, map[string]int{"a": 0})
	if !slices.Equal(order[0], []string{"a"}) {
		t.Errorf("rank 0 = %v, want [a]", order[0])
	}
}

func TestRunLinearChain(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "d"},
	)
	ranks := map[string]int{"a": 0, "b": 1, "c": 2, "d": 3}
	order := Run(g, ranks)

	// Each rank has exactly one node; order is trivially correct.
	for r, want := range map[int]string{0: "a", 1: "b", 2: "c", 3: "d"} {
		if !slices.Equal(order[r], []string{want}) {
			t.Errorf("rank %d = %v, want [%s]", r, order[r], want)
		}
	}
}

// --- Invariants ---

func TestRunPreservesAllNodes(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	ranks := map[string]int{"a": 0, "b": 1, "c": 1, "d": 2}
	order := Run(g, ranks)

	all := collectNodes(order)
	if !slices.Equal(all, []string{"a", "b", "c", "d"}) {
		t.Errorf("missing nodes: got %v", all)
	}
}

func TestRunPreservesRankAssignment(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"a", "c"}, [2]string{"b", "d"})
	ranks := map[string]int{"a": 0, "b": 1, "c": 1, "d": 2}
	order := Run(g, ranks)

	for r, nodes := range order {
		for _, n := range nodes {
			if ranks[n] != r {
				t.Errorf("node %s placed in rank %d but its assigned rank is %d",
					n, r, ranks[n])
			}
		}
	}
}

// --- Crossing reduction ---

func TestRunNoCrossingsInAcyclicFlow(t *testing.T) {
	// Diamond: naturally has no crossings.
	//   a
	//  / \
	// b   c
	//  \ /
	//   d
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	ranks := map[string]int{"a": 0, "b": 1, "c": 1, "d": 2}
	order := Run(g, ranks)
	if n := countCrossings(g, order); n != 0 {
		t.Errorf("diamond should have 0 crossings, got %d", n)
	}
}

func TestRunFixesSimpleCrossing(t *testing.T) {
	// Two layers, two nodes each. Edges cross if initial order is
	// alphabetical:
	//   a   c       a   c
	//    \ /    →    \ /
	//     X          |  \
	//    / \         |   \
	//   b   d   →   d   b
	//
	// With cross: a→d, c→b.
	// Initial alphabetical order: rank 0 = [a, c], rank 1 = [b, d]
	// Crossings: a→d (positions 0,1) vs c→b (positions 1,0) → cross.
	// Correct: rank 1 should be [d, b] (barycenter of d = 0 from a, of b = 1 from c).
	g := buildGraph([2]string{"a", "d"}, [2]string{"c", "b"})
	ranks := map[string]int{"a": 0, "c": 0, "b": 1, "d": 1}
	order := Run(g, ranks)

	if n := countCrossings(g, order); n != 0 {
		t.Errorf("simple crossing should be fixed, got %d crossings", n)
	}
}

func TestRunReducesCrossingsOnLargerGraph(t *testing.T) {
	// 3 nodes in each of 2 ranks, with cross-heavy edges.
	// Rank 0: a, b, c
	// Rank 1: x, y, z
	// Edges: a→z, b→y, c→x (all cross in alphabetical order)
	// Optimal: reverse rank 1 to [z, y, x]
	g := buildGraph(
		[2]string{"a", "z"},
		[2]string{"b", "y"},
		[2]string{"c", "x"},
	)
	ranks := map[string]int{
		"a": 0, "b": 0, "c": 0,
		"x": 1, "y": 1, "z": 1,
	}
	order := Run(g, ranks)

	// After ordering, crossings should be 0 (edges are parallel).
	if n := countCrossings(g, order); n != 0 {
		t.Errorf("expected 0 crossings after ordering, got %d", n)
	}
}

func TestRunMultiLayerReducesCrossings(t *testing.T) {
	// 4-layer graph with several crossings in initial ordering.
	g := buildGraph(
		[2]string{"a1", "b2"},
		[2]string{"a2", "b1"},
		[2]string{"b1", "c2"},
		[2]string{"b2", "c1"},
	)
	ranks := map[string]int{
		"a1": 0, "a2": 0,
		"b1": 1, "b2": 1,
		"c1": 2, "c2": 2,
	}

	// Count initial crossings (alphabetical ordering).
	initial := initOrder(g, ranks)
	initialCross := countCrossings(g, initial)

	order := Run(g, ranks)
	finalCross := countCrossings(g, order)

	if finalCross > initialCross {
		t.Errorf("Run should not increase crossings: %d → %d",
			initialCross, finalCross)
	}
}

// --- Determinism ---

func TestRunDeterministic(t *testing.T) {
	build := func() *graph.Graph {
		return buildGraph(
			[2]string{"a", "d"},
			[2]string{"b", "e"},
			[2]string{"c", "f"},
			[2]string{"a", "e"},
		)
	}
	ranks := map[string]int{"a": 0, "b": 0, "c": 0, "d": 1, "e": 1, "f": 1}

	o1 := Run(build(), ranks)
	o2 := Run(build(), ranks)

	for r := range o1 {
		if !slices.Equal(o1[r], o2[r]) {
			t.Errorf("determinism broken at rank %d: %v vs %v", r, o1[r], o2[r])
		}
	}
}

// --- Cross-counting ---

func TestCountCrossingsNoEdges(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	g.SetNode("b", graph.NodeAttrs{})
	order := Order{0: {"a"}, 1: {"b"}}
	if n := countCrossings(g, order); n != 0 {
		t.Errorf("expected 0 crossings, got %d", n)
	}
}

func TestCountCrossingsOnePair(t *testing.T) {
	// a→d, b→c with order {0: [a, b], 1: [c, d]} → one crossing
	g := buildGraph([2]string{"a", "d"}, [2]string{"b", "c"})
	order := Order{0: {"a", "b"}, 1: {"c", "d"}}
	if n := countCrossings(g, order); n != 1 {
		t.Errorf("expected 1 crossing, got %d", n)
	}
}

func TestCountCrossingsMultiple(t *testing.T) {
	// Two parallel edges that don't cross, plus one cross.
	// a→c, a→d, b→c
	// Order {0: [a, b], 1: [c, d]}
	// Crossings:
	//   a→c vs a→d: same source, never crosses
	//   a→c vs b→c: same target, never crosses
	//   a→d vs b→c: a pos 0 < b pos 1, d pos 1 > c pos 0, CROSS
	g := buildGraph(
		[2]string{"a", "c"},
		[2]string{"a", "d"},
		[2]string{"b", "c"},
	)
	order := Order{0: {"a", "b"}, 1: {"c", "d"}}
	if n := countCrossings(g, order); n != 1 {
		t.Errorf("expected 1 crossing, got %d", n)
	}
}

// --- Disconnected ---

func TestRunDisconnectedComponents(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"c", "d"},
	)
	ranks := map[string]int{"a": 0, "b": 1, "c": 0, "d": 1}
	order := Run(g, ranks)

	// Both components in the same layer.
	if len(order[0]) != 2 || len(order[1]) != 2 {
		t.Errorf("expected 2 nodes per layer: %v", order)
	}
	if countCrossings(g, order) != 0 {
		t.Error("disconnected components should have no crossings")
	}
}

// --- Isolated node without adjacent-layer neighbors ---

func TestRunNodeWithoutNeighbors(t *testing.T) {
	// Node "x" at rank 1 has no incoming or outgoing edges.
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	g.SetNode("x", graph.NodeAttrs{})
	ranks := map[string]int{"a": 0, "b": 1, "x": 1}

	order := Run(g, ranks)

	// Both b and x should appear in rank 1.
	if len(order[1]) != 2 {
		t.Errorf("rank 1 should have 2 nodes, got %v", order[1])
	}
}
