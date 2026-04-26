package rank

import (
	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// optimize tightens longest-path ranks by repeatedly cutting the
// tight subgraph along a slack edge and pulling the looser side
// closer. This is a simplified network simplex pass: each iteration
// picks one non-tight edge, identifies its source-side tight
// component (the "locked" set), and shifts the destination-side
// component down by the largest amount that keeps every edge
// crossing the cut feasible (rank diff >= minLen).
func optimize(g *graph.Graph, ranks map[string]int) {
	allEdges := g.Edges()
	// Safety cap: every successful iteration strictly decreases the
	// total slack of the chosen cut by >=1, and total slack is
	// bounded by len(edges)*max(rank). A cap proportional to that
	// product prevents runaway in case of a future bug breaking the
	// monotonic-progress invariant.
	maxIters := len(allEdges) * (len(g.Nodes()) + 1)
	for iter := 0; iter < maxIters; iter++ {
		tightSet := buildTightSet(allEdges, g, ranks)
		if len(tightSet) == 0 {
			break
		}

		type candidate struct {
			eid   graph.EdgeID
			slack int
		}
		var candidates []candidate
		for _, eid := range allEdges {
			if tightSet[eid] {
				continue
			}
			attrs, _ := g.EdgeAttrs(eid)
			slack := ranks[eid.To] - ranks[eid.From] - attrs.EffectiveMinLen()
			if slack > 0 {
				candidates = append(candidates, candidate{eid, slack})
			}
		}
		if len(candidates) == 0 {
			break
		}

		improved := false
		for _, c := range candidates {
			locked := reachLocked(g, c.eid.From, tightSet)
			visited := reachUnlocked(g, c.eid.To, locked)
			if len(visited) == 0 {
				continue
			}
			// Cap the shift by the smallest slack on any edge from
			// locked to visited. Shifting further would push that
			// edge's rank diff below minLen and corrupt the layout.
			shift := minCutSlack(g, locked, visited, ranks, c.slack)
			if shift <= 0 {
				continue
			}
			for n := range visited {
				ranks[n] -= shift
			}
			improved = true
			break
		}
		if !improved {
			break
		}
	}
	normalize(ranks)
}

// reachLocked returns the connected component of start in the tight
// subgraph (treating tight edges as undirected). These nodes anchor
// the cut and must not move.
func reachLocked(g *graph.Graph, start string, tightSet map[graph.EdgeID]bool) map[string]bool {
	locked := make(map[string]bool)
	stack := []string{start}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if locked[node] {
			continue
		}
		locked[node] = true
		for _, eid := range g.OutEdges(node) {
			if tightSet[eid] && !locked[eid.To] {
				stack = append(stack, eid.To)
			}
		}
		for _, eid := range g.InEdges(node) {
			if tightSet[eid] && !locked[eid.From] {
				stack = append(stack, eid.From)
			}
		}
	}
	return locked
}

// reachUnlocked walks the full graph (any edge direction, ignoring
// tightness) from start, stopping at locked nodes. The returned set
// is the partition that will move.
func reachUnlocked(g *graph.Graph, start string, locked map[string]bool) map[string]bool {
	if locked[start] {
		return nil
	}
	visited := make(map[string]bool)
	stack := []string{start}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[node] || locked[node] {
			continue
		}
		visited[node] = true
		for _, eid := range g.OutEdges(node) {
			if !locked[eid.To] && !visited[eid.To] {
				stack = append(stack, eid.To)
			}
		}
		for _, eid := range g.InEdges(node) {
			if !locked[eid.From] && !visited[eid.From] {
				stack = append(stack, eid.From)
			}
		}
	}
	return visited
}

// minCutSlack returns the largest safe shift amount: the minimum of
// candidateSlack and the slack on every edge crossing from locked to
// visited. Edges in the reverse direction (visited→locked) only gain
// slack when visited shifts down, so they don't constrain the shift.
func minCutSlack(g *graph.Graph, locked, visited map[string]bool, ranks map[string]int, candidateSlack int) int {
	best := candidateSlack
	for v := range visited {
		for _, eid := range g.InEdges(v) {
			if !locked[eid.From] {
				continue
			}
			attrs, _ := g.EdgeAttrs(eid)
			slack := ranks[eid.To] - ranks[eid.From] - attrs.EffectiveMinLen()
			if slack < best {
				best = slack
			}
		}
	}
	return best
}

func buildTightSet(allEdges []graph.EdgeID, g *graph.Graph, ranks map[string]int) map[graph.EdgeID]bool {
	tight := make(map[graph.EdgeID]bool, len(allEdges))
	for _, eid := range allEdges {
		attrs, _ := g.EdgeAttrs(eid)
		if ranks[eid.To]-ranks[eid.From] == attrs.EffectiveMinLen() {
			tight[eid] = true
		}
	}
	return tight
}
