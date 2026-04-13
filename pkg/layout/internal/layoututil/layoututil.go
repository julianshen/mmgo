// Package layoututil contains shared helpers used by multiple layout
// phases (rank, order, position). Each helper was originally duplicated
// across phase packages and promoted here once it appeared in a second
// phase.
package layoututil

import (
	"slices"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

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
