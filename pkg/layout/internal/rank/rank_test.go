package rank

import (
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

// assertInvariants verifies that the ranks satisfy the core contract:
//   - rank(v) - rank(u) >= minLen for every edge (u, v), excluding self-loops
//   - all ranks are non-negative
//   - the minimum rank is 0
func assertInvariants(t *testing.T, g *graph.Graph, ranks map[string]int) {
	t.Helper()

	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		attrs, _ := g.EdgeAttrs(eid)
		minLen := attrs.MinLen
		if minLen == 0 {
			minLen = 1
		}
		diff := ranks[eid.To] - ranks[eid.From]
		if diff < minLen {
			t.Errorf("edge %s->%s: rank diff %d < minLen %d",
				eid.From, eid.To, diff, minLen)
		}
	}

	minRank := -1
	for n, r := range ranks {
		if r < 0 {
			t.Errorf("node %q has negative rank %d", n, r)
		}
		if minRank == -1 || r < minRank {
			minRank = r
		}
	}
	if len(ranks) > 0 && minRank != 0 {
		t.Errorf("minimum rank should be 0, got %d", minRank)
	}
}

// --- Trivial cases ---

func TestRunEmptyGraph(t *testing.T) {
	g := graph.New()
	ranks := Run(g)
	if len(ranks) != 0 {
		t.Errorf("expected empty ranks, got %d entries", len(ranks))
	}
}

func TestRunSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	ranks := Run(g)
	if ranks["a"] != 0 {
		t.Errorf("single node should have rank 0, got %d", ranks["a"])
	}
}

func TestRunIsolatedNodes(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	g.SetNode("b", graph.NodeAttrs{})
	g.SetNode("c", graph.NodeAttrs{})

	ranks := Run(g)

	// All isolated nodes should be at rank 0.
	for _, n := range []string{"a", "b", "c"} {
		if ranks[n] != 0 {
			t.Errorf("isolated node %s: rank %d, want 0", n, ranks[n])
		}
	}
}

// --- Linear chain ---

func TestRunLinearChain(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)

	want := map[string]int{"a": 0, "b": 1, "c": 2, "d": 3}
	for n, w := range want {
		if ranks[n] != w {
			t.Errorf("rank[%s] = %d, want %d", n, ranks[n], w)
		}
	}
	assertInvariants(t, g, ranks)
}

// --- Diamond ---

func TestRunDiamond(t *testing.T) {
	// a → b, a → c, b → d, c → d
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)

	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0", ranks["a"])
	}
	if ranks["d"] != 2 {
		t.Errorf("d=%d, want 2 (longest path length)", ranks["d"])
	}
	if ranks["b"] != 1 {
		t.Errorf("b=%d, want 1", ranks["b"])
	}
	if ranks["c"] != 1 {
		t.Errorf("c=%d, want 1", ranks["c"])
	}
	assertInvariants(t, g, ranks)
}

// --- Multi-path graphs ---

func TestRunMultiPathGraph(t *testing.T) {
	// a → d (short path, length 1)
	// a → b → c → d (long path, length 3)
	g := buildGraph(
		[2]string{"a", "d"},
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)

	// Longest path determines d's rank: 3
	if ranks["d"] != 3 {
		t.Errorf("d=%d, want 3", ranks["d"])
	}
	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0", ranks["a"])
	}
	assertInvariants(t, g, ranks)
}

func TestRunTreeGraph(t *testing.T) {
	//       a
	//      / \
	//     b   c
	//    /|   |
	//   d e   f
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"b", "e"},
		[2]string{"c", "f"},
	)
	ranks := Run(g)

	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0", ranks["a"])
	}
	// All leaves should be at rank 2
	for _, leaf := range []string{"d", "e", "f"} {
		if ranks[leaf] != 2 {
			t.Errorf("leaf %s=%d, want 2", leaf, ranks[leaf])
		}
	}
	assertInvariants(t, g, ranks)
}

// --- MinLen ---

func TestRunMinLenRespected(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{MinLen: 3})

	ranks := Run(g)

	if ranks["b"]-ranks["a"] != 3 {
		t.Errorf("rank(b) - rank(a) = %d, want 3",
			ranks["b"]-ranks["a"])
	}
	assertInvariants(t, g, ranks)
}

func TestRunMinLenMultiplePaths(t *testing.T) {
	// a → b (minLen 1)
	// a → b (minLen 5) via multi-edge — dominant
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{MinLen: 1})
	g.SetEdge("a", "b", graph.EdgeAttrs{MinLen: 5})

	ranks := Run(g)

	if ranks["b"]-ranks["a"] < 5 {
		t.Errorf("rank diff should respect max minLen=5, got %d",
			ranks["b"]-ranks["a"])
	}
	assertInvariants(t, g, ranks)
}

// --- Disconnected components ---

func TestRunDisconnectedComponents(t *testing.T) {
	// Two independent chains: a→b, c→d
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)

	if len(ranks) != 4 {
		t.Errorf("expected 4 ranks, got %d", len(ranks))
	}
	// Both components should start at rank 0 (after normalization).
	// a and c should be 0, b and d should be 1.
	if ranks["a"] != 0 || ranks["c"] != 0 {
		t.Errorf("sources should be at rank 0: a=%d c=%d", ranks["a"], ranks["c"])
	}
	if ranks["b"] != 1 || ranks["d"] != 1 {
		t.Errorf("sinks should be at rank 1: b=%d d=%d", ranks["b"], ranks["d"])
	}
	assertInvariants(t, g, ranks)
}

// --- Self-loops ---

func TestRunSelfLoopIgnored(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	g.SetEdge("a", "b", graph.EdgeAttrs{})

	ranks := Run(g)

	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0 (self-loop should not affect rank)", ranks["a"])
	}
	if ranks["b"] != 1 {
		t.Errorf("b=%d, want 1", ranks["b"])
	}
}

// --- Determinism ---

func TestRunDeterministic(t *testing.T) {
	build := func() *graph.Graph {
		return buildGraph(
			[2]string{"a", "b"},
			[2]string{"a", "c"},
			[2]string{"b", "d"},
			[2]string{"c", "d"},
		)
	}
	r1 := Run(build())
	r2 := Run(build())

	if len(r1) != len(r2) {
		t.Fatalf("length mismatch: %d vs %d", len(r1), len(r2))
	}
	for n, v := range r1 {
		if r2[n] != v {
			t.Errorf("determinism broken: %s=%d vs %d", n, v, r2[n])
		}
	}
}

// --- Larger graph invariant test ---

func TestRunLargerGraphInvariants(t *testing.T) {
	// A more complex graph with cross edges.
	//   a → b → c → e
	//   a → d → e
	//   b → e (skip-level cross edge)
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "e"},
		[2]string{"a", "d"},
		[2]string{"d", "e"},
		[2]string{"b", "e"},
	)
	ranks := Run(g)

	// a is the only source, should be at rank 0
	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0", ranks["a"])
	}
	// e is the only sink, should be at max rank
	if ranks["e"] < ranks["b"] || ranks["e"] < ranks["c"] || ranks["e"] < ranks["d"] {
		t.Errorf("e should be after b, c, d: %v", ranks)
	}
	assertInvariants(t, g, ranks)
}
