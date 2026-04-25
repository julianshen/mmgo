package layout

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/layout/graph"
)

// End-to-end propagation: a cyclic graph through Layout() flags
// at least one EdgeLayout with BackEdge=true, with the original
// (From, To) preserved (no leaked reversed direction).
func TestLayoutBackEdgeFlagPropagates(t *testing.T) {
	g := buildGraph(
		[2]string{"a", "b"},
		[2]string{"b", "c"},
		[2]string{"c", "a"}, // closes the cycle
	)
	setWidths(g, 100, 50)

	r := Layout(g, defaultOpts())
	var backCount int
	seen := map[[2]string]bool{}
	for eid, el := range r.Edges {
		seen[[2]string{eid.From, eid.To}] = true
		if el.BackEdge {
			backCount++
		}
	}
	if backCount == 0 {
		t.Error("a 3-node cycle must produce at least one EdgeLayout with BackEdge=true")
	}
	for _, want := range [][2]string{{"a", "b"}, {"b", "c"}, {"c", "a"}} {
		if !seen[want] {
			t.Errorf("original edge %v missing — reversed direction leaked into output", want)
		}
	}
	_ = graph.EdgeID{} // import sentinel
}

func TestSelfLoopPoints_TBBowsUpward(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 60, Height: 40}
	pts := selfLoopPoints(nl, RankDirTB)
	if len(pts) != 4 {
		t.Fatalf("expected 4 control points, got %d", len(pts))
	}
	if !approxEqual(pts[0].Y, 30) {
		t.Errorf("exit.Y = %v, want ~30 (top edge)", pts[0].Y)
	}
	if !approxEqual(pts[3].Y, 30) {
		t.Errorf("entry.Y = %v, want ~30 (top edge)", pts[3].Y)
	}
	if pts[1].Y >= 30 {
		t.Errorf("cp1.Y = %v, expected < 30 (bowing upward)", pts[1].Y)
	}
	if pts[2].Y >= 30 {
		t.Errorf("cp2.Y = %v, expected < 30 (bowing upward)", pts[2].Y)
	}
}

func TestSelfLoopPoints_LRBowsLeftward(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 60, Height: 40}
	pts := selfLoopPoints(nl, RankDirLR)
	if len(pts) != 4 {
		t.Fatalf("expected 4 control points, got %d", len(pts))
	}
	if !approxEqual(pts[0].X, 70) {
		t.Errorf("exit.X = %v, want ~70 (left edge)", pts[0].X)
	}
	if !approxEqual(pts[3].X, 70) {
		t.Errorf("entry.X = %v, want ~70 (left edge)", pts[3].X)
	}
	if pts[1].X >= 70 {
		t.Errorf("cp1.X = %v, expected < 70 (bowing leftward)", pts[1].X)
	}
	if pts[2].X >= 70 {
		t.Errorf("cp2.X = %v, expected < 70 (bowing leftward)", pts[2].X)
	}
}

func TestSelfLoopAllRankDirsProduceFourPoints(t *testing.T) {
	nl := NodeLayout{X: 0, Y: 0, Width: 100, Height: 50}
	for _, dir := range []RankDir{RankDirTB, RankDirBT, RankDirLR, RankDirRL} {
		pts := selfLoopPoints(nl, dir)
		if len(pts) != 4 {
			t.Errorf("dir=%v: got %d points, want 4", dir, len(pts))
		}
	}
}
