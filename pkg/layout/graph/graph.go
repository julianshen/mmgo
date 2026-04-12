// Package graph provides a directed graph data structure with support for
// node/edge attributes, multi-edges, compound graphs (parent/child), and
// topological sorting. It serves as the foundation for the dagre layout engine.
package graph

import "fmt"

// EdgeID uniquely identifies an edge in the graph.
type EdgeID struct {
	From string
	To   string
	ID   int // disambiguates multi-edges between the same pair
}

// NodeAttrs holds attributes associated with a graph node.
type NodeAttrs struct {
	Label  string
	Width  float64
	Height float64
}

// EdgeAttrs holds attributes associated with a graph edge.
type EdgeAttrs struct {
	Label    string
	Weight   float64
	MinLen   int
	LabelPos string
}

type edgeEntry struct {
	id    EdgeID
	attrs EdgeAttrs
}

// Graph is a directed graph supporting multi-edges and compound (parent/child)
// relationships. It is not safe for concurrent use.
type Graph struct {
	nodes    map[string]NodeAttrs
	edges    []edgeEntry
	nextEdge int // monotonic counter for edge IDs

	// adjacency: node -> list of edge indices in edges slice
	outEdges map[string][]int
	inEdges  map[string][]int

	// compound graph
	parent   map[string]string   // child -> parent
	children map[string][]string // parent -> children
}

// New creates an empty directed graph.
func New() *Graph {
	return &Graph{
		nodes:    make(map[string]NodeAttrs),
		outEdges: make(map[string][]int),
		inEdges:  make(map[string][]int),
		parent:   make(map[string]string),
		children: make(map[string][]string),
	}
}

// --- Node operations ---

// SetNode adds or updates a node with the given attributes.
func (g *Graph) SetNode(id string, attrs NodeAttrs) {
	g.nodes[id] = attrs
}

// HasNode reports whether the graph contains a node with the given ID.
func (g *Graph) HasNode(id string) bool {
	_, ok := g.nodes[id]
	return ok
}

// NodeAttrs returns the attributes for the given node.
// Returns zero-value NodeAttrs if the node does not exist.
func (g *Graph) NodeAttrs(id string) NodeAttrs {
	return g.nodes[id]
}

// NodeCount returns the number of nodes in the graph.
func (g *Graph) NodeCount() int {
	return len(g.nodes)
}

// Nodes returns a slice of all node IDs.
func (g *Graph) Nodes() []string {
	result := make([]string, 0, len(g.nodes))
	for id := range g.nodes {
		result = append(result, id)
	}
	return result
}

// RemoveNode removes a node and all its incident edges.
// Also cleans up any parent/child relationships.
func (g *Graph) RemoveNode(id string) {
	if !g.HasNode(id) {
		return
	}
	// Remove all incident edges (collect indices first to avoid mutation during iteration).
	toRemove := make([]EdgeID, 0)
	for _, idx := range g.outEdges[id] {
		toRemove = append(toRemove, g.edges[idx].id)
	}
	for _, idx := range g.inEdges[id] {
		toRemove = append(toRemove, g.edges[idx].id)
	}
	for _, eid := range toRemove {
		g.RemoveEdge(eid)
	}

	// Clean up parent/child.
	if p := g.parent[id]; p != "" {
		g.removeChild(p, id)
	}
	// Orphan any children.
	for _, child := range g.children[id] {
		delete(g.parent, child)
	}
	delete(g.children, id)
	delete(g.parent, id)

	delete(g.nodes, id)
	delete(g.outEdges, id)
	delete(g.inEdges, id)
}

// --- Edge operations ---

// SetEdge adds a directed edge from -> to. If the nodes do not exist, they
// are auto-created with zero-value attributes. Multiple edges between the
// same pair are supported (multi-edges).
func (g *Graph) SetEdge(from, to string, attrs EdgeAttrs) EdgeID {
	if !g.HasNode(from) {
		g.SetNode(from, NodeAttrs{})
	}
	if !g.HasNode(to) {
		g.SetNode(to, NodeAttrs{})
	}

	eid := EdgeID{From: from, To: to, ID: g.nextEdge}
	g.nextEdge++

	idx := len(g.edges)
	g.edges = append(g.edges, edgeEntry{id: eid, attrs: attrs})
	g.outEdges[from] = append(g.outEdges[from], idx)
	g.inEdges[to] = append(g.inEdges[to], idx)
	return eid
}

// HasEdge reports whether at least one edge exists from -> to.
func (g *Graph) HasEdge(from, to string) bool {
	for _, idx := range g.outEdges[from] {
		if g.edges[idx].id.To == to {
			return true
		}
	}
	return false
}

// EdgeAttrs returns the attributes for the given edge.
func (g *Graph) EdgeAttrs(eid EdgeID) EdgeAttrs {
	for _, e := range g.edges {
		if e.id == eid {
			return e.attrs
		}
	}
	return EdgeAttrs{}
}

// SetEdgeAttrs updates the attributes for an existing edge.
func (g *Graph) SetEdgeAttrs(eid EdgeID, attrs EdgeAttrs) {
	for i, e := range g.edges {
		if e.id == eid {
			g.edges[i].attrs = attrs
			return
		}
	}
}

// EdgesBetween returns all edge IDs from -> to.
func (g *Graph) EdgesBetween(from, to string) []EdgeID {
	var result []EdgeID
	for _, idx := range g.outEdges[from] {
		if g.edges[idx].id.To == to {
			result = append(result, g.edges[idx].id)
		}
	}
	return result
}

// EdgeCount returns the total number of edges.
func (g *Graph) EdgeCount() int {
	return len(g.edges)
}

// Edges returns all edge IDs in the graph.
func (g *Graph) Edges() []EdgeID {
	result := make([]EdgeID, len(g.edges))
	for i, e := range g.edges {
		result[i] = e.id
	}
	return result
}

// RemoveEdge removes an edge by its ID.
func (g *Graph) RemoveEdge(eid EdgeID) {
	idx := -1
	for i, e := range g.edges {
		if e.id == eid {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	from := g.edges[idx].id.From
	to := g.edges[idx].id.To

	// Remove from edges slice.
	g.edges = append(g.edges[:idx], g.edges[idx+1:]...)

	// Rebuild adjacency indices for affected nodes (indices shifted).
	g.rebuildAdjacency(from)
	g.rebuildAdjacency(to)
}

// ReverseEdge reverses the direction of an edge, preserving its attributes.
func (g *Graph) ReverseEdge(eid EdgeID) {
	attrs := g.EdgeAttrs(eid)
	g.RemoveEdge(eid)
	g.SetEdge(eid.To, eid.From, attrs)
}

// InEdges returns all edge IDs pointing into the given node.
func (g *Graph) InEdges(id string) []EdgeID {
	var result []EdgeID
	for _, idx := range g.inEdges[id] {
		result = append(result, g.edges[idx].id)
	}
	return result
}

// OutEdges returns all edge IDs going out of the given node.
func (g *Graph) OutEdges(id string) []EdgeID {
	var result []EdgeID
	for _, idx := range g.outEdges[id] {
		result = append(result, g.edges[idx].id)
	}
	return result
}

// --- Adjacency queries ---

// Successors returns the IDs of all nodes reachable by a single outgoing edge.
func (g *Graph) Successors(id string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, idx := range g.outEdges[id] {
		to := g.edges[idx].id.To
		if !seen[to] {
			seen[to] = true
			result = append(result, to)
		}
	}
	return result
}

// Predecessors returns the IDs of all nodes with an edge pointing to id.
func (g *Graph) Predecessors(id string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, idx := range g.inEdges[id] {
		from := g.edges[idx].id.From
		if !seen[from] {
			seen[from] = true
			result = append(result, from)
		}
	}
	return result
}

// Neighbors returns the union of successors and predecessors (undirected view).
func (g *Graph) Neighbors(id string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range g.Successors(id) {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, p := range g.Predecessors(id) {
		if !seen[p] {
			seen[p] = true
			result = append(result, p)
		}
	}
	return result
}

// --- Compound graph ---

// SetParent sets the parent of a child node. Pass "" to remove the parent.
func (g *Graph) SetParent(child, parent string) {
	// Remove from old parent first.
	if old := g.parent[child]; old != "" {
		g.removeChild(old, child)
	}
	if parent == "" {
		delete(g.parent, child)
		return
	}
	g.parent[child] = parent
	g.children[parent] = append(g.children[parent], child)
}

// Parent returns the parent node ID, or "" if the node has no parent.
func (g *Graph) Parent(id string) string {
	return g.parent[id]
}

// Children returns the child node IDs of the given parent.
func (g *Graph) Children(id string) []string {
	c := g.children[id]
	if c == nil {
		return []string{}
	}
	return c
}

// --- Topological sort ---

// TopologicalSort returns a topological ordering of all nodes.
// Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	for _, e := range g.edges {
		inDegree[e.id.To]++
	}

	// Seed queue with zero in-degree nodes.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, idx := range g.outEdges[node] {
			to := g.edges[idx].id.To
			inDegree[to]--
			if inDegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}

	if len(order) != len(g.nodes) {
		return nil, fmt.Errorf("graph contains a cycle")
	}
	return order, nil
}

// --- Copy ---

// Copy returns a deep copy of the graph. Mutations to the copy do not affect
// the original.
func (g *Graph) Copy() *Graph {
	g2 := New()
	for id, attrs := range g.nodes {
		g2.SetNode(id, attrs)
	}
	for _, e := range g.edges {
		g2.SetEdge(e.id.From, e.id.To, e.attrs)
	}
	for child, parent := range g.parent {
		g2.SetParent(child, parent)
	}
	return g2
}

// --- Internal helpers ---

func (g *Graph) removeChild(parent, child string) {
	children := g.children[parent]
	for i, c := range children {
		if c == child {
			g.children[parent] = append(children[:i], children[i+1:]...)
			return
		}
	}
}

func (g *Graph) rebuildAdjacency(id string) {
	g.outEdges[id] = g.outEdges[id][:0]
	g.inEdges[id] = g.inEdges[id][:0]
	for i, e := range g.edges {
		if e.id.From == id {
			g.outEdges[id] = append(g.outEdges[id], i)
		}
		if e.id.To == id {
			g.inEdges[id] = append(g.inEdges[id], i)
		}
	}
}
