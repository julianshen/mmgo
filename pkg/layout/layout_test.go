package layout

import (
	"math"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/graphtest"
)

// --- Helpers ---

var buildGraph = graphtest.BuildGraph

func setWidths(g *graph.Graph, w, h float64) {
	for _, n := range g.Nodes() {
		attrs, _ := g.NodeAttrs(n)
		attrs.Width = w
		attrs.Height = h
		g.SetNode(n, attrs)
	}
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func defaultOpts() Options {
	return Options{NodeSep: 50, RankSep: 80}
}

// --- Trivial cases ---

func TestLayoutEmpty(t *testing.T) {
	g := graph.New()
	result := Layout(g, defaultOpts())

	if result == nil {
		t.Fatal("Layout should not return nil")
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 0 {
		t.Errorf("expected no edges, got %d", len(result.Edges))
	}
}

func TestLayoutSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{Width: 100, Height: 50})

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(result.Nodes))
	}
	nl := result.Nodes["a"]
	if nl.Width != 100 || nl.Height != 50 {
		t.Errorf("expected width=100 height=50, got %+v", nl)
	}
}

// --- Integration tests ---

func TestLayoutLinearChain(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(result.Edges))
	}
	// All three should have the same X (vertical alignment).
	ax := result.Nodes["a"].X
	bx := result.Nodes["b"].X
	cx := result.Nodes["c"].X
	if !approxEqual(ax, bx) || !approxEqual(bx, cx) {
		t.Errorf("linear chain should be vertically aligned: a=%f b=%f c=%f", ax, bx, cx)
	}
	// Y should increase down the chain in default TB direction.
	if !(result.Nodes["a"].Y < result.Nodes["b"].Y && result.Nodes["b"].Y < result.Nodes["c"].Y) {
		t.Error("TB direction: a should be above b should be above c")
	}
}

func TestLayoutDiamond(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 4 || len(result.Edges) != 4 {
		t.Fatalf("expected 4 nodes 4 edges, got %d %d",
			len(result.Nodes), len(result.Edges))
	}
	// a and d should be centered horizontally between b and c.
	midBC := (result.Nodes["b"].X + result.Nodes["c"].X) / 2
	if !approxEqual(result.Nodes["a"].X, midBC) {
		t.Errorf("a should be centered between b,c: a=%f mid=%f",
			result.Nodes["a"].X, midBC)
	}
	if !approxEqual(result.Nodes["d"].X, midBC) {
		t.Errorf("d should be centered between b,c: d=%f mid=%f",
			result.Nodes["d"].X, midBC)
	}
}

func TestLayoutCyclicGraphHandled(t *testing.T) {
	// Triangle cycle: a → b → c → a. The acyclic phase should break
	// one edge and lay out the rest successfully.
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "a"},
	)
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(result.Nodes))
	}
	// All edges should be present in the result (original direction restored).
	if len(result.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(result.Edges))
	}
}

func TestLayoutDoesNotMutateInput(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "a"}) // cycle
	setWidths(g, 100, 50)

	origEdgeCount := g.EdgeCount()
	origNodes := g.NodeCount()
	origHasAB := g.HasEdge("a", "b")
	origHasBA := g.HasEdge("b", "a")

	Layout(g, defaultOpts())

	if g.EdgeCount() != origEdgeCount {
		t.Errorf("edge count changed: %d → %d", origEdgeCount, g.EdgeCount())
	}
	if g.NodeCount() != origNodes {
		t.Errorf("node count changed: %d → %d", origNodes, g.NodeCount())
	}
	if g.HasEdge("a", "b") != origHasAB || g.HasEdge("b", "a") != origHasBA {
		t.Error("input graph edges mutated")
	}
}

// --- Options ---

func TestLayoutDefaultOptions(t *testing.T) {
	g := buildGraph([2]string{"a", "b"})
	setWidths(g, 100, 50)

	// All options zero → defaults applied.
	result := Layout(g, Options{})

	// Defaults should give reasonable positions (non-negative, b below a).
	if result.Nodes["a"].Y >= result.Nodes["b"].Y {
		t.Error("default TB direction: a should be above b")
	}
	for n, nl := range result.Nodes {
		if nl.X < 0 || nl.Y < 0 {
			t.Errorf("%s has negative coord: %+v", n, nl)
		}
	}
}

func TestLayoutResultBounds(t *testing.T) {
	g := buildGraph([2]string{"a", "b"})
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	// Bounds must enclose all nodes (including their half-widths/heights).
	for n, nl := range result.Nodes {
		right := nl.X + nl.Width/2
		bottom := nl.Y + nl.Height/2
		if right > result.Width+0.001 {
			t.Errorf("%s extends beyond Width: right=%f Width=%f",
				n, right, result.Width)
		}
		if bottom > result.Height+0.001 {
			t.Errorf("%s extends beyond Height: bottom=%f Height=%f",
				n, bottom, result.Height)
		}
	}
}

// --- Rank directions ---

func TestLayoutRankDirTB(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	setWidths(g, 100, 50)

	opts := defaultOpts()
	opts.RankDir = RankDirTB
	result := Layout(g, opts)

	// TB: a above b above c.
	if !(result.Nodes["a"].Y < result.Nodes["b"].Y && result.Nodes["b"].Y < result.Nodes["c"].Y) {
		t.Error("TB: a should be above b should be above c")
	}
}

func TestLayoutRankDirBT(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	setWidths(g, 100, 50)

	opts := defaultOpts()
	opts.RankDir = RankDirBT
	result := Layout(g, opts)

	// BT: a below b below c (rank 0 at the bottom).
	if !(result.Nodes["a"].Y > result.Nodes["b"].Y && result.Nodes["b"].Y > result.Nodes["c"].Y) {
		t.Errorf("BT: a should be below b should be below c, got a=%f b=%f c=%f",
			result.Nodes["a"].Y, result.Nodes["b"].Y, result.Nodes["c"].Y)
	}
}

func TestLayoutRankDirLR(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	setWidths(g, 100, 50)

	opts := defaultOpts()
	opts.RankDir = RankDirLR
	result := Layout(g, opts)

	// LR: a to the left of b to the left of c.
	if !(result.Nodes["a"].X < result.Nodes["b"].X && result.Nodes["b"].X < result.Nodes["c"].X) {
		t.Errorf("LR: a should be left of b left of c, got a=%f b=%f c=%f",
			result.Nodes["a"].X, result.Nodes["b"].X, result.Nodes["c"].X)
	}
}

func TestLayoutRankDirRL(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	setWidths(g, 100, 50)

	opts := defaultOpts()
	opts.RankDir = RankDirRL
	result := Layout(g, opts)

	// RL: a to the right of b to the right of c.
	if !(result.Nodes["a"].X > result.Nodes["b"].X && result.Nodes["b"].X > result.Nodes["c"].X) {
		t.Errorf("RL: a should be right of b right of c, got a=%f b=%f c=%f",
			result.Nodes["a"].X, result.Nodes["b"].X, result.Nodes["c"].X)
	}
}

// --- Edges ---

func TestLayoutEdgesHavePoints(t *testing.T) {
	g := buildGraph([2]string{"a", "b"})
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	if len(result.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(result.Edges))
	}
	for _, el := range result.Edges {
		if len(el.Points) < 2 {
			t.Errorf("edge should have at least 2 control points, got %d", len(el.Points))
		}
		// First point should be near a's position, last near b's.
		first := el.Points[0]
		last := el.Points[len(el.Points)-1]
		if !approxEqual(first.X, result.Nodes["a"].X) ||
			!approxEqual(first.Y, result.Nodes["a"].Y) {
			t.Errorf("first point should match source a: got %+v", first)
		}
		if !approxEqual(last.X, result.Nodes["b"].X) ||
			!approxEqual(last.Y, result.Nodes["b"].Y) {
			t.Errorf("last point should match target b: got %+v", last)
		}
	}
}

func TestLayoutEdgeLabelPosition(t *testing.T) {
	g := buildGraph([2]string{"a", "b"})
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	for _, el := range result.Edges {
		// Label midpoint should be between source and target.
		midX := (result.Nodes["a"].X + result.Nodes["b"].X) / 2
		midY := (result.Nodes["a"].Y + result.Nodes["b"].Y) / 2
		if !approxEqual(el.LabelPos.X, midX) || !approxEqual(el.LabelPos.Y, midY) {
			t.Errorf("label pos %+v should be midpoint (%f, %f)",
				el.LabelPos, midX, midY)
		}
	}
}

// --- Default widths ---

func TestLayoutDefaultWidthWhenUnset(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{}) // width/height unset
	g.SetNode("b", graph.NodeAttrs{})
	g.SetEdge("a", "b", graph.EdgeAttrs{})

	result := Layout(g, defaultOpts())

	// Should not panic or produce nodes at the same position.
	if len(result.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(result.Nodes))
	}
	// Both nodes should have non-zero default width/height.
	for n, nl := range result.Nodes {
		if nl.Width == 0 || nl.Height == 0 {
			t.Errorf("%s has zero dimensions: %+v", n, nl)
		}
	}
}

// --- Determinism ---

func TestLayoutDeterministic(t *testing.T) {
	build := func() *graph.Graph {
		g := buildGraph(
			[2]string{"a", "b"},
			[2]string{"a", "c"},
			[2]string{"b", "d"},
			[2]string{"c", "d"},
		)
		setWidths(g, 100, 50)
		return g
	}

	r1 := Layout(build(), defaultOpts())
	r2 := Layout(build(), defaultOpts())

	for n, nl := range r1.Nodes {
		if nl != r2.Nodes[n] {
			t.Errorf("determinism broken for %s: %+v vs %+v", n, nl, r2.Nodes[n])
		}
	}
}
