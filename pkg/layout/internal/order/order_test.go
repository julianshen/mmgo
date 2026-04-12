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

// collectNodes returns the sorted set of all nodes across all ranks in order.
func collectNodes(order Order) []string {
	var all []string
	for _, nodes := range order {
		all = append(all, nodes...)
	}
	slices.Sort(all)
	return all
}

// testCountCrossings is a convenience wrapper that derives ranks from the
// Order itself and rebuilds the precomputed layer-edge map. Tests don't
// run in a hot loop so the per-call overhead is acceptable.
func testCountCrossings(g *graph.Graph, order Order) int {
	ranks := make(map[string]int)
	for r, nodes := range order {
		for _, n := range nodes {
			ranks[n] = r
		}
	}
	ranksAsc := sortedRanks(order)
	layerEdges := buildLayerEdges(g, ranks, ranksAsc)
	var scratch []edgePos
	return countCrossings(order, ranksAsc, layerEdges, &scratch)
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
	if n := testCountCrossings(g, order); n != 0 {
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

	if n := testCountCrossings(g, order); n != 0 {
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
	if n := testCountCrossings(g, order); n != 0 {
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
	initialCross := testCountCrossings(g, initial)

	order := Run(g, ranks)
	finalCross := testCountCrossings(g, order)

	if finalCross > initialCross {
		t.Errorf("Run should not increase crossings: %d → %d",
			initialCross, finalCross)
	}
}

// --- Determinism ---

func TestRunDeterministic(t *testing.T) {
	// Run many times to stress Go's randomized map iteration (g.Nodes
	// and g.Edges range over internal maps). Any nondeterminism in
	// initOrder or sortByBarycenter tie-breaking will surface as
	// varying output across the loop iterations.
	build := func() *graph.Graph {
		return buildGraph(
			[2]string{"a", "d"},
			[2]string{"b", "e"},
			[2]string{"c", "f"},
			[2]string{"a", "e"},
		)
	}
	ranks := map[string]int{"a": 0, "b": 0, "c": 0, "d": 1, "e": 1, "f": 1}

	reference := Run(build(), ranks)
	for iter := 0; iter < 50; iter++ {
		got := Run(build(), ranks)
		for r := range reference {
			if !slices.Equal(reference[r], got[r]) {
				t.Fatalf("iter %d: determinism broken at rank %d: %v vs %v",
					iter, r, reference[r], got[r])
			}
		}
	}
}

// --- Non-contiguous ranks ---

// TestRunNonContiguousRanks verifies that the sweep loop handles rank
// numbers with gaps by iterating the sorted rank list rather than
// integer arithmetic (regression for a bug where the down sweep used
// order[r-1] and the up sweep used order[r+1], which silently broke
// when ranks had gaps).
func TestRunNonContiguousRanks(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "x"}, // a rank 0 → x rank 5
		[2]string{"b", "y"}, // b rank 0 → y rank 5
	)
	// Ranks deliberately skip 1, 2, 3, 4 — normally the dummy-node phase
	// would fill the gap, but that phase is not yet implemented.
	// Edges a→x and b→y are treated as adjacent-rank edges by
	// buildLayerEdges since they span the two ranks in ranksAsc.
	ranks := map[string]int{"a": 0, "b": 0, "x": 5, "y": 5}

	order := Run(g, ranks)

	if len(order[0]) != 2 || len(order[5]) != 2 {
		t.Fatalf("expected 2 nodes at ranks 0 and 5, got %v", order)
	}

	// Initial alphabetical order produces [a, b] / [x, y] which has
	// no crossings. The algorithm should preserve this.
	if n := testCountCrossings(g, order); n != 0 {
		t.Errorf("expected 0 crossings, got %d: %v", n, order)
	}
}

// TestRunNonContiguousRanksCrossingFix verifies the sweep actually runs
// across non-contiguous ranks and fixes crossings (not just preserves
// the initial state).
func TestRunNonContiguousRanksCrossingFix(t *testing.T) {
	// Pessimal alphabetical order: a→y, b→x. Alphabetical init gives
	// [a, b] / [x, y] which has one crossing (a→y crosses b→x).
	// Sweeping by barycenter should swap rank 5 to [y, x].
	g := buildGraph(
		[2]string{"a", "y"},
		[2]string{"b", "x"},
	)
	ranks := map[string]int{"a": 0, "b": 0, "x": 5, "y": 5}

	order := Run(g, ranks)
	if n := testCountCrossings(g, order); n != 0 {
		t.Errorf("sweep should have fixed crossings across non-contiguous ranks, got %d", n)
	}
}

// --- Preconditions ---

func TestRunPanicsOnUnrankedNode(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unranked node")
		}
	}()

	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	g.SetNode("b", graph.NodeAttrs{})
	// b is missing from ranks.
	Run(g, map[string]int{"a": 0})
}

// --- Monotonicity regression ---

// TestRunNeverIncreasesCrossings exercises the "best-so-far" latch
// across several graphs, asserting the result is never worse than the
// initial alphabetical ordering. This is the core correctness guarantee
// of the barycenter-with-best-tracking approach.
func TestRunNeverIncreasesCrossings(t *testing.T) {
	tests := []struct {
		name  string
		edges [][2]string
		ranks map[string]int
	}{
		{
			name: "dense 3x3 bipartite",
			edges: [][2]string{
				{"a", "x"}, {"a", "y"}, {"a", "z"},
				{"b", "x"}, {"b", "y"}, {"b", "z"},
				{"c", "x"}, {"c", "y"}, {"c", "z"},
			},
			ranks: map[string]int{
				"a": 0, "b": 0, "c": 0,
				"x": 1, "y": 1, "z": 1,
			},
		},
		{
			name: "4-layer zigzag",
			edges: [][2]string{
				{"a", "p"}, {"a", "q"},
				{"b", "p"}, {"b", "r"},
				{"p", "x"}, {"q", "x"}, {"q", "y"},
				{"r", "y"}, {"r", "z"},
				{"x", "m"}, {"y", "n"}, {"z", "m"},
			},
			ranks: map[string]int{
				"a": 0, "b": 0,
				"p": 1, "q": 1, "r": 1,
				"x": 2, "y": 2, "z": 2,
				"m": 3, "n": 3,
			},
		},
		{
			name: "star fan-out",
			edges: [][2]string{
				{"root", "c1"}, {"root", "c2"}, {"root", "c3"},
				{"root", "c4"}, {"root", "c5"}, {"root", "c6"},
			},
			ranks: map[string]int{
				"root": 0,
				"c1":   1, "c2": 1, "c3": 1, "c4": 1, "c5": 1, "c6": 1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := buildGraph(tc.edges...)
			initial := initOrder(g, tc.ranks)
			initialCross := testCountCrossings(g, initial)

			order := Run(g, tc.ranks)
			finalCross := testCountCrossings(g, order)

			if finalCross > initialCross {
				t.Errorf("Run increased crossings: initial=%d final=%d",
					initialCross, finalCross)
			}
		})
	}
}

// TestCountCrossingsSharedEndpoints verifies that edges sharing an
// endpoint (same source or same target) are correctly handled by the
// pairwise comparison — they should never count as crossings even when
// other edges in the same layer DO cross. Guards against a bug where
// changing strict `<` to `<=` would over-count shared-endpoint edges.
func TestCountCrossingsSharedEndpoints(t *testing.T) {
	// a has fan-out to x, y, z. b crosses through to x (between y and z).
	// Order: [a, b] / [x, y, z]
	//
	// Edge positions (upper, lower):
	//   a→x: (0, 0)
	//   a→y: (0, 1)
	//   a→z: (0, 2)
	//   b→x: (1, 0) — no wait, this goes from b to x at the START
	//
	// Let me reconsider. With order [a, b] and [x, y, z]:
	//   a at 0, b at 1, x at 0, y at 1, z at 2
	//   a→x: (0,0), a→y: (0,1), a→z: (0,2), b→y: (1,1)
	//
	// Crossings:
	//   a→x vs a→y: same source → no cross
	//   a→x vs a→z: same source → no cross
	//   a→x vs b→y: (0,0) vs (1,1) → 0<1 && 0<1 → no cross
	//   a→y vs a→z: same source → no cross
	//   a→y vs b→y: (0,1) vs (1,1) → 0<1 && 1<1 is false && 1>1 is false → no cross
	//   a→z vs b→y: (0,2) vs (1,1) → 0<1 && 2>1 → CROSS
	//
	// Expected: exactly 1 crossing despite 4 edges. A ≤ bug would count
	// the shared-target pair (a→y, b→y) as a crossing too, giving 2.
	g := buildGraph(
		[2]string{"a", "x"},
		[2]string{"a", "y"},
		[2]string{"a", "z"},
		[2]string{"b", "y"},
	)
	order := Order{0: {"a", "b"}, 1: {"x", "y", "z"}}
	if n := testCountCrossings(g, order); n != 1 {
		t.Errorf("expected exactly 1 crossing (shared endpoints don't cross), got %d", n)
	}
}

// --- Cross-counting ---

func TestCountCrossings(t *testing.T) {
	tests := []struct {
		name  string
		edges [][2]string
		order Order
		want  int
	}{
		{
			name:  "no edges",
			edges: nil,
			order: Order{0: {"a"}, 1: {"b"}},
			want:  0,
		},
		{
			name:  "one crossing",
			edges: [][2]string{{"a", "d"}, {"b", "c"}},
			order: Order{0: {"a", "b"}, 1: {"c", "d"}},
			want:  1,
		},
		{
			// a→c, a→d, b→c
			// a→c vs a→d: same source, never crosses
			// a→c vs b→c: same target, never crosses
			// a→d vs b→c: a<b and d>c, CROSS
			name:  "one real crossing among fan-in/out",
			edges: [][2]string{{"a", "c"}, {"a", "d"}, {"b", "c"}},
			order: Order{0: {"a", "b"}, 1: {"c", "d"}},
			want:  1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := graph.New()
			if len(tc.edges) == 0 {
				// No edges — declare nodes explicitly.
				for _, nodes := range tc.order {
					for _, n := range nodes {
						g.SetNode(n, graph.NodeAttrs{})
					}
				}
			} else {
				for _, e := range tc.edges {
					g.SetEdge(e[0], e[1], graph.EdgeAttrs{})
				}
			}
			if got := testCountCrossings(g, tc.order); got != tc.want {
				t.Errorf("got %d crossings, want %d", got, tc.want)
			}
		})
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
	if testCountCrossings(g, order) != 0 {
		t.Error("disconnected components should have no crossings")
	}
}

// --- buildLayerEdges internals ---

// TestBuildLayerEdgesFiltersSelfLoops verifies the defensive branch that
// skips self-loop edges when building the layer edge index.
func TestBuildLayerEdgesFiltersSelfLoops(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{}) // self-loop
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	ranks := map[string]int{"a": 0, "b": 1}

	le := buildLayerEdges(g, ranks, []int{0, 1})
	// Should contain only the a→b edge, not the self-loop.
	if len(le[0]) != 1 {
		t.Errorf("expected 1 edge, got %d: %v", len(le[0]), le[0])
	}
	if le[0][0].from != "a" || le[0][0].to != "b" {
		t.Errorf("unexpected edge: %v", le[0][0])
	}
}

// TestBuildLayerEdgesSkipsNonAdjacentSpans verifies edges that span more
// than one rank are skipped. (Dummy node insertion in a later phase
// normally prevents this, but the builder is defensive.)
func TestBuildLayerEdgesSkipsNonAdjacentSpans(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "c", graph.EdgeAttrs{}) // spans 2 ranks
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	g.SetEdge("b", "c", graph.EdgeAttrs{})
	ranks := map[string]int{"a": 0, "b": 1, "c": 2}

	le := buildLayerEdges(g, ranks, []int{0, 1, 2})
	if len(le[0]) != 1 || le[0][0].to != "b" {
		t.Errorf("rank 0 should only have a→b, got %v", le[0])
	}
	if len(le[1]) != 1 || le[1][0].from != "b" {
		t.Errorf("rank 1 should only have b→c, got %v", le[1])
	}
}

// TestBuildLayerEdgesPanicsOnUnrankedTarget verifies that unranked
// edge targets panic loudly rather than silently dropping.
func TestBuildLayerEdgesPanicsOnUnrankedTarget(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unranked target")
		}
	}()

	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	ranks := map[string]int{"a": 0, "b": 1} // c missing
	buildLayerEdges(g, ranks, []int{0, 1})
}

// TestBuildLayerEdgesPanicsOnUnrankedSource verifies that unranked
// edge sources panic loudly rather than silently dropping.
func TestBuildLayerEdgesPanicsOnUnrankedSource(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on unranked source")
		}
	}()

	g := buildGraph([2]string{"x", "a"}) // x unranked
	ranks := map[string]int{"a": 1}
	buildLayerEdges(g, ranks, []int{0, 1})
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
