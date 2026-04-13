// Package graphtest provides test helpers for layout phase packages.
// It is test-only infrastructure (used under _test.go files).
package graphtest

import "github.com/julianshen/mmgo/pkg/layout/graph"

// BuildGraph creates a new directed graph with the given edges. Each
// edge is a [2]string of {from, to}. Nodes are auto-created. Used by
// layout phase tests to eliminate the repeated local buildGraph helper.
func BuildGraph(edges ...[2]string) *graph.Graph {
	g := graph.New()
	for _, e := range edges {
		g.SetEdge(e[0], e[1], graph.EdgeAttrs{})
	}
	return g
}

// SetWidths sets Width and Height on every existing node in g. Used
// by layout tests to provide uniform node dimensions without hand-
// writing a SetNode call per node.
func SetWidths(g *graph.Graph, width, height float64) {
	for _, n := range g.Nodes() {
		attrs, _ := g.NodeAttrs(n)
		attrs.Width = width
		attrs.Height = height
		g.SetNode(n, attrs)
	}
}
