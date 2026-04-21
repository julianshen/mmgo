package dummy

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestNoDummiesForAdjacentEdges(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetNode("B", graph.NodeAttrs{})
	g.SetEdge("A", "B", graph.EdgeAttrs{})
	ranks := map[string]int{"A": 0, "B": 1}

	res := Run(g, ranks)
	if len(res.Chains) != 0 {
		t.Errorf("expected no chains for 1-span edge, got %d", len(res.Chains))
	}
	if len(res.Dummies) != 0 {
		t.Errorf("expected no dummies for 1-span edge, got %d", len(res.Dummies))
	}
	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes (no dummies), got %d", g.NodeCount())
	}
}

func TestInsertsDummiesForLongSpan(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetNode("B", graph.NodeAttrs{})
	g.SetNode("C", graph.NodeAttrs{})
	g.SetNode("D", graph.NodeAttrs{})
	g.SetEdge("A", "D", graph.EdgeAttrs{})
	ranks := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}

	res := Run(g, ranks)

	key := Key{From: "A", To: "D"}
	if len(res.Chains[key]) != 1 {
		t.Fatalf("expected 1 chain for A→D, got %d", len(res.Chains[key]))
	}
	chain := res.Chains[key][0]
	if len(chain.Dummies) != 2 {
		t.Errorf("expected 2 dummies for 3-span edge, got %d", len(chain.Dummies))
	}
	if len(res.Dummies) != 2 {
		t.Errorf("Result.Dummies should list both inserted ids, got %d", len(res.Dummies))
	}
	for i, d := range chain.Dummies {
		if ranks[d] != i+1 {
			t.Errorf("dummy %q: rank=%d, want %d", d, ranks[d], i+1)
		}
	}
	if g.HasEdge("A", "D") {
		t.Error("original long-span edge should be removed")
	}
	if !g.HasEdge("A", chain.Dummies[0]) {
		t.Error("first short edge missing")
	}
	if !g.HasEdge(chain.Dummies[1], "D") {
		t.Error("final short edge missing")
	}
}

func TestMultipleLongEdgesBetweenSamePair(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetNode("D", graph.NodeAttrs{})
	g.SetEdge("A", "D", graph.EdgeAttrs{Label: "first"})
	g.SetEdge("A", "D", graph.EdgeAttrs{Label: "second"})
	ranks := map[string]int{"A": 0, "D": 3}

	res := Run(g, ranks)
	key := Key{From: "A", To: "D"}
	if len(res.Chains[key]) != 2 {
		t.Fatalf("expected 2 chains for parallel A→D edges, got %d", len(res.Chains[key]))
	}
	seen := map[string]bool{}
	for _, chain := range res.Chains[key] {
		for _, d := range chain.Dummies {
			if seen[d] {
				t.Errorf("duplicate dummy id %q between parallel chains", d)
			}
			seen[d] = true
		}
	}
}

func TestDummyIDsAreReproducible(t *testing.T) {
	build := func() map[Key][]Chain {
		g := graph.New()
		g.SetNode("A", graph.NodeAttrs{})
		g.SetNode("D", graph.NodeAttrs{})
		g.SetEdge("A", "D", graph.EdgeAttrs{})
		ranks := map[string]int{"A": 0, "D": 3}
		return Run(g, ranks).Chains
	}
	first := build()
	for iter := 0; iter < 5; iter++ {
		next := build()
		for k, chains := range first {
			for ci, c := range chains {
				for pos, d := range c.Dummies {
					if d != next[k][ci].Dummies[pos] {
						t.Fatalf("iter %d: dummy id drift at key=%v chain=%d pos=%d", iter, k, ci, pos)
					}
				}
			}
		}
	}
}

func TestSelfLoopSkipped(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetEdge("A", "A", graph.EdgeAttrs{})
	ranks := map[string]int{"A": 0}

	res := Run(g, ranks)
	if len(res.Chains) != 0 {
		t.Errorf("expected no chains for self-loop, got %d", len(res.Chains))
	}
	if !g.HasEdge("A", "A") {
		t.Error("self-loop should be preserved")
	}
}

func TestIsDummy(t *testing.T) {
	cases := map[string]bool{
		"__dummy_0_0_1":    true,
		"__dummy_":         false,
		"A":                false,
		"__user_id":        false,
		"":                 false,
	}
	for id, want := range cases {
		if got := IsDummy(id); got != want {
			t.Errorf("IsDummy(%q) = %v, want %v", id, got, want)
		}
	}
}
