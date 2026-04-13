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
	"fmt"
	"slices"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/layoututil"
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
//
// Panics if any node in g is missing from ranks (a precondition violation
// typically caused by skipping the rank phase or passing a stale map).
func Run(g *graph.Graph, ranks map[string]int) Order {
	if g.NodeCount() == 0 {
		return Order{}
	}

	order := initOrder(g, ranks)

	// Precompute data that never changes during iteration.
	ranksAsc := layoututil.SortedRanks(order)
	layerEdges := buildLayerEdges(g, ranks, ranksAsc)
	preds, succs := layoututil.BuildAdjacency(g)

	// Reusable scratch buffer for countCrossingsInLayer.
	var scratch []edgePos

	best := copyOrder(order)
	bestCross := countCrossings(order, ranksAsc, layerEdges, &scratch)

	for i := 0; i < maxIterations; i++ {
		if i%2 == 0 {
			// Down sweep: sort each rank by its predecessors. Iterate
			// ranksAsc directly so that the "previous" rank is always
			// the sorted neighbor, even if ranks are non-contiguous.
			for idx := 1; idx < len(ranksAsc); idx++ {
				r, prev := ranksAsc[idx], ranksAsc[idx-1]
				order[r] = sortByBarycenter(order[r], order[prev], preds)
			}
		} else {
			// Up sweep: sort by successors.
			for idx := len(ranksAsc) - 2; idx >= 0; idx-- {
				r, next := ranksAsc[idx], ranksAsc[idx+1]
				order[r] = sortByBarycenter(order[r], order[next], succs)
			}
		}

		cross := countCrossings(order, ranksAsc, layerEdges, &scratch)
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
// within each rank (for determinism). Panics if any node is missing from
// ranks — a precondition violation the caller must fix.
func initOrder(g *graph.Graph, ranks map[string]int) Order {
	result := make(Order)
	for _, n := range g.Nodes() {
		r, ok := ranks[n]
		if !ok {
			panic(fmt.Sprintf("order: node %q has no rank (rank phase precondition violated)", n))
		}
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
// neighbors is either the predecessors or successors map (depending on
// sweep direction).
func sortByBarycenter(nodes, source []string, neighbors map[string][]string) []string {
	sourcePos := positionMap(source)

	type entry struct {
		id         string
		barycenter float64
		origIdx    int
	}
	entries := make([]entry, len(nodes))
	for i, n := range nodes {
		sum := 0.0
		count := 0
		for _, nbr := range neighbors[n] {
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

// edgePos is a cross-counting work item: the horizontal positions of an
// edge's endpoints in their respective ranks.
type edgePos struct{ u, l int }

// buildLayerEdges groups edges by (upperRank, lowerRank) pair. Called once
// before the main loop; the result is immutable across iterations since
// only positions change, not edges.
//
// Edges spanning more than one rank pair are silently dropped. This is
// deliberate: a later layout phase (not yet implemented) inserts dummy
// nodes on long-span edges so every edge spans exactly one rank pair.
// Until that phase exists, cross-counting ignores long-span edges —
// their crossings would otherwise be conflated with adjacent-rank
// crossings. This is tracked as part of TODO(features) at the top of
// this file.
//
// Panics if any edge references a node missing from ranks. This is a
// caller precondition violation (the rank phase is expected to rank
// every node that appears in any edge).
func buildLayerEdges(g *graph.Graph, ranks map[string]int, ranksAsc []int) map[int][]layerEdge {
	result := make(map[int][]layerEdge, len(ranksAsc))

	adjacent := make(map[[2]int]bool, len(ranksAsc)-1)
	for i := 0; i < len(ranksAsc)-1; i++ {
		adjacent[[2]int{ranksAsc[i], ranksAsc[i+1]}] = true
	}

	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		fromRank, okF := ranks[eid.From]
		if !okF {
			panic(fmt.Sprintf("order: edge %v has unranked source %q", eid, eid.From))
		}
		toRank, okT := ranks[eid.To]
		if !okT {
			panic(fmt.Sprintf("order: edge %v has unranked target %q", eid, eid.To))
		}
		if !adjacent[[2]int{fromRank, toRank}] {
			continue // long-span edge; see doc comment above
		}
		result[fromRank] = append(result[fromRank], layerEdge{from: eid.From, to: eid.To})
	}
	return result
}

// countCrossings returns the total number of edge crossings between all
// adjacent rank pairs in order. Builds a position map per rank once
// (instead of twice per layer pair) and reuses a scratch buffer for the
// cross-counting work items.
func countCrossings(order Order, ranksAsc []int, layerEdges map[int][]layerEdge, scratch *[]edgePos) int {
	if len(ranksAsc) < 2 {
		return 0
	}
	// Build one position map per rank. Each rank except the outermost
	// would otherwise be positioned twice (as lower of pair i and upper
	// of pair i+1).
	positions := make([]map[string]int, len(ranksAsc))
	for i, r := range ranksAsc {
		positions[i] = positionMap(order[r])
	}
	total := 0
	for i := 0; i < len(ranksAsc)-1; i++ {
		upperRank := ranksAsc[i]
		total += countCrossingsInLayer(
			positions[i], positions[i+1], layerEdges[upperRank], scratch,
		)
	}
	return total
}

// countCrossingsInLayer counts crossings for the given precomputed edges
// using position maps for upper and lower ranks. O(E^2) pairwise.
// scratch is a reusable buffer to avoid per-call slice allocations.
func countCrossingsInLayer(upperPos, lowerPos map[string]int, edges []layerEdge, scratch *[]edgePos) int {
	if len(edges) < 2 {
		return 0
	}
	positioned := (*scratch)[:0]
	for _, e := range edges {
		positioned = append(positioned, edgePos{u: upperPos[e.from], l: lowerPos[e.to]})
	}
	*scratch = positioned

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

