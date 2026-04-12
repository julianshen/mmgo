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
	"math"
	"slices"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// Run assigns a rank to every node in g and returns the resulting rank
// map. Every non-self-loop edge (u, v) in g satisfies
//
//	ranks[v] - ranks[u] >= minLen(u, v)
//
// where minLen defaults to 1 via EdgeAttrs.EffectiveMinLen.
//
// The graph is assumed to be acyclic (no cycles among non-self-loop edges).
// Self-loops are ignored for ranking purposes.
func Run(g *graph.Graph) map[string]int {
	s := &longestPathState{
		g:       g,
		ranks:   make(map[string]int, g.NodeCount()),
		visited: make(map[string]bool, g.NodeCount()),
	}
	// DFS from every node in sorted order. Already-visited nodes
	// short-circuit in dfs(). Starting from non-sources is harmless
	// because rank values are computed bottom-up from successors.
	nodes := g.Nodes()
	slices.Sort(nodes)
	for _, n := range nodes {
		s.dfs(n)
	}
	normalize(s.ranks)
	return s.ranks
}

// longestPathState groups the mutable state for the recursive DFS.
// Using a method on this struct avoids the `var dfs func; dfs = ...`
// self-reference dance a free closure would require.
//
// Recursion depth is bounded by the length of the longest path in g.
// Mermaid diagrams are typically tens to hundreds of nodes deep, well
// within Go's default goroutine stack growth. If mmgo ever needs to
// handle graphs with thousands of nodes on a single longest path, this
// should be converted to an iterative postorder traversal.
type longestPathState struct {
	g       *graph.Graph
	ranks   map[string]int
	visited map[string]bool
}

// dfs computes rank(v) = min over non-self successors w of
// (rank(w) - minLen(v, w)), with leaves getting rank 0.
func (s *longestPathState) dfs(v string) int {
	if s.visited[v] {
		return s.ranks[v]
	}
	s.visited[v] = true

	minChildRank := math.MaxInt
	for _, eid := range s.g.OutEdges(v) {
		if eid.To == v {
			continue
		}
		attrs, _ := s.g.EdgeAttrs(eid)
		candidate := s.dfs(eid.To) - attrs.EffectiveMinLen()
		if candidate < minChildRank {
			minChildRank = candidate
		}
	}

	if minChildRank == math.MaxInt {
		s.ranks[v] = 0
	} else {
		s.ranks[v] = minChildRank
	}
	return s.ranks[v]
}

// normalize shifts ranks so the minimum rank is 0.
func normalize(ranks map[string]int) {
	if len(ranks) == 0 {
		return
	}
	minRank := math.MaxInt
	for _, r := range ranks {
		if r < minRank {
			minRank = r
		}
	}
	if minRank == 0 {
		return
	}
	for n := range ranks {
		ranks[n] -= minRank
	}
}
