// Package layout is the top-level layout engine for directed graphs.
// It orchestrates the four layout phases (acyclic, rank, order, position)
// to produce x,y coordinates for every node and a set of control points
// for every edge.
//
// Typical usage:
//
//	g := graph.New()
//	// ... populate g with nodes and edges, setting NodeAttrs.Width/Height ...
//	result := layout.Layout(g, layout.Options{
//	    NodeSep: 50,
//	    RankSep: 50,
//	    RankDir: layout.RankDirTB,
//	})
//	for id, nl := range result.Nodes {
//	    // nl.X, nl.Y is the node's center point
//	    // nl.Width, nl.Height are its dimensions
//	}
//	for eid, el := range result.Edges {
//	    // el.Points is a polyline from source center to target center
//	}
//
// The input graph is not mutated. Layout operates on an internal copy
// and restores the original edge directions in the output.
//
// TODO(features): subgraph (compound graph) support, orthogonal edge
// routing, and edge-label collision avoidance are not yet implemented.
// See individual internal package TODOs for details.
package layout

import (
	"math"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/acyclic"
	"github.com/julianshen/mmgo/pkg/layout/internal/dummy"
	"github.com/julianshen/mmgo/pkg/layout/internal/layoututil"
	"github.com/julianshen/mmgo/pkg/layout/internal/order"
	"github.com/julianshen/mmgo/pkg/layout/internal/position"
	"github.com/julianshen/mmgo/pkg/layout/internal/rank"
)

// RankDir is the direction of rank progression in the layout. The zero
// value (RankDirTB) is top-to-bottom, matching dagre's default.
type RankDir int8

const (
	RankDirTB RankDir = iota // top to bottom (default)
	RankDirBT                // bottom to top
	RankDirLR                // left to right
	RankDirRL                // right to left
)

var rankDirNames = []string{"TB", "BT", "LR", "RL"}

// String returns the canonical two-letter keyword for the direction.
func (d RankDir) String() string {
	if int(d) < 0 || int(d) >= len(rankDirNames) {
		return "unknown"
	}
	return rankDirNames[d]
}

// Default layout spacing values, applied when Options fields are zero.
const (
	DefaultNodeSep = 50.0
	DefaultRankSep = 50.0
)

// Options configures the layout engine.
type Options struct {
	// NodeSep is the minimum horizontal gap between adjacent nodes in
	// the same rank (in pixels). Default DefaultNodeSep if zero.
	NodeSep float64
	// RankSep is the vertical distance between adjacent ranks (in pixels).
	// Default DefaultRankSep if zero.
	RankSep float64
	// RankDir is the direction of rank progression. Default RankDirTB
	// (which is also the zero value).
	RankDir RankDir
}

// Point is an x,y coordinate in pixels. Aliased from position.Point so
// callers don't need to import the internal position package.
type Point = position.Point

// NodeLayout holds the computed geometry of a single node. X,Y is the
// node's center point; Width and Height are the dimensions the caller
// supplied via graph.NodeAttrs.
type NodeLayout struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// EdgeLayout holds the computed geometry of a single edge.
type EdgeLayout struct {
	// Points is a polyline of control points from source center to
	// target center. For straight-line routing, this has exactly 2
	// points; future orthogonal routing may emit more.
	Points []Point
	// LabelPos is the suggested position for the edge's label.
	LabelPos Point
}

// Result is the output of the layout engine.
type Result struct {
	// Nodes maps node ID to its computed geometry.
	Nodes map[string]NodeLayout
	// Edges maps edge ID (from the original graph) to its computed
	// geometry. Edge directions match the original graph's directions,
	// regardless of any internal reversals done during the acyclic phase.
	Edges map[graph.EdgeID]EdgeLayout
	// Width and Height are the overall bounding box of the laid-out graph.
	Width  float64
	Height float64
}

const (
	defaultNodeWidth  = 100
	defaultNodeHeight = 50
)

// effectiveSize returns the node's width and height, falling back to
// defaultNodeWidth/defaultNodeHeight when attrs.Width or attrs.Height
// is zero or negative. Centralizes the default-dimension convention
// used by both the position pass and the final NodeLayout build.
func effectiveSize(attrs graph.NodeAttrs) (w, h float64) {
	w, h = attrs.Width, attrs.Height
	if w <= 0 {
		w = defaultNodeWidth
	}
	if h <= 0 {
		h = defaultNodeHeight
	}
	return w, h
}

// Layout computes positions for all nodes and edges in g. The input
// graph is not mutated.
//
// Each node's width and height are read from graph.NodeAttrs. Unset
// dimensions default to 100 × 50.
//
// Layout(nil, opts) returns an empty Result rather than panicking, so
// callers can treat "no graph" uniformly with "empty graph".
func Layout(g *graph.Graph, opts Options) *Result {
	if opts.NodeSep <= 0 {
		opts.NodeSep = DefaultNodeSep
	}
	if opts.RankSep <= 0 {
		opts.RankSep = DefaultRankSep
	}
	if g == nil {
		return &Result{
			Nodes: map[string]NodeLayout{},
			Edges: map[graph.EdgeID]EdgeLayout{},
		}
	}

	work := g.Copy()

	// Precompute node sizes once. packingDim is handed to the position
	// phase; the other dimension is reused later when building NodeLayout.
	sizes := precomputeSizes(work)

	// Position uses "width" to space nodes along its X axis. For LR/RL
	// rank directions, that axis becomes the final Y axis after
	// transformPoint swaps the coordinates, so we must pack by HEIGHT to
	// avoid vertical overlap of tall-narrow nodes in vertical columns.
	// For TB/BT the packing axis is horizontal and width is correct.
	packingDim := func(id string) float64 {
		sz := sizes[id]
		if opts.RankDir == RankDirLR || opts.RankDir == RankDirRL {
			return sz.height
		}
		return sz.width
	}
	// rankDim is the dimension PARALLEL to rank progression — height
	// for TB/BT, width for LR/RL. Without it, rank spacing would be
	// a fixed RankSep regardless of node size, causing wide nodes to
	// overlap across ranks in LR layouts.
	rankDim := func(id string) float64 {
		sz := sizes[id]
		if opts.RankDir == RankDirLR || opts.RankDir == RankDirRL {
			return sz.width
		}
		return sz.height
	}

	acyclic.Run(work)
	ranks := rank.Run(work)
	// Dummies route long-span edges through intermediate-rank
	// waypoints; bounds + the public Nodes map filter them out.
	dRes := dummy.Run(work, ranks)
	for _, id := range dRes.Dummies {
		sizes[id] = nodeSize{width: 1, height: 1}
	}
	ord := order.Run(work, ranks)
	coords := position.Run(work, ord, packingDim, position.Options{
		NodeSep: opts.NodeSep,
		RankSep: opts.RankSep,
		RankDim: rankDim,
	})

	nodes, offsetX, offsetY, width, height := buildNodesAndBounds(g, coords, sizes, opts.RankDir)
	edges := buildEdges(g, ranks, dRes.Chains, coords, opts.RankDir, nodes, offsetX, offsetY)

	return &Result{
		Nodes:  nodes,
		Edges:  edges,
		Width:  width,
		Height: height,
	}
}

// nodeSize bundles a node's width and height after default resolution.
type nodeSize struct{ width, height float64 }

// precomputeSizes returns the effective (width, height) for every node
// in g, resolving defaults once. Both the position phase (via widthFn)
// and the final NodeLayout construction consume this map.
func precomputeSizes(g *graph.Graph) map[string]nodeSize {
	sizes := make(map[string]nodeSize, g.NodeCount())
	for _, id := range g.Nodes() {
		attrs, _ := g.NodeAttrs(id)
		w, h := effectiveSize(attrs)
		sizes[id] = nodeSize{width: w, height: h}
	}
	return sizes
}

// buildNodesAndBounds fuses three former passes into one walk:
//  1. apply the rank-direction transform to each coord
//  2. build the public NodeLayout map
//  3. compute a proper min/max bounding box and translate so the
//     top-left corner is (0, 0)
//
// The bounding box is a real min/max scan rather than "max(X+W/2)"
// on the assumption that coords start at (0,0). That assumption was
// implicit before and would silently break if any future layout phase
// produced negative coordinates.
func buildNodesAndBounds(
	g *graph.Graph,
	coords position.Result,
	sizes map[string]nodeSize,
	dir RankDir,
) (nodes map[string]NodeLayout, offsetX, offsetY, width, height float64) {
	nodes = make(map[string]NodeLayout, g.NodeCount())
	if g.NodeCount() == 0 {
		return nodes, 0, 0, 0, 0
	}

	// For BT and RL the direction transform needs the post-TB max Y to
	// flip around. Include dummy Y values because dummies can lie
	// beyond any real node at intermediate ranks.
	var flipAround float64
	if dir == RankDirBT || dir == RankDirRL {
		for _, p := range coords {
			if p.Y > flipAround {
				flipAround = p.Y
			}
		}
	}

	// Bounds from REAL nodes only — dummies must not inflate the
	// viewBox past the rect extent of the real diagram.
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	transformed := make(map[string]Point, len(sizes))
	for id, sz := range sizes {
		p := transformPoint(coords[id], dir, flipAround)
		transformed[id] = p
		if dummy.IsDummy(id) {
			continue
		}
		left := p.X - sz.width/2
		right := p.X + sz.width/2
		top := p.Y - sz.height/2
		bottom := p.Y + sz.height/2
		if left < minX {
			minX = left
		}
		if right > maxX {
			maxX = right
		}
		if top < minY {
			minY = top
		}
		if bottom > maxY {
			maxY = bottom
		}
	}

	for id, sz := range sizes {
		if dummy.IsDummy(id) {
			continue
		}
		p := transformed[id]
		nodes[id] = NodeLayout{
			X:      p.X - minX,
			Y:      p.Y - minY,
			Width:  sz.width,
			Height: sz.height,
		}
	}
	return nodes, minX, minY, maxX - minX, maxY - minY
}

// transformPoint applies the rank-direction coordinate transform to a
// single point. TB is the identity. Geometry:
//   - BT: flip the Y axis (rank 0 ends at the bottom).
//   - LR: swap X and Y (rank progression runs horizontally).
//   - RL: swap and flip X (rank 0 ends at the right).
func transformPoint(p position.Point, dir RankDir, flipAround float64) Point {
	switch dir {
	case RankDirBT:
		return Point{X: p.X, Y: flipAround - p.Y}
	case RankDirLR:
		return Point{X: p.Y, Y: p.X}
	case RankDirRL:
		return Point{X: flipAround - p.Y, Y: p.X}
	default:
		// RankDirTB or any unknown value — identity.
		return p
	}
}

// buildEdges threads each original edge's polyline through any dummy
// chain inserted by dummy.Run. Two-point straight-line edges remain
// 2 points; longer edges become rank-by-rank polylines.
//
// Direction: dummies are inserted in rank-ascending order, so for a
// back-edge (ranks[origFrom] > ranks[origTo]) the chain runs against
// the polyline direction and must be reversed.
//
// Multi-edges between the same rank-ordered pair consume distinct
// chains in CompareEdgeIDs order — same order dummy.Run produced.
func buildEdges(
	g *graph.Graph,
	ranks map[string]int,
	chains map[dummy.Key][]dummy.Chain,
	coords position.Result,
	dir RankDir,
	nodes map[string]NodeLayout,
	offsetX, offsetY float64,
) map[graph.EdgeID]EdgeLayout {
	var flipAround float64
	if dir == RankDirBT || dir == RankDirRL {
		for _, p := range coords {
			if p.Y > flipAround {
				flipAround = p.Y
			}
		}
	}
	pointOf := func(id string) Point {
		if nl, ok := nodes[id]; ok {
			return Point{X: nl.X, Y: nl.Y}
		}
		p := transformPoint(coords[id], dir, flipAround)
		return Point{X: p.X - offsetX, Y: p.Y - offsetY}
	}

	origEdges := g.Edges()
	layoututil.SortEdges(origEdges)

	consumed := make(map[dummy.Key]int)

	edges := make(map[graph.EdgeID]EdgeLayout, len(origEdges))
	for _, eid := range origEdges {
		srcPt := pointOf(eid.From)
		dstPt := pointOf(eid.To)
		pts := []Point{srcPt, dstPt}

		switch {
		case eid.From == eid.To:
			// Self-loop: no chain.
		case ranks[eid.From] <= ranks[eid.To]:
			pts = applyChain(pts, srcPt, dstPt, chains, consumed,
				dummy.Key{From: eid.From, To: eid.To}, false, pointOf)
		default:
			pts = applyChain(pts, srcPt, dstPt, chains, consumed,
				dummy.Key{From: eid.To, To: eid.From}, true, pointOf)
		}

		edges[eid] = EdgeLayout{
			Points:   pts,
			LabelPos: midpointOf(pts),
		}
	}
	return edges
}

// applyChain returns the polyline for an edge, threading through the
// next chain available at key. Pass reversed=true when the chain's
// rank-ascending order is the OPPOSITE of the polyline direction
// (back-edges).
func applyChain(
	pts []Point, srcPt, dstPt Point,
	chains map[dummy.Key][]dummy.Chain, consumed map[dummy.Key]int,
	key dummy.Key, reversed bool,
	pointOf func(string) Point,
) []Point {
	idx := consumed[key]
	consumed[key] = idx + 1
	if idx >= len(chains[key]) {
		return pts
	}
	chain := chains[key][idx]
	out := make([]Point, 0, len(chain.Dummies)+2)
	out = append(out, srcPt)
	if reversed {
		for i := len(chain.Dummies) - 1; i >= 0; i-- {
			out = append(out, pointOf(chain.Dummies[i]))
		}
	} else {
		for _, d := range chain.Dummies {
			out = append(out, pointOf(d))
		}
	}
	out = append(out, dstPt)
	return out
}

// midpointOf returns the polyline midpoint by length. For a 2-point
// segment it's the average of the endpoints; for longer polylines
// it lands on the middle segment, which keeps edge labels readable
// when the polyline bends at a dummy.
func midpointOf(pts []Point) Point {
	if len(pts) < 2 {
		return Point{}
	}
	if len(pts) == 2 {
		return Point{X: (pts[0].X + pts[1].X) / 2, Y: (pts[0].Y + pts[1].Y) / 2}
	}
	mid := len(pts) / 2
	a, b := pts[mid-1], pts[mid]
	return Point{X: (a.X + b.X) / 2, Y: (a.Y + b.Y) / 2}
}
