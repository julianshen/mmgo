package graph

import (
	"slices"
	"testing"
)

// --- Node CRUD ---

func TestSetAndGetNode(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{Width: 100, Height: 50, Label: "Node A"})

	if !g.HasNode("a") {
		t.Fatal("expected node 'a' to exist")
	}
	attrs := g.NodeAttrs("a")
	if attrs.Width != 100 || attrs.Height != 50 || attrs.Label != "Node A" {
		t.Errorf("unexpected attrs: %+v", attrs)
	}
}

func TestNodeCount(t *testing.T) {
	g := New()
	if g.NodeCount() != 0 {
		t.Fatal("expected 0 nodes")
	}
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	if g.NodeCount() != 2 {
		t.Errorf("expected 2 nodes, got %d", g.NodeCount())
	}
}

func TestOverwriteNodeAttrs(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{Width: 10})
	g.SetNode("a", NodeAttrs{Width: 20})
	if g.NodeAttrs("a").Width != 20 {
		t.Error("expected attrs to be overwritten")
	}
	if g.NodeCount() != 1 {
		t.Error("overwrite should not increase node count")
	}
}

func TestRemoveNode(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	g.SetEdge("a", "b", EdgeAttrs{})
	g.RemoveNode("a")

	if g.HasNode("a") {
		t.Error("node 'a' should be removed")
	}
	if g.EdgeCount() != 0 {
		t.Error("edges incident to removed node should be removed")
	}
}

func TestNodes(t *testing.T) {
	g := New()
	g.SetNode("c", NodeAttrs{})
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	nodes := g.Nodes()
	slices.Sort(nodes)
	if !slices.Equal(nodes, []string{"a", "b", "c"}) {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

func TestHasNodeNonExistent(t *testing.T) {
	g := New()
	if g.HasNode("x") {
		t.Error("should not have node 'x'")
	}
}

// --- Edge CRUD ---

func TestSetAndGetEdge(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	g.SetEdge("a", "b", EdgeAttrs{Weight: 2, Label: "edge1"})

	if !g.HasEdge("a", "b") {
		t.Fatal("expected edge a->b")
	}
	edges := g.EdgesBetween("a", "b")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	attrs := g.EdgeAttrs(edges[0])
	if attrs.Weight != 2 || attrs.Label != "edge1" {
		t.Errorf("unexpected edge attrs: %+v", attrs)
	}
}

func TestSetEdgeAutoCreatesNodes(t *testing.T) {
	g := New()
	g.SetEdge("x", "y", EdgeAttrs{})
	if !g.HasNode("x") || !g.HasNode("y") {
		t.Error("SetEdge should auto-create nodes")
	}
}

func TestEdgeCount(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})
	if g.EdgeCount() != 2 {
		t.Errorf("expected 2 edges, got %d", g.EdgeCount())
	}
}

func TestRemoveEdge(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	edges := g.EdgesBetween("a", "b")
	g.RemoveEdge(edges[0])

	if g.HasEdge("a", "b") {
		t.Error("edge should be removed")
	}
	if !g.HasNode("a") || !g.HasNode("b") {
		t.Error("removing edge should not remove nodes")
	}
}

func TestMultiEdges(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{Label: "first"})
	g.SetEdge("a", "b", EdgeAttrs{Label: "second"})

	edges := g.EdgesBetween("a", "b")
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
	labels := []string{g.EdgeAttrs(edges[0]).Label, g.EdgeAttrs(edges[1]).Label}
	slices.Sort(labels)
	if !slices.Equal(labels, []string{"first", "second"}) {
		t.Errorf("unexpected labels: %v", labels)
	}
}

func TestSelfLoop(t *testing.T) {
	g := New()
	g.SetEdge("a", "a", EdgeAttrs{})
	if !g.HasEdge("a", "a") {
		t.Error("self-loop should exist")
	}
	if len(g.Successors("a")) != 1 || g.Successors("a")[0] != "a" {
		t.Error("self-loop should appear in successors")
	}
}

func TestEdges(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})
	edges := g.Edges()
	if len(edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(edges))
	}
}

// --- Adjacency queries ---

func TestSuccessors(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("a", "c", EdgeAttrs{})

	succ := g.Successors("a")
	slices.Sort(succ)
	if !slices.Equal(succ, []string{"b", "c"}) {
		t.Errorf("unexpected successors: %v", succ)
	}
}

func TestPredecessors(t *testing.T) {
	g := New()
	g.SetEdge("a", "c", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})

	pred := g.Predecessors("c")
	slices.Sort(pred)
	if !slices.Equal(pred, []string{"a", "b"}) {
		t.Errorf("unexpected predecessors: %v", pred)
	}
}

func TestNeighbors(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("c", "a", EdgeAttrs{})

	neighbors := g.Neighbors("a")
	slices.Sort(neighbors)
	if !slices.Equal(neighbors, []string{"b", "c"}) {
		t.Errorf("unexpected neighbors: %v", neighbors)
	}
}

func TestSuccessorsEmpty(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	if len(g.Successors("a")) != 0 {
		t.Error("isolated node should have no successors")
	}
}

func TestInEdges(t *testing.T) {
	g := New()
	g.SetEdge("a", "c", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})
	g.SetEdge("c", "d", EdgeAttrs{})

	inEdges := g.InEdges("c")
	if len(inEdges) != 2 {
		t.Errorf("expected 2 in-edges, got %d", len(inEdges))
	}
}

func TestOutEdges(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("a", "c", EdgeAttrs{})
	g.SetEdge("d", "a", EdgeAttrs{})

	outEdges := g.OutEdges("a")
	if len(outEdges) != 2 {
		t.Errorf("expected 2 out-edges, got %d", len(outEdges))
	}
}

// --- Compound graph (parent/child) ---

func TestSetParent(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	g.SetNode("child", NodeAttrs{})
	g.SetParent("child", "parent")

	if p := g.Parent("child"); p != "parent" {
		t.Errorf("expected parent 'parent', got %q", p)
	}
	children := g.Children("parent")
	if !slices.Equal(children, []string{"child"}) {
		t.Errorf("unexpected children: %v", children)
	}
}

func TestChildrenEmpty(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	if len(g.Children("a")) != 0 {
		t.Error("node with no children should return empty")
	}
}

func TestParentDefault(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	if p := g.Parent("a"); p != "" {
		t.Errorf("node without parent should return empty string, got %q", p)
	}
}

func TestRemoveParent(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	g.SetNode("child", NodeAttrs{})
	g.SetParent("child", "parent")
	g.SetParent("child", "") // remove parent

	if p := g.Parent("child"); p != "" {
		t.Errorf("expected no parent, got %q", p)
	}
	if len(g.Children("parent")) != 0 {
		t.Error("parent should have no children after removal")
	}
}

func TestRemoveNodeClearsParentChild(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	g.SetNode("child", NodeAttrs{})
	g.SetParent("child", "parent")
	g.RemoveNode("child")

	if len(g.Children("parent")) != 0 {
		t.Error("removing child should clean up parent's children list")
	}
}

// --- Topological sort ---

func TestTopologicalSort(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})
	g.SetEdge("a", "c", EdgeAttrs{})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// a must come before b and c; b must come before c.
	indexOf := func(s string) int {
		for i, v := range order {
			if v == s {
				return i
			}
		}
		return -1
	}
	if indexOf("a") > indexOf("b") || indexOf("a") > indexOf("c") || indexOf("b") > indexOf("c") {
		t.Errorf("invalid topological order: %v", order)
	}
}

func TestTopologicalSortCycleError(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("b", "c", EdgeAttrs{})
	g.SetEdge("c", "a", EdgeAttrs{})

	_, err := g.TopologicalSort()
	if err == nil {
		t.Error("expected error for cyclic graph")
	}
}

func TestTopologicalSortSingleNode(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(order, []string{"a"}) {
		t.Errorf("expected [a], got %v", order)
	}
}

func TestTopologicalSortDisconnected(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	g.SetNode("c", NodeAttrs{})

	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(order))
	}
}

// --- Copy ---

func TestCopy(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{Width: 10})
	g.SetNode("b", NodeAttrs{Width: 20})
	g.SetEdge("a", "b", EdgeAttrs{Weight: 5})
	g.SetParent("b", "a")

	g2 := g.Copy()

	// Verify copy has same structure.
	if g2.NodeCount() != 2 || g2.EdgeCount() != 1 {
		t.Error("copy should have same node/edge counts")
	}
	if g2.NodeAttrs("a").Width != 10 {
		t.Error("copy should preserve node attrs")
	}
	if g2.Parent("b") != "a" {
		t.Error("copy should preserve parent relationships")
	}

	// Verify independence: mutating copy doesn't affect original.
	g2.SetNode("c", NodeAttrs{})
	if g.HasNode("c") {
		t.Error("mutating copy should not affect original")
	}
}

// --- Edge cases ---

func TestEmptyGraph(t *testing.T) {
	g := New()
	if g.NodeCount() != 0 || g.EdgeCount() != 0 {
		t.Error("new graph should be empty")
	}
	if len(g.Nodes()) != 0 || len(g.Edges()) != 0 {
		t.Error("new graph should return empty slices")
	}
}

func TestNonExistentNodeQueries(t *testing.T) {
	g := New()
	// These should return empty slices, not panic.
	if len(g.Successors("x")) != 0 {
		t.Error("successors of non-existent node should be empty")
	}
	if len(g.Predecessors("x")) != 0 {
		t.Error("predecessors of non-existent node should be empty")
	}
	if len(g.Children("x")) != 0 {
		t.Error("children of non-existent node should be empty")
	}
}

func TestReverseEdge(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{Label: "fwd"})
	edges := g.EdgesBetween("a", "b")
	g.ReverseEdge(edges[0])

	if g.HasEdge("a", "b") {
		t.Error("original direction should not exist")
	}
	if !g.HasEdge("b", "a") {
		t.Error("reversed edge should exist")
	}
	revEdges := g.EdgesBetween("b", "a")
	if g.EdgeAttrs(revEdges[0]).Label != "fwd" {
		t.Error("reversed edge should preserve attrs")
	}
}
