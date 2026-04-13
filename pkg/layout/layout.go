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
//	    RankSep: 100,
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

// RankDir is the direction of rank progression in the layout.
type RankDir string

const (
	RankDirTB RankDir = "TB" // top to bottom (default)
	RankDirBT RankDir = "BT" // bottom to top
	RankDirLR RankDir = "LR" // left to right
	RankDirRL RankDir = "RL" // right to left
)

// Options configures the layout engine.
type Options struct {
	// NodeSep is the minimum horizontal gap between adjacent nodes in
	// the same rank (in pixels). Default 50 if zero.
	NodeSep float64
	// RankSep is the vertical distance between adjacent ranks (in pixels).
	// Default 50 if zero.
	RankSep float64
	// RankDir is the direction of rank progression. Default RankDirTB.
	RankDir RankDir
}

// Point is an x,y coordinate in pixels.
type Point struct {
	X float64
	Y float64
}

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

// defaultNodeWidth is used when a caller supplies NodeAttrs with no
// Width set. Avoids zero-width nodes that would collapse to a single
// column during position assignment.
const (
	defaultNodeWidth  = 100
	defaultNodeHeight = 50
)

// Layout computes positions for all nodes and edges in g. The input
// graph is not mutated.
//
// Each node's width and height are read from graph.NodeAttrs. Unset
// dimensions default to 100×50.
func Layout(g *graph.Graph, opts Options) *Result {
	// Apply option defaults.
	if opts.NodeSep <= 0 {
		opts.NodeSep = 50
	}
	if opts.RankSep <= 0 {
		opts.RankSep = 50
	}
	if opts.RankDir == "" {
		opts.RankDir = RankDirTB
	}

	if g.NodeCount() == 0 {
		return &Result{
			Nodes: map[string]NodeLayout{},
			Edges: map[graph.EdgeID]EdgeLayout{},
		}
	}

	// Work on a copy so the caller's graph is untouched.
	work := g.Copy()

	// Phase 1: break cycles by reversing back edges. The returned
	// reversed IDs aren't needed here — we build the edge output from
	// the original graph g, whose directions are already correct.
	acyclic.Run(work)

	// Phase 2: assign integer rank (layer) to each node.
	ranks := rank.Run(work)

	// Phase 3: order nodes within each rank to minimize crossings.
	ord := order.Run(work, ranks)

	// Phase 4: compute x,y coordinates.
	widthFn := func(id string) float64 {
		attrs, _ := work.NodeAttrs(id)
		if attrs.Width > 0 {
			return attrs.Width
		}
		return defaultNodeWidth
	}
	coords := position.Run(work, ord, widthFn, position.Options{
		NodeSep: opts.NodeSep,
		RankSep: opts.RankSep,
	})

	// Apply the direction transformation (LR/RL/BT). Default TB is a no-op.
	applyRankDir(coords, opts.RankDir)

	// Build the node output from the original graph (g) so the caller
	// sees their node IDs and attributes, not the work copy.
	nodes := buildNodeLayouts(g, coords)

	// Build edge output from the original graph (g), looking up
	// positions from coords. The original edge directions are preserved
	// regardless of any internal reversal.
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
		w := attrs.Width
		if w <= 0 {
			w = defaultNodeWidth
		}
		h := attrs.Height
		if h <= 0 {
			h = defaultNodeHeight
		}
		p := coords[id]
		nodes[id] = NodeLayout{
			X:      p.X,
			Y:      p.Y,
			Width:  w,
			Height: h,
		}
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
			Points: []Point{
				{X: src.X, Y: src.Y},
				{X: dst.X, Y: dst.Y},
			},
			LabelPos: Point{
				X: (src.X + dst.X) / 2,
				Y: (src.Y + dst.Y) / 2,
			},
		}
	}
	return edges
}

// applyRankDir transforms the coordinates produced by the position
// phase (which always lays out TB) into the requested rank direction.
// After transformation, coords are re-normalized so the minimum x,y is 0.
func applyRankDir(coords position.Result, dir RankDir) {
	if dir == RankDirTB || dir == "" {
		return
	}
	if len(coords) == 0 {
		return
	}

	// Find current bounds in TB space.
	var maxX, maxY float64
	for _, p := range coords {
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	for n, p := range coords {
		switch dir {
		case RankDirBT:
			// Flip y axis: rank 0 moves to the bottom.
			coords[n] = position.Point{X: p.X, Y: maxY - p.Y}
		case RankDirLR:
			// Rotate 90° clockwise: swap axes so rank progression is horizontal.
			coords[n] = position.Point{X: p.Y, Y: p.X}
		case RankDirRL:
			// Rotate and flip: rank 0 moves to the right.
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
