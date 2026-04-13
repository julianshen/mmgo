package layoututil

import (
	"slices"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestBuildAdjacency(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "b", graph.EdgeAttrs{})
	g.SetEdge("a", "c", graph.EdgeAttrs{})
	g.SetEdge("b", "c", graph.EdgeAttrs{})

	preds, succs := BuildAdjacency(g)

	if len(preds["a"]) != 0 {
		t.Errorf("a should have no predecessors, got %v", preds["a"])
	}
	aSuccs := slices.Clone(succs["a"])
	slices.Sort(aSuccs)
	if !slices.Equal(aSuccs, []string{"b", "c"}) {
		t.Errorf("a successors: got %v, want [b c]", aSuccs)
	}
	cPreds := slices.Clone(preds["c"])
	slices.Sort(cPreds)
	if !slices.Equal(cPreds, []string{"a", "b"}) {
		t.Errorf("c predecessors: got %v, want [a b]", cPreds)
	}
}

func TestBuildAdjacencyEmptyGraph(t *testing.T) {
	g := graph.New()
	preds, succs := BuildAdjacency(g)
	if len(preds) != 0 || len(succs) != 0 {
		t.Error("empty graph should produce empty adjacency maps")
	}
}

func TestSortedRanks(t *testing.T) {
	m := map[int][]string{
		2: {"c"},
		0: {"a"},
		1: {"b"},
	}
	got := SortedRanks(m)
	if !slices.Equal(got, []int{0, 1, 2}) {
		t.Errorf("got %v, want [0 1 2]", got)
	}
}

func TestSortedRanksEmpty(t *testing.T) {
	got := SortedRanks(map[int][]string{})
	if len(got) != 0 {
		t.Errorf("empty map should return empty slice, got %v", got)
	}
}

func TestSortedRanksNegative(t *testing.T) {
	m := map[int][]string{
		-2: {"a"},
		5:  {"b"},
		0:  {"c"},
	}
	got := SortedRanks(m)
	if !slices.Equal(got, []int{-2, 0, 5}) {
		t.Errorf("got %v, want [-2 0 5]", got)
	}
}
