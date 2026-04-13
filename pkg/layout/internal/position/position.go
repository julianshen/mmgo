// Package position assigns x,y coordinates to nodes after crossing
// minimization. This is the fourth phase of the dagre layout engine port.
//
// Given an Order from the order phase, Run computes a pixel position for
// each node such that:
//   - y is proportional to rank (rank * RankSep)
//   - Nodes within a rank don't overlap (respecting width + NodeSep)
//   - x minimizes edge lengths where possible by aligning nodes with the
//     median of their neighbors in adjacent ranks
//
// The current implementation uses a simplified median-based approach:
// initial per-rank centered packing, then iterative top-down/bottom-up
// median passes with order-preserving compaction. This produces good
// layouts for flowchart-shaped graphs where each node has few neighbors.
//
// TODO(perf): port dagre's Brandes-Kopf algorithm for optimal horizontal
// placement. Reference: Brandes & Kopf (2002), "Fast and Simple Horizontal
// Coordinate Assignment." The full algorithm does type-1 conflict
// detection, 4-orientation balancing (UL/UR/LL/LR), and produces tighter
// layouts for dense graphs. ~526 lines in dagre's position/bk.ts.
package position

import (
	"math"
	"slices"
	"sort"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/order"
)

// Point is an x,y coordinate in pixels.
type Point struct {
	X float64
	Y float64
}

// Result maps node IDs to their computed coordinates.
type Result map[string]Point

// NodeWidth returns the pixel width of a node. Callers typically build
// this from text measurement (width = label width + padding).
type NodeWidth func(id string) float64

// Options configures coordinate assignment.
type Options struct {
	// NodeSep is the minimum horizontal gap between adjacent nodes in
	// the same rank, in pixels.
	NodeSep float64
	// RankSep is the vertical distance between adjacent ranks, in pixels.
	RankSep float64
}

// alignmentPasses is the number of top-down/bottom-up median alignment
// passes. Typically converges within 2-3 iterations; 4 is a safe default.
const alignmentPasses = 4

// Run computes x,y coordinates for every node in ord.
//
// The algorithm:
//  1. Initial packing: each rank is laid out left-to-right with width +
//     NodeSep gaps, centered horizontally at x=0.
//  2. Alignment passes: alternately top-down (using predecessors) and
//     bottom-up (using successors), each node's x is shifted toward
//     the median of its neighbors' x in the adjacent rank.
//  3. Compaction: after each shift, enforce the order constraint with
//     a left-to-right pass that pushes overlapping nodes apart.
//  4. Normalize: shift all coordinates so the minimum x,y is 0.
func Run(g *graph.Graph, ord order.Order, widthFn NodeWidth, opts Options) Result {
	if len(ord) == 0 {
		return Result{}
	}

	ranksAsc := sortedRanks(ord)
	preds, succs := buildAdjacency(g)

	result := initialPacking(ord, ranksAsc, widthFn, opts)

	for pass := 0; pass < alignmentPasses; pass++ {
		if pass%2 == 0 {
			// Top-down: align each rank to predecessor medians.
			for i := 1; i < len(ranksAsc); i++ {
				r := ranksAsc[i]
				alignByMedian(result, ord[r], preds)
				compact(result, ord[r], widthFn, opts.NodeSep)
			}
		} else {
			// Bottom-up: align each rank to successor medians.
			for i := len(ranksAsc) - 2; i >= 0; i-- {
				r := ranksAsc[i]
				alignByMedian(result, ord[r], succs)
				compact(result, ord[r], widthFn, opts.NodeSep)
			}
		}
	}

	normalize(result)
	return result
}

// initialPacking lays out each rank left-to-right centered at x=0.
// y is set from the rank index.
func initialPacking(ord order.Order, ranksAsc []int, widthFn NodeWidth, opts Options) Result {
	result := make(Result)
	for _, r := range ranksAsc {
		nodes := ord[r]
		y := float64(r) * opts.RankSep

		totalWidth := 0.0
		for i, n := range nodes {
			if i > 0 {
				totalWidth += opts.NodeSep
			}
			totalWidth += widthFn(n)
		}

		x := -totalWidth / 2
		for _, n := range nodes {
			w := widthFn(n)
			x += w / 2
			result[n] = Point{X: x, Y: y}
			x += w/2 + opts.NodeSep
		}
	}
	return result
}

// alignByMedian shifts each node's x toward the median of its adjacent
// neighbors' x. Nodes with no neighbors in the adjacent rank are left
// unchanged. Order is not enforced here; compact handles that.
func alignByMedian(result Result, nodes []string, neighbors map[string][]string) {
	for _, n := range nodes {
		nbrs := neighbors[n]
		if len(nbrs) == 0 {
			continue
		}
		xs := make([]float64, 0, len(nbrs))
		for _, nbr := range nbrs {
			if p, ok := result[nbr]; ok {
				xs = append(xs, p.X)
			}
		}
		if len(xs) == 0 {
			continue
		}
		p := result[n]
		p.X = median(xs)
		result[n] = p
	}
}

// compact enforces the no-overlap order constraint and keeps the rank
// centered around the median of its target x values.
//
// A naive left-to-right push produces a rightward drift: when multiple
// nodes in a rank have the same target x (e.g., a fan-out where all
// children align to a single parent), pushing them apart pulls the
// compacted median to the right of the original target. To avoid this,
// we compact first, then translate the entire block so its compacted
// median matches the pre-compaction target median.
func compact(result Result, nodes []string, widthFn NodeWidth, nodeSep float64) {
	if len(nodes) == 0 {
		return
	}

	// Capture target positions (the x values set by alignByMedian)
	// before we start pushing nodes around.
	targets := make([]float64, len(nodes))
	for i, n := range nodes {
		targets[i] = result[n].X
	}
	targetMedian := median(targets)

	// Left-to-right compact: enforce the minimum gap between adjacent
	// nodes. The first node stays at its target; subsequent nodes move
	// right only if they would overlap.
	prevRight := math.Inf(-1)
	for _, n := range nodes {
		w := widthFn(n)
		p := result[n]
		halfW := w / 2
		if prevRight != math.Inf(-1) {
			minLeft := prevRight + nodeSep
			if p.X-halfW < minLeft {
				p.X = minLeft + halfW
				result[n] = p
			}
		}
		prevRight = p.X + halfW
	}

	// Recenter the compacted block so its median aligns with the
	// original target median. This cancels the rightward drift.
	compacted := make([]float64, len(nodes))
	for i, n := range nodes {
		compacted[i] = result[n].X
	}
	delta := targetMedian - median(compacted)
	if delta != 0 {
		for _, n := range nodes {
			p := result[n]
			p.X += delta
			result[n] = p
		}
	}
}

// median returns the median of a slice of floats. Uses lower-median for
// even counts (average of the two middle values).
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := slices.Clone(xs)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// normalize shifts all coordinates so the minimum x and minimum y are 0.
func normalize(result Result) {
	if len(result) == 0 {
		return
	}
	minX, minY := math.Inf(1), math.Inf(1)
	for _, p := range result {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
	}
	for n, p := range result {
		result[n] = Point{X: p.X - minX, Y: p.Y - minY}
	}
}

// buildAdjacency returns deduped predecessor and successor lists for every
// node in g. Called once before the alignment loop to avoid repeatedly
// calling g.Predecessors / g.Successors (which allocate fresh maps and
// slices on every invocation).
func buildAdjacency(g *graph.Graph) (preds, succs map[string][]string) {
	nodes := g.Nodes()
	preds = make(map[string][]string, len(nodes))
	succs = make(map[string][]string, len(nodes))
	for _, n := range nodes {
		preds[n] = g.Predecessors(n)
		succs[n] = g.Successors(n)
	}
	return preds, succs
}

// sortedRanks returns the rank numbers in ord in ascending order.
func sortedRanks(ord order.Order) []int {
	ranks := make([]int, 0, len(ord))
	for r := range ord {
		ranks = append(ranks, r)
	}
	slices.Sort(ranks)
	return ranks
}
