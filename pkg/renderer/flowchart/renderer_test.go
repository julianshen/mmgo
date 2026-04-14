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
	if rect.X != wantX {
		t.Errorf("rect.X = %f, want %f", rect.X, wantX)
	}
	if rect.Y != wantY {
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
