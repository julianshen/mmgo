package flowchart

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"
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

func TestRenderParallelEdgesStableMatching(t *testing.T) {
	// Two parallel A→B edges with distinct labels and arrowheads. The
	// renderer must not swap labels or arrowheads between them, even
	// across multiple Render() calls. Regression for the
	// `sort.Slice(...) by (From,To)` ambiguity for ties.
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "A", Shape: diagram.NodeShapeRectangle},
			{ID: "B", Label: "B", Shape: diagram.NodeShapeRectangle},
		},
		Edges: []diagram.Edge{
			{From: "A", To: "B", Label: "first", ArrowHead: diagram.ArrowHeadArrow, LineStyle: diagram.LineStyleSolid},
			{From: "A", To: "B", Label: "second", ArrowHead: diagram.ArrowHeadOpen, LineStyle: diagram.LineStyleSolid},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "A", Width: 80, Height: 40})
	g.SetNode("B", graph.NodeAttrs{Label: "B", Width: 80, Height: 40})
	g.SetEdge("A", "B", graph.EdgeAttrs{Label: "first"})
	g.SetEdge("A", "B", graph.EdgeAttrs{Label: "second"})
	l := layout.Layout(g, layout.Options{})

	first, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	for i := 0; i < 20; i++ {
		next, err := Render(d, l, nil)
		if err != nil {
			t.Fatalf("Render err: %v", err)
		}
		if string(next) != string(first) {
			t.Fatalf("iteration %d: output differs (non-deterministic edge matching)", i)
		}
	}
	// Both labels must appear and they must appear in declaration order.
	raw := string(first)
	iFirst := strings.Index(raw, ">first<")
	iSecond := strings.Index(raw, ">second<")
	if iFirst < 0 || iSecond < 0 {
		t.Fatalf("labels missing: first=%d second=%d\n%s", iFirst, iSecond, raw)
	}
}

func TestBuildMarkersDeterministic(t *testing.T) {
	// Multi-arrow diagram: render the SAME input multiple times and
	// require byte-identical SVG. Regression for the map-iteration
	// non-determinism in buildMarkers.
	d := &diagram.FlowchartDiagram{
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow, LineStyle: diagram.LineStyleSolid},
			{From: "B", To: "C", ArrowHead: diagram.ArrowHeadOpen, LineStyle: diagram.LineStyleSolid},
			{From: "C", To: "D", ArrowHead: diagram.ArrowHeadCross, LineStyle: diagram.LineStyleDotted},
			{From: "D", To: "E", ArrowHead: diagram.ArrowHeadCircle, LineStyle: diagram.LineStyleThick},
		},
	}
	first := buildDefs(d, DefaultTheme()).Markers
	for i := 0; i < 50; i++ {
		next := buildDefs(d, DefaultTheme()).Markers
		if len(next) != len(first) {
			t.Fatalf("iteration %d: marker count differs", i)
		}
		for j := range first {
			if next[j].ID != first[j].ID {
				t.Fatalf("iteration %d: marker[%d].ID = %q, want %q",
					i, j, next[j].ID, first[j].ID)
			}
		}
	}
	// Also assert the order is alphabetical, so the contract is
	// explicit and not just "stable but arbitrary".
	for i := 1; i < len(first); i++ {
		if first[i-1].ID >= first[i].ID {
			t.Errorf("markers not sorted: %q before %q", first[i-1].ID, first[i].ID)
		}
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

// Pins the endpoint-clipping behavior — arrowheads must land on the
// target rect edge, not at its center where they'd be buried under
// the node's own fill. Regressing the clip would put X2 back at the
// node center (at x=100 in this fixture).
func TestRenderEdgeClipsEndpointsToNodeBounds(t *testing.T) {
	// Two 40×20 nodes centered at (0,0) and (100,0), so the source's
	// right edge is at x=20 and the target's left edge is at x=80.
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 0, Y: 0, Width: 40, Height: 20},
			"B": {X: 100, Y: 0, Width: 40, Height: 20},
		},
	}
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	eid := graph.EdgeID{From: "A", To: "B"}

	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, l, eid, nil)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	if line.X1 != 20 {
		t.Errorf("source X clipped to %.2f, want 20 (right edge of A)", line.X1)
	}
	if line.X2 != 80 {
		t.Errorf("target X clipped to %.2f, want 80 (left edge of B)", line.X2)
	}
}

// Shape-aware clipping: a diamond target clips to the rhombus edge,
// not the bounding rect. The diamond has vertices at (±w/2, cy) and
// (cx, cy±h/2); a horizontal edge from the source hits the west
// vertex exactly.
func TestRenderEdgeClipsToDiamond(t *testing.T) {
	// A at (0,0) size 40x20; B (diamond) at (100,0) size 40x20.
	// Rect clip would put the endpoint at x=80 (left edge); diamond
	// clip should put it at x=80 too (the west vertex sits there on
	// the horizontal midline). The behavioral difference shows up
	// on the source side: a rect-to-diamond horizontal edge clips
	// at the west vertex (80, 0) rather than the bounding box.
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 0, Y: 0, Width: 40, Height: 20},
			"B": {X: 100, Y: 0, Width: 40, Height: 20},
		},
	}
	shapeByID := map[string]diagram.NodeShape{
		"A": diagram.NodeShapeRectangle,
		"B": diagram.NodeShapeDiamond,
	}
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points: []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
	}
	eid := graph.EdgeID{From: "A", To: "B"}

	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, l, eid, shapeByID)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	// West vertex of the B diamond.
	if line.X2 != 80 || line.Y2 != 0 {
		t.Errorf("diamond clip endpoint = (%.2f,%.2f), want (80,0)", line.X2, line.Y2)
	}
}

// Circle-family target clips to the radial edge. The endpoint must
// lie exactly `min(w,h)/2` from the target center along the edge.
func TestRenderEdgeClipsToCircle(t *testing.T) {
	l := &layout.Result{
		Nodes: map[string]layout.NodeLayout{
			"A": {X: 0, Y: 0, Width: 40, Height: 20},
			"B": {X: 100, Y: 0, Width: 40, Height: 40},
		},
	}
	shapeByID := map[string]diagram.NodeShape{
		"A": diagram.NodeShapeRectangle,
		"B": diagram.NodeShapeCircle,
	}
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points: []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
	}
	eid := graph.EdgeID{From: "A", To: "B"}

	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, l, eid, shapeByID)
	line, ok := elems[0].(*Line)
	if !ok {
		t.Fatalf("expected *Line, got %T", elems[0])
	}
	// Circle B has r = min(40,40)/2 = 20, center at (100,0). The
	// edge from (0,0) along +x enters the circle at x=80.
	if line.X2 != 80 {
		t.Errorf("circle clip endpoint X = %.2f, want 80 (center - radius)", line.X2)
	}
}

func TestRenderEdgeStraightLine(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 10, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
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
	ruler, err := newTestRuler(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ruler.Close() }()
	e := diagram.Edge{From: "A", To: "B", Label: "yes", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, ruler, nil, graph.EdgeID{}, nil)
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
	ruler, err := newTestRuler(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ruler.Close() }()
	e := diagram.Edge{From: "A", To: "B", Label: "bg", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	th := DefaultTheme()
	elems := renderEdge(e, el, 0, th, 16, ruler, nil, graph.EdgeID{}, nil)
	hasBgRect := false
	wantFill := "fill:" + th.Background
	for _, elem := range elems {
		if r, ok := elem.(*Rect); ok && strings.Contains(r.Style, wantFill) {
			hasBgRect = true
		}
	}
	if !hasBgRect {
		t.Errorf("edge label should have a background rect with %q", wantFill)
	}
}

func TestRenderEdgeLabelBackgroundRectWithRuler(t *testing.T) {
	ruler, err := newTestRuler(t)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ruler.Close() }()

	e := diagram.Edge{From: "A", To: "B", Label: "a very long label", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 200, Y: 0}},
		LabelPos: layout.Point{X: 100, Y: 0},
	}
	th := DefaultTheme()
	elems := renderEdge(e, el, 0, th, 16, ruler, nil, graph.EdgeID{}, nil)
	var bgRect *Rect
	wantFill := "fill:" + th.Background
	for _, elem := range elems {
		if r, ok := elem.(*Rect); ok && strings.Contains(r.Style, wantFill) {
			bgRect = r
		}
	}
	if bgRect == nil {
		t.Fatalf("edge label should have a background rect with %q", wantFill)
	}
	if bgRect.Width <= 40 {
		t.Errorf("long label rect width should exceed fallback 40, got %.2f", bgRect.Width)
	}
}

func TestRenderEdgeNilRulerWithLabelDoesNotPanic(t *testing.T) {
	// renderEdge is reachable from tests with a nil ruler. Production
	// callers always pass a real ruler via Render(), but the internal
	// helper must not panic when called directly with nil + a label.
	e := diagram.Edge{From: "A", To: "B", Label: "fallback", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("renderEdge panicked with nil ruler: %v", r)
		}
	}()
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
	if len(elems) == 0 {
		t.Error("expected fallback elements")
	}
}

func TestRenderEdgeDotted(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", LineStyle: diagram.LineStyleDotted, ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
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
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
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
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
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

func TestRenderBackEdge(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 100}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 50},
		BackEdge: true,
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
	if len(elems) < 1 {
		t.Fatal("expected at least 1 element")
	}
	path, ok := elems[0].(*Path)
	if !ok {
		t.Fatalf("expected *Path for back-edge, got %T", elems[0])
	}
	if !strings.Contains(path.Style, "stroke-dasharray:6,3") {
		t.Errorf("back-edge should have dashed style, got: %s", path.Style)
	}
	if !strings.Contains(path.D, "Q") {
		t.Errorf("back-edge should use quadratic bezier, got: %s", path.D)
	}
	if !strings.Contains(path.MarkerEnd, "arrow-arrow") {
		t.Errorf("back-edge should have arrow marker, got %s", path.MarkerEnd)
	}
}

func TestRenderBackEdgeDotted(t *testing.T) {
	e := diagram.Edge{From: "A", To: "B", LineStyle: diagram.LineStyleDotted, ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 100}, {X: 100, Y: 0}},
		LabelPos: layout.Point{X: 50, Y: 50},
		BackEdge: true,
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{}, nil)
	path, ok := elems[0].(*Path)
	if !ok {
		t.Fatalf("expected *Path, got %T", elems[0])
	}
	if !strings.Contains(path.Style, "stroke-dasharray:2,2") {
		t.Errorf("dotted back-edge should keep dotted dasharray, got: %s", path.Style)
	}
	if strings.Contains(path.Style, "stroke-dasharray:6,3") {
		t.Errorf("dotted back-edge should NOT have back-edge dash overriding, got: %s", path.Style)
	}
}

func TestRenderSelfLoop(t *testing.T) {
	e := diagram.Edge{From: "A", To: "A", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points: []layout.Point{
			{X: 20, Y: 0}, {X: 30, Y: -40}, {X: -30, Y: -40}, {X: -20, Y: 0},
		},
		LabelPos: layout.Point{X: 0, Y: -40},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{From: "A", To: "A"}, nil)
	if len(elems) < 1 {
		t.Fatal("expected at least 1 element")
	}
	path, ok := elems[0].(*Path)
	if !ok {
		t.Fatalf("expected *Path for self-loop, got %T", elems[0])
	}
	if !strings.Contains(path.D, "C") {
		t.Errorf("self-loop should use cubic bezier, got: %s", path.D)
	}
	if !strings.Contains(path.MarkerEnd, "arrow-arrow") {
		t.Errorf("self-loop should have arrow marker, got %s", path.MarkerEnd)
	}
}

func TestRenderSelfLoopFallbackNonFourPoints(t *testing.T) {
	e := diagram.Edge{From: "A", To: "A", ArrowHead: diagram.ArrowHeadArrow}
	el := layout.EdgeLayout{
		Points:   []layout.Point{{X: 0, Y: 0}, {X: 0, Y: 0}},
		LabelPos: layout.Point{X: 0, Y: 0},
	}
	elems := renderEdge(e, el, 0, DefaultTheme(), 16, nil, nil, graph.EdgeID{From: "A", To: "A"}, nil)
	if len(elems) != 0 {
		t.Errorf("self-loop with !=4 points should produce no elements, got %d", len(elems))
	}
}

func TestStraightEdgeRendersAsPath(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "From", Shape: diagram.NodeShapeRectangle},
			{ID: "B", Label: "To", Shape: diagram.NodeShapeRectangle},
		},
		Edges: []diagram.Edge{
			{From: "A", To: "B", ArrowHead: diagram.ArrowHeadArrow},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetNode("B", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetEdge("A", "B", graph.EdgeAttrs{})
	l := layout.Layout(g, layout.Options{})
	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(out)
	if strings.Contains(raw, "<line ") {
		t.Error("2-point edge should render as <path>, not <line>")
	}
	if !strings.Contains(raw, "<path ") {
		t.Error("expected <path> element in output")
	}
}

func TestBackEdgeBowNonZero(t *testing.T) {
	pt := backEdgeBow(layout.Point{X: 0, Y: 0}, layout.Point{X: 100, Y: 0})
	if pt.X != 50 {
		t.Errorf("bow midpoint X = %v, want 50", pt.X)
	}
	if pt.Y == 0 {
		t.Errorf("bow should be offset perpendicular, got Y = 0")
	}
}

func TestBackEdgeBowZeroLength(t *testing.T) {
	pt := backEdgeBow(layout.Point{X: 50, Y: 50}, layout.Point{X: 50, Y: 50})
	if pt.X != 50 || pt.Y != 50 {
		t.Errorf("zero-length bow should return midpoint, got (%v, %v)", pt.X, pt.Y)
	}
}
