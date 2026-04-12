// Package rank assigns integer rank (layer) values to graph nodes such that
// for every non-self-loop edge (u, v) the constraint
//
//	rank(v) - rank(u) >= minLen(u, v)
//
// holds. Ranks are non-negative integers starting at 0 for the topmost
// nodes (sources after acyclic processing).
//
// The current implementation uses longest-path ranking (the initialization
// phase of dagre's network simplex). This produces the minimum-height
// layering — for most flowcharts with uniform minLen=1 and weight=1 it is
// near-optimal. The graph is assumed to be acyclic; callers should run the
// acyclic phase first.
//
// TODO(perf): port dagre's full network simplex optimizer for cases where
// LP-optimal total edge length matters. See Gansner, Koutsofios, North, Vo
// (1993) "A Technique for Drawing Directed Graphs".
package rank

import (
	"slices"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// Run assigns a rank to every node in g and returns the resulting rank
// map. Every non-self-loop edge (u, v) in g satisfies
//
//	ranks[v] - ranks[u] >= minLen(u, v)
//
// where minLen defaults to 1 when EdgeAttrs.MinLen is 0.
//
// The graph is assumed to be acyclic (no cycles among non-self-loop edges).
// Self-loops are ignored for ranking purposes.
func Run(g *graph.Graph) map[string]int {
	ranks := longestPath(g)
	normalize(ranks)
	return ranks
}

// longestPath assigns each node its rank based on the longest path to a
// sink. Sinks get rank 0; sources get the most negative rank. The result
// is not yet normalized to start at 0.
//
// The algorithm DFS-walks from every source (and then any unvisited nodes
// for disconnected components). For each node v:
//
//	rank(v) = min over successors w of (rank(w) - minLen(v, w))
//
// or 0 if v has no successors. Sources are processed in sorted order for
// determinism, though the result is order-independent.
func longestPath(g *graph.Graph) map[string]int {
	ranks := make(map[string]int, g.NodeCount())
	visited := make(map[string]bool, g.NodeCount())

	var dfs func(v string) int
	dfs = func(v string) int {
		if visited[v] {
			return ranks[v]
		}
		visited[v] = true

		minChildRank := 0
		hasChildren := false
		for _, eid := range g.OutEdges(v) {
			if eid.To == v {
				continue // skip self-loops
			}
			attrs, _ := g.EdgeAttrs(eid)
			ml := effectiveMinLen(attrs)
			candidate := dfs(eid.To) - ml
			if !hasChildren || candidate < minChildRank {
				minChildRank = candidate
				hasChildren = true
			}
		}

		if hasChildren {
			ranks[v] = minChildRank
		} else {
			ranks[v] = 0
		}
		return ranks[v]
	}

	// Run DFS from all sources in sorted order for determinism.
	sources := findSources(g)
	slices.Sort(sources)
	for _, src := range sources {
		dfs(src)
	}

	// Handle disconnected components or isolated subgraphs that have no
	// acyclic entry point reachable from any source.
	allNodes := g.Nodes()
	slices.Sort(allNodes)
	for _, n := range allNodes {
		if !visited[n] {
			dfs(n)
		}
	}

	return ranks
}

// normalize shifts ranks so the minimum rank is 0.
func normalize(ranks map[string]int) {
	if len(ranks) == 0 {
		return
	}
	minRank := 0
	first := true
	for _, r := range ranks {
		if first || r < minRank {
			minRank = r
			first = false
		}
	}
	if minRank == 0 {
		return
	}
	for n := range ranks {
		ranks[n] -= minRank
	}
}

// findSources returns all nodes with no incoming non-self-loop edges.
func findSources(g *graph.Graph) []string {
	var sources []string
	for _, n := range g.Nodes() {
		if hasNonSelfIncoming(g, n) {
			continue
		}
		sources = append(sources, n)
	}
	return sources
}

// hasNonSelfIncoming reports whether n has any incoming edge that isn't a
// self-loop.
func hasNonSelfIncoming(g *graph.Graph, n string) bool {
	for _, eid := range g.InEdges(n) {
		if eid.From != n {
			return true
		}
	}
	return false
}

// effectiveMinLen returns the effective minimum edge length, defaulting
// to 1 when unset (zero value).
func effectiveMinLen(attrs graph.EdgeAttrs) int {
	if attrs.MinLen <= 0 {
		return 1
	}
	return attrs.MinLen
}
