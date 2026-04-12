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
//  5. Stop after a fixed iteration cap or when crossings reach 0.
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
	"slices"

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

	// Precompute data that never changes during iteration:
	//   - sortedRanks: the sorted list of rank numbers
	//   - layerEdges: edges indexed by (upperRank -> edges to lowerRank)
	// Rebuilding these every iteration would double the work.
	ranksAsc := sortedRanks(order)
	layerEdges := buildLayerEdges(g, ranks, ranksAsc)

	best := copyOrder(order)
	bestCross := countCrossings(order, ranksAsc, layerEdges)

	minR, maxR := ranksAsc[0], ranksAsc[len(ranksAsc)-1]

	for i := 0; i < maxIterations; i++ {
		if i%2 == 0 {
			// Down sweep: sort each rank by its predecessors.
			for r := minR + 1; r <= maxR; r++ {
				if nodes, ok := order[r]; ok {
					order[r] = sortByBarycenter(g, nodes, order[r-1], true)
				}
			}
		} else {
			// Up sweep: sort each rank by its successors.
			for r := maxR - 1; r >= minR; r-- {
				if nodes, ok := order[r]; ok {
					order[r] = sortByBarycenter(g, nodes, order[r+1], false)
				}
			}
		}

		cross := countCrossings(order, ranksAsc, layerEdges)
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
		slices.Sort(result[r])
	}
	return result
}

// sortByBarycenter reorders nodes in the target rank by the average
// position of their neighbors in the adjacent source rank. Nodes without
// any neighbor in the source rank keep their current relative position.
//
// If usePredecessors is true, neighbors are the node's predecessors
// (typical for a down sweep). Otherwise they are successors.
func sortByBarycenter(g *graph.Graph, nodes, source []string, usePredecessors bool) []string {
	sourcePos := positionMap(source)

	type entry struct {
		id         string
		barycenter float64
		origIdx    int
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

	slices.SortStableFunc(entries, func(a, b entry) int {
		if a.barycenter < b.barycenter {
			return -1
		}
		if a.barycenter > b.barycenter {
			return 1
		}
		return a.origIdx - b.origIdx
	})

	result := make([]string, len(entries))
	for i, e := range entries {
		result[i] = e.id
	}
	return result
}

// layerEdge is a precomputed edge between two adjacent ranks, stored as
// the source and target node IDs. Positions are looked up from the live
// order during countCrossings.
type layerEdge struct {
	from string
	to   string
}

// buildLayerEdges groups edges by (upperRank, lowerRank) pair. Called once
// before the main loop; the result is immutable across iterations since
// only positions change, not edges.
func buildLayerEdges(g *graph.Graph, ranks map[string]int, ranksAsc []int) map[int][]layerEdge {
	// Pre-size by the number of adjacent rank pairs.
	result := make(map[int][]layerEdge, len(ranksAsc))

	// Build a set of adjacent rank pairs we care about so we can filter
	// edges that span more than one rank (which shouldn't happen after
	// dummy node insertion in a later phase, but we're defensive).
	adjacent := make(map[[2]int]bool, len(ranksAsc)-1)
	for i := 0; i < len(ranksAsc)-1; i++ {
		adjacent[[2]int{ranksAsc[i], ranksAsc[i+1]}] = true
	}

	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		fromRank, okF := ranks[eid.From]
		toRank, okT := ranks[eid.To]
		if !okF || !okT {
			continue
		}
		if !adjacent[[2]int{fromRank, toRank}] {
			continue
		}
		result[fromRank] = append(result[fromRank], layerEdge{from: eid.From, to: eid.To})
	}
	return result
}

// countCrossings returns the total number of edge crossings between all
// adjacent rank pairs in order, using precomputed layerEdges.
func countCrossings(order Order, ranksAsc []int, layerEdges map[int][]layerEdge) int {
	if len(ranksAsc) < 2 {
		return 0
	}
	total := 0
	for i := 0; i < len(ranksAsc)-1; i++ {
		upperRank := ranksAsc[i]
		lowerRank := ranksAsc[i+1]
		total += countCrossingsInLayer(
			order[upperRank], order[lowerRank], layerEdges[upperRank],
		)
	}
	return total
}

// countCrossingsInLayer counts crossings for the given precomputed edges
// using the current positions of nodes in upper and lower ranks.
// O(E^2) pairwise comparison.
func countCrossingsInLayer(upper, lower []string, edges []layerEdge) int {
	if len(edges) < 2 {
		return 0
	}
	upperPos := positionMap(upper)
	lowerPos := positionMap(lower)

	type edgePos struct{ u, l int }
	positioned := make([]edgePos, 0, len(edges))
	for _, e := range edges {
		positioned = append(positioned, edgePos{u: upperPos[e.from], l: lowerPos[e.to]})
	}

	crossings := 0
	for i := 0; i < len(positioned); i++ {
		for j := i + 1; j < len(positioned); j++ {
			e1, e2 := positioned[i], positioned[j]
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
		dst[r] = slices.Clone(nodes)
	}
	return dst
}

// sortedRanks returns the rank numbers in ascending order.
func sortedRanks(order Order) []int {
	ranks := make([]int, 0, len(order))
	for r := range order {
		ranks = append(ranks, r)
	}
	slices.Sort(ranks)
	return ranks
}
