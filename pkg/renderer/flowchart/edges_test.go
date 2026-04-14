package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

func TestBuildMarkers(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow},
			{From: "B", To: "C", ArrowHead: diagram.ArrowHeadOpen},
			{From: "C", To: "D", ArrowHead: diagram.ArrowHeadCross},
			{From: "D", To: "E", ArrowHead: diagram.ArrowHeadCircle},
			{From: "E", To: "F", ArrowHead: diagram.ArrowHeadNone},
		},
	}
	defs := buildDefs(d, DefaultTheme())
	if len(defs.Markers) == 0 {
		t.Fatal("expected markers, got 0")
	}
	ids := map[string]bool{}
	for _, m := range defs.Markers {
		ids[m.ID] = true
	}
	if !ids["arrow-arrow-unknown"] {
		t.Error("expected marker arrow-arrow-unknown")
	}
	if !ids["arrow-open-unknown"] {
		t.Error("expected marker arrow-open-unknown")
	}
	if !ids["arrow-cross-unknown"] {
		t.Error("expected marker arrow-cross-unknown")
	}
	if !ids["arrow-circle-unknown"] {
		t.Error("expected marker arrow-circle-unknown")
	}
	if ids["arrow-none-unknown"] {
		t.Error("should not have marker for ArrowHeadNone")
	}
}

func TestBuildMarkersDotted(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow, LineStyle: diagram.LineStyleDotted},
		},
	}
	defs := buildDefs(d, DefaultTheme())
	if len(defs.Markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(defs.Markers))
	}
	if !strings.Contains(defs.Markers[0].ID, "dotted") {
		t.Errorf("marker ID should contain 'dotted': %s", defs.Markers[0].ID)
	}
}

func TestRenderEdgeStraightLine(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 10, DefaultTheme(), 16, nil)
	if len(elems) < 1 {
		t.Fatal("expected at least 1 element")
	}
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if line.X1 != 10 || line.Y1 != 10 {
		t.Errorf("start = (%.2f,%.2f), want (10,10)", line.X1, line.Y1)
	}
	if line.X2 != 110 || line.Y2 != 10 {
		t.Errorf("end = (%.2f,%.2f), want (110,10)", line.X2, line.Y2)
	}
	if !strings.Contains(line.MarkerEnd, "arrow-arrow") {
		t.Errorf("expected arrow marker, got %s", line.MarkerEnd)
	}
}

func TestRenderEdgeWithLabel(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", Label: "yes", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil)
	hasLabel := false
	for _, elem := range elems {
		if txt, ok := elem.(*Text); ok && txt.Content == "yes" {
			hasLabel = true
		}
	}
	if !hasLabel {
		t.Error("expected text element with label 'yes'")
	}
}

func TestRenderEdgeLabelBackgroundRect(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", Label: "bg", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil)
	hasBgRect := false
	for _, elem := range elems {
		if r, ok := elem.(*Rect); ok && strings.Contains(r.Style, "fill:white") {
			hasBgRect = true
		}
	}
	if !hasBgRect {
		t.Error("edge label should have a white background rect")
	}
}

func TestRenderEdgeLabelBackgroundRectWithRuler(t *testing.T) {
	ruler, err := newTestRuler(t)
	if err != nil {
		t.Fatal(err)
	}
	defer ruler.Close()

	e := diagram.Edge{From: "A", To: "B", Label: "a very long label", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 200, Y: 0}},
		LabelPos: layout.Point{X: 100, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, ruler)
	var bgRect *Rect
	for _, elem := range elems {
		if r, ok := elem.(*Rect); ok && strings.Contains(r.Style, "fill:white") {
			bgRect = r
		}
	}
	if bgRect == nil {
		t.Fatal("edge label should have a white background rect")
	}
	if bgRect.Width <= 40 {
		t.Errorf("long label rect width should exceed fallback 40, got %.2f", bgRect.Width)
	}
}

func TestRenderEdgeDotted(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", LineStyle: diagram.LineStyleDotted, ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if !strings.Contains(line.Style, "stroke-dasharray:2,2") {
		t.Errorf("dotted edge should have dasharray, got: %s", line.Style)
	}
}

func TestRenderEdgeNoMarker(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadNone}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if line.MarkerEnd != "" {
		t.Errorf("no-arrow edge should have empty marker-end, got %s", line.MarkerEnd)
	}
}

func TestRenderEdgeCurve(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 50, Y: 50}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 25},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil)
	path, ok := elems[0].(*Path)
	if !ok {
		t.Fatalf("expected *Path for 3-point edge, got %T", elems[0])
	}
	if !strings.Contains(path.D, " C") {
		t.Errorf("curve path should contain cubic bezier, got: %s", path.D)
	}
}

func newTestRuler(t *testing.T) (*textmeasure.Ruler, error) {
	t.Helper()
	return textmeasure.NewDefaultRuler()
}
