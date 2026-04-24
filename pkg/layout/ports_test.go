package layout

import (
	"math"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// Diamond with 3 outlets on TB: ports land on the three non-top
// vertices (left-midpoint, bottom, right-midpoint).
func TestPortPositions_DiamondTB3Edges(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 60, Height: 40}
	got := portPositions(nl, graph.ShapeDiamond, RankDirTB, 3)
	want := []Point{
		{X: 70, Y: 50},  // left vertex
		{X: 100, Y: 70}, // bottom vertex
		{X: 130, Y: 50}, // right vertex
	}
	if len(got) != 3 {
		t.Fatalf("want 3 ports, got %d", len(got))
	}
	for i := range want {
		if math.Abs(got[i].X-want[i].X) > 0.001 || math.Abs(got[i].Y-want[i].Y) > 0.001 {
			t.Errorf("port %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// Default/rect shape: 3 outlets on TB distribute evenly along the
// bottom edge, landing at x = left + (1,2,3)/4 * width.
func TestPortPositions_DefaultTB3Edges(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 40, Height: 20}
	got := portPositions(nl, graph.ShapeDefault, RankDirTB, 3)
	// Bottom edge y = 50 + 10 = 60; x spread over [80, 120].
	wantY := 60.0
	wantX := []float64{90, 100, 110}
	for i := 0; i < 3; i++ {
		if math.Abs(got[i].Y-wantY) > 0.001 {
			t.Errorf("port %d Y = %v, want %v", i, got[i].Y, wantY)
		}
		if math.Abs(got[i].X-wantX[i]) > 0.001 {
			t.Errorf("port %d X = %v, want %v", i, got[i].X, wantX[i])
		}
	}
}

// LR direction: ports sit on the RIGHT edge of the bbox.
func TestPortPositions_DefaultLR3Edges(t *testing.T) {
	nl := NodeLayout{X: 50, Y: 50, Width: 20, Height: 40}
	got := portPositions(nl, graph.ShapeDefault, RankDirLR, 3)
	wantX := 60.0 // right edge
	for i := 0; i < 3; i++ {
		if math.Abs(got[i].X-wantX) > 0.001 {
			t.Errorf("port %d X = %v, want %v (right edge)", i, got[i].X, wantX)
		}
	}
}

// bendPointFor moves along rank progression regardless of port position.
func TestBendPointFor(t *testing.T) {
	port := Point{X: 50, Y: 50}
	tests := []struct {
		dir  RankDir
		want Point
	}{
		{RankDirTB, Point{X: 50, Y: 60}},
		{RankDirBT, Point{X: 50, Y: 40}},
		{RankDirLR, Point{X: 60, Y: 50}},
		{RankDirRL, Point{X: 40, Y: 50}},
	}
	for _, tc := range tests {
		got := bendPointFor(port, tc.dir, 10)
		if got != tc.want {
			t.Errorf("dir=%v: got %+v want %+v", tc.dir, got, tc.want)
		}
	}
}

// End-to-end check: a node with 3 outgoing edges gets ExitPorts
// populated; a node with 2 outgoing edges keeps ExitPorts nil
// (backward-compat with pre-Phase-B behavior).
func TestAssignExitPorts_ThresholdAndSkip(t *testing.T) {
	g := graph.New()
	// Fork: S → A, S → B, S → C (3 outlets → gets ports).
	g.SetNode("S", graph.NodeAttrs{Width: 60, Height: 40, Shape: graph.ShapeDiamond})
	for _, n := range []string{"A", "B", "C"} {
		g.SetNode(n, graph.NodeAttrs{Width: 40, Height: 20})
		g.SetEdge("S", n, graph.EdgeAttrs{})
	}
	// Two-outlet fork: T → X, T → Y (no ports).
	g.SetNode("T", graph.NodeAttrs{Width: 60, Height: 40})
	g.SetNode("X", graph.NodeAttrs{Width: 40, Height: 20})
	g.SetNode("Y", graph.NodeAttrs{Width: 40, Height: 20})
	g.SetEdge("T", "X", graph.EdgeAttrs{})
	g.SetEdge("T", "Y", graph.EdgeAttrs{})

	r := Layout(g, Options{NodeSep: 50, RankSep: 80})
	if n := len(r.Nodes["S"].ExitPorts); n != 3 {
		t.Errorf("S (3 outlets) ExitPorts = %d, want 3", n)
	}
	if r.Nodes["T"].ExitPorts != nil {
		t.Errorf("T (2 outlets) ExitPorts = %v, want nil", r.Nodes["T"].ExitPorts)
	}
}
