package rank

import (
	"slices"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/graphtest"
)

// --- Helpers ---

var buildGraph = graphtest.BuildGraph

// assertRanks checks that each (node, expected rank) pair in want matches
// the computed ranks.
func assertRanks(t *testing.T, ranks, want map[string]int) {
	t.Helper()
	for n, w := range want {
		if ranks[n] != w {
			t.Errorf("rank[%s] = %d, want %d", n, ranks[n], w)
		}
	}
}

// assertInvariants verifies the core contract:
//   - rank(v) - rank(u) >= minLen for every non-self-loop edge (u, v)
//   - all ranks are non-negative
//   - the minimum rank is 0
func assertInvariants(t *testing.T, g *graph.Graph, ranks map[string]int) {
	t.Helper()

	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		attrs, _ := g.EdgeAttrs(eid)
		minLen := attrs.EffectiveMinLen()
		diff := ranks[eid.To] - ranks[eid.From]
		if diff < minLen {
			t.Errorf("edge %s->%s: rank diff %d < minLen %d",
				eid.From, eid.To, diff, minLen)
		}
	}

	if len(ranks) == 0 {
		return
	}
	values := make([]int, 0, len(ranks))
	for _, r := range ranks {
		if r < 0 {
			t.Errorf("negative rank %d", r)
		}
		values = append(values, r)
	}
	if m := slices.Min(values); m != 0 {
		t.Errorf("minimum rank should be 0, got %d", m)
	}
}

// --- Trivial cases ---

func TestRunTrivialCases(t *testing.T) {
	tests := []struct {
		name  string
		setup func() *graph.Graph
		want  map[string]int
	}{
		{
			name:  "empty",
			setup: func() *graph.Graph { return graph.New() },
			want:  map[string]int{},
		},
		{
			name: "single node",
			setup: func() *graph.Graph {
				g := graph.New()
				g.SetNode("a", graph.NodeAttrs{})
				return g
			},
			want: map[string]int{"a": 0},
		},
		{
			name: "three isolated nodes",
			setup: func() *graph.Graph {
				g := graph.New()
				g.SetNode("a", graph.NodeAttrs{})
				g.SetNode("b", graph.NodeAttrs{})
				g.SetNode("c", graph.NodeAttrs{})
				return g
			},
			want: map[string]int{"a": 0, "b": 0, "c": 0},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.setup()
			ranks := Run(g)
			if len(ranks) != len(tc.want) {
				t.Errorf("expected %d ranks, got %d", len(tc.want), len(ranks))
			}
			assertRanks(t, ranks, tc.want)
		})
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
	assertRanks(t, ranks, map[string]int{"a": 0, "b": 1, "c": 2, "d": 3})
	assertInvariants(t, g, ranks)
}

// --- Diamond ---

func TestRunDiamond(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)
	assertRanks(t, ranks, map[string]int{"a": 0, "b": 1, "c": 1, "d": 2})
	assertInvariants(t, g, ranks)
}

// --- Multi-path graphs ---

func TestRunMultiPathGraph(t *testing.T) {
	// a → d (short: length 1)
	// a → b → c → d (long: length 3)
	g := buildGraph(
		[2]string{"a", "d"},
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)
	// Longest path determines the sink's rank.
	assertRanks(t, ranks, map[string]int{"a": 0, "d": 3})
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
	assertRanks(t, ranks, map[string]int{
		"a": 0, "b": 1, "c": 1, "d": 2, "e": 2, "f": 2,
	})
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
	// Parallel edges a→b with differing minLens — largest wins.
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
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"c", "d"},
	)
	ranks := Run(g)

	if len(ranks) != 4 {
		t.Errorf("expected 4 ranks, got %d", len(ranks))
	}
	assertRanks(t, ranks, map[string]int{"a": 0, "b": 1, "c": 0, "d": 1})
	assertInvariants(t, g, ranks)
}

// --- Self-loops ---

func TestRunSelfLoopIgnored(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	g.SetEdge("a", "b", graph.EdgeAttrs{})

	ranks := Run(g)

	assertRanks(t, ranks, map[string]int{"a": 0, "b": 1})
	assertInvariants(t, g, ranks)
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

// --- Network simplex optimization ---

func TestRunTightensBranchRanks(t *testing.T) {
	g := buildGraph(
		[2]string{"A", "B"},
		[2]string{"B", "C"},
		[2]string{"C", "D"},
		[2]string{"D", "E"},
		[2]string{"C", "F"},
	)
	ranks := Run(g)
	if ranks["F"] != ranks["C"]+1 {
		t.Errorf("rank(F)=%d, want rank(C)+1=%d; ranks=%v", ranks["F"], ranks["C"]+1, ranks)
	}
	assertInvariants(t, g, ranks)
}

// --- Larger graph ---

func TestRunLargerGraphInvariants(t *testing.T) {
	// Cross-edge graph:
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

	// a is the only source, e is the only sink.
	if ranks["a"] != 0 {
		t.Errorf("a=%d, want 0", ranks["a"])
	}
	for _, predecessor := range []string{"b", "c", "d"} {
		if ranks["e"] < ranks[predecessor] {
			t.Errorf("e (rank %d) should come after %s (rank %d)",
				ranks["e"], predecessor, ranks[predecessor])
		}
	}
	assertInvariants(t, g, ranks)
}
