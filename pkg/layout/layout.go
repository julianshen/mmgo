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
// routing, dummy-node insertion for long-span edges, and edge-label
// collision avoidance are not yet implemented. Long-span edges are
// currently rendered as straight lines. See individual internal
// package TODOs for details.
package layout

import (
	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/acyclic"
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
// dimensions default to defaultNodeWidth × defaultNodeHeight.
func Layout(g *graph.Graph, opts Options) *Result {
	if opts.NodeSep <= 0 {
		opts.NodeSep = DefaultNodeSep
	}
	if opts.RankSep <= 0 {
		opts.RankSep = DefaultRankSep
	}

	if g.NodeCount() == 0 {
		return &Result{
			Nodes: map[string]NodeLayout{},
			Edges: map[graph.EdgeID]EdgeLayout{},
		}
	}

	work := g.Copy()

	// Precompute widths once so the position phase isn't calling
	// NodeAttrs in its hot loop.
	widths := make(map[string]float64, work.NodeCount())
	for _, id := range work.Nodes() {
		attrs, _ := work.NodeAttrs(id)
		w, _ := effectiveSize(attrs)
		widths[id] = w
	}
	widthFn := func(id string) float64 { return widths[id] }

	acyclic.Run(work)
	ranks := rank.Run(work)
	ord := order.Run(work, ranks)
	coords := position.Run(work, ord, widthFn, position.Options{
		NodeSep: opts.NodeSep,
		RankSep: opts.RankSep,
	})

	applyRankDir(coords, opts.RankDir)

	// Build the output from the original graph g so the caller sees
	// their node IDs and original edge directions — not the internals
	// of the work copy.
	nodes := buildNodeLayouts(g, coords)
	edges := buildEdgeLayouts(g, coords)
	width, height := computeBounds(nodes)

	return &Result{
		Nodes:  nodes,
		Edges:  edges,
		Width:  width,
		Height: height,
	}
}

// buildNodeLayouts creates the public NodeLayout map from the original
// graph and the computed coordinates.
func buildNodeLayouts(g *graph.Graph, coords position.Result) map[string]NodeLayout {
	nodes := make(map[string]NodeLayout, g.NodeCount())
	for _, id := range g.Nodes() {
		attrs, _ := g.NodeAttrs(id)
		w, h := effectiveSize(attrs)
		p := coords[id]
		nodes[id] = NodeLayout{X: p.X, Y: p.Y, Width: w, Height: h}
	}
	return nodes
}

// buildEdgeLayouts creates the public EdgeLayout map from the original
// graph and the computed coordinates. Uses straight-line routing from
// source center to target center; the label position is the midpoint.
//
// TODO(features): orthogonal polyline routing, curve fitting, self-loop
// geometry, and collision avoidance are not implemented. See package doc.
func buildEdgeLayouts(g *graph.Graph, coords position.Result) map[graph.EdgeID]EdgeLayout {
	edges := make(map[graph.EdgeID]EdgeLayout, g.EdgeCount())
	for _, eid := range g.Edges() {
		src := coords[eid.From]
		dst := coords[eid.To]
		edges[eid] = EdgeLayout{
			Points: []Point{src, dst},
			LabelPos: Point{
				X: (src.X + dst.X) / 2,
				Y: (src.Y + dst.Y) / 2,
			},
		}
	}
	return edges
}

// applyRankDir transforms coordinates produced by the position phase
// (which always lays out in TB form) into the requested rank direction.
// TB and zero value are no-ops.
//
// Geometry:
//   - BT: flip the Y axis so rank 0 ends up at the bottom.
//   - LR: swap X and Y so rank progression runs horizontally.
//   - RL: like LR but with X flipped so rank 0 ends up at the right.
func applyRankDir(coords position.Result, dir RankDir) {
	if dir == RankDirTB || len(coords) == 0 {
		return
	}

	// LR is a pure axis swap; no bounds scan needed.
	if dir == RankDirLR {
		for n, p := range coords {
			coords[n] = position.Point{X: p.Y, Y: p.X}
		}
		return
	}

	// BT and RL both need the post-TB maxY to flip around.
	var maxY float64
	for _, p := range coords {
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	switch dir {
	case RankDirBT:
		for n, p := range coords {
			coords[n] = position.Point{X: p.X, Y: maxY - p.Y}
		}
	case RankDirRL:
		// After swapping axes (LR), the old Y range becomes the new X
		// range. Flipping that range puts rank 0 on the right.
		for n, p := range coords {
			coords[n] = position.Point{X: maxY - p.Y, Y: p.X}
		}
	}
}

// computeBounds returns the overall width and height of the laid-out
// graph, including each node's half-dimensions beyond its center point.
func computeBounds(nodes map[string]NodeLayout) (width, height float64) {
	for _, nl := range nodes {
		right := nl.X + nl.Width/2
		bottom := nl.Y + nl.Height/2
		if right > width {
			width = right
		}
		if bottom > height {
			height = bottom
		}
	}
	return width, height
}
