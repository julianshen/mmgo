package layout

import "testing"

// Self-loops generate a 4-point cubic bezier whose endpoints sit on
// the node boundary on opposite sides of the rank-progression axis,
// and whose two control points bow against rank progression so the
// arc stays clear of downstream rows.
func TestSelfLoopPoints_TBBowsUpward(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 60, Height: 40}
	pts := selfLoopPoints(nl, RankDirTB)
	if len(pts) != 4 {
		t.Fatalf("expected 4 control points, got %d", len(pts))
	}
	// Exit/entry sit on the top edge (cy - h/2 = 30).
	for i, p := range []Point{pts[0], pts[3]} {
		if p.Y != 30 {
			t.Errorf("endpoint %d Y = %v, want 30 (top edge)", i, p.Y)
		}
	}
	// Control points are above the node (Y < top edge).
	for i, p := range []Point{pts[1], pts[2]} {
		if p.Y >= 30 {
			t.Errorf("cp %d Y = %v, expected < 30 (bowing upward)", i, p.Y)
		}
	}
}

func TestSelfLoopPoints_LRBowsLeftward(t *testing.T) {
	nl := NodeLayout{X: 100, Y: 50, Width: 60, Height: 40}
	pts := selfLoopPoints(nl, RankDirLR)
	if len(pts) != 4 {
		t.Fatalf("expected 4 control points, got %d", len(pts))
	}
	// Exit/entry on the left edge (cx - w/2 = 70).
	for i, p := range []Point{pts[0], pts[3]} {
		if p.X != 70 {
			t.Errorf("endpoint %d X = %v, want 70 (left edge)", i, p.X)
		}
	}
	// Control points further left.
	for i, p := range []Point{pts[1], pts[2]} {
		if p.X >= 70 {
			t.Errorf("cp %d X = %v, expected < 70 (bowing leftward)", i, p.X)
		}
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
