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

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/layoututil"
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
	// RankSep is the minimum gap between rank boundaries (not centers),
	// in pixels.
	RankSep float64
	// RankDim returns the rank-axis extent of a node — the height for
	// TB/BT and the width for LR/RL. Used to space ranks so that nodes
	// with large rank-axis extents don't overlap across rank boundaries.
	// If nil, ranks are placed at fixed `r * RankSep` intervals (the
	// pre-1.0 behavior, valid only when rank-axis node sizes are small
	// relative to RankSep).
	RankDim NodeWidth
}

// alignmentPasses is the number of top-down/bottom-up median alignment
// passes. Typically converges within 2-3 iterations; 4 is a safe default.
// Brandes-Kopf (tracked above as TODO) would replace this iterative
// approach with a closed-form solution.
const alignmentPasses = 4

// Run computes x,y coordinates for every node in ord.
func Run(g *graph.Graph, ord order.Order, widthFn NodeWidth, opts Options) Result {
	if len(ord) == 0 {
		return Result{}
	}

	ranksAsc := layoututil.SortedRanks(ord)
	preds, succs := layoututil.BuildAdjacency(g)

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
// y (the rank-axis coordinate) is accumulated from each rank's maximum
// rank-axis extent so that nodes with large perpendicular-to-pack
// dimensions don't overlap across rank boundaries.
func initialPacking(ord order.Order, ranksAsc []int, widthFn NodeWidth, opts Options) Result {
	result := make(Result)
	// Precompute the max rank-axis extent per rank. When RankDim is
	// nil (the pre-1.0 behavior), fall back to fixed-interval spacing.
	maxRankDim := make([]float64, len(ranksAsc))
	if opts.RankDim != nil {
		for i, r := range ranksAsc {
			for _, n := range ord[r] {
				if d := opts.RankDim(n); d > maxRankDim[i] {
					maxRankDim[i] = d
				}
			}
		}
	}
	y := 0.0
	for i, r := range ranksAsc {
		nodes := ord[r]
		if opts.RankDim == nil {
			y = float64(r) * opts.RankSep
		} else {
			if i == 0 {
				y = maxRankDim[i] / 2
			} else {
				y += maxRankDim[i-1]/2 + opts.RankSep + maxRankDim[i]/2
			}
		}

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
	// Scratch buffer reused across nodes in this rank — one allocation
	// per rank instead of per node.
	var xs []float64
	for _, n := range nodes {
		nbrs := neighbors[n]
		if len(nbrs) == 0 {
			continue
		}
		xs = xs[:0]
		for _, nbr := range nbrs {
			if p, ok := result[nbr]; ok {
				xs = append(xs, p.X)
			}
		}
		if len(xs) == 0 {
			continue
		}
		p := result[n]
		p.X = medianInPlace(xs)
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
	// before we start pushing nodes around. The same scratch buffer is
	// reused below to capture compacted positions — medianInPlace sorts
	// in place, so we compute targetMedian first, then reuse the backing
	// array.
	scratch := make([]float64, len(nodes))
	for i, n := range nodes {
		scratch[i] = result[n].X
	}
	targetMedian := medianInPlace(scratch)

	// Left-to-right compact: enforce the minimum gap between adjacent
	// nodes. The first node stays at its target.
	first := result[nodes[0]]
	prevRight := first.X + widthFn(nodes[0])/2
	for _, n := range nodes[1:] {
		w := widthFn(n)
		p := result[n]
		halfW := w / 2
		minLeft := prevRight + nodeSep
		if p.X-halfW < minLeft {
			p.X = minLeft + halfW
			result[n] = p
		}
		prevRight = p.X + halfW
	}

	// Recompute median and shift the whole block so the compacted
	// median aligns with the original target median.
	for i, n := range nodes {
		scratch[i] = result[n].X
	}
	delta := targetMedian - medianInPlace(scratch)
	if delta != 0 {
		for _, n := range nodes {
			p := result[n]
			p.X += delta
			result[n] = p
		}
	}
}

// medianInPlace sorts xs and returns its median. xs is mutated. Callers
// pass a scratch slice they no longer need.
func medianInPlace(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	n := len(xs)
	if n%2 == 1 {
		return xs[n/2]
	}
	return (xs[n/2-1] + xs[n/2]) / 2
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
	if minX == 0 && minY == 0 {
		return
	}
	for n, p := range result {
		result[n] = Point{X: p.X - minX, Y: p.Y - minY}
	}
}
