package layoututil

import (
	"slices"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

func TestCompareEdgeIDs(t *testing.T) {
	cases := []struct {
		name string
		a, b graph.EdgeID
		want int // sign comparison
	}{
		{"by From", graph.EdgeID{From: "a"}, graph.EdgeID{From: "b"}, -1},
		{"by To", graph.EdgeID{From: "a", To: "a"}, graph.EdgeID{From: "a", To: "b"}, -1},
		{"by ID", graph.EdgeID{From: "a", To: "b", ID: 1}, graph.EdgeID{From: "a", To: "b", ID: 2}, -1},
		{"equal", graph.EdgeID{From: "a", To: "b", ID: 1}, graph.EdgeID{From: "a", To: "b", ID: 1}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CompareEdgeIDs(tc.a, tc.b)
			if (got < 0) != (tc.want < 0) || (got == 0) != (tc.want == 0) {
				t.Errorf("CompareEdgeIDs(%v, %v) = %d, want sign %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestSortEdges(t *testing.T) {
	edges := []graph.EdgeID{
		{From: "b", To: "a", ID: 1},
		{From: "a", To: "b", ID: 2},
		{From: "a", To: "b", ID: 1},
	}
	SortEdges(edges)
	want := []graph.EdgeID{
		{From: "a", To: "b", ID: 1},
		{From: "a", To: "b", ID: 2},
		{From: "b", To: "a", ID: 1},
	}
	for i, e := range edges {
		if e != want[i] {
			t.Errorf("idx %d: got %v, want %v", i, e, want[i])
		}
	}
}

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
