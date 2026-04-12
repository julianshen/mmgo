package graph

import (
	"slices"
	"testing"
)

// mustSetParent calls SetParent and fails the test if it returns an error.
func mustSetParent(t *testing.T, g *Graph, child, parent string) {
	t.Helper()
	if err := g.SetParent(child, parent); err != nil {
		t.Fatalf("SetParent(%q, %q) unexpected error: %v", child, parent, err)
	}
}

// --- Node CRUD ---

func TestSetAndGetNode(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{Width: 100, Height: 50, Label: "Node A"})

	if !g.HasNode("a") {
		t.Fatal("expected node 'a' to exist")
	}
	attrs, ok := g.NodeAttrs("a")
	if !ok {
		t.Fatal("expected NodeAttrs ok=true for existing node")
	}
	if attrs.Width != 100 || attrs.Height != 50 || attrs.Label != "Node A" {
		t.Errorf("unexpected attrs: %+v", attrs)
	}
}

func TestNodeAttrsNonExistent(t *testing.T) {
	g := New()
	_, ok := g.NodeAttrs("missing")
	if ok {
		t.Error("expected ok=false for non-existent node")
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
	attrs, _ := g.NodeAttrs("a")
	if attrs.Width != 20 {
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

func TestRemoveNodeOrphansChildren(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	g.SetNode("child1", NodeAttrs{})
	g.SetNode("child2", NodeAttrs{})
	mustSetParent(t, g, "child1", "parent")
	mustSetParent(t, g, "child2", "parent")
	g.RemoveNode("parent")

	if g.Parent("child1") != "" {
		t.Error("child1 should be orphaned after parent removal")
	}
	if g.Parent("child2") != "" {
		t.Error("child2 should be orphaned after parent removal")
	}
	if !g.HasNode("child1") || !g.HasNode("child2") {
		t.Error("children should still exist after parent removal")
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
	attrs, ok := g.EdgeAttrs(edges[0])
	if !ok {
		t.Fatal("expected EdgeAttrs ok=true")
	}
	if attrs.Weight != 2 || attrs.Label != "edge1" {
		t.Errorf("unexpected edge attrs: %+v", attrs)
	}
}

func TestEdgeAttrsNonExistent(t *testing.T) {
	g := New()
	_, ok := g.EdgeAttrs(EdgeID{From: "x", To: "y", ID: 999})
	if ok {
		t.Error("expected ok=false for non-existent edge")
	}
}

func TestSetEdgeAttrs(t *testing.T) {
	g := New()
	eid := g.SetEdge("a", "b", EdgeAttrs{Weight: 1})
	ok := g.SetEdgeAttrs(eid, EdgeAttrs{Weight: 5, Label: "updated"})
	if !ok {
		t.Fatal("SetEdgeAttrs should return true for existing edge")
	}
	attrs, _ := g.EdgeAttrs(eid)
	if attrs.Weight != 5 || attrs.Label != "updated" {
		t.Errorf("expected updated attrs, got %+v", attrs)
	}
}

func TestSetEdgeAttrsNonExistent(t *testing.T) {
	g := New()
	ok := g.SetEdgeAttrs(EdgeID{From: "x", To: "y", ID: 999}, EdgeAttrs{Weight: 10})
	if ok {
		t.Error("SetEdgeAttrs should return false for non-existent edge")
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
	removed := g.RemoveEdge(edges[0])

	if !removed {
		t.Error("RemoveEdge should return true")
	}
	if g.HasEdge("a", "b") {
		t.Error("edge should be removed")
	}
	if !g.HasNode("a") || !g.HasNode("b") {
		t.Error("removing edge should not remove nodes")
	}
}

func TestRemoveEdgeNonExistent(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	removed := g.RemoveEdge(EdgeID{From: "x", To: "y", ID: 999})
	if removed {
		t.Error("RemoveEdge should return false for non-existent edge")
	}
	if g.EdgeCount() != 1 {
		t.Error("graph should be unchanged")
	}
}

func TestRemoveEdgeMultiEdgeGraph(t *testing.T) {
	// Regression test: removing an edge must not corrupt adjacency for other nodes.
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{Label: "ab"})
	g.SetEdge("c", "d", EdgeAttrs{Label: "cd"})
	g.SetEdge("a", "d", EdgeAttrs{Label: "ad"})

	// Remove the first edge.
	edges := g.EdgesBetween("a", "b")
	g.RemoveEdge(edges[0])

	// Verify other edges are intact and queryable.
	if !g.HasEdge("c", "d") {
		t.Fatal("edge c->d should still exist")
	}
	if !g.HasEdge("a", "d") {
		t.Fatal("edge a->d should still exist")
	}
	cdEdges := g.EdgesBetween("c", "d")
	if len(cdEdges) != 1 {
		t.Fatalf("expected 1 c->d edge, got %d", len(cdEdges))
	}
	attrs, _ := g.EdgeAttrs(cdEdges[0])
	if attrs.Label != "cd" {
		t.Errorf("expected label 'cd', got %q", attrs.Label)
	}

	// Verify adjacency queries work correctly.
	succ := g.Successors("c")
	if !slices.Equal(succ, []string{"d"}) {
		t.Errorf("expected successors [d], got %v", succ)
	}
	pred := g.Predecessors("d")
	slices.Sort(pred)
	if !slices.Equal(pred, []string{"a", "c"}) {
		t.Errorf("expected predecessors [a, c], got %v", pred)
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
	attrs0, _ := g.EdgeAttrs(edges[0])
	attrs1, _ := g.EdgeAttrs(edges[1])
	labels := []string{attrs0.Label, attrs1.Label}
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
	succ := g.Successors("a")
	if len(succ) != 1 || succ[0] != "a" {
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

func TestNeighborsDeduplication(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	g.SetEdge("b", "a", EdgeAttrs{})

	neighbors := g.Neighbors("a")
	if len(neighbors) != 1 || neighbors[0] != "b" {
		t.Errorf("expected [b] (deduplicated), got %v", neighbors)
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
	if err := g.SetParent("child", "parent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p := g.Parent("child"); p != "parent" {
		t.Errorf("expected parent 'parent', got %q", p)
	}
	children := g.Children("parent")
	if !slices.Equal(children, []string{"child"}) {
		t.Errorf("unexpected children: %v", children)
	}
}

func TestSetParentReassign(t *testing.T) {
	g := New()
	g.SetNode("parentA", NodeAttrs{})
	g.SetNode("parentB", NodeAttrs{})
	g.SetNode("child", NodeAttrs{})
	mustSetParent(t, g, "child", "parentA")
	mustSetParent(t, g, "child", "parentB")

	if p := g.Parent("child"); p != "parentB" {
		t.Errorf("expected parent 'parentB', got %q", p)
	}
	if len(g.Children("parentA")) != 0 {
		t.Error("parentA should have no children after reassignment")
	}
	children := g.Children("parentB")
	if !slices.Equal(children, []string{"child"}) {
		t.Errorf("unexpected children: %v", children)
	}
}

func TestSetParentNonExistentChild(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	err := g.SetParent("ghost", "parent")
	if err == nil {
		t.Error("expected error for non-existent child")
	}
}

func TestSetParentNonExistentParent(t *testing.T) {
	g := New()
	g.SetNode("child", NodeAttrs{})
	err := g.SetParent("child", "ghost")
	if err == nil {
		t.Error("expected error for non-existent parent")
	}
}

func TestSetParentSelfCycle(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	err := g.SetParent("a", "a")
	if err == nil {
		t.Error("expected error for self-parenting")
	}
}

func TestSetParentCycleDetection(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	g.SetNode("b", NodeAttrs{})
	g.SetNode("c", NodeAttrs{})
	mustSetParent(t, g, "b", "a")
	mustSetParent(t, g, "c", "b")
	err := g.SetParent("a", "c") // would create a->b->c->a cycle
	if err == nil {
		t.Error("expected error for circular parent-child hierarchy")
	}
}

func TestChildrenEmpty(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{})
	if len(g.Children("a")) != 0 {
		t.Error("node with no children should return empty")
	}
}

func TestChildrenReturnsCopy(t *testing.T) {
	g := New()
	g.SetNode("parent", NodeAttrs{})
	g.SetNode("child", NodeAttrs{})
	mustSetParent(t, g, "child", "parent")

	children := g.Children("parent")
	children[0] = "hacked" // mutate the returned slice

	actual := g.Children("parent")
	if actual[0] != "child" {
		t.Error("mutating returned Children slice should not affect graph")
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
	mustSetParent(t, g, "child", "parent")
	mustSetParent(t, g, "child", "") // remove parent

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
	mustSetParent(t, g, "child", "parent")
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

func TestTopologicalSortEmptyGraph(t *testing.T) {
	g := New()
	order, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 0 {
		t.Errorf("expected empty order, got %v", order)
	}
}

// --- Copy ---

func TestCopy(t *testing.T) {
	g := New()
	g.SetNode("a", NodeAttrs{Width: 10})
	g.SetNode("b", NodeAttrs{Width: 20})
	g.SetEdge("a", "b", EdgeAttrs{Weight: 5})
	mustSetParent(t, g, "b", "a")

	g2 := g.Copy()

	// Verify copy has same structure.
	if g2.NodeCount() != 2 || g2.EdgeCount() != 1 {
		t.Error("copy should have same node/edge counts")
	}
	attrs, _ := g2.NodeAttrs("a")
	if attrs.Width != 10 {
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

	// Verify edge independence.
	g2.SetEdge("c", "a", EdgeAttrs{})
	if g.EdgeCount() != 1 {
		t.Error("adding edge to copy should not affect original")
	}

	// Verify edge IDs are preserved.
	origEdges := g.Edges()
	copyEdges := g2.EdgesBetween(origEdges[0].From, origEdges[0].To)
	found := false
	for _, ce := range copyEdges {
		if ce.ID == origEdges[0].ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("copy should preserve original edge IDs")
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
	newID, ok := g.ReverseEdge(edges[0])
	if !ok {
		t.Fatal("ReverseEdge should return ok=true")
	}

	if g.HasEdge("a", "b") {
		t.Error("original direction should not exist")
	}
	if !g.HasEdge("b", "a") {
		t.Error("reversed edge should exist")
	}
	attrs, _ := g.EdgeAttrs(newID)
	if attrs.Label != "fwd" {
		t.Error("reversed edge should preserve attrs")
	}
}

func TestReverseEdgeNonExistent(t *testing.T) {
	g := New()
	g.SetEdge("a", "b", EdgeAttrs{})
	_, ok := g.ReverseEdge(EdgeID{From: "x", To: "y", ID: 999})
	if ok {
		t.Error("ReverseEdge should return ok=false for non-existent edge")
	}
	// Verify no phantom edge was created.
	if g.HasNode("x") || g.HasNode("y") {
		t.Error("non-existent reverse should not create phantom nodes")
	}
	if g.EdgeCount() != 1 {
		t.Error("graph should be unchanged")
	}
}
