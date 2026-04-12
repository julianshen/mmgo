// Package graph provides a directed graph data structure with support for
// node/edge attributes, multi-edges, compound graphs (parent/child), and
// topological sorting. It serves as the foundation for the dagre layout engine.
package graph

import "fmt"

// EdgeID uniquely identifies an edge in the graph. The ID field is assigned
// by Graph.SetEdge and must not be fabricated by callers.
type EdgeID struct {
	From string
	To   string
	ID   int // assigned by Graph.SetEdge; disambiguates multi-edges between the same pair
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

// EffectiveMinLen returns the minimum edge length, defaulting to 1 when
// MinLen is zero or negative. This is the single source of truth for the
// project-wide "unset MinLen means 1" convention used across the layout
// engine.
func (a EdgeAttrs) EffectiveMinLen() int {
	if a.MinLen <= 0 {
		return 1
	}
	return a.MinLen
}

// Graph is a directed graph supporting multi-edges and compound (parent/child)
// relationships. It is not safe for concurrent use.
type Graph struct {
	nodes    map[string]NodeAttrs
	edges    map[EdgeID]EdgeAttrs
	nextEdge int // monotonic counter for edge IDs

	// adjacency: node -> set of edge IDs
	outEdges map[string][]EdgeID
	inEdges  map[string][]EdgeID

	// compound graph
	parent   map[string]string   // child -> parent
	children map[string][]string // parent -> children
}

// New creates an empty directed graph.
func New() *Graph {
	return &Graph{
		nodes:    make(map[string]NodeAttrs),
		edges:    make(map[EdgeID]EdgeAttrs),
		outEdges: make(map[string][]EdgeID),
		inEdges:  make(map[string][]EdgeID),
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

// NodeAttrs returns the attributes for the given node and whether it exists.
func (g *Graph) NodeAttrs(id string) (NodeAttrs, bool) {
	attrs, ok := g.nodes[id]
	return attrs, ok
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
	// Remove all incident edges (collect first to avoid mutation during iteration).
	toRemove := make([]EdgeID, 0, len(g.outEdges[id])+len(g.inEdges[id]))
	toRemove = append(toRemove, g.outEdges[id]...)
	toRemove = append(toRemove, g.inEdges[id]...)
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

	g.edges[eid] = attrs
	g.outEdges[from] = append(g.outEdges[from], eid)
	g.inEdges[to] = append(g.inEdges[to], eid)
	return eid
}

// HasEdge reports whether at least one edge exists from -> to.
func (g *Graph) HasEdge(from, to string) bool {
	for _, eid := range g.outEdges[from] {
		if eid.To == to {
			return true
		}
	}
	return false
}

// EdgeAttrs returns the attributes for the given edge and whether it exists.
func (g *Graph) EdgeAttrs(eid EdgeID) (EdgeAttrs, bool) {
	attrs, ok := g.edges[eid]
	return attrs, ok
}

// SetEdgeAttrs updates the attributes for an existing edge.
// Returns false if the edge does not exist.
func (g *Graph) SetEdgeAttrs(eid EdgeID, attrs EdgeAttrs) bool {
	if _, ok := g.edges[eid]; !ok {
		return false
	}
	g.edges[eid] = attrs
	return true
}

// EdgesBetween returns all edge IDs from -> to.
func (g *Graph) EdgesBetween(from, to string) []EdgeID {
	var result []EdgeID
	for _, eid := range g.outEdges[from] {
		if eid.To == to {
			result = append(result, eid)
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
	result := make([]EdgeID, 0, len(g.edges))
	for eid := range g.edges {
		result = append(result, eid)
	}
	return result
}

// RemoveEdge removes an edge by its ID. Returns true if the edge was found
// and removed.
func (g *Graph) RemoveEdge(eid EdgeID) bool {
	if _, ok := g.edges[eid]; !ok {
		return false
	}
	delete(g.edges, eid)
	g.outEdges[eid.From] = removeEdgeID(g.outEdges[eid.From], eid)
	g.inEdges[eid.To] = removeEdgeID(g.inEdges[eid.To], eid)
	return true
}

// ReverseEdge reverses the direction of an edge, preserving its attributes.
// Returns the new EdgeID and true, or a zero EdgeID and false if the original
// edge does not exist.
func (g *Graph) ReverseEdge(eid EdgeID) (EdgeID, bool) {
	attrs, ok := g.EdgeAttrs(eid)
	if !ok {
		return EdgeID{}, false
	}
	g.RemoveEdge(eid)
	newID := g.SetEdge(eid.To, eid.From, attrs)
	return newID, true
}

// InEdges returns all edge IDs pointing into the given node.
func (g *Graph) InEdges(id string) []EdgeID {
	result := make([]EdgeID, len(g.inEdges[id]))
	copy(result, g.inEdges[id])
	return result
}

// OutEdges returns all edge IDs going out of the given node.
func (g *Graph) OutEdges(id string) []EdgeID {
	result := make([]EdgeID, len(g.outEdges[id]))
	copy(result, g.outEdges[id])
	return result
}

// --- Adjacency queries ---

// Successors returns the IDs of all nodes reachable by a single outgoing edge.
func (g *Graph) Successors(id string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, eid := range g.outEdges[id] {
		if !seen[eid.To] {
			seen[eid.To] = true
			result = append(result, eid.To)
		}
	}
	return result
}

// Predecessors returns the IDs of all nodes with an edge pointing to id.
func (g *Graph) Predecessors(id string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, eid := range g.inEdges[id] {
		if !seen[eid.From] {
			seen[eid.From] = true
			result = append(result, eid.From)
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
// Both child and parent (if non-empty) must exist as nodes in the graph.
// Returns an error if they don't, or if setting the parent would create a cycle.
func (g *Graph) SetParent(child, parent string) error {
	if !g.HasNode(child) {
		return fmt.Errorf("child node %q does not exist", child)
	}
	if parent != "" && !g.HasNode(parent) {
		return fmt.Errorf("parent node %q does not exist", parent)
	}
	if parent == child {
		return fmt.Errorf("node %q cannot be its own parent", child)
	}
	// Check for cycles: walk up from parent to ensure child is not an ancestor.
	if parent != "" {
		for p := parent; p != ""; p = g.parent[p] {
			if p == child {
				return fmt.Errorf("setting %q as parent of %q would create a cycle", parent, child)
			}
		}
	}

	// Remove from old parent first.
	if old := g.parent[child]; old != "" {
		g.removeChild(old, child)
	}
	if parent == "" {
		delete(g.parent, child)
		return nil
	}
	g.parent[child] = parent
	g.children[parent] = append(g.children[parent], child)
	return nil
}

// Parent returns the parent node ID, or "" if the node has no parent.
func (g *Graph) Parent(id string) string {
	return g.parent[id]
}

// Children returns the child node IDs of the given parent.
// Returns a copy; mutating the returned slice does not affect the graph.
func (g *Graph) Children(id string) []string {
	c := g.children[id]
	if len(c) == 0 {
		return []string{}
	}
	result := make([]string, len(c))
	copy(result, c)
	return result
}

// --- Topological sort ---

// TopologicalSort returns a topological ordering of all nodes.
// Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range g.nodes {
		inDegree[id] = 0
	}
	for eid := range g.edges {
		inDegree[eid.To]++
	}

	// Seed queue with zero in-degree nodes.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var order []string
	for head := 0; head < len(queue); head++ {
		node := queue[head]
		order = append(order, node)

		for _, eid := range g.outEdges[node] {
			inDegree[eid.To]--
			if inDegree[eid.To] == 0 {
				queue = append(queue, eid.To)
			}
		}
	}

	if len(order) != len(g.nodes) {
		return nil, fmt.Errorf("graph contains a cycle")
	}
	return order, nil
}

// --- Copy ---

// Copy returns a deep copy of the graph. Edge IDs are preserved so that
// references held by callers remain valid on the copy. Mutations to the copy
// do not affect the original.
func (g *Graph) Copy() *Graph {
	g2 := New()
	g2.nextEdge = g.nextEdge

	for id, attrs := range g.nodes {
		g2.SetNode(id, attrs)
	}
	// Copy edges directly to preserve IDs (SetEdge would assign new IDs).
	for eid, attrs := range g.edges {
		g2.edges[eid] = attrs
		g2.outEdges[eid.From] = append(g2.outEdges[eid.From], eid)
		g2.inEdges[eid.To] = append(g2.inEdges[eid.To], eid)
	}
	for child, parent := range g.parent {
		g2.SetParent(child, parent) //nolint:errcheck // nodes guaranteed to exist from loop above
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

func removeEdgeID(slice []EdgeID, eid EdgeID) []EdgeID {
	for i, e := range slice {
		if e == eid {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
