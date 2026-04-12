// Package acyclic breaks cycles in a directed graph by reversing a set of
// feedback edges. The reversed edges can later be restored via Undo.
//
// This is a port of the acyclic phase of dagrejs/dagre, implemented via the
// Greedy Feedback Arc Set heuristic (Eades, Lin, Smyth 1993). The exact
// minimum feedback arc set is NP-hard; this heuristic runs in O(|V| + |E|)
// and typically reverses a near-minimal set of edges in practice.
package acyclic

import (
	"fmt"
	"slices"

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
		newID, ok := g.ReverseEdge(eid)
		if !ok {
			// Unreachable under normal Run flow: backEdges comes from
			// g.Edges() with no intervening mutation. Defensive skip.
			continue
		}
		reversed = append(reversed, newID)
	}
	return reversed
}

// Undo reverses the edges listed in reversed, restoring their original
// directions. Safe to call with a nil or empty slice.
//
// Panics if any EdgeID in reversed no longer exists in g. This indicates
// that g was mutated between Run and Undo — a programming error that would
// otherwise produce silent, hard-to-diagnose layout bugs.
func Undo(g *graph.Graph, reversed []graph.EdgeID) {
	for _, eid := range reversed {
		if _, ok := g.ReverseEdge(eid); !ok {
			panic(fmt.Sprintf("acyclic.Undo: edge %v not found; graph was mutated between Run and Undo", eid))
		}
	}
}

// greedyOrdering computes a node ordering using the Eades-Lin-Smyth
// greedy feedback arc set heuristic. The ordering is deterministic:
// ties are broken by alphabetical node ID.
//
// The algorithm repeatedly:
//  1. Removes all sinks (nodes with non-self out-degree 0), appending
//     them to a list that will form the tail of the final ordering.
//  2. Removes all sources (nodes with non-self in-degree 0), appending
//     them to a list that forms the head.
//  3. If neither exists, picks the node with the highest
//     (out-degree - in-degree) and treats it as a source.
//
// Edges going "backward" in the final ordering approximate a minimum
// feedback arc set.
func greedyOrdering(g *graph.Graph) []string {
	work := g.Copy()

	var head []string      // sources in discovery order
	var tailRev []string   // sinks in discovery order (reversed before emit)

	for work.NodeCount() > 0 {
		// Snapshot degrees once per iteration. Re-snapshotting after each
		// drain batch keeps counts consistent as we remove nodes.
		for {
			degs := computeDegrees(work)
			drained := false

			// Drain all sinks.
			for _, n := range sortedNodes(work) {
				if degs[n].out == 0 {
					tailRev = append(tailRev, n)
					work.RemoveNode(n)
					drained = true
				}
			}
			if drained {
				continue
			}

			// Drain all sources.
			for _, n := range sortedNodes(work) {
				if degs[n].in == 0 {
					head = append(head, n)
					work.RemoveNode(n)
					drained = true
				}
			}
			if drained {
				continue
			}

			break
		}

		// If the graph still has nodes, pick the one with the highest
		// out-in degree delta and treat it as a source.
		if work.NodeCount() > 0 {
			best := pickMaxDelta(work)
			head = append(head, best)
			work.RemoveNode(best)
		}
	}

	// Assemble the final order: head ++ reverse(tailRev).
	result := make([]string, 0, len(head)+len(tailRev))
	result = append(result, head...)
	for i := len(tailRev) - 1; i >= 0; i-- {
		result = append(result, tailRev[i])
	}
	return result
}

// degrees holds the non-self-loop in- and out-degree of a node.
type degrees struct {
	in  int
	out int
}

// computeDegrees returns a map of non-self-loop in/out degrees for every
// node in g. Self-loops are excluded so they don't interfere with source
// and sink detection.
func computeDegrees(g *graph.Graph) map[string]degrees {
	result := make(map[string]degrees, g.NodeCount())
	for _, n := range g.Nodes() {
		result[n] = degrees{}
	}
	for _, eid := range g.Edges() {
		if eid.From == eid.To {
			continue
		}
		d := result[eid.From]
		d.out++
		result[eid.From] = d
		d = result[eid.To]
		d.in++
		result[eid.To] = d
	}
	return result
}

// pickMaxDelta returns the node with the highest (out_degree - in_degree),
// excluding self-loops. Ties break alphabetically.
func pickMaxDelta(g *graph.Graph) string {
	degs := computeDegrees(g)
	nodes := sortedNodes(g)
	best := nodes[0]
	bestDelta := degs[best].out - degs[best].in
	for _, n := range nodes[1:] {
		d := degs[n].out - degs[n].in
		if d > bestDelta {
			best = n
			bestDelta = d
		}
	}
	return best
}

func sortedNodes(g *graph.Graph) []string {
	nodes := g.Nodes()
	slices.Sort(nodes)
	return nodes
}
