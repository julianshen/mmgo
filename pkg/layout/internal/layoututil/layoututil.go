// Package layoututil contains shared helpers used by multiple layout
// phases (rank, order, position). Each helper was originally duplicated
// across phase packages and promoted here once it appeared in a second
// phase.
package layoututil

import (
	"slices"
	"sort"
	"strings"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// CompareEdgeIDs orders edges lexicographically by From, then To,
// then ID. Used wherever a phase needs a deterministic edge iteration
// order — Go map iteration is randomized, and without this several
// phases would produce non-reproducible output across runs.
func CompareEdgeIDs(a, b graph.EdgeID) int {
	if a.From != b.From {
		return strings.Compare(a.From, b.From)
	}
	if a.To != b.To {
		return strings.Compare(a.To, b.To)
	}
	return a.ID - b.ID
}

// SortEdges sorts edges in place using CompareEdgeIDs.
func SortEdges(edges []graph.EdgeID) {
	sort.Slice(edges, func(i, j int) bool {
		return CompareEdgeIDs(edges[i], edges[j]) < 0
	})
}

// BuildAdjacency returns deduped predecessor and successor lists for
// every node in g. Called once before a layout phase's main loop to
// avoid repeatedly calling g.Predecessors / g.Successors, which
// allocate fresh maps and slices on every invocation.
func BuildAdjacency(g *graph.Graph) (preds, succs map[string][]string) {
	nodes := g.Nodes()
	preds = make(map[string][]string, len(nodes))
	succs = make(map[string][]string, len(nodes))
	for _, n := range nodes {
		preds[n] = g.Predecessors(n)
		succs[n] = g.Successors(n)
	}
	return preds, succs
}

// SortedRanks returns the rank numbers in m in ascending order.
func SortedRanks(m map[int][]string) []int {
	ranks := make([]int, 0, len(m))
	for r := range m {
		ranks = append(ranks, r)
	}
	slices.Sort(ranks)
	return ranks
}
