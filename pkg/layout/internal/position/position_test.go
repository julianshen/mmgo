package position

import (
	"math"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
	"github.com/julianshen/mmgo/pkg/layout/internal/graphtest"
	"github.com/julianshen/mmgo/pkg/layout/internal/order"
)

// --- Helpers ---

var buildGraph = graphtest.BuildGraph

// uniformWidth returns a fixed-width function for tests.
func uniformWidth(w float64) NodeWidth {
	return func(string) float64 { return w }
}

func defaultOpts() Options {
	return Options{NodeSep: 50, RankSep: 80}
}

// assertNoOverlaps checks that no two nodes in the same rank overlap
// and that the X ordering matches the slice order (i.e., the order
// phase's chosen ordering is preserved by the position phase).
func assertNoOverlaps(t *testing.T, ord order.Order, widthFn NodeWidth, result Result) {
	t.Helper()
	for r, nodes := range ord {
		// Check X ordering matches slice order and adjacent pairs don't overlap.
		for i := 0; i < len(nodes)-1; i++ {
			a, b := nodes[i], nodes[i+1]
			if result[a].X > result[b].X {
				t.Errorf("rank %d: X order does not match slice order: %s(%f) > %s(%f)",
					r, a, result[a].X, b, result[b].X)
			}
			aRight := result[a].X + widthFn(a)/2
			bLeft := result[b].X - widthFn(b)/2
			if aRight > bLeft {
				t.Errorf("rank %d: %s (right=%f) overlaps %s (left=%f)",
					r, a, aRight, b, bLeft)
			}
		}
		// Also check non-adjacent pairs don't overlap (defense against bugs
		// where slice order and X order diverge).
		for i := 0; i < len(nodes); i++ {
			for j := i + 2; j < len(nodes); j++ {
				a, b := nodes[i], nodes[j]
				aRight := result[a].X + widthFn(a)/2
				bLeft := result[b].X - widthFn(b)/2
				if aRight > bLeft {
					t.Errorf("rank %d: non-adjacent %s (right=%f) overlaps %s (left=%f)",
						r, a, aRight, b, bLeft)
				}
			}
		}
	}
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

// --- Trivial cases ---

func TestRunEmpty(t *testing.T) {
	g := graph.New()
	result := Run(g, order.Order{}, uniformWidth(100), defaultOpts())
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestRunSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	ord := order.Order{0: {"a"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	p, ok := result["a"]
	if !ok {
		t.Fatal("missing node a")
	}
	if p.X < 0 {
		t.Errorf("x should be non-negative after normalization, got %f", p.X)
	}
	if p.Y != 0 {
		t.Errorf("y should be 0 for rank 0 node, got %f", p.Y)
	}
}

// --- Rank spacing ---

func TestRunRankSpacing(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	ord := order.Order{0: {"a"}, 1: {"b"}, 2: {"c"}}
	opts := Options{NodeSep: 50, RankSep: 80}

	result := Run(g, ord, uniformWidth(100), opts)

	// y coordinates should be proportional to rank.
	if !approxEqual(result["b"].Y-result["a"].Y, 80) {
		t.Errorf("expected Y gap of 80, got %f", result["b"].Y-result["a"].Y)
	}
	if !approxEqual(result["c"].Y-result["b"].Y, 80) {
		t.Errorf("expected Y gap of 80, got %f", result["c"].Y-result["b"].Y)
	}
}

// --- Node spacing within rank ---

func TestRunNodeSpacing(t *testing.T) {
	g := buildGraph(
		[2]string{"root", "a"},
		[2]string{"root", "b"},
		[2]string{"root", "c"},
	)
	ord := order.Order{0: {"root"}, 1: {"a", "b", "c"}}
	opts := Options{NodeSep: 50, RankSep: 80}

	result := Run(g, ord, uniformWidth(100), opts)

	// a, b, c should be ordered left-to-right with at least 50 px gap.
	assertNoOverlaps(t, ord, uniformWidth(100), result)

	// Specifically: centers should be (width + nodeSep) = 150 apart.
	if !approxEqual(result["b"].X-result["a"].X, 150) {
		t.Errorf("expected 150 center-to-center, got %f", result["b"].X-result["a"].X)
	}
	if !approxEqual(result["c"].X-result["b"].X, 150) {
		t.Errorf("expected 150 center-to-center, got %f", result["c"].X-result["b"].X)
	}
}

// --- Linear chain ---

func TestRunLinearChain(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"b", "c"})
	ord := order.Order{0: {"a"}, 1: {"b"}, 2: {"c"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	// All three nodes should be vertically aligned (same x) since
	// each has exactly one neighbor in the adjacent rank.
	if !approxEqual(result["a"].X, result["b"].X) {
		t.Errorf("a and b should be vertically aligned: a.x=%f b.x=%f",
			result["a"].X, result["b"].X)
	}
	if !approxEqual(result["b"].X, result["c"].X) {
		t.Errorf("b and c should be vertically aligned: b.x=%f c.x=%f",
			result["b"].X, result["c"].X)
	}
}

// --- Diamond ---

func TestRunDiamond(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"a", "c"},
		[2]string{"b", "d"},
		[2]string{"c", "d"},
	)
	ord := order.Order{
		0: {"a"},
		1: {"b", "c"},
		2: {"d"},
	}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	assertNoOverlaps(t, ord, uniformWidth(100), result)

	// a should be roughly centered between b and c.
	midBC := (result["b"].X + result["c"].X) / 2
	if !approxEqual(result["a"].X, midBC) {
		t.Errorf("a should be centered between b and c: a.x=%f midBC=%f",
			result["a"].X, midBC)
	}

	// d should also be centered between b and c (its two predecessors).
	if !approxEqual(result["d"].X, midBC) {
		t.Errorf("d should be centered between b and c: d.x=%f midBC=%f",
			result["d"].X, midBC)
	}
}

// --- Varying node widths ---

func TestRunVaryingWidths(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"a", "c"})
	ord := order.Order{0: {"a"}, 1: {"b", "c"}}

	widths := map[string]float64{"a": 200, "b": 50, "c": 300}
	widthFn := func(id string) float64 { return widths[id] }

	result := Run(g, ord, widthFn, defaultOpts())

	assertNoOverlaps(t, ord, widthFn, result)

	// Gap between b's right edge and c's left edge should be exactly NodeSep.
	bRight := result["b"].X + widths["b"]/2
	cLeft := result["c"].X - widths["c"]/2
	if !approxEqual(cLeft-bRight, 50) {
		t.Errorf("expected NodeSep=50 gap between b and c, got %f", cLeft-bRight)
	}
}

// --- Non-negative coordinates ---

func TestRunCoordsAreNonNegative(t *testing.T) {
	g := buildGraph([2]string{"a", "b"}, [2]string{"a", "c"}, [2]string{"a", "d"})
	ord := order.Order{0: {"a"}, 1: {"b", "c", "d"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	for n, p := range result {
		if p.X < 0 {
			t.Errorf("%s has negative x: %f", n, p.X)
		}
		if p.Y < 0 {
			t.Errorf("%s has negative y: %f", n, p.Y)
		}
	}
}

// --- Disconnected components ---

func TestRunDisconnectedComponents(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"c", "d"},
	)
	ord := order.Order{0: {"a", "c"}, 1: {"b", "d"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	if len(result) != 4 {
		t.Errorf("expected 4 positions, got %d", len(result))
	}
	assertNoOverlaps(t, ord, uniformWidth(100), result)
}

// --- Determinism ---

func TestRunDeterministic(t *testing.T) {
	build := func() (*graph.Graph, order.Order) {
		g := buildGraph(
			[2]string{"a", "x"},
			[2]string{"a", "y"},
			[2]string{"b", "x"},
			[2]string{"b", "y"},
		)
		return g, order.Order{0: {"a", "b"}, 1: {"x", "y"}}
	}

	g1, o1 := build()
	g2, o2 := build()

	r1 := Run(g1, o1, uniformWidth(100), defaultOpts())
	r2 := Run(g2, o2, uniformWidth(100), defaultOpts())

	for n, p := range r1 {
		if !approxEqual(p.X, r2[n].X) || !approxEqual(p.Y, r2[n].Y) {
			t.Errorf("determinism broken: %s %v vs %v", n, p, r2[n])
		}
	}
}

// --- Isolated rank (no predecessors or successors) ---

func TestRunIsolatedRank(t *testing.T) {
	g := graph.New()
	g.SetNode("a", graph.NodeAttrs{})
	g.SetNode("b", graph.NodeAttrs{})
	g.SetNode("c", graph.NodeAttrs{})
	ord := order.Order{0: {"a", "b", "c"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	assertNoOverlaps(t, ord, uniformWidth(100), result)
	// All nodes at rank 0 → y = 0.
	for n, p := range result {
		if p.Y != 0 {
			t.Errorf("%s should be at y=0, got %f", n, p.Y)
		}
	}
}

// --- Internal helpers ---

func TestMedianInPlace(t *testing.T) {
	cases := []struct {
		name string
		in   []float64
		want float64
	}{
		{"empty", nil, 0},
		{"single", []float64{5}, 5},
		{"odd", []float64{1, 3, 5}, 3},
		{"even", []float64{2, 4, 6, 8}, 5}, // (4+6)/2
		{"unsorted", []float64{9, 1, 5}, 5},
		{"negatives", []float64{-3, -1, 2}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clone because medianInPlace sorts in place.
			in := append([]float64(nil), tc.in...)
			if got := medianInPlace(in); got != tc.want {
				t.Errorf("medianInPlace(%v) = %f, want %f", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeEmpty(t *testing.T) {
	result := Result{}
	normalize(result) // must not panic
	if len(result) != 0 {
		t.Error("normalize should leave empty result unchanged")
	}
}

// --- Large fan-out centering ---

func TestRunWideFanOutCentering(t *testing.T) {
	// Single parent with 5 children. Parent should be centered over
	// the middle of the children after the median alignment pass.
	g := buildGraph(
		[2]string{"root", "c1"},
		[2]string{"root", "c2"},
		[2]string{"root", "c3"},
		[2]string{"root", "c4"},
		[2]string{"root", "c5"},
	)
	ord := order.Order{0: {"root"}, 1: {"c1", "c2", "c3", "c4", "c5"}}

	result := Run(g, ord, uniformWidth(100), defaultOpts())

	// root should be positioned near c3 (the middle child).
	diff := math.Abs(result["root"].X - result["c3"].X)
	if diff > 1 {
		t.Errorf("root should align with middle child c3: root.x=%f c3.x=%f",
			result["root"].X, result["c3"].X)
	}
}
