package mindmap

import (
	"testing"
)

func TestShapeFillColorRoot(t *testing.T) {
	th := DefaultTheme()
	got := shapeFillColor(0, 0, th)
	if got != th.RootColor {
		t.Errorf("root fill = %q, want %q", got, th.RootColor)
	}
}

func TestShapeFillColorSection(t *testing.T) {
	th := DefaultTheme()
	got := shapeFillColor(0, 1, th)
	if got != th.SectionColors[0] {
		t.Errorf("section 0 fill = %q, want %q", got, th.SectionColors[0])
	}
}

func TestShapeFillColorWrapsAround(t *testing.T) {
	th := DefaultTheme()
	got := shapeFillColor(len(th.SectionColors), 1, th)
	if got != th.SectionColors[0] {
		t.Errorf("wrapped fill = %q, want %q", got, th.SectionColors[0])
	}
}

func TestShapeFillColorEmptyColors(t *testing.T) {
	th := Theme{RootColor: "#root", EdgeStroke: "#edge"}
	got := shapeFillColor(0, 1, th)
	if got != "#edge" {
		t.Errorf("empty SectionColors fill = %q, want #edge", got)
	}
}

func TestShapeTextColorRoot(t *testing.T) {
	th := Theme{RootText: "#rt"}
	got := shapeTextColor(0, th)
	if got != "#rt" {
		t.Errorf("root text color = %q, want #rt", got)
	}
}

func TestShapeTextColorNode(t *testing.T) {
	th := Theme{NodeText: "#nt"}
	got := shapeTextColor(1, th)
	if got != "#nt" {
		t.Errorf("node text color = %q, want #nt", got)
	}
}

func TestEdgeStrokeWidth(t *testing.T) {
	cases := []struct {
		depth int
		want  float64
	}{
		{0, 17},
		{1, 17},
		{2, 14},
		{3, 11},
		{5, 5},
		{10, 2},
	}
	for _, tc := range cases {
		got := edgeStrokeWidth(tc.depth)
		if got != tc.want {
			t.Errorf("edgeStrokeWidth(%d) = %.0f, want %.0f", tc.depth, got, tc.want)
		}
	}
}

func TestEdgeColor(t *testing.T) {
	th := DefaultTheme()
	got := edgeColor(1, th)
	if got != th.SectionColors[1] {
		t.Errorf("edgeColor(1) = %q, want %q", got, th.SectionColors[1])
	}
}

func TestEdgeColorEmptyColors(t *testing.T) {
	th := Theme{EdgeStroke: "#fallback"}
	got := edgeColor(0, th)
	if got != "#fallback" {
		t.Errorf("edgeColor with empty colors = %q, want #fallback", got)
	}
}

func TestHexagonPointsNotSelfIntersecting(t *testing.T) {
	pts := hexagonPoints(20, 200)
	if pts == "" {
		t.Error("hexagonPoints returned empty string")
	}
}

func TestDefaultNodePath(t *testing.T) {
	p := defaultNodePath(100, 40)
	if p == "" {
		t.Error("defaultNodePath returned empty string")
	}
}

func TestCloudPath(t *testing.T) {
	p := cloudPath(100, 40)
	if p == "" {
		t.Error("cloudPath returned empty string")
	}
}

func TestBangPath(t *testing.T) {
	p := bangPath(100, 40)
	if p == "" {
		t.Error("bangPath returned empty string")
	}
}

func TestCurvedEdgePathNormal(t *testing.T) {
	p := curvedEdgePath(0, 0, 100, 100)
	if p == "" {
		t.Error("curvedEdgePath returned empty string")
	}
}

func TestCurvedEdgePathDegenerate(t *testing.T) {
	p := curvedEdgePath(50, 50, 50, 50)
	if p == "" {
		t.Error("curvedEdgePath returned empty for degenerate input")
	}
}
