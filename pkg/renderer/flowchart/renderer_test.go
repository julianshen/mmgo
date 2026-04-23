package flowchart

import (
	"encoding/xml"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/layout"
	"github.com/julianshen/mmgo/pkg/layout/graph"

	fcparser "github.com/julianshen/mmgo/pkg/parser/flowchart"
)

func TestRenderNilInputs(t *testing.T) {
	_, err := Render(nil, &layout.Result{}, nil)
	if err == nil {
		t.Fatal("expected error for nil diagram")
	}
	if !strings.Contains(err.Error(), "diagram is nil") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = Render(&diagram.FlowchartDiagram{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for nil layout")
	}
	if !strings.Contains(err.Error(), "layout is nil") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSingleNode(t *testing.T) {
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Hello", Width: 100, Height: 50})
	l := layout.Layout(g, layout.Options{})
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}},
	}

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.Contains(raw, "<rect") {
		t.Errorf("SVG should contain <rect>:\n%s", raw)
	}
	if !strings.Contains(raw, ">Hello<") {
		t.Errorf("SVG should contain label text:\n%s", raw)
	}
}

func TestRenderRectangleNodeGeometry(t *testing.T) {
	n := diagram.Node{ID: "A", Label: "Hello", Shape: diagram.NodeShapeRectangle}
	nl := layout.NodeLayout{X: 100, Y: 50, Width: 80, Height: 40}
	pad := 10.0

	elems := renderNode(n, nl, pad, DefaultTheme(), 16)
	if len(elems) < 2 {
		t.Fatalf("expected at least 2 elements (rect + text), got %d", len(elems))
	}

	rect, ok := elems[0].(*Rect)
	if !ok {
		t.Fatalf("first element should be *Rect, got %T", elems[0])
	}
	wantX := nl.X - nl.Width/2 + pad
	wantY := nl.Y - nl.Height/2 + pad
	if float64(rect.X) != wantX {
		t.Errorf("rect.X = %f, want %f", rect.X, wantX)
	}
	if float64(rect.Y) != wantY {
		t.Errorf("rect.Y = %f, want %f", rect.Y, wantY)
	}

	txt, ok := elems[1].(*Text)
	if !ok {
		t.Fatalf("second element should be *Text, got %T", elems[1])
	}
	if txt.Content != "Hello" {
		t.Errorf("text.Content = %q, want %q", txt.Content, "Hello")
	}
	if txt.Anchor != "middle" {
		t.Errorf("text-anchor = %q, want middle", txt.Anchor)
	}
}

func TestRenderEmptyDiagramProducesValidSVG(t *testing.T) {
	d := &diagram.FlowchartDiagram{}
	l := layout.Layout(graph.New(), layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	raw := string(svgBytes)
	if !strings.HasPrefix(raw, "<?xml") {
		t.Fatalf("SVG should start with XML declaration, got: %q", raw[:min(len(raw), 60)])
	}

	var svg SVG
	xmlStart := strings.Index(raw, "<svg")
	if xmlStart < 0 {
		t.Fatalf("no <svg> element in output:\n%s", raw)
	}
	if err := xml.Unmarshal([]byte(raw[xmlStart:]), &svg); err != nil {
		t.Fatalf("invalid SVG XML: %v\n%s", err, raw)
	}
	if svg.XMLNS != "http://www.w3.org/2000/svg" {
		t.Errorf("xmlns = %q, want SVG namespace", svg.XMLNS)
	}
	if svg.ViewBox == "" {
		t.Error("viewBox should be set")
	}
}

func TestRenderStyledNode(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Styled", Shape: diagram.NodeShapeRectangle},
		},
		Styles: []diagram.StyleDef{
			{NodeID: "A", CSS: "fill:#ff0000;stroke:#00ff00"},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Styled", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	raw := string(svgBytes)
	if !strings.Contains(raw, "fill:#ff0000") {
		t.Errorf("styled node should have custom fill:\n%s", raw)
	}
}

func TestRenderStyledNodePreservesBaseStyle(t *testing.T) {
	// User `style` overrides must MERGE into the base shape style, not
	// REPLACE it. After applying `fill:#f9f`, the rect should still
	// have its theme stroke and stroke-width.
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Pink", Shape: diagram.NodeShapeRectangle},
		},
		Styles: []diagram.StyleDef{
			{NodeID: "A", CSS: "fill:#f9f"},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Pink", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "fill:#f9f") {
		t.Errorf("override missing:\n%s", raw)
	}
	if !strings.Contains(raw, "stroke:") {
		t.Errorf("base stroke clobbered by override:\n%s", raw)
	}
	if !strings.Contains(raw, "stroke-width:") {
		t.Errorf("base stroke-width clobbered by override:\n%s", raw)
	}
}

func TestRenderMultipleStyleDirectivesAccumulate(t *testing.T) {
	// Mermaid permits multiple `style` directives on the same node;
	// the renderer must concatenate them all (later wins per CSS) and
	// must NOT silently drop subsequent directives.
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Multi", Shape: diagram.NodeShapeRectangle},
		},
		Styles: []diagram.StyleDef{
			{NodeID: "A", CSS: "fill:#aaa"},
			{NodeID: "A", CSS: "stroke-dasharray:4,4"},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Multi", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "fill:#aaa") {
		t.Errorf("first directive missing:\n%s", raw)
	}
	if !strings.Contains(raw, "stroke-dasharray:4,4") {
		t.Errorf("second directive dropped:\n%s", raw)
	}
}

func TestRenderClassNode(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: "Classy", Shape: diagram.NodeShapeRectangle, Classes: []string{"highlight"}},
		},
		Classes: map[string]string{"highlight": "fill:#ffff00"},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Label: "Classy", Width: 80, Height: 40})
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	raw := string(svgBytes)
	if !strings.Contains(raw, `class="highlight"`) {
		t.Errorf("class node should have class attr:\n%s", raw)
	}
	if !strings.Contains(raw, ".highlight") {
		t.Errorf("CSS should include class rule:\n%s", raw)
	}
}

func TestRenderEscapesXMLSpecialCharsInLabels(t *testing.T) {
	// Node, edge, and subgraph labels with `&`, `<`, `>`, `"` and a
	// literal `<script>` tag must be escaped by encoding/xml. Pinned so
	// a future refactor can't accidentally introduce a manual string
	// builder that bypasses the escaper.
	d := &diagram.FlowchartDiagram{
		Nodes: []diagram.Node{
			{ID: "A", Label: `Tom & Jerry <script>x</script>`, Shape: diagram.NodeShapeRectangle},
		},
		Edges: []diagram.Edge{
			{From: "A", To: "B", Label: `if x < y && y > "z"`, ArrowHead: diagram.ArrowHeadArrow},
		},
		Subgraphs: []*diagram.Subgraph{
			{
				ID:    "sg1",
				Label: `Group "<one>"`,
				Nodes: []diagram.Node{{ID: "B", Label: "B", Shape: diagram.NodeShapeRectangle}},
			},
		},
	}
	g := graph.New()
	g.SetNode("A", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetNode("B", graph.NodeAttrs{Width: 80, Height: 40})
	g.SetEdge("A", "B", graph.EdgeAttrs{})
	l := layout.Layout(g, layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)

	// Forbidden: any literal angle-bracket script tag must NOT survive.
	if strings.Contains(raw, "<script>") {
		t.Errorf("literal <script> tag in output (XML escaping bypassed):\n%s", raw)
	}
	// Required: the escaped forms must be present for the special chars.
	for _, want := range []string{"&amp;", "&lt;", "&gt;"} {
		if !strings.Contains(raw, want) {
			t.Errorf("expected escaped sequence %q in output", want)
		}
	}
}

func TestRenderTitleAndDesc(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Title:    "Diagram Title",
		AccDescr: "A description of the diagram",
	}
	l := layout.Layout(graph.New(), layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Diagram Title</title>") {
		t.Errorf("expected <title> element in output:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>A description of the diagram</desc>") {
		t.Errorf("expected <desc> element in output:\n%s", raw)
	}
}

func TestRenderAccTitle(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		AccTitle: "Accessible Title",
	}
	l := layout.Layout(graph.New(), layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Accessible Title</title>") {
		t.Errorf("expected <title> with AccTitle in output:\n%s", raw)
	}
}

func TestRenderTitleAndAccTitleBothSet(t *testing.T) {
	d := &diagram.FlowchartDiagram{
		Title:    "Main Title",
		AccTitle: "Accessibility Title",
	}
	l := layout.Layout(graph.New(), layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if strings.Count(raw, "<title>") != 2 {
		t.Errorf("expected 2 <title> elements when both Title and AccTitle set, got %d:\n%s", strings.Count(raw, "<title>"), raw)
	}
	if !strings.Contains(raw, "<title>Main Title</title>") {
		t.Errorf("expected Title element:\n%s", raw)
	}
	if !strings.Contains(raw, "<title>Accessibility Title</title>") {
		t.Errorf("expected AccTitle element:\n%s", raw)
	}
}

func TestRenderNoTitleOrDescWhenEmpty(t *testing.T) {
	d := &diagram.FlowchartDiagram{}
	l := layout.Layout(graph.New(), layout.Options{})

	out, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render err: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "<title>") {
		t.Errorf("unexpected <title> in output when no title set:\n%s", raw)
	}
	if strings.Contains(raw, "<desc>") {
		t.Errorf("unexpected <desc> in output when no descr set:\n%s", raw)
	}
}

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGoldenSimple(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("testdata", "simple.mmd"))
	if err != nil {
		t.Skip("testdata/simple.mmd not found")
	}

	d, err := fcparser.Parse(strings.NewReader(string(input)))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	g := graph.New()
	for _, n := range d.Nodes {
		g.SetNode(n.ID, graph.NodeAttrs{Label: n.Label, Width: 100, Height: 50})
	}
	for _, e := range d.Edges {
		g.SetEdge(e.From, e.To, graph.EdgeAttrs{Label: e.Label})
	}
	l := layout.Layout(g, layout.Options{})

	svgBytes, err := Render(d, l, nil)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	goldenPath := filepath.Join("testdata", "simple.golden.svg")
	if *updateGolden {
		if err := os.WriteFile(goldenPath, svgBytes, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (run with -update to create)", err)
	}
	if string(svgBytes) != string(golden) {
		t.Errorf("output does not match golden file")
	}
}
