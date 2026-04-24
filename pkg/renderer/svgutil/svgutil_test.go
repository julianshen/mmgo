package svgutil

import (
	"math"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/layout"
)

func TestClipToRectEdge(t *testing.T) {
	cases := []struct {
		name                 string
		cx, cy, w, h, ox, oy float64
		wantX, wantY         float64
	}{
		{"east", 0, 0, 10, 6, 100, 0, 5, 0},
		{"west", 0, 0, 10, 6, -100, 0, -5, 0},
		{"north", 0, 0, 10, 6, 0, -100, 0, -3},
		{"south", 0, 0, 10, 6, 0, 100, 0, 3},
		{"NE-w-limited", 0, 0, 10, 100, 50, 50, 5, 5},
		{"NE-h-limited", 0, 0, 100, 10, 50, 50, 5, 5},
		{"coincident", 3, 4, 10, 6, 3, 4, 3, 4},
		{"interior-clamped", 0, 0, 100, 100, 5, 5, 5, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := ClipToRectEdge(tc.cx, tc.cy, tc.w, tc.h, tc.ox, tc.oy)
			if math.Abs(x-tc.wantX) > 1e-9 || math.Abs(y-tc.wantY) > 1e-9 {
				t.Errorf("ClipToRectEdge=(%v,%v) want=(%v,%v)", x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestClipToCircleEdge(t *testing.T) {
	cases := []struct {
		name              string
		cx, cy, r, ox, oy float64
		wantX, wantY      float64
	}{
		{"east", 0, 0, 5, 100, 0, 5, 0},
		{"west", 0, 0, 5, -100, 0, -5, 0},
		{"north", 0, 0, 5, 0, -100, 0, -5},
		{"south", 0, 0, 5, 0, 100, 0, 5},
		{"diagonal-NE", 0, 0, 10, 30, 40, 6, 8},
		{"offset-center", 3, 4, 5, 3, 4, 3, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := ClipToCircleEdge(tc.cx, tc.cy, tc.r, tc.ox, tc.oy)
			if math.Abs(x-tc.wantX) > 1e-9 || math.Abs(y-tc.wantY) > 1e-9 {
				t.Errorf("ClipToCircleEdge=(%v,%v) want=(%v,%v)", x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestClipToDiamondEdge(t *testing.T) {
	cases := []struct {
		name                 string
		cx, cy, w, h, ox, oy float64
		wantX, wantY         float64
	}{
		// Diamond of w=20, h=10 has vertices at (±10, 0) and (0, ±5).
		{"east-vertex", 0, 0, 20, 10, 100, 0, 10, 0},
		{"west-vertex", 0, 0, 20, 10, -100, 0, -10, 0},
		{"north-vertex", 0, 0, 20, 10, 0, -100, 0, -5},
		{"south-vertex", 0, 0, 20, 10, 0, 100, 0, 5},
		// Off-axis: with w=h=10 (square diamond), a reference at
		// (10,10) hits the NE edge at (2.5, 2.5) — 1/(2+2) = 0.25,
		// 10 * 0.25 = 2.5.
		{"diagonal-NE", 0, 0, 10, 10, 10, 10, 2.5, 2.5},
		// Reference already inside: result clamps to the reference
		// (parallel to ClipToRectEdge's inside-clamp safety).
		{"inside-clamps", 0, 0, 20, 10, 1, 0, 1, 0},
		{"at-center", 0, 0, 20, 10, 0, 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x, y := ClipToDiamondEdge(tc.cx, tc.cy, tc.w, tc.h, tc.ox, tc.oy)
			if math.Abs(x-tc.wantX) > 1e-9 || math.Abs(y-tc.wantY) > 1e-9 {
				t.Errorf("ClipToDiamondEdge=(%v,%v) want=(%v,%v)", x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestNegCoord(t *testing.T) {
	cases := map[float64]string{0: "0.00", 9: "-9.00", -4: "4.00", 1.234: "-1.23"}
	for in, want := range cases {
		if got := NegCoord(in); got != want {
			t.Errorf("NegCoord(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestInlineMarkerAt(t *testing.T) {
	// SVG uses Y-down, so atan2(dy, dx) gives angles where:
	//   east  +X = 0°, south +Y = 90°, west -X = 180°, north -Y = -90°.
	cases := []struct {
		name          string
		start, next   [2]float64
		wantRotateDeg string
	}{
		{"east", [2]float64{0, 0}, [2]float64{10, 0}, "rotate(0.00)"},
		{"south", [2]float64{0, 0}, [2]float64{0, 10}, "rotate(90.00)"},
		{"west", [2]float64{0, 0}, [2]float64{-10, 0}, "rotate(180.00)"},
		{"north", [2]float64{0, 0}, [2]float64{0, -10}, "rotate(-90.00)"},
		{"ne-diagonal", [2]float64{0, 0}, [2]float64{10, -10}, "rotate(-45.00)"},
		{"coincident", [2]float64{5, 5}, [2]float64{5, 5}, "rotate(0.00)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := InlineMarkerAt(tc.start[0], tc.start[1], tc.next[0], tc.next[1], 0, 9, nil)
			if !strings.Contains(g.Transform, tc.wantRotateDeg) {
				t.Errorf("expected %q in %q", tc.wantRotateDeg, g.Transform)
			}
		})
	}

	// Threaded fields: anchor translate, negCoord on refX/refY, children.
	g := InlineMarkerAt(10, 20, 10, 40, 0, 9, []any{&Rect{Width: 1, Height: 1}})
	if !strings.Contains(g.Transform, "translate(10.00,20.00)") {
		t.Errorf("missing anchor translate: %q", g.Transform)
	}
	if !strings.Contains(g.Transform, "translate(0.00,-9.00)") {
		t.Errorf("expected final translate(0.00,-9.00): %q", g.Transform)
	}
	if len(g.Children) != 1 {
		t.Errorf("children not threaded through")
	}
}

func TestRound2(t *testing.T) {
	if Round2(1.456) != 1.46 {
		t.Errorf("Round2(1.456) = %v", Round2(1.456))
	}
	if Round2(math.NaN()) != 0 {
		t.Error("NaN should round to 0")
	}
	if Round2(math.Inf(1)) != 0 {
		t.Error("Inf should round to 0")
	}
}

func TestSanitize(t *testing.T) {
	if Sanitize(-1) != 0 {
		t.Error("negative should sanitize to 0")
	}
	if Sanitize(42) != 42 {
		t.Error("positive should pass through")
	}
}

func TestViewBox(t *testing.T) {
	vb := ViewBox(100.5, 200)
	if vb != "0 0 100.50 200.00" {
		t.Errorf("ViewBox = %q", vb)
	}
}

func TestCatmullRomPath(t *testing.T) {
	// Fewer than 3 points: nothing to curve.
	if got := CatmullRomPath(nil, CatmullRomTension); got != "" {
		t.Errorf("nil pts: got %q, want empty", got)
	}
	if got := CatmullRomPath([]layout.Point{{X: 0, Y: 0}, {X: 10, Y: 0}}, CatmullRomTension); got != "" {
		t.Errorf("2 pts: got %q, want empty", got)
	}

	// 3+ points: starts with a moveto and emits one cubic per segment.
	pts := []layout.Point{{X: 0, Y: 0}, {X: 10, Y: 5}, {X: 20, Y: 0}}
	d := CatmullRomPath(pts, CatmullRomTension)
	if !strings.HasPrefix(d, "M0.00,0.00") {
		t.Errorf("missing moveto: %q", d)
	}
	if strings.Count(d, " C") != len(pts)-1 {
		t.Errorf("expected %d cubic segments in %q", len(pts)-1, d)
	}
}

func TestLabelChip(t *testing.T) {
	chip := LabelChip(50, 30, 20, 10, 4, "#fff", 3)
	if chip.X != 36 || chip.Y != 21 {
		t.Errorf("origin = (%v,%v), want (36,21)", chip.X, chip.Y)
	}
	if chip.Width != 28 || chip.Height != 18 {
		t.Errorf("size = (%v,%v), want (28,18)", chip.Width, chip.Height)
	}
	if chip.RX != 3 || chip.RY != 3 {
		t.Errorf("corners = (%v,%v), want (3,3)", chip.RX, chip.RY)
	}
	if !strings.Contains(chip.Style, "fill:#fff") || !strings.Contains(chip.Style, "stroke:none") {
		t.Errorf("style = %q, want fill:#fff and stroke:none", chip.Style)
	}

	// Square chip (cornerR=0) leaves rx/ry off via omitempty.
	square := LabelChip(0, 0, 10, 10, 2, "#000", 0)
	if square.RX != 0 || square.RY != 0 {
		t.Errorf("square corners = (%v,%v), want (0,0)", square.RX, square.RY)
	}
}

func TestMarshalSVG(t *testing.T) {
	doc := Doc{XMLNS: "http://www.w3.org/2000/svg", ViewBox: "0 0 100 100"}
	out, err := MarshalSVG(doc)
	if err != nil {
		t.Fatalf("MarshalSVG: %v", err)
	}
	if len(out) == 0 {
		t.Error("output should not be empty")
	}
}
