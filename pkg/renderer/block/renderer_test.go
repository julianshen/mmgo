package block

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.BlockDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSimpleBlocks(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b", Label: "flows"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") {
		t.Error("block labels missing")
	}
	if !strings.Contains(raw, ">flows<") {
		t.Error("edge label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderAllShapes(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "r", Label: "R", Shape: diagram.BlockShapeRect},
			{ID: "o", Label: "O", Shape: diagram.BlockShapeRound},
			{ID: "d", Label: "D", Shape: diagram.BlockShapeDiamond},
			{ID: "s", Label: "S", Shape: diagram.BlockShapeStadium},
			{ID: "c", Label: "C", Shape: diagram.BlockShapeCircle},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<polygon") {
		t.Error("diamond should produce polygon")
	}
	if !strings.Contains(raw, "<circle") {
		t.Error("circle shape should produce circle element")
	}
	assertValidSVG(t, out)
}

func TestRenderColumnsHint(t *testing.T) {
	d := &diagram.BlockDiagram{
		Columns: 3,
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderEdgeLabels(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
			{ID: "d", Label: "D"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b", Label: "step1"},
			{From: "a", To: "c", Label: "step2"},
			{From: "b", To: "d"},
			{From: "c", To: "d"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">step1<") || !strings.Contains(raw, ">step2<") {
		t.Error("edge labels missing")
	}
	assertValidSVG(t, out)
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{{ID: "a", Label: "A"}},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{
			{ID: "a", Label: "A"},
			{ID: "b", Label: "B"},
			{ID: "c", Label: "C"},
		},
		Edges: []diagram.BlockEdge{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}
	first, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for i := 0; i < 10; i++ {
		next, err := Render(d, nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func assertValidSVG(t *testing.T, svgBytes []byte) {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
}

func TestRenderAppliesCustomTheme(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{{ID: "A", Label: "A"}, {ID: "B", Label: "B"}},
		Edges: []diagram.BlockEdge{{From: "A", To: "B", Label: "go"}},
	}
	out, err := Render(d, &Options{Theme: Theme{
		NodeFill:   "#111111",
		NodeStroke: "#aabbcc",
		NodeText:   "#ddeeff",
		EdgeStroke: "#223344",
		EdgeText:   "#556677",
		Background: "#000000",
	}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	for _, want := range []string{`fill:#000000`, `fill:#111111;stroke:#aabbcc`, `fill:#ddeeff`, `stroke:#223344`, `fill:#556677`} {
		if !strings.Contains(raw, want) {
			t.Errorf("themed output missing %q", want)
		}
	}
}

func TestDefaultThemeStable(t *testing.T) {
	got := DefaultTheme()
	want := Theme{
		NodeFill:   "#ECECFF",
		NodeStroke: "#9370DB",
		NodeText:   "#333",
		EdgeStroke: "#333",
		EdgeText:   "#333",
		Background: "#fff",
	}
	if got != want {
		t.Errorf("DefaultTheme drifted:\n got  %+v\n want %+v", got, want)
	}
}

// AccTitle/AccDescr emit as <title>/<desc> SVG children.
func TestRenderBlockAccessibility(t *testing.T) {
	d := &diagram.BlockDiagram{
		AccTitle: "System layout",
		AccDescr: "Top-level boxes",
		Nodes: []diagram.BlockNode{
			{ID: "A", Label: "A"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>System layout</title>") {
		t.Errorf("expected <title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Top-level boxes</desc>") {
		t.Errorf("expected <desc> in:\n%s", raw)
	}
}

// Each new shape produces its hallmark SVG element.
func TestRenderBlockExtendedShapes(t *testing.T) {
	cases := []struct {
		shape diagram.BlockShape
		want  string
	}{
		{diagram.BlockShapeHexagon, "<polygon"},
		{diagram.BlockShapeSubroutine, "<line"},
		{diagram.BlockShapeDoubleCircle, "<circle"},
		{diagram.BlockShapeCylinder, "<path"},
	}
	for _, tc := range cases {
		d := &diagram.BlockDiagram{
			Nodes: []diagram.BlockNode{
				{ID: "A", Label: "A", Shape: tc.shape},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("shape %v render: %v", tc.shape, err)
		}
		if !strings.Contains(string(out), tc.want) {
			t.Errorf("shape %v: expected %q in output", tc.shape, tc.want)
		}
	}
}

// Each new shape produces a polygon glyph (asymmetric flag,
// parallelograms, trapezoids).
func TestRenderBlockPhaseBShapes(t *testing.T) {
	for _, shape := range []diagram.BlockShape{
		diagram.BlockShapeAsymmetric,
		diagram.BlockShapeParallelogram,
		diagram.BlockShapeParallelogramAlt,
		diagram.BlockShapeTrapezoid,
		diagram.BlockShapeTrapezoidAlt,
	} {
		d := &diagram.BlockDiagram{
			Nodes: []diagram.BlockNode{{ID: "A", Label: "A", Shape: shape}},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("shape %v: %v", shape, err)
		}
		if !strings.Contains(string(out), "<polygon") {
			t.Errorf("shape %v: expected <polygon> in output", shape)
		}
	}
}

// Each LineStyle/ArrowHead combination produces the expected
// marker reference / dasharray / stroke-width in the SVG.
func TestRenderBlockEdgeStyles(t *testing.T) {
	cases := []struct {
		style diagram.LineStyle
		head  diagram.ArrowHead
		want  []string
	}{
		{diagram.LineStyleSolid, diagram.ArrowHeadArrow, []string{"url(#block-arrow)"}},
		{diagram.LineStyleSolid, diagram.ArrowHeadCross, []string{"url(#block-cross)"}},
		{diagram.LineStyleSolid, diagram.ArrowHeadCircle, []string{"url(#block-circle)"}},
		{diagram.LineStyleThick, diagram.ArrowHeadArrow, []string{"stroke-width:3"}},
		{diagram.LineStyleDotted, diagram.ArrowHeadArrow, []string{"stroke-dasharray:4 4"}},
	}
	for _, tc := range cases {
		d := &diagram.BlockDiagram{
			Nodes: []diagram.BlockNode{{ID: "A", Label: "A"}, {ID: "B", Label: "B"}},
			Edges: []diagram.BlockEdge{{From: "A", To: "B", LineStyle: tc.style, ArrowHead: tc.head}},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("style %v head %v: %v", tc.style, tc.head, err)
		}
		raw := string(out)
		for _, want := range tc.want {
			if !strings.Contains(raw, want) {
				t.Errorf("style %v head %v: expected %q in output", tc.style, tc.head, want)
			}
		}
	}
}

// LineStyleInvisible suppresses the edge entirely.
func TestRenderBlockInvisibleEdge(t *testing.T) {
	d := &diagram.BlockDiagram{
		Nodes: []diagram.BlockNode{{ID: "A", Label: "A"}, {ID: "B", Label: "B"}},
		Edges: []diagram.BlockEdge{{From: "A", To: "B", LineStyle: diagram.LineStyleInvisible}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Both nodes still render but neither anchor marker should
	// appear because the edge was suppressed.
	if strings.Contains(string(out), "url(#block-arrow)") {
		t.Errorf("invisible edge should not emit arrow marker")
	}
}
