package svgutil

import (
	"math"
	"strings"
	"testing"
)

func TestClipToRectEdge(t *testing.T) {
	cases := []struct {
		name                   string
		cx, cy, w, h, ox, oy   float64
		wantX, wantY           float64
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

func TestNegCoord(t *testing.T) {
	cases := map[float64]string{0: "0.00", 9: "-9.00", -4: "4.00", 1.234: "-1.23"}
	for in, want := range cases {
		if got := NegCoord(in); got != want {
			t.Errorf("NegCoord(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestInlineMarkerAt(t *testing.T) {
	g := InlineMarkerAt(10, 20, 10, 40, 0, 9, []any{&Rect{Width: 1, Height: 1}})
	if !strings.Contains(g.Transform, "translate(10.00,20.00)") {
		t.Errorf("missing anchor translate: %q", g.Transform)
	}
	// path going straight down → angle = 90°.
	if !strings.Contains(g.Transform, "rotate(90.00)") {
		t.Errorf("expected rotate(90.00): %q", g.Transform)
	}
	// refX=0 → "0.00" (not "-0.00"), refY=9 → "-9.00".
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
