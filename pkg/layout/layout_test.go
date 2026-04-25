package layout

import (
	"reflect"
	"math"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/graphtest"
)

// --- Helpers ---

var (
	buildGraph = graphtest.BuildGraph
	setWidths  = graphtest.SetWidths
)

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
	// one edge internally; the result must still expose all three
	// edges with their ORIGINAL directions and EdgeIDs.
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "a"},
	)
	setWidths(g, 100, 50)

	// Capture the original (From, To) pairs — these are what the
	// caller passed in and what the output must preserve regardless
	// of any internal reversal.
	want := map[[2]string]bool{
		{"a", "b"}: true,
		{"b", "c"}: true,
		{"c", "a"}: true,
	}

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(result.Edges))
	}

	for eid := range result.Edges {
		key := [2]string{eid.From, eid.To}
		if !want[key] {
			t.Errorf("edge %v is not in the original input; acyclic reversal leaked into output", eid)
		}
		delete(want, key)
	}
	if len(want) > 0 {
		t.Errorf("missing original edges: %v", want)
	}
}

// graphSnapshot captures enough of a graph's state to detect any mutation
// Layout might perform via aliasing. Used to strengthen the non-mutation
// invariant check beyond simple node/edge counts.
type graphSnapshot struct {
	nodes map[string]graph.NodeAttrs
	edges map[graph.EdgeID]graph.EdgeAttrs
}

func snapshotGraph(g *graph.Graph) graphSnapshot {
	s := graphSnapshot{
		nodes: make(map[string]graph.NodeAttrs),
		edges: make(map[graph.EdgeID]graph.EdgeAttrs),
	}
	for _, n := range g.Nodes() {
		attrs, _ := g.NodeAttrs(n)
		s.nodes[n] = attrs
	}
	for _, eid := range g.Edges() {
		attrs, _ := g.EdgeAttrs(eid)
		s.edges[eid] = attrs
	}
	return s
}

func (a graphSnapshot) equal(b graphSnapshot) bool {
	if len(a.nodes) != len(b.nodes) || len(a.edges) != len(b.edges) {
		return false
	}
	for n, av := range a.nodes {
		if bv, ok := b.nodes[n]; !ok || av != bv {
			return false
		}
	}
	for eid, av := range a.edges {
		if bv, ok := b.edges[eid]; !ok || av != bv {
			return false
		}
	}
	return true
}

func TestLayoutDoesNotMutateInput(t *testing.T) {
	tests := []struct {
		name  string
		build func() *graph.Graph
	}{
		{
			name: "cycle",
			build: func() *graph.Graph {
				g := buildGraph([2]string{"a", "b"}, [2]string{"b", "a"})
				setWidths(g, 100, 50)
				return g
			},
		},
		{
			name: "diamond with distinct node and edge attrs",
			build: func() *graph.Graph {
				g := graph.New()
				// Set distinct node attrs so aliasing shows up clearly.
				for i, n := range []string{"a", "b", "c", "d"} {
					g.SetNode(n, graph.NodeAttrs{
						Label:  n,
						Width:  float64(100 + i*10),
						Height: float64(50 + i*5),
					})
				}
				// Distinct edge attrs so edge-attr aliasing is caught.
				pairs := [][2]string{
					{"a", "b"}, {"a", "c"}, {"b", "d"}, {"c", "d"},
				}
				for i, p := range pairs {
					g.SetEdge(p[0], p[1], graph.EdgeAttrs{
						Label:  p[0] + "→" + p[1],
						Weight: float64(i + 1),
					})
				}
				return g
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := tc.build()
			before := snapshotGraph(g)
			Layout(g, defaultOpts())
			after := snapshotGraph(g)
			if !before.equal(after) {
				t.Errorf("input graph mutated during Layout")
			}
		})
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

func TestLayoutRankDirs(t *testing.T) {
	// For a→b→c, each direction should produce a consistent ordering
	// along one axis.
	tests := []struct {
		name  string
		dir   RankDir
		check func(a, b, c NodeLayout) (ok bool, msg string)
	}{
		{
			name: "TB",
			dir:  RankDirTB,
			check: func(a, b, c NodeLayout) (bool, string) {
				return a.Y < b.Y && b.Y < c.Y, "a should be above b above c (Y ascending)"
			},
		},
		{
			name: "BT",
			dir:  RankDirBT,
			check: func(a, b, c NodeLayout) (bool, string) {
				return a.Y > b.Y && b.Y > c.Y, "a should be below b below c (Y descending)"
			},
		},
		{
			name: "LR",
			dir:  RankDirLR,
			check: func(a, b, c NodeLayout) (bool, string) {
				return a.X < b.X && b.X < c.X, "a should be left of b left of c (X ascending)"
			},
		},
		{
			name: "RL",
			dir:  RankDirRL,
			check: func(a, b, c NodeLayout) (bool, string) {
				return a.X > b.X && b.X > c.X, "a should be right of b right of c (X descending)"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
			setWidths(g, 100, 50)

			opts := defaultOpts()
			opts.RankDir = tc.dir
			result := Layout(g, opts)

			ok, msg := tc.check(result.Nodes["a"], result.Nodes["b"], result.Nodes["c"])
			if !ok {
				t.Errorf("%s: %s — got a=%+v b=%+v c=%+v",
					tc.dir, msg, result.Nodes["a"], result.Nodes["b"], result.Nodes["c"])
			}
		})
	}
}

func TestRankDirString(t *testing.T) {
	cases := map[RankDir]string{
		RankDirTB: "TB",
		RankDirBT: "BT",
		RankDirLR: "LR",
		RankDirRL: "RL",
	}
	for d, want := range cases {
		if got := d.String(); got != want {
			t.Errorf("RankDir(%d).String() = %q, want %q", d, got, want)
		}
	}
	if RankDir(99).String() != "unknown" {
		t.Error("out-of-range RankDir should stringify as 'unknown'")
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
		if !reflect.DeepEqual(nl, r2.Nodes[n]) {
			t.Errorf("determinism broken for node %s: %+v vs %+v", n, nl, r2.Nodes[n])
		}
	}
	// Edge control points and label positions must also be deterministic.
	for eid, el := range r1.Edges {
		other, ok := r2.Edges[eid]
		if !ok {
			t.Errorf("edge %v missing from r2", eid)
			continue
		}
		if len(el.Points) != len(other.Points) {
			t.Errorf("edge %v point count differs: %d vs %d",
				eid, len(el.Points), len(other.Points))
			continue
		}
		for i := range el.Points {
			if el.Points[i] != other.Points[i] {
				t.Errorf("edge %v point %d differs: %+v vs %+v",
					eid, i, el.Points[i], other.Points[i])
			}
		}
		if el.LabelPos != other.LabelPos {
			t.Errorf("edge %v label pos differs: %+v vs %+v",
				eid, el.LabelPos, other.LabelPos)
		}
	}
}

// --- Nil safety ---

// TestLayoutNilGraph verifies that Layout(nil, opts) degrades
// gracefully to an empty Result rather than panicking on g.Copy().
func TestLayoutNilGraph(t *testing.T) {
	result := Layout(nil, defaultOpts())
	if result == nil {
		t.Fatal("Layout(nil) should return non-nil empty Result")
	}
	if len(result.Nodes) != 0 || len(result.Edges) != 0 {
		t.Errorf("expected empty Result, got nodes=%d edges=%d",
			len(result.Nodes), len(result.Edges))
	}
}

// --- Self-loops ---

// TestLayoutSelfLoop documents the current straight-line routing
// behavior for self-loops: the edge collapses to two identical points
// (the node's center). This is a known limitation tracked in the
// buildEdges TODO(features) comment. When orthogonal/spline routing
// is added, self-loops should become proper loop-back arcs; this test
// will then need to be updated.
func TestLayoutSelfLoop(t *testing.T) {
	g := graph.New()
	g.SetEdge("a", "a", graph.EdgeAttrs{})
	setWidths(g, 100, 50)

	result := Layout(g, defaultOpts())

	if len(result.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(result.Nodes))
	}
	if len(result.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(result.Edges))
	}
	// Self-loop is now a 4-point cubic-bezier (exit, cp1, cp2, entry)
	// generated by selfLoopPoints — no longer the degenerate 2-point
	// collapse. The exit and entry sit on the node boundary; the two
	// control points bow upstream of the node.
	for _, el := range result.Edges {
		if len(el.Points) != 4 {
			t.Fatalf("self-loop should have 4 bezier points, got %d", len(el.Points))
		}
		if el.Points[0] == el.Points[3] {
			t.Errorf("exit and entry must differ, got %+v", el.Points)
		}
	}
}

// --- LR/RL dimension packing ---

// TestLayoutLRUsesHeightForPacking verifies that in LR rank direction,
// tall-narrow nodes in the same column don't overlap vertically. This
// is a regression guard for the bug where position.Run was always
// packed by width, causing vertical overlap for tall nodes in LR/RL
// layouts.
func TestLayoutLRUsesHeightForPacking(t *testing.T) {
	// Two tall, narrow nodes in the same rank (column after LR rotation).
	g := buildGraph([2]string{"root", "a"}, [2]string{"root", "b"})
	g.SetNode("root", graph.NodeAttrs{Width: 20, Height: 20})
	g.SetNode("a", graph.NodeAttrs{Width: 20, Height: 200})
	g.SetNode("b", graph.NodeAttrs{Width: 20, Height: 200})

	opts := defaultOpts()
	opts.RankDir = RankDirLR
	result := Layout(g, opts)

	// In LR, a and b are in the same column (rank 1) stacked vertically.
	// With heights of 200 each and NodeSep of 50, their centers should
	// be at least 250 apart vertically to avoid overlap.
	aY := result.Nodes["a"].Y
	bY := result.Nodes["b"].Y
	if math.Abs(aY-bY) < 200 {
		t.Errorf("a and b should not overlap vertically in LR mode (need gap >= 200, got %f)",
			math.Abs(aY-bY))
	}
}
