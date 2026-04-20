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

	chains := Run(g, ranks)
	if len(chains) != 0 {
		t.Errorf("expected no chains for 1-span edge, got %d", len(chains))
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
	// Long-span edge A → D spans 3 ranks; should get 2 dummies.
	g.SetEdge("A", "D", graph.EdgeAttrs{})
	ranks := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}

	chains := Run(g, ranks)

	key := Key{From: "A", To: "D"}
	if len(chains[key]) != 1 {
		t.Fatalf("expected 1 chain for A→D, got %d", len(chains[key]))
	}
	if got := len(chains[key][0].Dummies); got != 2 {
		t.Errorf("expected 2 dummies for 3-span edge, got %d", got)
	}
	// Dummies assigned ranks 1 and 2 respectively.
	for i, d := range chains[key][0].Dummies {
		wantRank := i + 1
		if ranks[d] != wantRank {
			t.Errorf("dummy %q: rank=%d, want %d", d, ranks[d], wantRank)
		}
	}
	// Original edge removed; chain of short edges inserted.
	if g.HasEdge("A", "D") {
		t.Error("original long-span edge should be removed")
	}
	if !g.HasEdge("A", chains[key][0].Dummies[0]) {
		t.Error("first short edge missing")
	}
	if !g.HasEdge(chains[key][0].Dummies[1], "D") {
		t.Error("final short edge missing")
	}
}

func TestMultipleLongEdgesBetweenSamePair(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{})
	g.SetNode("D", graph.NodeAttrs{})
	// Two parallel edges A → D across 3 ranks.
	g.SetEdge("A", "D", graph.EdgeAttrs{Label: "first"})
	g.SetEdge("A", "D", graph.EdgeAttrs{Label: "second"})
	ranks := map[string]int{"A": 0, "D": 3}

	chains := Run(g, ranks)
	key := Key{From: "A", To: "D"}
	if len(chains[key]) != 2 {
		t.Fatalf("expected 2 chains for parallel A→D edges, got %d", len(chains[key]))
	}
	// Dummy IDs must be distinct between the two chains.
	dummies := map[string]bool{}
	for _, chain := range chains[key] {
		for _, d := range chain.Dummies {
			if dummies[d] {
				t.Errorf("duplicate dummy id %q between parallel chains", d)
			}
			dummies[d] = true
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
		return Run(g, ranks)
	}
	first := build()
	for i := 0; i < 5; i++ {
		next := build()
		for k, chains := range first {
			for i, c := range chains {
				for j, d := range c.Dummies {
					if d != next[k][i].Dummies[j] {
						t.Fatalf("iter %d: dummy id drift at %s chain %d pos %d", i, d, i, j)
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

	chains := Run(g, ranks)
	if len(chains) != 0 {
		t.Errorf("expected no chains for self-loop, got %d", len(chains))
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
