// Package acyclic breaks cycles in a directed graph by reversing a set of
// feedback edges. The reversed edges can later be restored via Undo.
//
// This is a port of the acyclic phase of dagrejs/dagre, implemented via the
// Greedy Feedback Arc Set heuristic (Eades, Lin, Smyth 1993). The exact
// minimum feedback arc set is NP-hard; this heuristic runs in O(|V| + |E|)
// and typically reverses a near-minimal set of edges in practice.
package acyclic

import (
	"sort"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// Run reverses a set of feedback edges in g so that the non-self-loop
// portion of the graph becomes acyclic. Self-loops are preserved.
//
// The returned slice contains the current EdgeIDs of the reversed edges
// (i.e., their direction after reversal). Pass this slice to Undo to
// restore the original edge directions.
func Run(g *graph.Graph) []graph.EdgeID {
	order := greedyOrdering(g)
	orderIdx := make(map[string]int, len(order))
	for i, n := range order {
		orderIdx[n] = i
	}

	// Edges whose source comes after their target in the ordering are
	// "back edges" — reverse them to break cycles. Self-loops are skipped
	// because reversing them has no effect.
	var backEdges []graph.EdgeID
	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		if orderIdx[eid.From] > orderIdx[eid.To] {
			backEdges = append(backEdges, eid)
		}
	}

	reversed := make([]graph.EdgeID, 0, len(backEdges))
	for _, eid := range backEdges {
		if newID, ok := g.ReverseEdge(eid); ok {
			reversed = append(reversed, newID)
		}
	}
	return reversed
}

// Undo reverses the edges listed in reversed, restoring their original
// directions. Safe to call with a nil or empty slice.
func Undo(g *graph.Graph, reversed []graph.EdgeID) {
	for _, eid := range reversed {
		g.ReverseEdge(eid)
	}
}

// greedyOrdering computes a node ordering using the Eades-Lin-Smyth
// greedy feedback arc set heuristic. The ordering is deterministic:
// ties are broken by alphabetical node ID.
//
// The algorithm repeatedly:
//  1. Removes sinks (out-degree 0 excluding self-loops), prepending them
//     to the right side of the result.
//  2. Removes sources (in-degree 0 excluding self-loops), appending them
//     to the left side.
//  3. If neither exists, picks the node with the highest
//     (out-degree - in-degree) and treats it as a source.
//
// Edges going "backward" in the final ordering approximate a minimum
// feedback arc set.
func greedyOrdering(g *graph.Graph) []string {
	work := g.Copy()

	var left, right []string

	for work.NodeCount() > 0 {
		for {
			sink := findSink(work)
			if sink == "" {
				break
			}
			right = append([]string{sink}, right...)
			work.RemoveNode(sink)
		}

		for {
			src := findSource(work)
			if src == "" {
				break
			}
			left = append(left, src)
			work.RemoveNode(src)
		}

		if work.NodeCount() > 0 {
			best := pickMaxDelta(work)
			left = append(left, best)
			work.RemoveNode(best)
		}
	}

	result := make([]string, 0, len(left)+len(right))
	result = append(result, left...)
	result = append(result, right...)
	return result
}

// findSink returns the alphabetically-first node with zero non-self-loop
// out-degree, or "" if none exists.
func findSink(g *graph.Graph) string {
	for _, n := range sortedNodes(g) {
		if outDegreeNoSelf(g, n) == 0 {
			return n
		}
	}
	return ""
}

// findSource returns the alphabetically-first node with zero non-self-loop
// in-degree, or "" if none exists.
func findSource(g *graph.Graph) string {
	for _, n := range sortedNodes(g) {
		if inDegreeNoSelf(g, n) == 0 {
			return n
		}
	}
	return ""
}

// pickMaxDelta returns the node with the highest (out_degree - in_degree),
// excluding self-loops. Ties break alphabetically. Panics if g is empty.
func pickMaxDelta(g *graph.Graph) string {
	nodes := sortedNodes(g)
	best := nodes[0]
	bestDelta := outDegreeNoSelf(g, best) - inDegreeNoSelf(g, best)
	for _, n := range nodes[1:] {
		d := outDegreeNoSelf(g, n) - inDegreeNoSelf(g, n)
		if d > bestDelta {
			best = n
			bestDelta = d
		}
	}
	return best
}

// outDegreeNoSelf counts outgoing edges from n excluding self-loops.
func outDegreeNoSelf(g *graph.Graph, n string) int {
	count := 0
	for _, eid := range g.OutEdges(n) {
		if eid.To != n {
			count++
		}
	}
	return count
}

// inDegreeNoSelf counts incoming edges to n excluding self-loops.
func inDegreeNoSelf(g *graph.Graph, n string) int {
	count := 0
	for _, eid := range g.InEdges(n) {
		if eid.From != n {
			count++
		}
	}
	return count
}

func sortedNodes(g *graph.Graph) []string {
	nodes := g.Nodes()
	sort.Strings(nodes)
	return nodes
}
