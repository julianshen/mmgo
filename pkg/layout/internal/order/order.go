// Package order determines the horizontal ordering of nodes within each
// rank to minimize edge crossings between adjacent ranks.
//
// This is the third phase of the dagre layout engine port. The input is a
// ranked graph (from the rank phase) and the output is an ordering of
// nodes within each rank. The core algorithm is the barycenter heuristic
// (Sugiyama 1981) with iterative up/down sweeping:
//
//  1. Start with an initial ordering (alphabetical within each rank).
//  2. Sweep down: for each rank r from top to bottom, sort nodes by the
//     average position of their predecessors in rank r-1.
//  3. Sweep up: for each rank r from bottom to top, sort by successors
//     in rank r+1.
//  4. After each sweep, count total crossings; keep the best ordering seen.
//  5. Stop after a fixed iteration cap.
//
// Cross counting uses an O(E^2) pairwise comparison, adequate for the
// target graph size (10-100 nodes typical, 500 worst case).
//
// TODO(perf): port dagre's O(|E| log |V|) Barth-Jünger-Mutzel (2004)
// cross counter for large graphs.
//
// TODO(features): compound-graph-aware ordering (subgraph support) and
// flat-edge conflict resolution are not yet implemented.
package order

import (
	"math"
	"sort"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// Order maps a rank number to the left-to-right ordered list of nodes in
// that rank.
type Order map[int][]string

// maxIterations matches dagre's default. Convergence is typically much
// faster (4-8 iterations) for diagrams of mmgo's target size.
const maxIterations = 24

// Run computes a node ordering within each rank that minimizes edge
// crossings between adjacent ranks.
//
// ranks must map every node in g to its rank (as produced by the rank
// phase). The returned Order contains the same set of nodes grouped by
// rank, in the order that minimizes crossings.
func Run(g *graph.Graph, ranks map[string]int) Order {
	if g.NodeCount() == 0 {
		return Order{}
	}

	order := initOrder(g, ranks)
	best := copyOrder(order)
	bestCross := countCrossings(g, best)

	minR, maxR := rankRange(order)

	for i := 0; i < maxIterations; i++ {
		if i%2 == 0 {
			// Down sweep: sort each rank by its predecessors.
			for r := minR + 1; r <= maxR; r++ {
				if nodes, ok := order[r]; ok {
					order[r] = sortByBarycenter(g, order, nodes, r-1, true)
				}
			}
		} else {
			// Up sweep: sort each rank by its successors.
			for r := maxR - 1; r >= minR; r-- {
				if nodes, ok := order[r]; ok {
					order[r] = sortByBarycenter(g, order, nodes, r+1, false)
				}
			}
		}

		cross := countCrossings(g, order)
		if cross < bestCross {
			best = copyOrder(order)
			bestCross = cross
			if bestCross == 0 {
				break
			}
		}
	}

	return best
}

// initOrder groups nodes by rank with alphabetical initial ordering
// within each rank (for determinism).
func initOrder(g *graph.Graph, ranks map[string]int) Order {
	result := make(Order)
	for _, n := range g.Nodes() {
		r := ranks[n]
		result[r] = append(result[r], n)
	}
	for r := range result {
		sort.Strings(result[r])
	}
	return result
}

// sortByBarycenter reorders nodes in the target rank by the average
// position of their neighbors in the adjacent source rank. Nodes without
// any neighbor in the source rank keep their current relative position.
//
// If usePredecessors is true, neighbors are the node's predecessors
// (typical for a down sweep). Otherwise they are successors.
func sortByBarycenter(g *graph.Graph, order Order, nodes []string, sourceRank int, usePredecessors bool) []string {
	sourcePos := positionMap(order[sourceRank])

	type entry struct {
		id         string
		barycenter float64
		// origIdx is the node's current position in the target rank;
		// used as the tie-breaker and as the default for nodes with no
		// adjacent-rank neighbors.
		origIdx int
	}
	entries := make([]entry, len(nodes))
	for i, n := range nodes {
		var neighbors []string
		if usePredecessors {
			neighbors = g.Predecessors(n)
		} else {
			neighbors = g.Successors(n)
		}

		sum := 0.0
		count := 0
		for _, nbr := range neighbors {
			if p, ok := sourcePos[nbr]; ok {
				sum += float64(p)
				count++
			}
		}

		bary := float64(i) // default: preserve current position
		if count > 0 {
			bary = sum / float64(count)
		}
		entries[i] = entry{id: n, barycenter: bary, origIdx: i}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].barycenter != entries[j].barycenter {
			return entries[i].barycenter < entries[j].barycenter
		}
		return entries[i].origIdx < entries[j].origIdx
	})

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.id
	}
	return result
}

// countCrossings returns the total number of edge crossings between all
// adjacent rank pairs in order.
func countCrossings(g *graph.Graph, order Order) int {
	if len(order) < 2 {
		return 0
	}
	ranks := sortedRanks(order)
	total := 0
	for i := 0; i < len(ranks)-1; i++ {
		total += countCrossingsBetween(g, order, ranks[i], ranks[i+1])
	}
	return total
}

// countCrossingsBetween counts crossings for edges spanning the two
// given adjacent ranks. O(E^2) pairwise comparison.
func countCrossingsBetween(g *graph.Graph, order Order, upperRank, lowerRank int) int {
	upperPos := positionMap(order[upperRank])
	lowerPos := positionMap(order[lowerRank])

	type edge struct{ u, l int }
	var edges []edge
	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		uPos, okU := upperPos[eid.From]
		lPos, okL := lowerPos[eid.To]
		if !okU || !okL {
			continue
		}
		edges = append(edges, edge{u: uPos, l: lPos})
	}

	crossings := 0
	for i := 0; i < len(edges); i++ {
		for j := i + 1; j < len(edges); j++ {
			e1, e2 := edges[i], edges[j]
			if (e1.u < e2.u && e1.l > e2.l) || (e1.u > e2.u && e1.l < e2.l) {
				crossings++
			}
		}
	}
	return crossings
}

// positionMap returns a map from node ID to its index in the slice.
func positionMap(nodes []string) map[string]int {
	pos := make(map[string]int, len(nodes))
	for i, n := range nodes {
		pos[n] = i
	}
	return pos
}

// copyOrder returns a deep copy of order.
func copyOrder(src Order) Order {
	dst := make(Order, len(src))
	for r, nodes := range src {
		clone := make([]string, len(nodes))
		copy(clone, nodes)
		dst[r] = clone
	}
	return dst
}

// rankRange returns the minimum and maximum rank numbers in order.
func rankRange(order Order) (minR, maxR int) {
	minR, maxR = math.MaxInt, math.MinInt
	for r := range order {
		if r < minR {
			minR = r
		}
		if r > maxR {
			maxR = r
		}
	}
	return
}

// sortedRanks returns the rank numbers in ascending order.
func sortedRanks(order Order) []int {
	ranks := make([]int, 0, len(order))
	for r := range order {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)
	return ranks
}
